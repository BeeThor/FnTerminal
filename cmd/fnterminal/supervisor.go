package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// A private FD communicates startup outcome to the service. PTY output is never
// used as a readiness signal, and errors are not hidden behind an empty close.
func superviseShell(mode string) int {
	ready := os.NewFile(3, "startup-status")
	defer ready.Close()
	syscall.CloseOnExec(3) // private status pipe must never reach Bash/SSH
	failure := func(stage string, e error) int {
		_ = json.NewEncoder(ready).Encode(map[string]any{"ok": false, "stage": stage, "error": e.Error()})
		return 1
	}
	if os.Geteuid() == 0 {
		return failure("identity", fmt.Errorf("root shell is prohibited"))
	}
	if err := prepareSupervisor(); err != nil {
		return failure("subreaper", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGHUP)
	defer stop()
	var child *exec.Cmd
	switch mode {
	case "application":
		child = exec.Command("/bin/bash", "--noprofile", "--norc", "-i")
	case "nas":
		path := os.Getenv("FNTERMINAL_LOGIN_HELPER")
		if path == "" {
			return failure("nas-helper", fmt.Errorf("NAS login helper is not configured"))
		}
		child = exec.Command(path)
	default:
		return failure("mode", fmt.Errorf("unsupported terminal mode"))
	}
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.SysProcAttr = childAttributes()
	if err := child.Start(); err != nil {
		return failure("shell-exec", err)
	}
	_ = json.NewEncoder(ready).Encode(map[string]any{"ok": true})
	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	exitCode := 0
	select {
	case <-ctx.Done():
		cleanupSupervisorChildren(child.Process.Pid)
		<-done
	case err := <-done:
		if err != nil {
			if status, ok := err.(*exec.ExitError); ok {
				exitCode = status.ExitCode()
			} else {
				exitCode = 1
			}
		}
	}
	finishSupervisorChildren()
	_ = json.NewEncoder(ready).Encode(map[string]any{"type": "exit", "exitCode": exitCode})
	return 0
}
