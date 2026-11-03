//go:build !windows

package lifecycle

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func newUpReadinessOrchestrator(t *testing.T, entry *config.LifecycleEntry) *Orchestrator {
	t.Helper()
	base := newWaitReadyTestOrchestrator(t)
	base.cfg.Stack = map[string]*config.LifecycleEntry{
		"app": entry,
		"next": {
			Order:  1,
			Script: &config.ScriptPluginConfig{Up: "touch next-started"},
		},
	}
	return NewOrchestrator(base.cfg, base.env)
}

func assertReadinessStoppedUp(t *testing.T, o *Orchestrator) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(o.cfg.FileDir(), "next-started")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("next entry must not start after readiness failure: stat error = %v", err)
	}
}

func TestOrchestratorUpReadinessTimeout(t *testing.T) {
	for _, slowProbe := range []bool{false, true} {
		t.Run(strconv.FormatBool(slowProbe), func(t *testing.T) {
			check := config.HealthCheckConfig{Type: "command", Command: "false", ReadyTimeout: 1}
			if slowProbe {
				// The readiness ceiling must also bound an in-flight health probe.
				release := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					<-release
					w.WriteHeader(http.StatusServiceUnavailable)
				}))
				defer server.Close()
				defer close(release)
				check = config.HealthCheckConfig{Type: "http", URL: server.URL, Timeout: 10, ReadyTimeout: 1}
			}
			o := newUpReadinessOrchestrator(t, &config.LifecycleEntry{
				Script:       &config.ScriptPluginConfig{Up: "true"},
				HealthChecks: map[string]config.HealthCheckConfig{"check": check},
			})
			// A fallback bounds a regression without supplying the readiness deadline.
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
			defer cancel()
			start := time.Now()
			err := o.Up(ctx, UpOptions{Wait: true})
			elapsed := time.Since(start)
			if err == nil || !strings.Contains(err.Error(), `entry "app" not ready within 1s (ready_timeout)`) {
				t.Fatalf("Up error = %v, want the declared readiness timeout", err)
			}
			if elapsed < time.Second || elapsed >= 4*time.Second {
				t.Errorf("Up took %s, want the 1s readiness ceiling with scheduling slack", elapsed)
			}
			assertReadinessStoppedUp(t, o)
		})
	}
}

func TestOrchestratorUpReadinessDeadNativeProcess(t *testing.T) {
	o := newUpReadinessOrchestrator(t, &config.LifecycleEntry{
		Process: &config.ProcessPluginConfig{Command: "exit 0"},
		HealthChecks: map[string]config.HealthCheckConfig{
			"check": {Type: "command", Command: "false", ReadyTimeout: 60},
		},
	})
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := o.Down(ctx, DownOptions{}); err != nil {
			t.Errorf("clean up native fixture: %v", err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()
	start := time.Now()
	err := o.Up(ctx, UpOptions{Wait: true})
	elapsed := time.Since(start)
	pidBytes, readErr := os.ReadFile(entryPidPath(o.cfg.FileDir(), "app"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	pid, parseErr := strconv.Atoi(strings.TrimSpace(string(pidBytes)))
	if parseErr != nil || pid <= 0 {
		t.Fatalf("invalid native fixture PID %q: %v", pidBytes, parseErr)
	}
	want := `entry "app" native process (pid ` + strconv.Itoa(pid) +
		") exited before becoming ready — see " + entryLogPath(o.cfg.FileDir(), "app")
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Up error = %v, want diagnostic %q", err, want)
	}
	if elapsed >= livenessPollInterval*3 {
		t.Errorf("dead native process took %s to fail, want one liveness interval plus slack", elapsed)
	}
	assertReadinessStoppedUp(t, o)
}

func TestOrchestratorUpReadinessCallerCancellation(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(strconv.FormatBool(deadline), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			wantErr := context.Canceled
			if deadline {
				cancel()
				ctx, cancel = context.WithTimeout(t.Context(), 300*time.Millisecond)
				wantErr = context.DeadlineExceeded
			}
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if !deadline {
					cancel() // Cancel during readiness, after plugin.Up has succeeded.
				}
				w.WriteHeader(http.StatusServiceUnavailable)
			}))
			defer server.Close()
			o := newUpReadinessOrchestrator(t, &config.LifecycleEntry{
				Script: &config.ScriptPluginConfig{Up: "true"},
				HealthChecks: map[string]config.HealthCheckConfig{
					"check": {Type: "http", URL: server.URL, ReadyTimeout: 60},
				},
			})
			start := time.Now()
			err := o.Up(ctx, UpOptions{Wait: true})
			if !errors.Is(err, wantErr) || !strings.Contains(err.Error(), `entry "app" readiness wait cancelled`) {
				t.Fatalf("Up error = %v, want readiness cancellation wrapping %v", err, wantErr)
			}
			if elapsed := time.Since(start); elapsed >= time.Second {
				t.Errorf("caller cancellation took %s to propagate", elapsed)
			}
			assertReadinessStoppedUp(t, o)
		})
	}
}

func TestOrchestratorUpReadinessNoWait(t *testing.T) {
	o := newUpReadinessOrchestrator(t, &config.LifecycleEntry{
		Script: &config.ScriptPluginConfig{Up: "true"},
		HealthChecks: map[string]config.HealthCheckConfig{
			"check": {Type: "command", Command: "touch health-probed; false", ReadyTimeout: 1},
		},
	})
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := o.Up(ctx, UpOptions{Wait: false}); err != nil {
		t.Fatalf("Up without readiness wait: %v", err)
	}
	if _, err := os.Stat(filepath.Join(o.cfg.FileDir(), "health-probed")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Wait=false must skip health probes: stat error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(o.cfg.FileDir(), "next-started")); err != nil {
		t.Fatalf("Wait=false must continue to the next entry: %v", err)
	}
}
