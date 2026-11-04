//go:build !windows

package lifecycle

import (
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestStatusTunnelCleanupEscalatesOnlyItsOwnedProcessGroup(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "started")
	cmd := exec.Command("sh", "-c", fmt.Sprintf("trap '' TERM; : > %q; /bin/sleep 30 & wait", marker))
	if err := configureProcessGroup(cmd); err != nil {
		t.Fatal(err)
	}
	cmd.Stdout, cmd.Stderr = io.Discard, io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	killProcessGroupInCleanup(t, cmd.Process.Pid)
	waitForTunnelCancellationFile(t, marker)
	manager := newTunnelManager(slog.New(slog.NewTextHandler(io.Discard, nil)))
	manager.open["owned"] = &tunnelProcess{cmd: cmd, pid: cmd.Process.Pid}
	done := make(chan struct{})
	go func() {
		manager.CloseStatus()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("status tunnel cleanup waited indefinitely for a TERM-ignoring child")
	}
	if len(manager.open) != 0 {
		t.Fatalf("owned tunnel was not reclaimed: %d remaining", len(manager.open))
	}
}
