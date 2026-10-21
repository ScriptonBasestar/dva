package lifecycle

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestProcessGroupPIDError(t *testing.T) {
	for _, tc := range []struct {
		name      string
		pid       int
		supported bool
		ownGroup  int
		want      error
	}{
		{name: "no PID is harmless", pid: 0, supported: false},
		{name: "negative PID is harmless", pid: -1, supported: false},
		{name: "no PID never matches an absent own group", pid: 0, supported: true},
		{name: "supported positive PID", pid: 42, supported: true, ownGroup: 7},
		{name: "unsupported positive PID", pid: 42, supported: false, want: errProcessGroupsUnsupported},
		// ISSUE-525: negated, 1 is kill(-1) and the own group is DVA itself, on any platform.
		{name: "PID 1", pid: 1, supported: true, want: errUnsafeProcessGroup},
		{name: "unsupported PID 1", pid: 1, supported: false, want: errUnsafeProcessGroup},
		{name: "own process group", pid: 7, supported: true, ownGroup: 7, want: errUnsafeProcessGroup},
	} {
		t.Run(tc.name, func(t *testing.T) {
			const pidFile = "/project/.dva/pids/api.pid"
			err := processGroupPIDError(tc.pid, pidFile, tc.supported, tc.ownGroup)
			if tc.want == nil && err != nil || tc.want != nil && !errors.Is(err, tc.want) {
				t.Fatalf("processGroupPIDError(%d, %v, %d) error = %v, want %v", tc.pid, tc.supported, tc.ownGroup, err, tc.want)
			}
			// The sentinel tells the user to remove the pidfile, so the error must say which.
			if errors.Is(err, errUnsafeProcessGroup) && !strings.Contains(err.Error(), pidFile) {
				t.Fatalf("processGroupPIDError(%d) error = %q, want it to name %s", tc.pid, err, pidFile)
			}
		})
	}
}

func TestSignalableProcessGroupPID(t *testing.T) {
	for _, pid := range []int{-1, 0} {
		if signalableProcessGroupPID(pid) {
			t.Fatalf("pid %d is signalable", pid)
		}
	}
	if !signalableProcessGroupPID(1) {
		t.Fatal("positive PID is not signalable")
	}
}

// TestRetainedPIDOneIsReportedNotSignalled drives the stop, down and status paths with a pidfile
// holding 1 (ISSUE-525): each must return errUnsafeProcessGroup naming that pidfile instead of
// signalling kill(-1) or reporting success, and must keep the pidfile so the corrupt state stays
// visible.
func TestRetainedPIDOneIsReportedNotSignalled(t *testing.T) {
	p := &ProcessPlugin{}
	for _, tc := range []struct {
		name string
		run  func(*PluginContext) error
	}{
		{"stop", func(pctx *PluginContext) error { return p.Stop(context.Background(), pctx) }},
		{"down", func(pctx *PluginContext) error { return p.Down(context.Background(), pctx) }},
		{"status", func(pctx *PluginContext) error { _, err := p.Status(context.Background(), pctx); return err }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			pidFile := filepath.Join(dir, config.DotDirName, config.PidsDirName, "api.pid")
			if err := os.MkdirAll(filepath.Dir(pidFile), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(pidFile, []byte("1\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			pctx := &PluginContext{Entry: &config.LifecycleEntry{Name: "api"}, ConfigDir: dir, Logger: slog.Default()}
			err := tc.run(pctx)
			if !errors.Is(err, errUnsafeProcessGroup) {
				t.Fatalf("%s with pidfile 1 error = %v, want errUnsafeProcessGroup", tc.name, err)
			}
			if !strings.Contains(err.Error(), pidFile) {
				t.Fatalf("%s with pidfile 1 error = %q, want it to name %s", tc.name, err, pidFile)
			}
			if _, err := os.Stat(pidFile); err != nil {
				t.Fatalf("%s removed the pidfile holding 1: %v", tc.name, err)
			}
		})
	}
}
