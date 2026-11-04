//go:build !windows

package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
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
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("auth error = %v, want context.Canceled", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("auth command did not return after cancellation")
	}
	waitForStatusChildGone(t, childPID)
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
