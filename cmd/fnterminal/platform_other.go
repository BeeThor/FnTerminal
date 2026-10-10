//go:build !linux

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"syscall"
)

// Host tests can exercise PTYs, but production authentication is Linux-only.
func socketPeer(net.Conn) (uint32, error) { return 0, fmt.Errorf("SO_PEERCRED requires Linux") }
func enableReaper() error                 { return fmt.Errorf("production service requires Linux fnOS") }
func cleanupTerminal(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = syscall.Kill(pid, syscall.SIGKILL)
}
func cleanupAdopted() {}
func shellCommand(mode string) (*exec.Cmd, error) {
	path, err := os.Executable()
	if err != nil {
		return nil, err
	}
	return exec.Command(path, "--supervise-shell", mode), nil
}
func prepareSupervisor() error { return nil }
func childAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true, Foreground: true, Ctty: 0}
}
func cleanupSupervisorChildren(pid int) { cleanupTerminal(pid) }
func finishSupervisorChildren()         {}
