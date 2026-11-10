package lifecycle

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// The ordinary run (no instance) keeps the pid and log paths it always had; an instance moves
// both into its own subdirectory.
func TestPluginContextInstancePaths(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "proj")
	plain := &PluginContext{ConfigDir: dir}
	if got, want := plain.PidFile("web"), filepath.Join(dir, ".sb", "dva", "pids", "web.pid"); got != want {
		t.Errorf("plain pid = %s, want %s", got, want)
	}
	if got, want := plain.LogFile("web"), filepath.Join(dir, ".sb", "dva", "logs", "web.log"); got != want {
		t.Errorf("plain log = %s, want %s", got, want)
	}
	a := &PluginContext{ConfigDir: dir, Instance: "a"}
	if got, want := a.PidFile("web"), filepath.Join(dir, ".sb", "dva", "pids", "a", "web.pid"); got != want {
		t.Errorf("instance pid = %s, want %s", got, want)
	}
	if got, want := a.LogFile("web"), filepath.Join(dir, ".sb", "dva", "logs", "a", "web.log"); got != want {
		t.Errorf("instance log = %s, want %s", got, want)
	}
}

// Two instances of the same entry in one directory are two processes with two pid files: the
// second Up must not see the first's pid file as its own (and so skip or kill it), and stopping
// one must leave the other running.
func TestProcessPluginInstancesDoNotShareState(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process plugin requires Unix process groups")
	}
	dir := t.TempDir()
	mk := func(instance string) *PluginContext {
		return &PluginContext{
			Entry:     &config.LifecycleEntry{Name: "web", Process: &config.ProcessPluginConfig{Command: "sleep 300"}},
			ConfigDir: dir,
			Instance:  instance,
			Env:       config.NewEnvironment(nil, dir, dir),
			Logger:    slog.Default(),
		}
	}
	a, b := mk("a"), mk("b")
	p := &ProcessPlugin{}
	for _, pctx := range []*PluginContext{a, b} {
		if _, err := p.Up(context.Background(), pctx); err != nil {
			t.Fatalf("Up %s: %v", pctx.Instance, err)
		}
	}
	pidA, pidB := readPIDFile(t, a.PidFile("web")), readPIDFile(t, b.PidFile("web"))
	killProcessGroupInCleanup(t, pidA)
	killProcessGroupInCleanup(t, pidB)
	if pidA == pidB {
		t.Fatalf("both instances share pid %d", pidA)
	}
	if _, err := os.Stat((&PluginContext{ConfigDir: dir}).PidFile("web")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("an instance must not write the ordinary pid file: %v", err)
	}

	if err := p.Down(context.Background(), a); err != nil {
		t.Fatalf("Down a: %v", err)
	}
	if IsProcessRunning(pidA) {
		t.Errorf("instance a (pid %d) still running after its Down", pidA)
	}
	if !IsProcessRunning(pidB) {
		t.Errorf("stopping instance a stopped instance b (pid %d)", pidB)
	}
	if err := p.Down(context.Background(), b); err != nil {
		t.Fatalf("Down b: %v", err)
	}
}

// An instance is its own compose project, so two of them in one directory cannot recreate each
// other's containers; the ordinary run keeps the project name it had.
func TestInstanceComposeProjectName(t *testing.T) {
	env := config.NewEnvironment(map[string]string{"PROJ": "Shop"}, "", "")
	dir := filepath.Join(t.TempDir(), "MyApp")
	declared := &config.ComposePluginConfig{ProjectName: "${PROJ}"}

	plain := &PluginContext{ConfigDir: dir, Env: env}
	if got := instanceComposeConfig(plain, declared); got != declared {
		t.Error("the ordinary run must keep its compose config untouched")
	}
	if got := instanceComposeConfig(&PluginContext{ConfigDir: dir, Env: env, Instance: "a"}, declared); got.ProjectName != "Shop-a" {
		t.Errorf("declared project = %q, want Shop-a", got.ProjectName)
	}
	if declared.ProjectName != "${PROJ}" {
		t.Errorf("the declaration was modified: %q", declared.ProjectName)
	}
	if got := instanceComposeConfig(&PluginContext{ConfigDir: dir, Env: env, Instance: "b"}, &config.ComposePluginConfig{}); got.ProjectName != "myapp-b" {
		t.Errorf("derived project = %q, want myapp-b", got.ProjectName)
	}
	if got := instanceComposeConfig(&PluginContext{ConfigDir: dir, Env: env, Instance: "b"}, nil); got.ProjectName != "myapp-b" {
		t.Errorf("nil config project = %q, want myapp-b", got.ProjectName)
	}
}
