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
// outlives the test package (ISSUE-522). ESRCH means the group is already gone. EPERM is treated
// the same: macOS answers EPERM instead of ESRCH when the group holds only zombies (TASK-598
// reproduced it; TASK-599 reproduced it for this helper's own call, measured at 8–40% of
// kills of a group whose leader died unreaped, depending on load). The group is our own child's, so a genuine permission
// problem is not expected; any other error still fails the test. pid comes from a pidfile, so it is checked before it is
// negated: 1 would become kill(-1) and this test binary's own group would kill the test itself.
func killLeftoverProcessGroup(t *testing.T, pid int) {
	t.Helper()
	if pid <= 1 || pid == syscall.Getpgrp() {
		t.Errorf("refusing to kill process group %d", pid)
		return
	}
	if err := killGroupTolerantly(pid, syscall.Kill); err != nil {
		t.Errorf("kill process group %d: %v", pid, err)
	}
}

// killGroupTolerantly SIGKILLs the group of pid through kill and drops the errors that mean the
// group is already gone (ESRCH, and EPERM for a zombie-only group on macOS).
func killGroupTolerantly(pid int, kill func(pid int, sig syscall.Signal) error) error {
	if err := kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EPERM) {
		return err
	}
	return nil
}
