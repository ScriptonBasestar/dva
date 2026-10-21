//go:build windows

package cli

import (
	"os"
	"testing"
)

// killLeftoverProcessGroup kills only pid: os/exec gives DVA no process group it can signal on
// Windows, and the process plugin refuses to start local background entries there anyway.
func killLeftoverProcessGroup(t *testing.T, pid int) {
	t.Helper()
	if proc, err := os.FindProcess(pid); err == nil {
		_ = proc.Kill()
	}
}
