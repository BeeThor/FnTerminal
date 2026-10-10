package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const prefix = "/app/fnterminal"

type peerKey struct{}
type peer struct {
	uid   uint32
	valid bool
}

func main() {
	if len(os.Args) == 3 && os.Args[1] == "--supervise-shell" {
		os.Exit(superviseShell(os.Args[2]))
	}
	socket := flag.String("socket", "", "gateway Unix socket (required)")
	web := flag.String("web", "", "offline frontend directory (required)")
	home := flag.String("home", "", "package home (required)")
	gateway := flag.Uint("gateway-uid", 0, "verified gateway process UID; never the package UID")
	group := flag.Int("socket-gid", -1, "verified gateway group; -1 keeps socket mode 0600")
	idle := flag.Duration("idle-timeout", 15*time.Minute, "timeout without terminal input")
	lifetime := flag.Duration("session-timeout", 8*time.Hour, "maximum terminal lifetime")
	flag.Parse()
	if os.Geteuid() == 0 {
		log.Fatal("FnTerminal must run as a non-root package user")
	}
	if uint64(*gateway) > 1<<32-1 || uint32(*gateway) == uint32(os.Geteuid()) {
		log.Fatal("gateway UID must differ from package UID")
	}
	if *idle <= 0 || *lifetime <= 0 {
		log.Fatal("timeouts must be positive")
	}
	if *socket == "" || *web == "" || *home == "" {
		log.Fatal("--socket, --web and --home are required")
	}
	if err := enableReaper(); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(*home, 0700); err != nil {
		log.Fatal(err)
	}
	if err := os.Chmod(*home, 0700); err != nil {
		log.Fatal(err)
	}
	listener, err := listenSocket(*socket, *group)
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	defer os.Remove(*socket)
	resolvedHome, err := filepath.EvalSymlinks(*home)
	if err != nil {
		log.Fatal(err)
	}
	m := newManager(resolvedHome, *idle, *lifetime)
	m.audit.add("", "service_start", version, "服务以独立应用账户运行")
	server := &http.Server{
		Handler: m.handler(*web, uint32(*gateway)), ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 10 * time.Second, WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 8192,
		ConnContext: func(ctx context.Context, c net.Conn) context.Context {
			p, err := socketPeer(c)
			if err != nil {
				return context.WithValue(ctx, peerKey{}, peer{})
			}
			return context.WithValue(ctx, peerKey{}, peer{p, true})
		},
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	log.Printf("event=service_start package_uid=%d gateway_uid=%d socket=%q", os.Geteuid(), *gateway, *socket)
	select {
	case <-ctx.Done():
	case err := <-served:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("event=serve_error error=%q", err)
		}
	}
	// Hijacked WebSockets are not closed by http.Server.Shutdown.
	m.shutdown()
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(shutdown)
	cleanupAdopted()
	log.Print("event=service_stop")
}

func listenSocket(path string, gid int) (net.Listener, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("socket path must be absolute")
	}
	// Fail closed on writable target directories, including symlink-resolved target.
	st, err := os.Stat(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if err := secureSocketDirectory(filepath.Dir(path), st); err != nil {
		return nil, err
	}
	if st, err := os.Lstat(path); err == nil {
		if st.Mode()&os.ModeSocket == 0 {
			return nil, fmt.Errorf("refuse to unlink non-socket")
		}
		c, err := net.DialTimeout("unix", path, 300*time.Millisecond)
		if err == nil {
			c.Close()
			return nil, fmt.Errorf("another instance is listening")
		}
		if !errors.Is(err, syscall.ECONNREFUSED) {
			return nil, fmt.Errorf("cannot verify stale socket: %w", err)
		}
		if err := os.Remove(path); err != nil {
			return nil, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	// Avoid even a brief world-accessible socket creation window.
	old := syscall.Umask(0077)
	l, err := net.Listen("unix", path)
	syscall.Umask(old)
	if err != nil {
		return nil, err
	}
	mode := os.FileMode(0600)
	if gid >= 0 {
		err = os.Chown(path, -1, gid)
		mode = 0660
	}
	if err == nil {
		err = os.Chmod(path, mode)
	}
	if err != nil {
		l.Close()
		os.Remove(path)
		return nil, err
	}
	return l, nil
}
