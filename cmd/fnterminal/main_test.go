package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 3 && os.Args[1] == "--supervise-shell" {
		os.Exit(superviseShell(os.Args[2]))
	}
	os.Exit(m.Run())
}

func request(path, uid, admin string, p peer) *http.Request {
	r := httptest.NewRequest("GET", "http://nas.local"+path, nil)
	r.Header.Set("X-Trim-Userid", uid)
	r.Header.Set("X-Trim-Isadmin", admin)
	return r.WithContext(context.WithValue(r.Context(), peerKey{}, p))
}
func TestAuthorization(t *testing.T) {
	m := newManager(t.TempDir(), time.Minute, time.Hour)
	defer m.shutdown()
	for _, tc := range []struct {
		name, uid, admin string
		p                peer
		code             int
	}{
		{"admin", "1000", "true", peer{0, true}, 307},
		{"ordinary", "1000", "false", peer{0, true}, 403},
		{"missing_uid", "", "true", peer{0, true}, 403},
		{"malformed_uid", "+1000", "true", peer{0, true}, 403},
		{"overflow_uid", "4294967296", "true", peer{0, true}, 403},
		{"strict_admin", "1000", "TRUE", peer{0, true}, 403},
		{"spoof_package_peer", "1000", "true", peer{1000, true}, 403},
		{"no_credentials", "1000", "true", peer{}, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			m.handler(t.TempDir(), 0).ServeHTTP(w, request(prefix, tc.uid, tc.admin, tc.p))
			if w.Code != tc.code {
				t.Fatalf("status %d want %d", w.Code, tc.code)
			}
		})
	}
	r := request(prefix, "1000", "true", peer{0, true})
	r.Header.Add("X-Trim-Isadmin", "false")
	if _, ok := identity(r, 0); ok {
		t.Fatal("duplicate admin headers accepted")
	}
}
func TestOrigin(t *testing.T) {
	for _, o := range []string{"https://evil.test", "null", "", "http://nas.local/", "http://user@nas.local", "http://nas.local?x=1"} {
		r := request(prefix, "1000", "true", peer{})
		r.Header.Set("Origin", o)
		if sameOrigin(r) {
			t.Fatalf("accepted %q", o)
		}
	}
	r := request(prefix, "1000", "true", peer{})
	r.Header.Set("Origin", "https://nas.local")
	if !sameOrigin(r) {
		t.Fatal("same host rejected")
	}
}
func TestStaticAllowlist(t *testing.T) {
	web := t.TempDir()
	os.WriteFile(filepath.Join(web, "index.html"), []byte("hello"), 0600)
	os.WriteFile(filepath.Join(web, "secret"), []byte("SECRET"), 0600)
	m := newManager(t.TempDir(), time.Minute, time.Hour)
	defer m.shutdown()
	for _, p := range []string{prefix + "/assets/secret", prefix + "/assets/../secret", prefix + "/assets/", "/main.js"} {
		w := httptest.NewRecorder()
		m.handler(web, 0).ServeHTTP(w, request(p, "1000", "true", peer{0, true}))
		if w.Code != 404 {
			t.Fatalf("%s: %d", p, w.Code)
		}
	}
	w := httptest.NewRecorder()
	m.handler(web, 0).ServeHTTP(w, request(prefix+"/", "1000", "true", peer{0, true}))
	if w.Code != 200 || w.Body.String() != "hello" {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestConcurrentReservation(t *testing.T) {
	m := newManager(t.TempDir(), time.Minute, time.Hour)
	var wg sync.WaitGroup
	var mu sync.Mutex
	opened := 0
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if m.reserve("1000") {
				mu.Lock()
				opened++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	if opened != 6 {
		t.Fatalf("opened %d", opened)
	}
	for i := 0; i < opened; i++ {
		m.release("1000", "")
	}
	m.shutdown()
	if m.reserve("1000") {
		t.Fatal("accepted during shutdown")
	}
}
func TestGlobalAndCreationLimits(t *testing.T) {
	m := newManager(t.TempDir(), time.Minute, time.Hour)
	for i := 0; i < 12; i++ {
		if !m.reserve(fmt.Sprint(i)) {
			t.Fatal("reservation rejected")
		}
	}
	if m.reserve("999") {
		t.Fatal("global limit bypassed")
	}
	for i := 0; i < 12; i++ {
		m.release(fmt.Sprint(i), "")
	}
	for i := 0; i < 10; i++ {
		if !m.reserve("same") {
			t.Fatal("premature rate limit")
		}
		m.release("same", "")
	}
	if m.reserve("same") {
		t.Fatal("creation rate bypassed")
	}
	m.shutdown()
}
func TestSocketSafety(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.sock")
	os.WriteFile(path, []byte("important"), 0600)
	if _, e := listenSocket(path, -1); e == nil {
		t.Fatal("replaced regular file")
	}
	os.Remove(path)
	l, e := listenSocket(path, -1)
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	st, _ := os.Stat(path)
	if st.Mode().Perm() != 0600 {
		t.Fatal(st.Mode())
	}
	if _, e := listenSocket(path, -1); e == nil {
		t.Fatal("replaced live socket")
	}
	other := t.TempDir()
	os.Chmod(other, 0777)
	if _, e := listenSocket(filepath.Join(other, "app.sock"), -1); e == nil {
		t.Fatal("unsafe parent accepted")
	}
}

// Integration harness simulates only the verified gateway context; production
// has no HTTP/TCP listener and no option to disable peer authentication.
func harness(t *testing.T, idle, life time.Duration) (*manager, *httptest.Server) {
	t.Helper()
	m := newManager(t.TempDir(), idle, life)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), peerKey{}, peer{0, true}))
		m.handler("../../web", 0).ServeHTTP(w, r)
	}))
	t.Cleanup(func() { m.shutdown(); s.Close() })
	return m, s
}
func dial(t *testing.T, s *httptest.Server) (*websocket.Conn, string) {
	t.Helper()
	h := http.Header{"X-Trim-Userid": []string{"1000"}, "X-Trim-Isadmin": []string{"true"}, "Origin": []string{s.URL}}
	token := prepareTicket(t, s, uidHeader(h), s.URL, "application")
	c, _, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+prefix+"/ws?ticket="+token, h)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { c.Close() })
	c.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, raw, e := c.ReadMessage()
	if e != nil {
		t.Fatal(e)
	}
	var f struct{ Type, ID string }
	json.Unmarshal(raw, &f)
	if f.Type != "ready" || f.ID == "" {
		t.Fatal(string(raw))
	}
	return c, f.ID
}
func until(t *testing.T, c *websocket.Conn, needle string) string {
	t.Helper()
	all := ""
	for i := 0; i < 100; i++ {
		kind, raw, e := c.ReadMessage()
		if e != nil {
			t.Fatalf("read %v, output %q", e, all)
		}
		if kind == websocket.BinaryMessage {
			c.WriteJSON(frame{Type: "ack", Bytes: int64(len(raw))})
			all += string(raw)
		}
		if strings.Contains(all, needle) {
			return all
		}
	}
	t.Fatal("no expected output", needle)
	return all
}
func TestPTYWebSocket(t *testing.T) {
	_, s := harness(t, time.Minute, time.Hour)
	c, _ := dial(t, s)
	c.WriteJSON(frame{Type: "resize", Cols: 91, Rows: 31})
	c.WriteJSON(frame{Type: "input", Data: "printf '\\nREADY_%s\\n' 终端; stty size; printf 'TOKEN_%s_END\\n' \"${TRIM_API_TOKEN-unset}\"\r"})
	out := until(t, c, "TOKEN_unset_END\r\n")
	if !strings.Contains(out, "READY_终端") || !strings.Contains(out, "31 91") {
		t.Fatal(out)
	}
	c.WriteJSON(frame{Type: "input", Data: "sleep 30\r"})
	time.Sleep(50 * time.Millisecond)
	c.WriteJSON(frame{Type: "input", Data: "\x03"})
	c.WriteJSON(frame{Type: "input", Data: "printf '\\nCTRL_%s\\n' OK\r"})
	until(t, c, "CTRL_OK\r\n")
	c.WriteJSON(frame{Type: "input", Data: "exit\r"})
	for {
		_, _, e := c.ReadMessage()
		if e != nil {
			break
		}
	}
}
func TestLeaseBoundToUserAndOrigin(t *testing.T) {
	m, s := harness(t, time.Minute, time.Hour)
	c, id := dial(t, s)
	defer c.Close()
	renew := func(uid, origin, body string) int {
		r, _ := http.NewRequest("POST", s.URL+prefix+"/lease", strings.NewReader(body))
		r.Header.Set("X-Trim-Userid", uid)
		r.Header.Set("X-Trim-Isadmin", "true")
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		res, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		io.Copy(io.Discard, res.Body)
		res.Body.Close()
		return res.StatusCode
	}
	body := fmt.Sprintf(`{"id":%q}`, id)
	if code := renew("1001", s.URL, body); code != 403 {
		t.Fatal(code)
	}
	if code := renew("1000", "https://evil.test", body); code != 403 {
		t.Fatal(code)
	}
	if code := renew("1000", s.URL, body); code != 204 {
		t.Fatal(code)
	}
	if code := renew("1000", s.URL, body+` {}`); code != 400 {
		t.Fatal(code)
	}
	m.mu.Lock()
	m.sessions[id].lease.Store(time.Now().Add(-time.Second).UnixNano())
	m.mu.Unlock()
	if code := renew("1000", s.URL, body); code != 403 {
		t.Fatal("expired lease resurrected", code)
	}
}
func TestTimeouts(t *testing.T) {
	for _, tc := range []struct {
		name              string
		idle, life, lease time.Duration
	}{{"idle", 100 * time.Millisecond, time.Minute, time.Minute}, {"lifetime", time.Minute, 100 * time.Millisecond, time.Minute}, {"lease", time.Minute, time.Minute, 100 * time.Millisecond}} {
		t.Run(tc.name, func(t *testing.T) {
			m, s := harness(t, tc.idle, tc.life)
			m.leaseTime = tc.lease
			c, _ := dial(t, s)
			for {
				_, _, e := c.ReadMessage()
				if e != nil {
					if !websocket.IsCloseError(e, 1008) {
						t.Fatal(e)
					}
					expected := map[string]string{"idle": "idle timeout", "lifetime": "session timeout", "lease": "authentication lease expired"}[tc.name]
					if e.(*websocket.CloseError).Text != expected {
						t.Fatal("wrong close reason", e)
					}
					deadline := time.Now().Add(time.Second)
					found := false
					for time.Now().Before(deadline) {
						for _, event := range m.audit.visible("1000") {
							if event.Event == "session_end" && event.Code == expected {
								found = true
							}
						}
						if found {
							break
						}
						time.Sleep(10 * time.Millisecond)
					}
					if !found {
						t.Fatal("timeout cause missing from audit", m.audit.visible("1000"))
					}
					break
				}
			}
		})
	}
}
func TestShutdownAndProtocol(t *testing.T) {
	for _, mode := range []string{"shutdown", "invalid_json", "invalid_ack", "invalid_resize", "oversized"} {
		t.Run(mode, func(t *testing.T) {
			m, s := harness(t, time.Minute, time.Hour)
			c, _ := dial(t, s)
			switch mode {
			case "shutdown":
				done := make(chan struct{})
				go func() { m.shutdown(); close(done) }()
				select {
				case <-done:
				case <-time.After(3 * time.Second):
					t.Fatal("shutdown blocked")
				}
			case "invalid_json":
				c.WriteMessage(websocket.TextMessage, []byte("{"))
			case "invalid_ack":
				c.WriteJSON(frame{Type: "ack", Bytes: 999999})
			case "invalid_resize":
				c.WriteJSON(frame{Type: "resize", Cols: 65535, Rows: 2})
			case "oversized":
				c.WriteJSON(frame{Type: "input", Data: strings.Repeat("x", 17000)})
			}
			for {
				_, _, e := c.ReadMessage()
				if e != nil {
					if mode != "shutdown" && !websocket.IsCloseError(e, 1008) {
						t.Fatal("policy violation must close with explicit reason", e)
					}
					break
				}
			}
		})
	}
}
func TestWebSocketRejection(t *testing.T) {
	_, s := harness(t, time.Minute, time.Hour)
	for _, h := range []http.Header{{"Origin": []string{s.URL}}, {"X-Trim-Userid": []string{"1000"}, "X-Trim-Isadmin": []string{"false"}, "Origin": []string{s.URL}}, {"X-Trim-Userid": []string{"1000"}, "X-Trim-Isadmin": []string{"true"}, "Origin": []string{"https://evil.test"}}} {
		c, res, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+prefix+"/ws", h)
		if c != nil {
			c.Close()
			t.Fatal("unauthorized upgraded")
		}
		if e == nil || res == nil || res.StatusCode != 403 {
			t.Fatal(e, res)
		}
	}
}
func TestBucket(t *testing.T) {
	b := newBucket(0, 10)
	if !b.take(10) || b.take(1) {
		t.Fatal("burst broken")
	}
}
func TestPeerRejectTCP(t *testing.T) {
	a, b := net.Pipe()
	defer a.Close()
	defer b.Close()
	if _, e := socketPeer(a); e == nil {
		t.Fatal("trusted TCP")
	}
}

