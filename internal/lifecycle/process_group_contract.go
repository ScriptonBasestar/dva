package lifecycle

import (
	"errors"
	"fmt"
)

var errProcessGroupsUnsupported = errors.New("local background process groups are unsupported on Windows; run the command outside DVA or use a supported remote runner")

var errUnsafeProcessGroup = errors.New("refusing to signal a process group DVA did not start; inspect and remove the pidfile")

// processGroupPIDError is pure so every caller makes the same decision for a retained PID.
// A retained PID of 1 or of DVA's own process group never names a group DVA started: negated,
// 1 becomes kill(-1), every process the user owns, and the own group would signal DVA and its
// caller (ISSUE-525). Both are errors on every platform, so a corrupt pidfile is reported
// instead of being dropped or reported as stopped.
func processGroupPIDError(pid int, supported bool, ownGroup int) error {
	if pid == 1 || (pid > 0 && pid == ownGroup) {
		return fmt.Errorf("%w: pid %d", errUnsafeProcessGroup, pid)
	}
	if pid > 0 && !supported {
		return errProcessGroupsUnsupported
	}
	return nil
}

func requireProcessGroupPID(pid int) error {
	return processGroupPIDError(pid, processGroupsSupported(), ownProcessGroup())
}

// signalableProcessGroupPID only says the pidfile names a process; whether that PID may be
// signalled is requireProcessGroupPID's decision, so keep safety checks there.
func signalableProcessGroupPID(pid int) bool { return pid > 0 }

func managedProcessStatus(name string, pid int) ([]ServiceStatus, error) {
	if err := requireProcessGroupPID(pid); err != nil {
		return nil, fmt.Errorf("status %s: %w", name, err)
	}
	if pid > 0 && IsProcessRunning(pid) {
		return []ServiceStatus{{Name: name, State: "running"}}, nil
	}
	return []ServiceStatus{{Name: name, State: "stopped"}}, nil
}
