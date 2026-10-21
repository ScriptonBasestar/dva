//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestRunDoctorCommandTimeoutTerminatesShellChildren(t *testing.T) {
	dir := t.TempDir()
	pidFile := filepath.Join(dir, "child.pid")
	passed, evidence := runDoctorCommand(config.DoctorCheck{
		Name:    "bounded",
		Type:    "command",
		Timeout: "200ms",
		Command: "sleep 5 & echo $! > child.pid; wait",
	}, dir)
	if passed || !evidence.TimedOut || evidence.ErrorClassification != "timeout" {
		t.Fatalf("timeout result = passed=%t evidence=%+v", passed, evidence)
	}
	data, err := os.ReadFile(pidFile)
	if err != nil {
		t.Fatalf("read child pid: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		err = syscall.Kill(pid, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		// Linux reports success for kill(pid, 0) while an exited child waits as a
		// zombie to be reaped. That is terminated, not a leaked running process.
		if runtime.GOOS == "linux" {
			status, readErr := os.ReadFile(filepath.Join("/proc", strconv.Itoa(pid), "status"))
			if os.IsNotExist(readErr) || strings.Contains(string(status), "\nState:\tZ") {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out command left child pid %d running", pid)
}

func TestRunDoctorCommandNamesTerminatingSignal(t *testing.T) {
	const command = "kill -TERM $$"
	passed, evidence := runDoctorCommand(config.DoctorCheck{Name: "killed", Type: "command", Command: command}, t.TempDir())
	if passed || evidence.ErrorClassification != "signal" || evidence.Signal != "SIGTERM" {
		t.Fatalf("signal result = passed=%t evidence=%+v", passed, evidence)
	}
	if evidence.ExitCode == nil || *evidence.ExitCode != -1 {
		t.Fatalf("exit_code = %v, want pointer to -1", evidence.ExitCode)
	}
	if got, want := doctorCommandFinding(command, evidence), "command terminated by SIGTERM: "+command; got != want {
		t.Fatalf("finding = %q, want %q", got, want)
	}
	encoded, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"signal":"SIGTERM"`) {
		t.Fatalf("signal name missing from JSON: %s", encoded)
	}
}
