//go:build darwin || linux || freebsd || netbsd || openbsd

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestMetricsOuterSignalHelper(_ *testing.T) {
	if os.Getenv("CIMETRICS_SIGNAL_HELPER") != "1" {
		return
	}
	args := []string{"--check", "unit", "--", "/bin/sh", "-c", `trap 'exit 0' TERM; printf '%s' "$$" > "$1"; sleep 30 & printf '%s' "$!" > "$2"; wait`, "sh", os.Getenv("CIMETRICS_SHELL_PID_FILE"), os.Getenv("CIMETRICS_CHILD_PID_FILE")}
	r := runCommand(args, os.Stdout, os.Stderr)
	if r.signal != nil {
		rethrow(r.signal)
	}
	os.Exit(r.code)
}

// Model DVA's outer process-group TERM. The wrapper's separately-owned child
// group must be terminated within DVA's grace period, even if its shell traps
// TERM and exits successfully.
func TestMetricsOuterGroupSignalCleansChildGroup(t *testing.T) {
	dir := t.TempDir()
	shellFile := filepath.Join(dir, "shell.pid")
	childFile := filepath.Join(dir, "child.pid")
	cmd := exec.Command(os.Args[0], "-test.run=^TestMetricsOuterSignalHelper$")
	cmd.Env = append(os.Environ(), "CIMETRICS_SIGNAL_HELPER=1", "CIMETRICS_SHELL_PID_FILE="+shellFile, "CIMETRICS_CHILD_PID_FILE="+childFile, "DVA_PROJECT_CI_METRICS_DIR="+dir)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	outerPID := cmd.Process.Pid
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	finished := false
	shellPID := 0
	defer func() {
		if finished {
			return
		}
		_ = syscall.Kill(-outerPID, syscall.SIGTERM)
		select {
		case <-done:
		case <-time.After(time.Second):
			_ = syscall.Kill(-outerPID, syscall.SIGKILL)
			if shellPID > 0 {
				_ = syscall.Kill(-shellPID, syscall.SIGKILL)
			}
			<-done
		}
	}()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if s, e := os.ReadFile(shellFile); e == nil {
			shellPID, _ = strconv.Atoi(strings.TrimSpace(string(s)))
		}
		if shellPID > 0 {
			if _, e := os.Stat(childFile); e == nil {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	childBytes, err := os.ReadFile(childFile)
	if err != nil || shellPID <= 0 {
		t.Fatal("owned child group did not start")
	}
	childPID, err := strconv.Atoi(strings.TrimSpace(string(childBytes)))
	if err != nil || childPID <= 0 {
		t.Fatalf("invalid owned child pid: %q", childBytes)
	}
	started := time.Now()
	if err := syscall.Kill(-outerPID, syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		finished = true
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("outer cancellation became PASS: %v", err)
		}
		status, ok := exit.Sys().(syscall.WaitStatus)
		if !ok || !status.Signaled() || status.Signal() != syscall.SIGTERM {
			t.Fatalf("outer signal lost: %v", exit)
		}
		if elapsed := time.Since(started); elapsed > 750*time.Millisecond {
			t.Fatalf("outer cancellation exceeded DVA grace: %s", elapsed)
		}
	case <-time.After(750 * time.Millisecond):
		t.Fatal("outer cancellation did not finish within DVA grace")
	}
	// Kill(pid, 0) is a read-only existence query for the exact child PID that
	// the owned shell recorded. Allow a brief interval for OS reaping.
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		err := syscall.Kill(childPID, 0)
		if errors.Is(err, syscall.ESRCH) {
			return
		}
		if b, e := exec.Command("/bin/ps", "-o", "stat=", "-p", strconv.Itoa(childPID)).Output(); e == nil && strings.HasPrefix(strings.TrimSpace(string(b)), "Z") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("owned descendant %d still exists after group cleanup", childPID)
}
