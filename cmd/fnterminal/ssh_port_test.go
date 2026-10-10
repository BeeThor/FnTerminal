package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSSHPortValidation(t *testing.T) {
	_, s := harness(t, time.Minute, time.Hour)
	for _, port := range []string{"0", "-1", "65536", "1.5", `"10022"`, `"-oProxyCommand=bad"`} {
		t.Run(port, func(t *testing.T) {
			request := func(method, path, body, csrf string) *http.Response {
				r, _ := http.NewRequest(method, s.URL+prefix+path, strings.NewReader(body))
				r.Header.Set("X-Trim-Userid", "1000")
				r.Header.Set("X-Trim-Isadmin", "true")
				r.Header.Set("Origin", s.URL)
				r.Header.Set("Content-Type", "application/json")
				if csrf != "" {
					r.Header.Set("X-Fnterminal-Csrf", csrf)
				}
				response, e := http.DefaultClient.Do(r)
				if e != nil {
					t.Fatal(e)
				}
				return response
			}
			bootstrap := request("GET", "/api/bootstrap", "", "")
			var data struct {
				CSRF string `json:"csrf"`
			}
			json.NewDecoder(bootstrap.Body).Decode(&data)
			bootstrap.Body.Close()
			response := request("POST", "/api/connect", fmt.Sprintf(`{"mode":"nas","cols":80,"rows":24,"sshPort":%s}`, port), data.CSRF)
			defer response.Body.Close()
			if response.StatusCode != 400 {
				t.Fatal("invalid port accepted", response.StatusCode)
			}
		})
	}
}

func TestSSHHostValidation(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "localhost", "nas.example", "::1", "[::1]"} {
		if _, ok := validSSHHost(host); !ok {
			t.Fatal(host)
		}
	}
	for _, host := range []string{"https://nas", "-oProxyCommand=bad", "user@host", "host/path", "host;command", "host..example"} {
		if _, ok := validSSHHost(host); ok {
			t.Fatal("invalid host accepted", host)
		}
	}
}

func TestSSHUserValidation(t *testing.T) {
	for _, user := range []string{"10001", "nasdemo", "user-name", "_service"} {
		if !sshUserPattern.MatchString(user) {
			t.Fatal(user)
		}
	}
	for _, user := range []string{"-oProxyCommand=bad", "user;id", "user@host", "user name"} {
		if sshUserPattern.MatchString(user) {
			t.Fatal("invalid username", user)
		}
	}
}
