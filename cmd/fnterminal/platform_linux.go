//go:build linux

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func socketPeer(c net.Conn) (uint32, error) {
	u, ok := c.(*net.UnixConn)
	if !ok {
		return 0, fmt.Errorf("not a Unix connection")
	}
	raw, err := u.SyscallConn()
	if err != nil {
		return 0, err
	}
	var cred *syscall.Ucred
	var inner error
	err = raw.Control(func(fd uintptr) {
		cred, inner = syscall.GetsockoptUcred(int(fd), syscall.SOL_SOCKET, syscall.SO_PEERCRED)
	})
	if err != nil {
		return 0, err
	}
	if inner != nil {
		return 0, inner
	}
	return cred.Uid, nil
}
func enableReaper() error {
	_, _, e := syscall.Syscall6(syscall.SYS_PRCTL, 36, 1, 0, 0, 0, 0) // PR_SET_CHILD_SUBREAPER
	if e != 0 {
		return e
	}
	return nil
}

type process struct {
	pid, ppid, sid int
	start          string
	state          string
}

func processes() []process {
	entries, _ := os.ReadDir("/proc")
	var out []process
	for _, entry := range entries {
		pid, e := strconv.Atoi(entry.Name())
		if e != nil {
			continue
		}
		data, e := os.ReadFile("/proc/" + entry.Name() + "/stat")
		if e != nil {
			continue
		}
		// comm may itself contain spaces and parentheses.
		i := strings.LastIndexByte(string(data), ')')
		if i < 0 {
			continue
		}
		f := strings.Fields(string(data)[i+1:])
		if len(f) < 20 {
			continue
		}
		ppid, _ := strconv.Atoi(f[1])
		sid, _ := strconv.Atoi(f[3])
		out = append(out, process{pid: pid, ppid: ppid, sid: sid, start: f[19], state: f[0]})
	}
	return out
}
func killTree(root int, adopted bool) {
	// Each pass discovers job-control process groups and double-fork children.
	// Per-terminal cleanup uses the PTY session; service stop also cleans adopted daemons.
	for pass := 0; pass < 3; pass++ {
		ps := processes()
		targets := map[int]process{}
		for _, p := range ps {
			if p.pid == root || (!adopted && p.sid == root) || (adopted && p.ppid == root) {
				targets[p.pid] = p
			}
		}
		for changed := true; changed; {
			changed = false
			for _, p := range ps {
				if _, yes := targets[p.ppid]; yes {
					if _, found := targets[p.pid]; !found {
						targets[p.pid] = p
						changed = true
					}
				}
			}
		}
		current := map[int]string{}
		for _, p := range processes() {
			current[p.pid] = p.start
		}
		for pid, p := range targets {
			if pid != os.Getpid() && current[pid] == p.start {
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
}
func cleanupTerminal(pid int) {
	_ = syscall.Kill(pid, syscall.SIGTERM)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		found := false
		for _, p := range processes() {
			if p.pid == pid && p.state != "Z" {
				found = true
				break
			}
		}
		if !found {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	killTree(pid, false)
}
func cleanupAdopted() {
	killTree(os.Getpid(), true)
	for {
		var status syscall.WaitStatus
		pid, _ := syscall.Wait4(-1, &status, syscall.WNOHANG, nil)
		if pid <= 0 {
			return
		}
	}
}

func shellCommand(mode string) (*exec.Cmd, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return exec.Command(path, "--supervise-shell", mode), nil
}
func prepareSupervisor() error { return enableReaper() }
func childAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true, Foreground: true, Ctty: 0}
}
func cleanupSupervisorChildren(_ int) { killTree(os.Getpid(), true) }
func finishSupervisorChildren()       { cleanupAdopted() }
