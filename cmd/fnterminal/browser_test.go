package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// Opt-in local QA fixture compiled only into a Go test binary. The production
// service cannot inject identities or listen on TCP.
func TestBrowserHarness(t *testing.T) {
	if os.Getenv("FNTERMINAL_BROWSER_TEST") != "1" {
		t.Skip("opt-in browser fixture")
	}
	m := newManager(t.TempDir(), 15*time.Minute, 8*time.Hour)
	web, _ := os.Getwd()
	web += "/.build/web"
	h := m.handler(web, 0)
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Header.Set("X-Trim-Userid", "1000")
		r.Header.Set("X-Trim-Isadmin", "true")
		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), peerKey{}, peer{0, true})))
	}))
	defer s.Close()
	defer m.shutdown()
	fmt.Println("BROWSER_FIXTURE=" + s.URL + prefix + "/")
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
}
