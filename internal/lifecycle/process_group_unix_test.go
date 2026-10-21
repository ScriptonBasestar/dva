//go:build !windows

package lifecycle

import (
	"errors"
	"syscall"
	"testing"
)

// TestTerminateProcessGroupRefusesUnsafePIDs proves the last guard before kill(2): none of these
// values may reach syscall.Kill, because -1 signals every process the user owns and the own
// group would signal DVA itself (ISSUE-525).
func TestTerminateProcessGroupRefusesUnsafePIDs(t *testing.T) {
	for _, pid := range []int{-1, 0, 1, syscall.Getpgrp()} {
		if err := terminateProcessGroup(pid); !errors.Is(err, errUnsafeProcessGroup) {
			t.Fatalf("terminateProcessGroup(%d) error = %v, want errUnsafeProcessGroup", pid, err)
		}
	}
}
