package lifecycle

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// newWaitReadyTestOrchestrator builds a minimal Orchestrator whose cfg.FileDir() is a
// temp config dir, so entryPidPath/entryLogPath resolve inside the test sandbox.
func newWaitReadyTestOrchestrator(t *testing.T) *Orchestrator {
	t.Helper()
	dir := t.TempDir()
	writeImportedPlanConfig(t, dir, "version: \"0.1.0\"\n")
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	env := config.NewEnvironment(nil, cfg.FileDir(), cfg.FileDir())
	return &Orchestrator{cfg: cfg, env: env, hc: &HealthChecker{}}
}

// deadPid returns the pid of a process that has already been reaped. Go's
// os.Process.Signal returns ErrProcessDone for a reaped pid without a syscall,
// so IsProcessRunning on it is deterministic on every Unix.
func deadPid(t *testing.T) int {
	t.Helper()
	cmd := exec.Command("true")
	if err := cmd.Run(); err != nil {
		t.Fatalf("spawn fixture process: %v", err)
	}
	return cmd.Process.Pid
}

func TestEntryReadyTimeout(t *testing.T) {
	tests := []struct {
		name   string
		checks map[string]config.HealthCheckConfig
		want   time.Duration
	}{
		{
			name:   "no checks falls back to the default ceiling",
			checks: nil,
			want:   30 * time.Second,
		},
		{
			name: "no ready_timeout declared falls back to the default ceiling",
			checks: map[string]config.HealthCheckConfig{
				"http": {Type: "http", URL: "http://localhost:1/x"},
			},
			want: 30 * time.Second,
		},
		{
			name: "largest declared ready_timeout wins",
			checks: map[string]config.HealthCheckConfig{
				"slow": {Type: "command", Command: "true", ReadyTimeout: 12},
				"fast": {Type: "command", Command: "true", ReadyTimeout: 5},
			},
			want: 12 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := entryReadyTimeout(config.LifecycleEntry{Name: "e", HealthChecks: tt.checks})
			if got != tt.want {
				t.Errorf("entryReadyTimeout() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestWaitEntryReadySucceedsWhenCheckPasses(t *testing.T) {
	o := newWaitReadyTestOrchestrator(t)
	entry := config.LifecycleEntry{
		Name: "ok",
		HealthChecks: map[string]config.HealthCheckConfig{
			"check": {Type: "command", Command: "true", ReadyTimeout: 5},
		},
	}
	if err := o.waitEntryReady(context.Background(), entry, o.env); err != nil {
		t.Fatalf("waitEntryReady() error = %v, want nil", err)
	}
}

// TestWaitEntryReadyHonorsReadyTimeout covers TASK-402 gap 2: the composition
// readiness wait must end within the entry's ready_timeout instead of polling
// forever, and the error must name the ceiling that fired.
func TestWaitEntryReadyHonorsReadyTimeout(t *testing.T) {
	o := newWaitReadyTestOrchestrator(t)
	entry := config.LifecycleEntry{
		Name: "never-ready",
		HealthChecks: map[string]config.HealthCheckConfig{
			"check": {Type: "command", Command: "false", ReadyTimeout: 1},
		},
	}
	start := time.Now()
	err := o.waitEntryReady(context.Background(), entry, o.env)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("waitEntryReady() error = nil, want a ready_timeout failure")
	}
	if !strings.Contains(err.Error(), "not ready within 1s (ready_timeout)") {
		t.Errorf("error = %q, want it to name the ready_timeout ceiling", err)
	}
	if elapsed >= livenessPollInterval*3 {
		t.Errorf("wait took %s, want it bounded by the 1s ready_timeout", elapsed)
	}
}

// TestWaitEntryReadyFastFailsOnDeadPidfile covers TASK-402 gap 1: a native run
// command that died right after spawn must fail the wait within one poll
// interval — long before the ready_timeout deadline — with the pid and the log
// path in the error.
func TestWaitEntryReadyFastFailsOnDeadPidfile(t *testing.T) {
	o := newWaitReadyTestOrchestrator(t)
	entry := config.LifecycleEntry{
		Name: "dead-on-arrival",
		// A long ceiling proves the fast-fail fires on liveness, not the deadline.
		Process: &config.ProcessPluginConfig{},
		HealthChecks: map[string]config.HealthCheckConfig{
			"check": {Type: "command", Command: "false", ReadyTimeout: 60},
		},
	}
	pidPath := entryPidPath(o.cfg.FileDir(), entry.Name)
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	pid := deadPid(t)
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(pid)), 0o644); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	err := o.waitEntryReady(context.Background(), entry, o.env)
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("waitEntryReady() error = nil, want a fast-fail on the dead pidfile")
	}
	want := "native process (pid " + strconv.Itoa(pid) + ") exited before becoming ready — see " +
		entryLogPath(o.cfg.FileDir(), entry.Name)
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want it to contain %q", err, want)
	}
	if elapsed >= 10*time.Second {
		t.Errorf("wait took %s, want a fast-fail within one poll interval + slack", elapsed)
	}
}

// TestWaitEntryReadyIgnoresNonProcessEntryPidfiles pins the entryPid guard: without
// a Process plugin there is no pidfile contract, so the wait rides on the health
// checks alone even if a stray file sits at the entry's pid path.
func TestWaitEntryReadyIgnoresNonProcessEntryPidfiles(t *testing.T) {
	o := newWaitReadyTestOrchestrator(t)
	entry := config.LifecycleEntry{
		Name: "compose-entry",
		HealthChecks: map[string]config.HealthCheckConfig{
			"check": {Type: "command", Command: "true", ReadyTimeout: 5},
		},
	}
	pidPath := entryPidPath(o.cfg.FileDir(), entry.Name)
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(deadPid(t))), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := o.waitEntryReady(context.Background(), entry, o.env); err != nil {
		t.Fatalf("waitEntryReady() error = %v, want nil (no Process plugin, no liveness probe)", err)
	}
}
