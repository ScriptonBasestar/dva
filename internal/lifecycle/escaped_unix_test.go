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

// stoppedEntryFixture starts a real process for a tcp-checked entry, stops it with `dva stop`
// semantics (haltProcess) and then opens an unrelated listener on the entry's health address.
func stoppedEntryFixture(t *testing.T) (*PluginContext, config.LifecycleEntry) {
	t.Helper()
	probe, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := probe.Addr().String()
	_ = probe.Close()
	dir := t.TempDir()
	entry := config.LifecycleEntry{
		Name:         "web",
		Process:      &config.ProcessPluginConfig{Command: "sleep 60"},
		HealthChecks: map[string]config.HealthCheckConfig{"ready": {Type: "tcp", Address: address}},
	}
	pctx := &PluginContext{ConfigDir: dir, Entry: &entry, Env: config.NewEnvironment(nil, dir, dir)}
	if err := startLocalProcess(entry.Name, entry.Process.Command, dir, pctx); err != nil {
		t.Fatal(err)
	}
	if err := (&ProcessPlugin{}).Stop(context.Background(), pctx); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", address)
	if err != nil {
		t.Skipf("health address %s taken before the stand-in listener: %v", address, err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	return pctx, entry
}

// After `dva stop` ended the process, an unrelated listener on the health port is not an
// escaped server (TASK-632).
func TestStatusNotEscapedAfterStop(t *testing.T) {
	pctx, _ := stoppedEntryFixture(t)
	statuses, err := (&ProcessPlugin{}).Status(context.Background(), pctx)
	if err != nil || len(statuses) != 1 || statuses[0].State != "stopped" || statuses[0].Extra[escapedExtraKey] != "" {
		t.Fatalf("statuses = %+v err = %v, want stopped without escaped", statuses, err)
	}
}

// A new start forgets the earlier stop, so a server that later daemonizes is reported again.
func TestStatusEscapedAfterRestartClearsHalted(t *testing.T) {
	pctx, entry := stoppedEntryFixture(t)
	if err := startLocalProcess(entry.Name, "true", pctx.ConfigDir, pctx); err != nil {
		t.Fatal(err)
	}
	if wasHalted(pctx, entry.Name) {
		t.Fatal("halted marker survived a new start")
	}
	pidData, err := os.ReadFile(pctx.PidFile(entry.Name))
	if err != nil {
		t.Fatal(err)
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if !waitForProcessExit(context.Background(), pid, haltExitTimeout) {
		t.Fatalf("pid %d did not exit", pid)
	}
	statuses, err := (&ProcessPlugin{}).Status(context.Background(), pctx)
	if err != nil || len(statuses) != 1 || statuses[0].State != "error" || statuses[0].Extra[escapedExtraKey] == "" {
		t.Fatalf("statuses = %+v err = %v, want escaped after restart", statuses, err)
	}
}

// Stopping an entry whose pid was already dead does not mark it halted: the stop did not end
// whatever still answers, so status keeps reporting the escape.
func TestStopOfDeadPidDoesNotMarkHalted(t *testing.T) {
	dir := t.TempDir()
	entry := escapedFixture(t, dir)
	pctx := &PluginContext{ConfigDir: dir, Entry: &entry}
	if err := (&ProcessPlugin{}).Stop(context.Background(), pctx); err != nil {
		t.Fatal(err)
	}
	if wasHalted(pctx, entry.Name) {
		t.Fatal("stop of a dead pid recorded a halted marker")
	}
}