func TestBackpressureShutdown(t *testing.T) {
	m, s := harness(t, time.Minute, time.Hour)
	c, _ := dial(t, s)
	c.WriteJSON(frame{Type: "input", Data: "yes BACKPRESSURE\r"})
	// Deliberately never acknowledge output; terminal output stops at the window.
	time.Sleep(150 * time.Millisecond)
	done := make(chan struct{})
	go func() { m.shutdown(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown blocked by output backpressure")
	}
}

func TestDedicatedPackageGroup(t *testing.T) {
	passwd := "root:x:0:0::/root:/bin/bash\nfnterminal:x:700:700::/home/fnterminal:/bin/bash\n"
	if err := dedicatedPackageGroup(700, 700, "fnterminal:x:700:fnterminal,root\n", passwd); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ group, passwd string }{
		{"fnterminal:x:700:fnterminal,guest\n", passwd},
		{"fnterminal:x:700:fnterminal\n", passwd + "guest:x:701:700::/:/bin/bash\n"},
		{"other:x:701:fnterminal\n", passwd},
		{"fnterminal:x:700:fnterminal\n", "root:x:0:0::/root:/bin/bash\n"},
	} {
		if err := dedicatedPackageGroup(700, 700, tc.group, tc.passwd); err == nil {
			t.Fatal("unsafe package group accepted")
		}
	}
}

