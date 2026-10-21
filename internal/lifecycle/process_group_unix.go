//go:build !windows

package lifecycle

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func configureProcessGroup(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return nil
}

// terminateProcessGroup sends SIGTERM to the process group led by pid. It is the last guard
// before kill(2), for callers that skip requireProcessGroupPID: pid <= 1 would become kill(-1),
// every process the user owns, and the own group would signal DVA and its caller.
func terminateProcessGroup(pid int) error {
	if pid <= 1 || pid == ownProcessGroup() {
		return fmt.Errorf("%w: pid %d", errUnsafeProcessGroup, pid)
	}
	return syscall.Kill(-pid, syscall.SIGTERM)
}

func processGroupsSupported() bool { return true }

func ownProcessGroup() int { return syscall.Getpgrp() }

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}
