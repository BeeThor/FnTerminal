package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

type frame struct {
	Type  string `json:"type"`
	Data  string `json:"data,omitempty"`
	Cols  uint16 `json:"cols,omitempty"`
	Rows  uint16 `json:"rows,omitempty"`
	Bytes int64  `json:"bytes,omitempty"`
}

// Token buckets bound input, output and creation rates without growing queues.
type bucket struct {
	tokens, rate, burst float64
	last                time.Time
}

func newBucket(rate, burst float64) *bucket { return &bucket{burst, rate, burst, time.Now()} }
func (b *bucket) take(n float64) bool {
	now := time.Now()
	b.tokens = min(b.burst, b.tokens+now.Sub(b.last).Seconds()*b.rate)
	b.last = now
	if n > b.tokens {
		return false
	}
	b.tokens -= n
	return true
}

type manager struct {
	mu                        sync.Mutex
	wg                        sync.WaitGroup
	ctx                       context.Context
	cancel                    context.CancelFunc
	stopped                   bool
	count                     int
	users                     map[string]int
	rates                     map[string]*bucket
	global                    *bucket
	sessions                  map[string]*session
	home                      string
	appDir                    string
	audit                     *auditLog
	grants                    map[string]csrfGrant
	tickets                   map[string]connectSpec
	idle, lifetime, leaseTime time.Duration
}
type session struct {
	id, uid                  string
	origin                   string
	lease, activity, pending atomic.Int64
	ack                      chan struct{}
	endReason                atomic.Pointer[string]
}

// Keep the first concrete termination cause across timer/read/write goroutines.
func (s *session) setEndReason(reason string) { s.endReason.CompareAndSwap(nil, &reason) }
func (s *session) endCause() string {
	if reason := s.endReason.Load(); reason != nil {
		return *reason
	}
	return "disconnect"
}

