package main

import (
	"bytes"
	"encoding/json"
	"github.com/gorilla/websocket"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestTicketCannotBeReplayedOrUsedAcrossOrigin(t *testing.T) {
	_, s := harness(t, time.Minute, time.Hour)
	token := prepareTicket(t, s, "1000", s.URL, "application")
	base := "ws" + strings.TrimPrefix(s.URL, "http") + prefix + "/ws?ticket=" + token
	h := http.Header{"X-Trim-Userid": []string{"1000"}, "X-Trim-Isadmin": []string{"true"}, "Origin": []string{"https://evil.test"}}
	_, r, e := websocket.DefaultDialer.Dial(base, h)
	if e == nil || r.StatusCode != 403 {
		t.Fatal("cross-origin ticket accepted")
	}
	h.Set("Origin", s.URL)
	c, _, e := websocket.DefaultDialer.Dial(base, h)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	_, r, e = websocket.DefaultDialer.Dial(base, h)
	if e == nil || r.StatusCode != 403 {
		t.Fatal("ticket reused")
	}
}
func TestCreateRequiresCsrf(t *testing.T) {
	_, s := harness(t, time.Minute, time.Hour)
	r, _ := http.NewRequest("POST", s.URL+prefix+"/api/connect", strings.NewReader(`{"mode":"application","cols":80,"rows":24}`))
	r.Header.Set("Origin", s.URL)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Trim-Userid", "1000")
	r.Header.Set("X-Trim-Isadmin", "true")
	result, e := http.DefaultClient.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	defer result.Body.Close()
	if result.StatusCode != 403 || result.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("CSRF or CORS failed")
	}
}
func TestStartupErrorIsVisibleAndAudited(t *testing.T) {
	m, s := harness(t, time.Minute, time.Hour)
	token := prepareTicket(t, s, "1000", s.URL, "application")
	m.home = "/definitely-missing-fnterminal-home"
	h := http.Header{"X-Trim-Userid": []string{"1000"}, "X-Trim-Isadmin": []string{"true"}, "Origin": []string{s.URL}}
	c, _, e := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(s.URL, "http")+prefix+"/ws?ticket="+token, h)
	if e != nil {
		t.Fatal(e)
	}
	defer c.Close()
	c.SetReadDeadline(time.Now().Add(3 * time.Second))
	_, raw, e := c.ReadMessage()
	if e != nil {
		t.Fatal(e)
	}
	var frame map[string]string
	json.Unmarshal(raw, &frame)
	if frame["type"] != "error" || frame["code"] != "pty_start" || frame["detail"] == "" {
		t.Fatal(string(raw))
	}
	if len(m.audit.visible("1000")) == 0 {
		t.Fatal("no diagnostic event")
	}
}
func TestDiagnosticsDoNotIncludeTerminalInput(t *testing.T) {
	m, s := harness(t, time.Minute, time.Hour)
	c, _ := dial(t, s)
	secret := "sensitive-not-for-audit"
	c.WriteJSON(frame{Type: "input", Data: secret})
	r, _ := http.NewRequest("GET", s.URL+prefix+"/api/diagnostics", nil)
	r.Header.Set("X-Trim-Userid", "1000")
	r.Header.Set("X-Trim-Isadmin", "true")
	result, e := http.DefaultClient.Do(r)
	if e != nil {
		t.Fatal(e)
	}
	defer result.Body.Close()
	data, _ := io.ReadAll(result.Body)
	if bytes.Contains(data, []byte(secret)) {
		t.Fatal("input leaked into diagnostics")
	}
	m.audit.add("1001", "test_other_uid", "", "")

	for _, event := range m.audit.visible("1000") {
		if event.UID == "1001" {
			t.Fatal("another user event visible")
		}
	}
}
