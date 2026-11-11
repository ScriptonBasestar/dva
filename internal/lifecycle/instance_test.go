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
	for _, pctx := range []*PluginContext{a, b} {
		if _, err := os.Stat(pctx.LogFile("web")); err != nil {
			t.Errorf("instance %s left no log in its own directory: %v", pctx.Instance, err)
		}
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
	if got := instanceComposeConfig(plain, declared, dir); got != declared {
		t.Error("the ordinary run must keep its compose config untouched")
	}
	if got := instanceComposeConfig(&PluginContext{ConfigDir: dir, Env: env, Instance: "a"}, declared, dir); got.ProjectName != "shop-a" {
		t.Errorf("declared project = %q, want shop-a", got.ProjectName)
	}
	if declared.ProjectName != "${PROJ}" {
		t.Errorf("the declaration was modified: %q", declared.ProjectName)
	}
	if got := instanceComposeConfig(&PluginContext{ConfigDir: dir, Env: env, Instance: "b"}, &config.ComposePluginConfig{}, dir); got.ProjectName != "myapp-b" {
		t.Errorf("derived project = %q, want myapp-b", got.ProjectName)
	}
	if got := instanceComposeConfig(&PluginContext{ConfigDir: dir, Env: env, Instance: "b"}, nil, dir); got.ProjectName != "myapp-b" {
		t.Errorf("nil config project = %q, want myapp-b", got.ProjectName)
	}
}

// Compose requires an explicit project name to be lowercase letters, digits, - and _; it only
// normalizes the name it derives itself. A directory name with dots or spaces and a variant name
// with capitals must not reach it as typed.
func TestInstanceProjectNameIsNormalized(t *testing.T) {
	env := config.NewEnvironment(nil, "", "")
	for _, tc := range []struct{ dir, instance, want string }{
		{"my.app", "A", "my-app-a"},
		{"my app", "b", "my-app-b"},
		{"MyApp", "x_1", "myapp-x_1"},
		{".hidden", "v", "hidden-v"},
	} {
		pctx := &PluginContext{ConfigDir: filepath.Join("/w", tc.dir), Env: env, Instance: tc.instance}
		if got := instanceProjectName(pctx, "", nil, pctx.ConfigDir); got != tc.want {
			t.Errorf("dir %q instance %q = %q, want %q", tc.dir, tc.instance, got, tc.want)
		}
	}
}

// The argv a compose plugin really runs carries the instance's project, for docker compose and
// podman-compose alike; the ordinary run carries none it did not declare.
func TestComposePluginsRunInstanceProject(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "myapp")
	env := config.NewEnvironment(nil, dir, dir)
	mk := func(instance string, entry *config.LifecycleEntry) *PluginContext {
		return &PluginContext{Entry: entry, ConfigDir: dir, Instance: instance, Env: env, Logger: slog.Default()}
	}
	hasProject := func(args []string, want string) bool {
		for i := 0; i+1 < len(args); i++ {
			if (args[i] == "--project-name" || args[i] == "-p") && args[i+1] == want {
				return true
			}
		}
		return false
	}
	hasAnyProject := func(args []string) bool {
		for _, a := range args {
			if a == "--project-name" || a == "-p" {
				return true
			}
		}
		return false
	}

	docker := &config.LifecycleEntry{Name: "db", Compose: &config.ComposePluginConfig{Files: []string{"compose.yml"}}}
	_, args, err := (&ComposePlugin{}).buildArgs(mk("a", docker), []string{"up"})
	if err != nil {
		t.Fatal(err)
	}
	if !hasProject(args, "myapp-a") {
		t.Errorf("docker compose argv has no instance project: %v", args)
	}
	_, plain, err := (&ComposePlugin{}).buildArgs(mk("", docker), []string{"up"})
	if err != nil {
		t.Fatal(err)
	}
	if hasAnyProject(plain) {
		t.Errorf("ordinary docker compose run gained a project name: %v", plain)
	}

	podman := &config.LifecycleEntry{Name: "db", PodmanCompose: &config.PodmanComposePluginConfig{Files: []string{"compose.yml"}}}
	_, pargs := (&PodmanComposePlugin{}).buildArgs(mk("a", podman), []string{"up"})
	_, pargsB := (&PodmanComposePlugin{}).buildArgs(mk("b", podman), []string{"up"})
	if !hasProject(pargs, "myapp-a") || !hasProject(pargsB, "myapp-b") {
		t.Errorf("podman-compose argv has no instance project: %v / %v", pargs, pargsB)
	}
	_, pplain := (&PodmanComposePlugin{}).buildArgs(mk("", podman), []string{"up"})
	if hasAnyProject(pplain) {
		t.Errorf("ordinary podman-compose run gained a project name: %v", pplain)
	}
}

// An instance's process leaves its log in the instance's own directory (two instances writing
// one log file would interleave), and the orchestrator reads readiness from the instance's pid.
func TestOrchestratorReadsInstancePidFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte("version: \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	o := NewOrchestrator(cfg, config.NewEnvironment(nil, dir, dir))
	entry := config.LifecycleEntry{Name: "web", Process: &config.ProcessPluginConfig{Command: "x"}}
	pctx := &PluginContext{ConfigDir: dir, Instance: "a"}
	if err := os.MkdirAll(filepath.Dir(pctx.PidFile("web")), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pctx.PidFile("web"), []byte("4242\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, known := o.entryPid(entry); known {
		t.Error("an orchestrator without an instance read an instance's pid file")
	}
	o.SetInstance("a")
	if pid, known := o.entryPid(entry); !known || pid != 4242 {
		t.Errorf("instance a pid = %d known=%v, want 4242 true", pid, known)
	}
	if got, want := o.entryLogPath("web"), pctx.LogFile("web"); got != want {
		t.Errorf("instance log path = %s, want %s", got, want)
	}
}

// An instance's project is what compose derives for the plain run plus the instance: entries whose
// compose files share a directory share one project (one network), as they do in a plain run;
// entries in different directories stay separate; a top-level `name:` wins over the directory.
func TestInstanceProjectFollowsComposeDerivation(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		t.Helper()
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("infra/db.yml", "services: {}\n")
	write("infra/cache.yml", "services: {}\n")
	write("other/web.yml", "services: {}\n")
	write("named/c.yml", "name: Fancy.Shop\nservices: {}\n")
	env := config.NewEnvironment(nil, root, root)
	name := func(instance string, files ...string) string {
		return instanceProjectName(&PluginContext{ConfigDir: root, Env: env, Instance: instance}, "", files, root)
	}
	if a, b := name("a", "infra/db.yml"), name("a", "infra/cache.yml"); a != b || a != "infra-a" {
		t.Errorf("entries in one directory: %q vs %q, want both infra-a", a, b)
	}
	if name("a", "infra/db.yml") == name("a", "other/web.yml") {
		t.Error("entries in different directories share a project")
	}
	if name("a", "infra/db.yml") == name("b", "infra/db.yml") {
		t.Error("instances a and b share a project")
	}
	if got := name("a", "named/c.yml"); got != "fancy-shop-a" {
		t.Errorf("top-level name: project = %q, want fancy-shop-a", got)
	}
	if got := instanceProjectName(&PluginContext{Env: env, Instance: "a"}, "", nil, "/w/Plain"); got != "plain-a" {
		t.Errorf("no files: project = %q, want plain-a", got)
	}
}
