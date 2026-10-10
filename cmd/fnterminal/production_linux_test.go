//go:build linux

package main

import (
	"context"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// Run in the isolated Linux VM as root only to emulate the trusted gateway.
// The real production binary and every terminal run with UID/GID 700.
func TestLinuxProductionGateway(t *testing.T) {
	binary := os.Getenv("FNTERMINAL_PRODUCTION_BIN")
	if os.Geteuid() != 0 || binary == "" {
		t.Skip("requires isolated root gateway fixture and production binary")
	}
	dir, e := os.MkdirTemp("/tmp", "fnterminal-production-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	os.Chown(dir, 700, 700)
	os.Chmod(dir, 0755)
	home := filepath.Join(dir, "home")
	web := filepath.Join(dir, "web")
	os.Mkdir(home, 0700)
	os.Chown(home, 700, 700)
	os.Mkdir(web, 0755)
	os.WriteFile(filepath.Join(web, "index.html"), []byte("offline frontend"), 0644)
	socket := filepath.Join(dir, "app.sock")
	bin := filepath.Join(dir, "bin")
	os.Mkdir(bin, 0755)
	helper, helperErr := os.ReadFile("/nas-login")
	if helperErr == nil {
		os.WriteFile(filepath.Join(bin, "nas-login"), helper, 0755)
	}
	command := exec.Command(binary, "--socket", socket, "--home", home, "--web", web)
	command.Env = append(os.Environ(), "TRIM_APPDEST="+dir)
	command.SysProcAttr = &syscall.SysProcAttr{Credential: &syscall.Credential{Uid: 700, Gid: 700}}
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if e := command.Start(); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		command.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- command.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			command.Process.Kill()
			t.Error("production service did not stop")
		}
	})
	for i := 0; i < 100; i++ {
		if _, e := os.Stat(socket); e == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	target, _ := url.Parse("http://gateway.internal")
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = &http.Transport{DialContext: func(context.Context, string, string) (net.Conn, error) { return net.Dial("unix", socket) }}
	original := proxy.Director
	proxy.Director = func(r *http.Request) {
		original(r)
		r.Host = "gateway.internal"
		r.Header.Del("X-Trim-Userid")
		r.Header.Del("X-Trim-Isadmin")
		r.Header.Set("X-Trim-Userid", "1000")
		r.Header.Set("X-Trim-Isadmin", "true")
	}
	gateway := httptest.NewServer(proxy)
	defer gateway.Close()
	response, e := http.Get(gateway.URL + prefix + "/")
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := io.ReadAll(response.Body)
	response.Body.Close()
	if response.StatusCode != 200 || string(raw) != "offline frontend" {
		t.Fatal(response.StatusCode, string(raw))
	}
	c, id := dial(t, gateway)
	c.WriteJSON(frame{Type: "input", Data: "printf '\\nREAL_%s\\n' LINUX; id -u\r"})
	output := until(t, c, "REAL_LINUX\r\n")
	if !strings.Contains(output, "700") {
		output += until(t, c, "700\r\n")
	}
	t.Log("real gateway with rewritten Host, Unix credentials and UID700 PTY passed", id)
	result, e := http.Get(gateway.URL + prefix + "/api/diagnostics")
	if e != nil {
		t.Fatal(e)
	}
	defer result.Body.Close()
	var data struct {
		UID           int          `json:"uid"`
		GatewayUID    int          `json:"gatewayUID"`
		ForwardedHost string       `json:"forwardedHost"`
		Events        []auditEvent `json:"events"`
	}
	if json.NewDecoder(result.Body).Decode(&data) != nil || data.UID != 700 || data.GatewayUID != 0 || data.ForwardedHost != "gateway.internal" || len(data.Events) == 0 {
		t.Fatal(data)
	}
	c.Close()
	// Exercise the actual nas-login -> OpenSSH -> independently authenticated user.
	if helperErr == nil && os.Getenv("FNTERMINAL_TEST_SSH") == "1" {
		sshDir := filepath.Join(home, ".ssh")
		os.Mkdir(sshDir, 0700)
		os.Chown(sshDir, 700, 700)
		for _, name := range []string{"id_ed25519", "id_ed25519.pub"} {
			key, err := os.ReadFile("/test-keys/" + name)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(sshDir, name)
			os.WriteFile(path, key, 0600)
			os.Chown(path, 700, 700)
		}
		custom := 10022
		for _, test := range []struct {
			name string
			port *int
			user string
			uid  string
		}{{"default22", nil, "nasdemo", "701"}, {"custom10022", &custom, "nasdemo", "701"}, {"numeric10022", &custom, "10001", "702"}} {
			requested, label := test.port, test.name
			host, username := "", ""
			if requested != nil {
				host = "localhost"
				username = test.user
			}
			ticket := prepareTicketOptions(t, gateway, "1000", gateway.URL, "nas", requested, host, username)
			headers := http.Header{"X-Trim-Userid": []string{"1000"}, "X-Trim-Isadmin": []string{"true"}, "Origin": []string{gateway.URL}}
			nas, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(gateway.URL, "http")+prefix+"/ws?ticket="+ticket, headers)
			if err != nil {
				t.Fatal(err)
			}
			defer nas.Close()
			nas.SetReadDeadline(time.Now().Add(10 * time.Second))
			nas.ReadMessage()
			if requested == nil {
				until(t, nas, "用户名")
				nas.WriteJSON(frame{Type: "input", Data: "nasdemo\r"})
			}
			if label != "numeric10022" {
				until(t, nas, "(yes/no")
				nas.WriteJSON(frame{Type: "input", Data: "yes\r"})
			}
			until(t, nas, "$ ")
			nas.WriteJSON(frame{Type: "input", Data: "printf '\\nNAS_USER_%s\\n' \"$(id -u)\"\r"})
			until(t, nas, "NAS_USER_"+test.uid+"\r\n")
			t.Logf("NAS SSH key authentication and first-use host verification passed as UID%s; port=%v", test.uid, label)
			nas.Close()
		}
	}
}
