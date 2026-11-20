//go:build darwin || linux || freebsd || netbsd || openbsd

package cimetrics

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMetricsCancellationCannotBecomePassWhenChildTrapsTERM(t *testing.T) {
	state := t.TempDir()
	if err := os.Chmod(state, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", state)
	ready := filepath.Join(t.TempDir(), "ready")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- observedRun(t, ctx, []string{"/bin/sh", "-c", `trap 'exit 0' TERM; printf ready > "$1"; sleep 30 & wait`, "sh", ready}, &stdout, &stderr)
	}()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(ready); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := os.Stat(ready); err != nil {
		t.Fatal("owned child did not start")
	}
	start := time.Now()
	cancel()
	select {
	case err := <-done:
		if err == nil || !errors.Is(err, context.Canceled) {
			t.Fatalf("canceled check became PASS: %v", err)
		}
		if elapsed := time.Since(start); elapsed > 700*time.Millisecond {
			t.Fatalf("cancellation exceeded DVA grace: %s", elapsed)
		}
	case <-time.After(700 * time.Millisecond):
		t.Fatal("cancellation did not return within DVA grace")
	}
	rec, _ := awaitRecord(t, state)
	if rec.Outcome != "canceled" {
		t.Fatalf("cancellation outcome lost: %+v", rec)
	}
}

func TestMetricsCompletedCheckWinsLaterCancellation(t *testing.T) {
	state := t.TempDir()
	if err := os.Chmod(state, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", state)
	ctx, cancel := context.WithCancel(context.Background())
	var out, stderr bytes.Buffer
	if err := observedRun(t, ctx, []string{"/bin/sh", "-c", "exit 0"}, &out, &stderr); err != nil {
		t.Fatal(err)
	}
	cancel()
	rec, _ := awaitRecord(t, state)
	if rec.Outcome != "succeeded" || rec.ExitCode == nil || *rec.ExitCode != 0 {
		t.Fatalf("completed check lost result: %+v", rec)
	}
}
