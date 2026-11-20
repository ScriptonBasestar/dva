//go:build !windows

package lifecycle

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func identityPctx(dir, startedBy string) *PluginContext {
	return &PluginContext{
		Entry:     &config.LifecycleEntry{Name: "api", Process: &config.ProcessPluginConfig{Command: "sleep 300"}},
		Env:       config.NewEnvironment(nil, dir, dir),
		ConfigDir: dir,
		StartedBy: startedBy,
		Logger:    slog.Default(),
	}
}

func readPid(t *testing.T, pctx *PluginContext, name string) int {
	t.Helper()
	data, err := os.ReadFile(pctx.PidFile(name))
	if err != nil {
		t.Fatalf("read pid file: %v", err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("pid file: %v", err)
	}
	return pid
}

// The starter is recorded with the pid, survives a stop (the pid file stays for a restart) and
// goes away with the pid file on down.
func TestIdentityFollowsThePidFile(t *testing.T) {
	dir := t.TempDir()
	pctx := identityPctx(dir, "compare:review")
	p := &ProcessPlugin{}
	if _, err := p.Up(context.Background(), pctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	pid := readPid(t, pctx, "api")
	killProcessGroupInCleanup(t, pid)

	id, err := ReadIdentity(dir, "", "api")
	if err != nil {
		t.Fatalf("ReadIdentity: %v", err)
	}
	if id.StartedBy != "compare:review" || id.PID != pid || id.Path != filepath.Clean(dir) {
		t.Fatalf("identity = %+v, want compare:review pid %d in %s", id, pid, dir)
	}

	if err := p.Stop(context.Background(), pctx); err != nil {
		t.Fatalf("stop: %v", err)
	}
	if _, err := ReadIdentity(dir, "", "api"); err != nil {
		t.Fatalf("identity after stop: %v (stop keeps the pid file, so it keeps the identity)", err)
	}
	if err := p.Down(context.Background(), pctx); err != nil {
		t.Fatalf("down: %v", err)
	}
	if _, err := ReadIdentity(dir, "", "api"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("identity after down: err = %v, want not exist", err)
	}
}

func TestIdentityEmptyStarterIsCwd(t *testing.T) {
	dir := t.TempDir()
	pctx := identityPctx(dir, "")
	if _, err := (&ProcessPlugin{}).Up(context.Background(), pctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	killProcessGroupInCleanup(t, readPid(t, pctx, "api"))
	id, err := ReadIdentity(dir, "", "api")
	if err != nil {
		t.Fatalf("ReadIdentity: %v", err)
	}
	if id.StartedBy != StartedByCwd {
		t.Fatalf("started_by = %q, want %q", id.StartedBy, StartedByCwd)
	}
}

// An identity is only trusted while it still describes the pid file beside it.
func TestReadIdentityRejectsStale(t *testing.T) {
	dir := t.TempDir()
	pctx := identityPctx(dir, "cwd")
	if err := os.MkdirAll(pctx.StateDir(config.PidsDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeIdentity(pctx, "api", 4242); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIdentity(dir, "", "api"); !errors.Is(err, ErrIdentityStale) {
		t.Fatalf("no pid file: err = %v, want ErrIdentityStale", err)
	}
	if err := os.WriteFile(pctx.PidFile("api"), []byte("4343"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIdentity(dir, "", "api"); !errors.Is(err, ErrIdentityStale) {
		t.Fatalf("other pid: err = %v, want ErrIdentityStale", err)
	}
	if err := os.WriteFile(pctx.PidFile("api"), []byte("4242"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIdentity(dir, "", "api"); err != nil {
		t.Fatalf("matching pid: %v", err)
	}
	// A copied state directory names the checkout it was written in, not this one.
	other := t.TempDir()
	if err := os.CopyFS(other, os.DirFS(dir)); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadIdentity(other, "", "api"); !errors.Is(err, ErrIdentityStale) {
		t.Fatalf("other directory: err = %v, want ErrIdentityStale", err)
	}
}

// TestStartedProcessOutlivesTheCallingSession runs `up` in a helper process that stands for
// the dva invocation of an agent session, then hangs up the helper's whole process group the
// way a closing terminal does. The started entry leads its own group, so it keeps running:
// a session ending is never an automatic stop (TASK-538).
func TestStartedProcessOutlivesTheCallingSession(t *testing.T) {
	if dir := os.Getenv("DVA_TEST_SESSION_UP_DIR"); dir != "" {
		pctx := identityPctx(dir, "")
		if _, err := (&ProcessPlugin{}).Up(context.Background(), pctx); err != nil {
			os.Exit(3)
		}
		// Stay alive until hung up, like a session that is still open.
		time.Sleep(time.Minute)
		os.Exit(0)
	}

	dir := t.TempDir()
	session := exec.Command(os.Args[0], "-test.run=^TestStartedProcessOutlivesTheCallingSession$")
	session.Env = append(os.Environ(), "DVA_TEST_SESSION_UP_DIR="+dir)
	session.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := session.Start(); err != nil {
		t.Fatal(err)
	}
	sessionGroup := session.Process.Pid
	killProcessGroupInCleanup(t, sessionGroup)

	pctx := identityPctx(dir, "")
	var pid int
	deadline := time.Now().Add(10 * time.Second)
	for pid == 0 {
		if _, err := ReadIdentity(dir, "", "api"); err == nil {
			pid = readPid(t, pctx, "api")
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the session did not start the entry")
		}
		time.Sleep(20 * time.Millisecond)
	}
	killProcessGroupInCleanup(t, pid)
	if pid == sessionGroup {
		t.Fatalf("entry pid %d is the session's group leader", pid)
	}

	if err := syscall.Kill(-sessionGroup, syscall.SIGHUP); err != nil {
		t.Fatalf("hang up session group: %v", err)
	}
	if err := session.Wait(); err == nil {
		t.Fatal("session exited cleanly; it should have died of SIGHUP")
	}
	time.Sleep(200 * time.Millisecond)
	if !IsProcessRunning(pid) {
		t.Fatalf("entry pid %d died with the session that started it", pid)
	}
}
