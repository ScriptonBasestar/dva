package cirun

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestRunFailFastFalseContinuesIndependentAndSkipsFailureDescendants(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	independent := filepath.Join(root, "independent-ran")
	child := filepath.Join(root, "child-ran")
	grandchild := filepath.Join(root, "grandchild-ran")
	p := profile(
		config.CIStep{Name: "failed", Run: "exit 4"},
		config.CIStep{Name: "independent", Run: "touch " + shellQuote(independent)},
		config.CIStep{Name: "child", Run: "touch " + shellQuote(child), DependsOn: []string{"failed"}},
		config.CIStep{Name: "grandchild", Run: "touch " + shellQuote(grandchild), DependsOn: []string{"child"}},
	)
	p.MaxParallel = 1
	p.FailFast = new(false)

	r, err := Run(context.Background(), Options{Root: root, ProfileName: "continue", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: p})
	if err == nil || r.Status != "failed" {
		t.Fatalf("report=%#v err=%v, want failed nonzero run", r, err)
	}
	if !strings.Contains(err.Error(), `CI step "failed"`) || !strings.Contains(err.Error(), "exit status 4") || !strings.Contains(r.Error, "exit status 4") {
		t.Fatalf("concrete failure lost from top-level result: report=%#v err=%v", r, err)
	}
	if got, want := []string{r.Steps[0].Status, r.Steps[1].Status, r.Steps[2].Status, r.Steps[3].Status}, []string{"failed", "succeeded", "skipped", "skipped"}; !sameStrings(got, want) {
		t.Fatalf("step statuses = %v, want %v", got, want)
	}
	if _, err := os.Stat(independent); err != nil {
		t.Fatalf("independent step did not continue: %v", err)
	}
	for _, path := range []string{child, grandchild} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("failed dependency descendant ran at %s: %v", path, err)
		}
	}
}

func TestRunDefaultFailFastCancelsIndependentSibling(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	p := profile(
		config.CIStep{Name: "failed", Run: "exit 4"},
		config.CIStep{Name: "slow", Run: "sleep 5"},
	)
	if p.FailFast != nil {
		t.Fatal("profile helper must exercise the omitted default")
	}
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "default-fail-fast", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: p})
	if err == nil || r.Status != "failed" {
		t.Fatalf("report=%#v err=%v, want failed run", r, err)
	}
	if got := r.Steps[1].Status; got != "canceled" {
		t.Fatalf("default fail_fast sibling status = %q, want canceled", got)
	}
}

func TestFailFastFalseStillHonorsExternalCancellationAndDeadline(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		context    func() (context.Context, context.CancelFunc)
	}{
		{
			name: "cancellation", want: "canceled",
			context: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
		},
		{
			name: "deadline", want: "timed_out",
			context: func() (context.Context, context.CancelFunc) {
				return context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.context()
			defer cancel()
			root, state := t.TempDir(), t.TempDir()
			marker := filepath.Join(root, "must-not-run")
			p := profile(config.CIStep{Name: "independent", Run: "touch " + shellQuote(marker)})
			p.FailFast = new(false)

			r, err := Run(ctx, Options{Root: root, ProfileName: "external-" + tc.name, StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: p})
			if err == nil || r.Status != tc.want {
				t.Fatalf("report=%#v err=%v, want %s", r, err, tc.want)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("external %s ran a step: %v", tc.name, err)
			}
		})
	}
}

func TestFailFastFalseExternalCancellationStopsRunningStep(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	started := filepath.Join(root, "started")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := profile(config.CIStep{Name: "slow", Run: "touch " + shellQuote(started) + "; sleep 5"})
	p.FailFast = new(false)
	type result struct {
		report Report
		err    error
	}
	done := make(chan result, 1)
	go func() {
		r, err := Run(ctx, Options{Root: root, ProfileName: "running-cancel", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: p})
		done <- result{r, err}
	}()
	waitForFailFastFile(t, started)
	cancel()
	select {
	case result := <-done:
		if result.err == nil || result.report.Status != "canceled" || result.report.Steps[0].Status != "canceled" {
			t.Fatalf("report=%#v err=%v, want cancelled running step", result.report, result.err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("external cancellation did not stop running fail_fast:false step")
	}
}

func TestFailFastFalseProfileDeadlineStopsRunningStep(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	started := filepath.Join(root, "started")
	p := profile(config.CIStep{Name: "slow", Run: "touch " + shellQuote(started) + "; sleep 5"})
	p.Timeout = "40ms"
	p.FailFast = new(false)
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "running-deadline", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: p})
	if err == nil || r.Status != "timed_out" || r.Steps[0].Status != "timed_out" {
		t.Fatalf("report=%#v err=%v, want timed-out running step", r, err)
	}
	if _, err := os.Stat(started); err != nil {
		t.Fatalf("deadline test never started its step: %v", err)
	}
}

func TestFailFastFalseCheckpointFailureStillAborts(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first-ran")
	second := filepath.Join(root, "second-ran")
	p := profile(
		config.CIStep{Name: "first", Run: "touch " + shellQuote(first)},
		config.CIStep{Name: "second", Run: "touch " + shellQuote(second)},
	)
	p.FailFast = new(false)
	sentinel := errors.New("checkpoint unavailable")

	reports, err := execute(context.Background(), root, p, nil, io.Discard, func([]StepReport) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("execute error = %v, want checkpoint failure", err)
	}
	for i, report := range reports {
		if report.Status != "skipped" {
			t.Fatalf("report %d status = %q, want skipped after integrity failure", i, report.Status)
		}
	}
	for _, marker := range []string{first, second} {
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatalf("checkpoint failure ran independent step %s: %v", marker, err)
		}
	}
}

func sameStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func waitForFailFastFile(t *testing.T, path string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("step did not create %s", path)
}