func newManager(home string, idle, life time.Duration) *manager {
	if resolved, e := filepath.EvalSymlinks(home); e == nil {
		home = resolved
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &manager{ctx: ctx, cancel: cancel, home: home, idle: idle, lifetime: life, leaseTime: 60 * time.Second, users: make(map[string]int), rates: make(map[string]*bucket), sessions: make(map[string]*session), global: newBucket(.5, 30), appDir: os.Getenv("TRIM_APPDEST"), audit: &auditLog{}, grants: make(map[string]csrfGrant), tickets: make(map[string]connectSpec)}
}
func (m *manager) reserve(uid string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.stopped || m.count >= 12 || m.users[uid] >= 6 {
		return false
	}
	for id, b := range m.rates {
		if time.Since(b.last) > 2*time.Minute {
			delete(m.rates, id)
		}
	}
	b := m.rates[uid]
	if b == nil {
		if len(m.rates) >= 128 {
			return false
		}
		b = newBucket(1.0/6, 10)
		m.rates[uid] = b
	}
	if !b.take(1) || !m.global.take(1) {
		return false
	}
	m.count++
	m.users[uid]++
	m.wg.Add(1)
	return true
}
func (m *manager) release(uid, id string) {
	m.mu.Lock()
	m.count--
	m.users[uid]--
	if m.users[uid] == 0 {
		delete(m.users, uid)
	}
	delete(m.sessions, id)
	m.mu.Unlock()
	m.wg.Done()
}
func (m *manager) shutdown() {
	m.mu.Lock()
	m.stopped = true
	m.cancel()
	m.mu.Unlock()
	m.wg.Wait()
}

func (m *manager) terminal(w http.ResponseWriter, r *http.Request, uid string) {
	spec, valid := m.consumeTicket(r, uid)
	if !valid {
		m.audit.add(uid, "websocket_rejected", "ticket", "连接凭据缺失、过期或Origin不一致")
		failAPI(w, 403, "ticket", "连接凭据缺失、过期或Origin不一致，请重新连接", "")
		return
	}
	if !m.reserve(uid) {
		failAPI(w, 429, "session_limit", "会话数量或创建频率超过限制", "")
		return
	}
	s := &session{id: spec.ID, uid: uid, origin: spec.origin, ack: make(chan struct{}, 1)}
	defer func() { m.release(uid, s.id) }()
	u := websocket.Upgrader{ReadBufferSize: 4096, WriteBufferSize: 8192, HandshakeTimeout: 5 * time.Second, CheckOrigin: func(req *http.Request) bool { origin, ok := requestOrigin(req); return ok && origin == spec.origin }}
	conn, err := u.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	conn.SetReadLimit(32 * 1024)
	ctx, cancel := context.WithCancel(m.ctx)
	defer cancel()
	var writes sync.Mutex
	write := func(kind int, data []byte) error {
		writes.Lock()
		defer writes.Unlock()
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		return conn.WriteMessage(kind, data)
	}
	reject := func(reason string) {
		s.setEndReason(reason)
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1008, reason), time.Now().Add(time.Second))
	}
	// This also unblocks reads/writes on shutdown and timer expiry.
	var workers sync.WaitGroup
	workers.Add(1)
	go func() {
		defer workers.Done()
		<-ctx.Done()
		if m.ctx.Err() != nil {
			s.setEndReason("service shutdown")
			_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1001, "service shutdown"), time.Now().Add(time.Second))
		}
		conn.Close()
	}()
	defer workers.Wait() // cancel explicitly before waiting below
	defer cancel()
	s.activity.Store(time.Now().UnixNano())
	s.lease.Store(time.Now().Add(m.leaseTime).UnixNano())
	shell, err := shellCommand(spec.Mode)
	if err != nil {
		return
	}
	shell.Dir = m.home
	// Do not pass TRIM_API_TOKEN, installation secrets, loader variables or service configuration to the shell.
	shell.Env = []string{"HOME=" + m.home, "TERM=xterm-256color", "COLORTERM=truecolor", "PATH=" + filepath.Join(os.Getenv("TRIM_APPDEST"), "bin") + ":/usr/local/bin:/usr/bin:/bin", "LANG=C.UTF-8", "LC_ALL=C.UTF-8", "SHELL=/bin/bash", "BASH_SILENCE_DEPRECATION_WARNING=1", "PS1=\\u@\\h:\\w\\$ "}
	readyRead, readyWrite, pipeErr := os.Pipe()
	if pipeErr != nil {
		m.report(uid, "start_failed", "startup_pipe", pipeErr)
		_ = write(websocket.TextMessage, startupMessage("startup_pipe", pipeErr))
		return
	}
	defer readyRead.Close()
	defer readyWrite.Close()
	shell.ExtraFiles = []*os.File{readyWrite}
	if spec.Mode == "nas" {
		port := 22
		if spec.SSHPort != nil {
			port = *spec.SSHPort
		}
		shell.Env = append(shell.Env, "FNTERMINAL_SSH_PORT="+strconv.Itoa(port), "FNTERMINAL_SSH_HOST="+spec.SSHHost, "FNTERMINAL_SSH_USER="+spec.SSHUser)
	}
	shell.Env = append(shell.Env, "FNTERMINAL_LOGIN_HELPER="+filepath.Join(m.appDir, "bin/nas-login"))
	term, err := pty.StartWithSize(shell, &pty.Winsize{Rows: spec.Rows, Cols: spec.Cols})
	readyWrite.Close()
	if err != nil {
		m.report(uid, "start_failed", "pty_start", err)
		_ = write(websocket.TextMessage, startupMessage("pty_start", err))
		log.Printf("event=pty_error uid=%s error=%q", uid, err)
		return
	}
	// PTYs start in blocking mode. Rewrap after enabling nonblocking mode so
	// os.File uses the runtime poller and Close interrupts both Read and Write.
	fd, dupErr := syscall.Dup(int(term.Fd()))
	if dupErr != nil {
		cleanupTerminal(shell.Process.Pid)
		term.Close()
		_ = shell.Wait()
		return
	}
	_ = term.Close()
	syscall.CloseOnExec(fd)
	if err := syscall.SetNonblock(fd, true); err != nil {
		syscall.Close(fd)
		cleanupTerminal(shell.Process.Pid)
		_ = shell.Wait()
		return
	}
	term = os.NewFile(uintptr(fd), "pty")
	// Wait for the actual Bash/SSH child, not merely the supervisor exec.
	_ = readyRead.SetReadDeadline(time.Now().Add(5 * time.Second))
	var startup struct {
		OK    bool   `json:"ok"`
		Stage string `json:"stage"`
		Error string `json:"error"`
	}
	if e := json.NewDecoder(readyRead).Decode(&startup); e != nil || !startup.OK {
		if e == nil {
			e = fmt.Errorf("%s: %s", startup.Stage, startup.Error)
		}
		m.report(uid, "start_failed", "shell_start", e)
		_ = write(websocket.TextMessage, startupMessage("shell_start", e))
		cleanupTerminal(shell.Process.Pid)
		term.Close()
		_ = shell.Wait()
		return
	}
	_ = readyRead.SetReadDeadline(time.Time{})
	exitStatus := make(chan int, 1)
	workers.Add(1)
	go func() {
		defer workers.Done()
		var event struct {
			Type     string `json:"type"`
			ExitCode int    `json:"exitCode"`
		}
		if json.NewDecoder(readyRead).Decode(&event) == nil && event.Type == "exit" {
			exitStatus <- event.ExitCode
		}
		close(exitStatus)
	}()
	// Closing startup/status pipe on cancellation joins the supervisor reader too.
	workers.Add(1)
	go func() { defer workers.Done(); <-ctx.Done(); readyRead.Close() }()
	// Cleanup precedes worker join; closing both descriptors unblocks all pumps.
	defer func() { cancel(); conn.Close(); cleanupTerminal(shell.Process.Pid); term.Close(); _ = shell.Wait() }()
	workers.Add(1)
	go func() { defer workers.Done(); <-ctx.Done(); term.Close() }()
	if ctx.Err() != nil {
		return
	}
	m.mu.Lock()
	m.sessions[s.id] = s
	m.mu.Unlock()
	started := time.Now()
	m.audit.add(uid, "session_ready", spec.Mode, "")
	log.Printf("event=session_start uid=%s session=%s pid=%d peer=%q", uid, s.id, shell.Process.Pid, r.RemoteAddr)
	defer func() { m.audit.add(uid, "session_end", s.endCause(), "") }()
	defer func() {
		log.Printf("event=session_end uid=%s session=%s reason=%q seconds=%.1f", uid, s.id, s.endCause(), time.Since(started).Seconds())
	}()
	ready, _ := json.Marshal(map[string]string{"type": "ready", "id": s.id})
	if write(websocket.TextMessage, ready) != nil {
		return
	}
	workers.Add(1)
	go func() {
		defer workers.Done()
		defer cancel()
		buffer := make([]byte, 8192)
		output := newBucket(1024*1024, 2*1024*1024)
		for {
			for s.pending.Load() >= 128*1024 {
				select {
				case <-s.ack:
				case <-ctx.Done():
					return
				}
			}
			n, e := term.Read(buffer)
			if n > 0 {
				for !output.take(float64(n)) {
					select {
					case <-time.After(10 * time.Millisecond):
					case <-ctx.Done():
						return
					}
				}
				s.pending.Add(int64(n))
				if write(websocket.BinaryMessage, buffer[:n]) != nil {
					return
				}
			}
			if e != nil {
				select {
				case code := <-exitStatus:
					s.setEndReason("terminal exited")
					m.audit.add(uid, "session_exit", fmt.Sprint(code), spec.Mode)
					event, _ := json.Marshal(map[string]any{"type": "exit", "exitCode": code, "mode": spec.Mode})
					_ = write(websocket.TextMessage, event)
					_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1000, "terminal exited"), time.Now().Add(time.Second))
				case <-ctx.Done():
				}
				return
			}
		}
	}()
	_ = conn.SetReadDeadline(time.Now().Add(45 * time.Second))
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(45 * time.Second)) })
	workers.Add(1)
	go func() {
		defer workers.Done()
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-tick.C:
				cause := ""
				if now.Sub(started) >= m.lifetime {
					cause = "session timeout"
				} else if now.Sub(time.Unix(0, s.activity.Load())) >= m.idle {
					cause = "idle timeout"
				} else if now.UnixNano() >= s.lease.Load() {
					cause = "authentication lease expired"
				}
				if cause != "" {
					s.setEndReason(cause)
					_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(1008, cause), now.Add(time.Second))
					cancel()
					return
				}
				if now.Unix()%20 == 0 {
					if conn.WriteControl(websocket.PingMessage, nil, now.Add(time.Second)) != nil {
						cancel()
						return
					}
				}
			}
		}
	}()
	input, events := newBucket(64*1024, 128*1024), newBucket(120, 240)
	for {
		kind, raw, e := conn.ReadMessage()
		if e != nil {
			return
		}
		if !events.take(1) || kind != websocket.TextMessage {
			reject("protocol or rate violation")
			return
		}
		var msg frame
		if json.Unmarshal(raw, &msg) != nil {
			reject("malformed frame")
			return
		}
		switch msg.Type {
		case "input":
			if len(msg.Data) > 16*1024 || !input.take(float64(len(msg.Data))) {
				reject("input limit")
				return
			}
			if len(msg.Data) > 0 {
				s.activity.Store(time.Now().UnixNano())
			}
			// A blocked PTY write is interrupted by cancellation closing the PTY.
			if _, e = io.WriteString(term, msg.Data); e != nil {
				return
			}
		case "resize":
			if msg.Cols < 2 || msg.Cols > 500 || msg.Rows < 2 || msg.Rows > 200 {
				reject("invalid size")
				return
			}
			if pty.Setsize(term, &pty.Winsize{Cols: msg.Cols, Rows: msg.Rows}) != nil {
				return
			}
		case "ack":
			if msg.Bytes <= 0 || msg.Bytes > s.pending.Load() {
				reject("invalid acknowledgment")
				return
			}
			s.pending.Add(-msg.Bytes)
			select {
			case s.ack <- struct{}{}:
			default:
			}
		default:
			reject("unknown frame")
			return
		}
	}
}
