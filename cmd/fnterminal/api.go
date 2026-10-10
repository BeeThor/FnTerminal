package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
)

const version = "0.9.0"

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

func respondJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func failAPI(w http.ResponseWriter, status int, code, message, detail string) {
	respondJSON(w, status, apiError{code, message, detail})
}
func randomID() string {
	var b [24]byte
	if _, e := rand.Read(b[:]); e != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}
func requestOrigin(r *http.Request) (string, bool) {
	origins := r.Header.Values("Origin")
	if len(origins) != 1 {
		return "", false
	}
	u, e := url.Parse(origins[0])
	if e != nil || u.User != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return "", false
	}
	return strings.ToLower(u.Scheme + "://" + u.Host), true
}

type csrfGrant struct {
	uid     string
	expires time.Time
}
type connectSpec struct {
	ID          string `json:"id"`
	Mode        string `json:"mode"`
	Cols        uint16 `json:"cols"`
	Rows        uint16 `json:"rows"`
	SSHPort     *int   `json:"sshPort,omitempty"`
	SSHHost     string `json:"sshHost,omitempty"`
	SSHUser     string `json:"sshUser,omitempty"`
	uid, origin string
	expires     time.Time
}
type auditEvent struct {
	Time   string `json:"time"`
	Event  string `json:"event"`
	UID    string `json:"uid,omitempty"`
	Code   string `json:"code,omitempty"`
	Detail string `json:"detail,omitempty"`
}
type auditLog struct {
	mu     sync.Mutex
	events []auditEvent
}

