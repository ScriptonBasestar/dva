//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"errors"
	"os/exec"
	"syscall"
)

func resourceConfigureProcess(cmd *exec.Cmd) { cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} }
func resourceTermGroup(pid int) error {
	err := syscall.Kill(-pid, syscall.SIGTERM)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
func resourceKillGroup(pid int) error {
	err := syscall.Kill(-pid, syscall.SIGKILL)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	return err
}
func resourceGroupGone(pid int) bool {
	err := syscall.Kill(-pid, 0)
	return errors.Is(err, syscall.ESRCH)
}
func resourceSignalExit(err *exec.ExitError) int {
	if s, ok := err.Sys().(syscall.WaitStatus); ok && s.Signaled() {
		return 128 + int(s.Signal())
	}
	return err.ExitCode()
}
