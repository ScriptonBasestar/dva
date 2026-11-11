//go:build !windows

package cli

import (
	"os/exec"
	"syscall"
	"testing"
)

// killGroupTolerantly drops ESRCH and EPERM (the macOS answer for a zombie-only group) and
// reports every other error.
func TestKillGroupTolerantlyToleratesOnlyGroupAlreadyGone(t *testing.T) {
	for _, tc := range []struct {
		name    string
		err     error
		wantErr bool
	}{
		{"none", nil, false},
		{"ESRCH", syscall.ESRCH, false},
		{"EPERM", syscall.EPERM, false},
		{"EINVAL", syscall.EINVAL, true},
	} {
		got := killGroupTolerantly(4242, func(pid int, _ syscall.Signal) error {
			if pid != -4242 {
				t.Errorf("%s: signalled %d, want -4242", tc.name, pid)
			}
			return tc.err
		})
		if (got != nil) != tc.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", tc.name, got, tc.wantErr)
		}
	}
}

// Against real zombie-only groups: a leader killed but not yet reaped, then the helper's own
// kill. On macOS the second kill answers EPERM in 8–40% of rounds
// depending on load (measured in the TASK-599 review), so the
// rounds are many; with the helper tolerating only ESRCH this test fails there.
func TestKillGroupTolerantlyAcceptsZombieOnlyGroups(t *testing.T) {
	rounds := 1500
	if testing.Short() {
		rounds = 200
	}
	for i := 0; i < rounds; i++ {
		c := exec.Command("sleep", "30")
		c.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := c.Start(); err != nil {
			t.Fatal(err)
		}
		pid := c.Process.Pid
		_ = syscall.Kill(-pid, syscall.SIGKILL)
		killLeftoverProcessGroup(t, pid)
		_ = c.Wait()
	}
}
