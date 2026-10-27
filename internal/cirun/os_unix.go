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
	// The wait error is expected here: the group is being signalled to exit.
	// errcheck flags the bare receive and staticcheck S1005 flags `_ =`, so the
	// discard is marked for errcheck instead.
	select {
	case <-waited: //nolint:errcheck // expected exit error, see above
	case <-time.After(750 * time.Millisecond):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		<-waited //nolint:errcheck // expected exit error, see above
	}
	cleanupGroup(pid)
}

func signalCompositionGroup(pid int) { _ = syscall.Kill(-pid, syscall.SIGTERM) }

// TERM was already sent before draining child supervisors. Do not start another
// 750ms grace per ancestor after the shared drain; charge elapsed time once.
func finishCompositionGroup(pid int, waited <-chan error, signaledAt time.Time) {
	remaining := max(time.Duration(0), 750*time.Millisecond-time.Since(signaledAt))
	select {
	case <-waited: //nolint:errcheck // expected signalled exit
	case <-time.After(remaining):
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		<-waited //nolint:errcheck // expected signalled exit
	}
	cleanupGroup(pid)
}