func (a *auditLog) add(uid, event, code, detail string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(detail) > 512 {
		detail = detail[:512]
	}
	entry := auditEvent{time.Now().UTC().Format(time.RFC3339), event, uid, code, detail}
	a.events = append(a.events, entry)
	if raw, err := json.Marshal(entry); err == nil {
		log.Printf("audit=%s", raw)
	}
	if len(a.events) > 200 {
		a.events = a.events[len(a.events)-200:]
	}
}
func (a *auditLog) visible(uid string) []auditEvent {
	a.mu.Lock()
	defer a.mu.Unlock()
	events := []auditEvent{}
	for _, e := range a.events {
		if e.UID == "" || e.UID == uid {
			events = append(events, e)
		}
	}
	return events
}
func (m *manager) cleanGrants() {
	now := time.Now()
	for id, g := range m.grants {
		if now.After(g.expires) {
			delete(m.grants, id)
		}
	}
	for id, g := range m.tickets {
		if now.After(g.expires) {
			delete(m.tickets, id)
		}
	}
}
func (m *manager) bootstrap(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != "GET" {
		failAPI(w, 405, "method", "仅支持GET", "")
		return
	}
	m.mu.Lock()
	m.cleanGrants()
	if len(m.grants) >= 256 {
		m.mu.Unlock()
		failAPI(w, 429, "busy", "连接准备过于频繁，请稍后重试", "")
		return
	}
	id := randomID()
	if id == "" {
		m.mu.Unlock()
		failAPI(w, 503, "entropy", "系统无法生成连接凭据", "")
		return
	}
	m.grants[id] = csrfGrant{uid, time.Now().Add(2 * time.Minute)}
	m.mu.Unlock()
	_, sshErr := exec.LookPath("ssh")
	respondJSON(w, 200, map[string]any{"version": version, "protocol": 2, "csrf": id, "capabilities": map[string]bool{"application": true, "nasSSH": sshErr == nil}, "limits": map[string]int{"tabs": 6, "idleMinutes": 15}})
}
func (m *manager) connect(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != "POST" {
		failAPI(w, 405, "method", "仅支持POST", "")
		return
	}
	origin, ok := requestOrigin(r)
	if !ok || r.Header.Get("Content-Type") != "application/json" {
		failAPI(w, 403, "origin", "连接请求缺少有效Origin或JSON类型", "")
		return
	}
	tokens := r.Header.Values("X-Fnterminal-Csrf")
	m.mu.Lock()
	m.cleanGrants()
	valid := false
	if len(tokens) == 1 {
		g, exists := m.grants[tokens[0]]
		valid = exists && g.uid == uid
		delete(m.grants, tokens[0])
	}
	m.mu.Unlock()
	if !valid {
		failAPI(w, 403, "csrf", "连接准备凭据已失效，请重试", "")
		return
	}
	var spec connectSpec
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024))
	d.DisallowUnknownFields()
	if d.Decode(&spec) != nil || d.Decode(&struct{}{}) != io.EOF {
		failAPI(w, 400, "invalid_request", "连接参数格式错误", "")
		return
	}
	if spec.Mode != "application" && spec.Mode != "nas" {
		failAPI(w, 400, "mode", "不支持的终端类型", "")
		return
	}
	if spec.Cols < 2 || spec.Cols > 500 || spec.Rows < 2 || spec.Rows > 200 {
		failAPI(w, 400, "size", "终端尺寸无效", "")
		return
	}
	if _, e := os.Stat("/bin/bash"); e != nil {
		failAPI(w, 503, "shell_missing", "系统缺少Bash，无法创建终端", e.Error())
		return
	}
	if spec.Mode == "nas" {
		port := 22
		if spec.SSHPort != nil {
			port = *spec.SSHPort
		}
		if port < 1 || port > 65535 {
			failAPI(w, 400, "invalid_ssh_port", "SSH端口必须是1到65535的整数", "")
			return
		}
		spec.SSHPort = &port
		host, hostOK := validSSHHost(spec.SSHHost)
		if !hostOK {
			failAPI(w, 400, "invalid_ssh_host", "SSH主机必须是有效IP或域名，不包含协议、路径或用户名", "")
			return
		}
		spec.SSHHost = host
		if spec.SSHUser != "" && !sshUserPattern.MatchString(spec.SSHUser) {
			failAPI(w, 400, "invalid_ssh_user", "SSH用户名格式无效", "")
			return
		}
		endpoint := net.JoinHostPort(host, fmt.Sprint(port))
		if _, e := exec.LookPath("ssh"); e != nil {
			failAPI(w, 503, "ssh_client_missing", "NAS未提供SSH客户端", "")
			return
		}
		if st, e := os.Stat(filepath.Join(m.appDir, "bin/nas-login")); e != nil || st.Mode().Perm()&0111 == 0 {
			failAPI(w, 503, "ssh_helper_missing", "NAS登录组件缺失，请重新安装完整应用", "")
			return
		}
		c, e := net.DialTimeout("tcp", endpoint, 2*time.Second)
		if e != nil {
			m.audit.add(uid, "connect_failed", "ssh_unavailable", endpoint+": "+e.Error())
			failAPI(w, 503, "ssh_unavailable", fmt.Sprintf("无法连接 SSH %s", endpoint), fmt.Sprintf("请核对目标 %s 的地址、SSH监听端口和网络连通性。端口应从运行此应用的NAS视角可达；应用不会自动开启SSH。", endpoint))
			return
		}
		c.Close()
	}
	spec.ID = randomID()
	if spec.ID == "" {
		failAPI(w, 503, "entropy", "系统无法生成连接凭据", "")
		return
	}
	spec.uid = uid
	spec.origin = origin
	spec.expires = time.Now().Add(15 * time.Second)
	m.mu.Lock()
	pending := 0
	for _, ticket := range m.tickets {
		if ticket.uid == uid {
			pending++
		}
	}
	if m.stopped || m.count >= 12 || m.users[uid]+pending >= 6 || len(m.tickets) >= 32 {
		m.mu.Unlock()
		failAPI(w, 429, "session_limit", "会话数量已达上限，请关闭其它终端", "")
		return
	}
	m.tickets[spec.ID] = spec
	m.mu.Unlock()
	detail := ""
	if spec.Mode == "nas" {
		detail = net.JoinHostPort(spec.SSHHost, fmt.Sprint(*spec.SSHPort))
	}
	m.audit.add(uid, "connect_prepared", spec.Mode, detail)
	respondJSON(w, 201, map[string]any{"ticket": spec.ID, "expiresIn": 15})
}
func (m *manager) consumeTicket(r *http.Request, uid string) (connectSpec, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanGrants()
	values := r.URL.Query()["ticket"]
	if len(values) != 1 {
		return connectSpec{}, false
	}
	spec, exists := m.tickets[values[0]]
	origin, ok := requestOrigin(r)
	if !exists || !ok || spec.uid != uid || spec.origin != origin {
		return connectSpec{}, false
	}
	delete(m.tickets, values[0])
	return spec, true
}
func (m *manager) diagnostics(w http.ResponseWriter, r *http.Request, uid string) {
	if r.Method != "GET" {
		failAPI(w, 405, "method", "仅支持GET", "")
		return
	}
	name := "unknown"
	if u, e := user.Current(); e == nil {
		name = u.Username
	}
	peer, _ := r.Context().Value(peerKey{}).(peer)
	m.mu.Lock()
	count, own := m.count, m.users[uid]
	m.mu.Unlock()
	respondJSON(w, 200, map[string]any{"version": version, "runtime": runtime.GOOS + "/" + runtime.GOARCH, "account": name, "uid": os.Geteuid(), "gatewayUID": peer.uid, "forwardedHost": r.Host, "activeSessions": own, "totalSessions": count, "events": m.audit.visible(uid), "note": "日志仅包含连接阶段与错误元信息，不记录命令、密码或终端输出。"})
}
func (m *manager) report(uid, stage, code string, err error) {
	detail := ""
	if err != nil {
		detail = err.Error()
	}
	m.audit.add(uid, stage, code, detail)
}
func startupMessage(stage string, err error) []byte {
	raw, _ := json.Marshal(map[string]string{"type": "error", "code": stage, "message": "应用终端启动失败", "detail": fmt.Sprint(err)})
	return raw
}

var sshUserPattern = regexp.MustCompile(`^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,63}$`)
var sshHostPattern = regexp.MustCompile(`^[a-zA-Z0-9](?:[a-zA-Z0-9.-]{0,251}[a-zA-Z0-9])?$`)

func validSSHHost(raw string) (string, bool) {
	host := strings.TrimSpace(raw)
	if host == "" {
		return "127.0.0.1", true
	}
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		host = host[1 : len(host)-1]
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), true
	}
	if !sshHostPattern.MatchString(host) || strings.Contains(host, "..") {
		return "", false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", false
		}
	}
	return strings.ToLower(host), true
}
