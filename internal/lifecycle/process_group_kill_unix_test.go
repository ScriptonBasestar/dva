//go:build !windows

package lifecycle

import (
	"errors"
	"syscall"
	"testing"
)

// killProcessGroupInCleanup SIGKILLs the process group led by pid when the test ends.
//
// It signals the group directly rather than exec'ing kill(1). On hosted runners the exec'd
// `kill -9 -<pid>` was observed to leave TERM-ignoring shell loops alive, for a reason not yet
// known, and that variant correlated with the job losing its runner (ISSUE-515). ESRCH means
// the group is already gone. EPERM is treated the same: macOS answers EPERM instead of ESRCH
// when the group holds only zombies, which is what a group the code under test already killed
// and reaped looks like (TASK-598; inferred from timing, not reproduced in isolation). The group
// is our own child's, so a genuine permission problem is not expected. Any other error fails the
// test so a cleanup that leaves processes behind is visible instead of swallowed.
//
// pid comes from a pidfile, so it is checked before it is ever negated: 0 would signal this
// test binary's own group and 1 would become kill(-1), every process the runner user owns.
func killProcessGroupInCleanup(t *testing.T, pid int) {
	t.Helper()
	if pid <= 1 || pid == syscall.Getpgrp() {
		t.Fatalf("refusing to register a process-group kill for pid %d", pid)
	}
	t.Cleanup(func() {
		t.Helper()
		if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) && !errors.Is(err, syscall.EPERM) {
			t.Errorf("kill process group %d: %v", pid, err)
		}
	})
}
