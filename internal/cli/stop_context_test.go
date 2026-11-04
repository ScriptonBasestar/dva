package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

func TestStopRoutesHonorCallerCancellation(t *testing.T) {
	for _, route := range []string{"whole-stack", "plan"} {
		t.Run(route, func(t *testing.T) {
			started := filepath.Join(t.TempDir(), "started")
			yml := fmt.Sprintf(`version: "0.1.0"
stack:
  blocking:
    default_runner: script
    runners:
      script:
        stop: %q
`, ": > "+started+"; exec sleep 30")
			args := []string(nil)
			if route != "whole-stack" {
				yml += `plans:
  preview:
    entries:
      - name: blocking
  composed:
    composes:
      - plan: preview
`
				args = []string{"preview"}
				if route == "composition" {
					args = []string{"composed"}
				}
			}
			useConfig(t, yml)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			cmd := &cobra.Command{}
			cmd.SetContext(ctx)
			done := make(chan error, 1)
			go func() { done <- stopCmd.RunE(cmd, args) }()

			waitForStopMarker(t, started)
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("stop error = %v, want context.Canceled", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("stop did not return after caller cancellation")
			}
		})
	}
}

func TestStopRoutesDoNotStartForCancelledCaller(t *testing.T) {
	for _, route := range []string{"whole-stack", "plan", "composition"} {
		t.Run(route, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "executed")
			yml := fmt.Sprintf(`version: "0.1.0"
stack:
  app:
    default_runner: script
    runners:
      script:
        stop: %q
`, ": > "+marker)
			if route != "whole-stack" {
				yml += `plans:
  preview:
    entries:
      - name: app
  composed:
    composes:
      - plan: preview
`
			}
			useConfig(t, yml)
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			cmd := &cobra.Command{}
			cmd.SetContext(ctx)
			args := []string(nil)
			if route == "plan" {
				args = []string{"preview"}
			} else if route == "composition" {
				args = []string{"composed"}
			}
			err := stopCmd.RunE(cmd, args)
			if !errors.Is(err, context.Canceled) {
				t.Fatalf("stop error = %v, want context.Canceled", err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("cancelled stop ran an entry: marker stat = %v", err)
			}
		})
	}
}

func waitForStopMarker(t *testing.T, marker string) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("stop command did not start external script: %s", marker)
}
