//go:build !windows

package cli

import (
	"errors"
	"syscall"
	"testing"
)

// killLeftoverProcessGroup SIGKILLs the process group led by pid. Native entries run as
// `sh -c <run>` leading their own group, and a shell may fork its command instead of exec'ing
// it (Debian's dash does for `sleep 999`), so killing only the leader orphans the child and it
// outlives the test package (ISSUE-522). pid comes from a pidfile, so it is checked before it is
// negated: 1 would become kill(-1) and this test binary's own group would kill the test itself.
func killLeftoverProcessGroup(t *testing.T, pid int) {
	t.Helper()
	if pid <= 1 || pid == syscall.Getpgrp() {
		t.Errorf("refusing to kill process group %d", pid)
		return
	}
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		t.Errorf("kill process group %d: %v", pid, err)
	}
}
