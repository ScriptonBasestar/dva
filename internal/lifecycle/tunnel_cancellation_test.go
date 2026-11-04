//go:build !windows

package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestCloudflaredAuthCancellationTerminatesItsProcessGroup(t *testing.T) {
	dir := t.TempDir()
	started := filepath.Join(dir, "started")
	childPID := filepath.Join(dir, "child.pid")
	script := "#!/bin/sh\nPATH=/bin:/usr/bin\ntrap '' TERM\nsleep 30 &\necho $! > \"" + childPID + "\"\n: > \"" + started + "\"\nwait\n"
	if err := os.WriteFile(filepath.Join(dir, "cloudflared"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- runCloudflaredDiscardContext(ctx, "app.example.com") }()
	waitForTunnelCancellationFile(t, started)
	raw, err := os.ReadFile(childPID)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(string(raw[:len(raw)-1]))
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("auth error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("auth command did not return after cancellation")
	}
	if err := syscall.Kill(pid, 0); err == nil {
		t.Fatalf("auth child %d survived cancellation", pid)
	}
}

func waitForTunnelCancellationFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("tunnel shim did not start: %s", path)
}
