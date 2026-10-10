package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func identity(r *http.Request, gateway uint32) (string, bool) {
	p, ok := r.Context().Value(peerKey{}).(peer)
	if !ok || !p.valid || p.uid != gateway {
		return "", false
	}
	uids, admins := r.Header.Values("X-Trim-Userid"), r.Header.Values("X-Trim-Isadmin")
	if len(uids) != 1 || len(admins) != 1 || admins[0] != "true" {
		return "", false
	}
	uid := uids[0]
	if uid == "" || len(uid) > 10 {
		return "", false
	}
	for _, c := range uid {
		if c < '0' || c > '9' {
			return "", false
		}
	}
	id, err := strconv.ParseUint(uid, 10, 32)
	return strconv.FormatUint(id, 10), err == nil
}

func sameOrigin(r *http.Request) bool {
	v := r.Header.Values("Origin")
	if len(v) != 1 {
		return false
	}
	u, err := url.Parse(v[0])
	return err == nil && (u.Scheme == "https" || u.Scheme == "http") && u.User == nil && u.Path == "" && u.RawQuery == "" && u.Fragment == "" && strings.EqualFold(u.Host, r.Host)
}

func (m *manager) handler(web string, gateway uint32) http.Handler {
	// Serve only bundled public files; never enumerate a directory or expose logs.
	assets := make(map[string]string)
	// The compiled Uniterm bundle uses hashed filenames. Discover only regular
	// public assets at startup; no arbitrary file server or directory listing.
	_ = filepath.WalkDir(filepath.Join(web, "assets"), func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !entry.Type().IsRegular() {
			return nil
		}
		rel, e := filepath.Rel(filepath.Join(web, "assets"), path)
		if e != nil {
			return nil
		}
		switch filepath.Ext(rel) {
		case ".js", ".css", ".woff2", ".png":
			assets[filepath.ToSlash(rel)] = filepath.Join("assets", rel)
		}
		return nil
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; connect-src 'self'; img-src 'self' data:; font-src 'self'; object-src 'none'; base-uri 'none'; frame-ancestors 'self'; form-action 'none'")
		uid, ok := identity(r, gateway)
		if !ok {
			if r.URL.Path == prefix+"/ws" {
				m.mu.Lock()
				ticket, exists := m.tickets[r.URL.Query().Get("ticket")]
				m.mu.Unlock()
				if exists {
					m.audit.add(ticket.uid, "websocket_rejected", "gateway_auth", "网关未转发有效WebSocket管理员身份或对端UID不符合配置")
				}
			}
			failAPI(w, 403, "gateway_auth", "仅允许通过飞牛网关登录的管理员访问", "")
			return
		}
		switch r.URL.Path {
		case prefix + "/api/bootstrap":
			m.bootstrap(w, r, uid)
			return
		case prefix + "/api/connect":
			m.connect(w, r, uid)
			return
		case prefix + "/api/diagnostics":
			m.diagnostics(w, r, uid)
			return
		case prefix + "/ws":
			if r.Method != "GET" {
				w.WriteHeader(405)
				return
			}
			m.terminal(w, r, uid)
			return
		case prefix + "/lease":
			if r.Method != "POST" {
				w.WriteHeader(405)
				return
			}
			if r.Header.Get("Content-Type") != "application/json" {
				w.WriteHeader(403)
				return
			}
			var req struct {
				ID string `json:"id"`
			}
			d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512))
			d.DisallowUnknownFields()
			if d.Decode(&req) != nil || d.Decode(&struct{}{}) != io.EOF {
				w.WriteHeader(400)
				return
			}
			m.mu.Lock()
			s, exists := m.sessions[req.ID]
			origin, originOK := requestOrigin(r)
			if exists && originOK && origin == s.origin && s.uid == uid && time.Now().UnixNano() < s.lease.Load() {
				s.lease.Store(time.Now().Add(m.leaseTime).UnixNano())
			} else {
				exists = false
			}
			m.mu.Unlock()
			if !exists {
				w.WriteHeader(403)
				return
			}
			w.WriteHeader(204)
			return
		case prefix:
			if r.Method != "GET" && r.Method != "HEAD" {
				w.WriteHeader(405)
				return
			}
			http.Redirect(w, r, prefix+"/", 307)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(405)
			return
		}
		name := ""
		if r.URL.Path == prefix+"/" {
			name = "index.html"
		} else if strings.HasPrefix(r.URL.Path, prefix+"/assets/") {
			name = assets[strings.TrimPrefix(r.URL.Path, prefix+"/assets/")]
		}
		if name == "" {
			http.NotFound(w, r)
			return
		}
		path := filepath.Join(web, name)
		st, err := os.Lstat(path)
		if err != nil || !st.Mode().IsRegular() {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, path)
	})
}
