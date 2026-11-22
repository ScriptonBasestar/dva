//go:build darwin

package lifecycle

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// lookupListenerPids asks lsof for the pids listening on a TCP port. lsof exits 1 both when
// nothing listens and on some errors, so an empty result is treated as unverified by the
// caller rather than as "nobody owns it".
func lookupListenerPids(port int) ([]int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "lsof", "-nP", "-t", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN").Output()
	if err != nil && len(out) == 0 {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil, true
		}
		return nil, false
	}
	return parsePidLines(string(out)), true
}

func lookupProcessGroup(pid int) (int, bool) {
	group, err := syscall.Getpgid(pid)
	return group, err == nil
}

func parsePidLines(text string) []int {
	var pids []int
	seen := map[int]bool{}
	for field := range strings.FieldsSeq(text) {
		pid, err := strconv.Atoi(field)
		if err == nil && pid > 0 && !seen[pid] {
			seen[pid] = true
			pids = append(pids, pid)
		}
	}
	return pids
}