func uidHeader(h http.Header) string { return h.Get("X-Trim-Userid") }
func prepareTicket(t *testing.T, s *httptest.Server, uid, origin, mode string) string {
	return prepareTicketPort(t, s, uid, origin, mode, nil)
}
func prepareTicketPort(t *testing.T, s *httptest.Server, uid, origin, mode string, port *int) string {
	return prepareTicketOptions(t, s, uid, origin, mode, port, "", "")
}
func prepareTicketOptions(t *testing.T, s *httptest.Server, uid, origin, mode string, port *int, host, username string) string {
	t.Helper()
	request := func(method, path, body, csrf string) *http.Response {
		r, _ := http.NewRequest(method, s.URL+prefix+path, strings.NewReader(body))
		r.Header.Set("X-Trim-Userid", uid)
		r.Header.Set("X-Trim-Isadmin", "true")
		r.Header.Set("Origin", origin)
		r.Header.Set("Content-Type", "application/json")
		if csrf != "" {
			r.Header.Set("X-Fnterminal-Csrf", csrf)
		}
		resp, e := http.DefaultClient.Do(r)
		if e != nil {
			t.Fatal(e)
		}
		return resp
	}
	b := request("GET", "/api/bootstrap", "", "")
	defer b.Body.Close()
	var bootstrap struct {
		CSRF string `json:"csrf"`
	}
	if json.NewDecoder(b.Body).Decode(&bootstrap) != nil || bootstrap.CSRF == "" {
		t.Fatal("bootstrap failed", b.StatusCode)
	}
	payload := map[string]any{"mode": mode, "cols": 91, "rows": 31}
	if host != "" {
		payload["sshHost"] = host
	}
	if username != "" {
		payload["sshUser"] = username
	}
	if port != nil {
		payload["sshPort"] = *port
	}
	raw, _ := json.Marshal(payload)
	r := request("POST", "/api/connect", string(raw), bootstrap.CSRF)
	defer r.Body.Close()
	var spec struct {
		Ticket string `json:"ticket"`
	}
	if json.NewDecoder(r.Body).Decode(&spec) != nil || r.StatusCode != 201 || spec.Ticket == "" {
		t.Fatal("prepare failed", r.StatusCode)
	}
	return spec.Ticket
}
