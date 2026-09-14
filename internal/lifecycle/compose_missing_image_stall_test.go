//go:build unix

package lifecycle

// The stall test needs a named pipe, so it lives behind a unix build tag: syscall.Mkfifo
// does not exist on Windows, and the /bin/sh shim it drives would not run there either.

import (
	"context"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"errors"
)

// Given a daemon that answers `docker info` and then stalls on inspect, When the probe
// runs, Then the deadline ends it and the original failure stands. This is the bound
// docker_daemon.go established for the same post-failure path: a probe added to improve
// on a bare exit status must not be able to replace it with a hang. The shim blocks on a
// fifo nobody writes to, so the only thing that can end the call is the timeout.
func TestComposeUp_StalledInspect_DoesNotHangAndKeepsOriginalError(t *testing.T) {
	installImageShim(t)
	t.Setenv(shimComposeExitVar, "1")
	t.Setenv(shimInfoExitVar, "0")
	t.Setenv(shimConfigJSONVar, fixtureMixedProject)

	fifo := filepath.Join(t.TempDir(), "stall")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("mkfifo unavailable: %v", err)
	}
	t.Setenv(shimInspectFifoVar, fifo)

	saved := missingImageProbeTimeout
	t.Cleanup(func() { missingImageProbeTimeout = saved })
	missingImageProbeTimeout = 300 * time.Millisecond

	p, pctx := upFailureContext(t, "")

	done := make(chan error, 1)
	go func() {
		_, err := p.Up(context.Background(), pctx)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("Up() error = nil, want the compose failure")
		}
		if _, ok := errors.AsType[*MissingLocalImageError](err); ok {
			t.Fatalf("a probe that never got an answer still produced a diagnosis: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("Up() did not return: the probe turned a failed command into a hang")
	}
}
