//go:build !windows

package lifecycle

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// foreignOwnerFixture stands in for an entry whose health port another process holds: the
// pidfile names a live `sleep` leading its own process group, while this test process — a
// different group — listens on the probed port. The real OS lookup is used end to end.
func foreignOwnerFixture(t *testing.T, configDir string) config.LifecycleEntry {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	if _, verified := lookupListenerPids(ln.Addr().(*net.TCPAddr).Port); !verified {
		t.Skip("owner lookup unavailable here (no lsof or /proc access)")
	}
	sleeper := exec.Command("sleep", "60")
	sleeper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := sleeper.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = sleeper.Process.Kill()
		_ = sleeper.Wait()
	})
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
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(sleeper.Process.Pid)), 0o644); err != nil {
		t.Fatal(err)
	}
	return entry
}

func declareExternal(entry *config.LifecycleEntry) {
	ready := entry.HealthChecks["ready"]
	ready.Owner = config.HealthOwnerExternal
	entry.HealthChecks["ready"] = ready
}

func TestWaitEntryReadyPortOwnerMismatch(t *testing.T) {
	o := newWaitReadyTestOrchestrator(t)
	entry := foreignOwnerFixture(t, o.cfg.FileDir())
	err := o.waitEntryReady(context.Background(), entry, o.env)
	if err == nil || !strings.Contains(err.Error(), "owner mismatch") {
		t.Fatalf("waitEntryReady() = %v, want an owner mismatch error", err)
	}
}

// A foreground `docker run -p` entry: Docker serves the port, and the entry says so.
func TestWaitEntryReadyPortOwnerExternal(t *testing.T) {
	o := newWaitReadyTestOrchestrator(t)
	entry := foreignOwnerFixture(t, o.cfg.FileDir())
	declareExternal(&entry)
	if err := o.waitEntryReady(context.Background(), entry, o.env); err != nil {
		t.Fatalf("waitEntryReady() = %v, want ready with owner: external", err)
	}
}

func TestStatusPortOwnerMismatchAndExternal(t *testing.T) {
	dir := t.TempDir()
	entry := foreignOwnerFixture(t, dir)
	address := entry.HealthChecks["ready"].Address

	statuses, err := (&ProcessPlugin{}).Status(context.Background(), &PluginContext{ConfigDir: dir, Entry: &entry})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].State != "error" || !strings.HasPrefix(statuses[0].Extra[portOwnerExtraKey], address) {
		t.Fatalf("statuses = %+v, want error with port_owner naming %s", statuses, address)
	}

	declareExternal(&entry)
	statuses, err = (&ProcessPlugin{}).Status(context.Background(), &PluginContext{ConfigDir: dir, Entry: &entry})
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 || statuses[0].State != "running" || statuses[0].Extra[portOwnerExternalExtraKey] != address ||
		statuses[0].Extra[portOwnerExtraKey] != "" {
		t.Fatalf("statuses = %+v, want running with port_owner_external=%s", statuses, address)
	}
}
