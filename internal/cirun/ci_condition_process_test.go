//go:build darwin || linux || freebsd || netbsd || openbsd

package cirun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestConditionCancellationStopsRecordedBackgroundChild(t *testing.T) {
	root := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	step := config.CIStep{Name: "unit", Run: "exit 77", Timeout: "10s", Condition: &config.CICondition{
		Run: "sleep 20 & echo $! > child.pid; wait", Timeout: "5s"}}
	type result struct {
		report StepReport
		err    error
	}
	done := make(chan result, 1)
	go func() {
		r, err := runConditionedStep(ctx, root, step, os.Environ(), 1, "test", nil, time.Now())
		done <- result{r, err}
	}()
	child := waitCompositionFile(t, filepath.Join(root, "child.pid"))
	cancel()
	select {
	case result := <-done:
		if result.err == nil || result.report.Status != "canceled" {
			t.Fatalf("report=%+v err=%v", result.report, result.err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("condition did not terminate after cancellation")
	}
	assertCompositionProcessStopped(t, child)
}
