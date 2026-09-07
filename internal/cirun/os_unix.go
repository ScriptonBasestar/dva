//go:build darwin || linux || freebsd || netbsd || openbsd

package cirun

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func tryLock(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB) }
func lockContended(err error) bool {
	return errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN)
}
func unlock(f *os.File) error { return syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }
func startProcess(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd.Start()
}
func cleanupGroup(pid int) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	time.Sleep(20 * time.Millisecond)
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
func terminateProcessGroup(pid int, waited <-chan error) {
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	select {
	case <-waited:
	case <-time.After(750 * time.Millisecond):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		<-waited
	}
	cleanupGroup(pid)
}
