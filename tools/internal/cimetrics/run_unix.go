//go:build darwin || linux || freebsd || netbsd || openbsd

package cimetrics

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

func configureGroup(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func signalGroup(pid int, sig syscall.Signal) error {
	err := syscall.Kill(-pid, sig)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
func stopGroup(pid int) error { return signalGroup(pid, syscall.SIGTERM) }
func killGroup(pid int) error { return signalGroup(pid, syscall.SIGKILL) }
func signalName(state *os.ProcessState) string {
	if state == nil {
		return ""
	}
	if status, ok := state.Sys().(syscall.WaitStatus); ok && status.Signaled() {
		return status.Signal().String()
	}
	return ""
}
func groupSupported() bool { return true }
