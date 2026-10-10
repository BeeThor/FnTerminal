//go:build linux

package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLinuxSocketPeer(t *testing.T) {
	dir, e := os.MkdirTemp("/tmp", "ft-peer-")
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(dir)
	l, e := net.Listen("unix", filepath.Join(dir, "app.sock"))
	if e != nil {
		t.Fatal(e)
	}
	defer l.Close()
	c, e := net.Dial("unix", filepath.Join(dir, "app.sock"))
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	server, e := l.Accept()
	if e != nil {
		t.Fatal(e)
	}
	defer server.Close()
	uid, e := socketPeer(server)
	if e != nil || uid != uint32(os.Getuid()) {
		t.Fatal(uid, e)
	}
}
func TestLinuxDetachedCleanupAndSessionIsolation(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root execution is refused by the production supervisor")
	}
	m, s := harness(t, time.Minute, time.Hour)
	c, _ := dial(t, s)
	other, _ := dial(t, s)
	marker := filepath.Join(t.TempDir(), "detached.pid")
	c.WriteJSON(frame{Type: "input", Data: fmt.Sprintf("setsid sh -c 'echo $$ > %s; exec sleep 120' >/dev/null 2>&1 &\r", marker)})
	var pid int
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		data, _ := os.ReadFile(marker)
		pid, _ = strconv.Atoi(strings.TrimSpace(string(data)))
		if pid > 0 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid == 0 {
		t.Fatal("detached process did not start")
	}
	c.Close()
	deadline = time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, e := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(e) {
			pid = 0
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if pid != 0 {
		t.Fatalf("detached process still exists: %d", pid)
	}
	other.WriteJSON(frame{Type: "input", Data: "printf '\\nISOLATION_%s\\n' OK\r"})
	until(t, other, "ISOLATION_OK\r\n")
	m.shutdown()
}

func TestLinuxPrivateSupervisorFD(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("non-root PTY required")
	}
	_, server := harness(t, time.Minute, time.Hour)
	conn, _ := dial(t, server)
	conn.WriteJSON(frame{Type: "input", Data: "if [ -e /proc/$$/fd/3 ]; then printf 'PRIVATE_FD_%s\\n' LEAK; else printf 'PRIVATE_FD_%s\\n' CLOSED; fi\r"})
	until(t, conn, "PRIVATE_FD_CLOSED\r\n")
}
