//go:build !windows

package lifecycle

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// escapedFixture writes a dead pid into the entry's pidfile and opens a listener standing in
// for the server that outlived it. It returns the entry, whose tcp health check probes that
// listener.
func escapedFixture(t *testing.T, configDir string) config.LifecycleEntry {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	entry := config.LifecycleEntry{
		Name:    "web",
		Process: &config.ProcessPluginConfig{Command: "true"},
		HealthChecks: map[string]config.HealthCheckConfig{
			"ready": {Type: "tcp", Address: ln.Addr().String(), ReadyTimeout: 60},
		},
	}
	pidPath := entryPidPath(configDir, entry.Name)
	if err := os.MkdirAll(filepath.Dir(pidPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(deadPid(t))), 0o644); err != nil {
		t.Fatal(err)
	}
	return entry
}

// A probe that passes after the tracked pid exited is served by an escaped process: `up` must
// fail naming the address instead of reporting the entry ready.
func TestWaitEntryReadyEscapedServer(t *testing.T) {
	o := newWaitReadyTestOrchestrator(t)
	entry := escapedFixture(t, o.cfg.FileDir())
	err := o.waitEntryReady(context.Background(), entry, o.env)
	if err == nil {
		t.Fatal("waitEntryReady() = nil, want an escaped-from-tracking error")
	}
	address := entry.HealthChecks["ready"].Address
	if !strings.Contains(err.Error(), "escaped from tracking") || !strings.Contains(err.Error(), address) {
		t.Fatalf("error = %q, want escaped from tracking naming %s", err, address)
	}
}

// A dead pid with no answering endpoint stays an ordinary exit, not an escape.
func TestWaitEntryReadyEscapedOnlyWhenEndpointAnswers(t *testing.T) {
	o := newWaitReadyTestOrchestrator(t)
	entry := escapedFixture(t, o.cfg.FileDir())
	entry.HealthChecks = map[string]config.HealthCheckConfig{
		"ready": {Type: "command", Command: "false", ReadyTimeout: 60},
	}
	err := o.waitEntryReady(context.Background(), entry, o.env)
	if err == nil || strings.Contains(err.Error(), "escaped") || !strings.Contains(err.Error(), "exited before becoming ready") {
		t.Fatalf("error = %v, want the plain exited-before-ready error", err)
	}
}

func TestStatusEscapedServer(t *testing.T) {
	dir := t.TempDir()
	entry := escapedFixture(t, dir)
	pctx := &PluginContext{ConfigDir: dir, Entry: &entry}
	statuses, err := (&ProcessPlugin{}).Status(context.Background(), pctx)
	if err != nil {
		t.Fatal(err)
	}
	address := entry.HealthChecks["ready"].Address
	if len(statuses) != 1 || statuses[0].State != "error" || statuses[0].Extra[escapedExtraKey] != address {
		t.Fatalf("statuses = %+v, want state error with escaped=%s", statuses, address)
	}

	out := captureStdout(func() { printServiceTable(statuses) })
	if !strings.Contains(out, "! web: escaped from tracking") || !strings.Contains(out, address) {
		t.Fatalf("table = %q, want the escaped line", out)
	}
}

func TestStatusEscapedNotWhenEndpointSilent(t *testing.T) {
	dir := t.TempDir()
	entry := escapedFixture(t, dir)
	entry.HealthChecks = nil
	statuses, err := (&ProcessPlugin{}).Status(context.Background(), &PluginContext{ConfigDir: dir, Entry: &entry})
	if err != nil || len(statuses) != 1 || statuses[0].State != "stopped" {
		t.Fatalf("statuses = %+v err = %v, want stopped", statuses, err)
	}
}
