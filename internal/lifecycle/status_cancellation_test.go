package lifecycle

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestHealthCheckerCheckContextCancelsCommandProbe(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "started")
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan []HealthCheckResult, 1)
	go func() {
		done <- (&HealthChecker{}).CheckContext(ctx, map[string]config.HealthCheckConfig{
			"slow": {Type: "command", Command: ": > started; exec sleep 30", Timeout: 30},
		}, dir, nil)
	}()
	waitForStatusCancellationMarker(t, marker)
	cancel()
	select {
	case results := <-done:
		if len(results) != 0 {
			t.Fatalf("cancelled results = %+v, want omitted cancelled probe", results)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("CheckContext did not return after cancellation")
	}
}

func TestOrchestratorStatusCancellationReturnsPartialResults(t *testing.T) {
	dir := t.TempDir()
	started := filepath.Join(dir, "started")
	later := filepath.Join(dir, "later")
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(`version: "0.1.0"
stack:
  first:
    default_runner: script
    runners:
      script: {}
    health_checks:
      slow:
        type: command
        command: ": > started; exec sleep 30"
        timeout: 30
  later:
    default_runner: script
    runners:
      script: {}
    health_checks:
      later:
        type: command
        command: ": > later"
        timeout: 30
`), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	orch := NewOrchestrator(cfg, config.NewEnvironment(nil, dir, dir))
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan struct {
		status *AggregatedStatus
		err    error
	}, 1)
	go func() {
		status, err := orch.Status(ctx)
		done <- struct {
			status *AggregatedStatus
			err    error
		}{status, err}
	}()
	waitForStatusCancellationMarker(t, started)
	cancel()
	select {
	case got := <-done:
		if !errors.Is(got.err, context.Canceled) {
			t.Fatalf("Status error = %v, want context.Canceled", got.err)
		}
		if got.status == nil || len(got.status.Entries) != 1 || got.status.Entries[0].Name != "first" {
			t.Fatalf("partial status = %+v, want first entry only", got.status)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Status did not return after cancellation")
	}
	if _, err := os.Stat(later); !os.IsNotExist(err) {
		t.Fatalf("later health probe ran after cancellation: %v", err)
	}
}

func waitForStatusCancellationMarker(t *testing.T, marker string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("status probe did not start: %s", marker)
}
