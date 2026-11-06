package cirun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestStepReuseUsesDeclaredInputAndFreshBypasses(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("count\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	state := t.TempDir()
	step := config.CIStep{Name: "cached", Run: "printf x >> count", Reuse: &config.CIReuse{Inputs: []string{"input"}, Tools: []string{"sh"}, Outputs: []string{"count"}, MaxAge: "1h"}}
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)})
	if err != nil || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != "executed" {
		t.Fatalf("first run report=%+v err=%v", r.Steps, err)
	}
	r, err = Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)})
	if err != nil || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != "reused" {
		t.Fatalf("second run report=%+v err=%v", r.Steps, err)
	}
	count, err := os.ReadFile(filepath.Join(root, "count"))
	if err != nil || string(count) != "x" {
		t.Fatalf("cache did not avoid command: %q err=%v", count, err)
	}
	r, err = Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Fresh: true, Profile: profile(step)})
	if err != nil || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != "bypassed" {
		t.Fatalf("fresh report=%+v err=%v", r.Steps, err)
	}
	count, _ = os.ReadFile(filepath.Join(root, "count"))
	if string(count) != "xx" {
		t.Fatalf("fresh failed to execute: %q", count)
	}
}

func TestStepReuseRejectsChangedInputsAndMissingOutputs(t *testing.T) {
	root := t.TempDir()
	if err := runGit(root, "init"); err != nil {
		t.Skipf("git unavailable: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "input"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("count\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	state := t.TempDir()
	step := config.CIStep{Name: "cached", Run: "printf x >> count", Reuse: &config.CIReuse{Inputs: []string{"input"}, Tools: []string{"sh"}, Outputs: []string{"count"}, MaxAge: "1h"}}
	for _, mutation := range []func() error{
		func() error { return os.WriteFile(filepath.Join(root, "input"), []byte("two"), 0o600) },
		func() error { return os.Remove(filepath.Join(root, "count")) },
	} {
		if _, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)}); err != nil {
			t.Fatal(err)
		}
		if err := mutation(); err != nil {
			t.Fatal(err)
		}
		r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)})
		if err != nil || r.Steps[0].Reuse == nil || r.Steps[0].Reuse.Disposition != "executed" {
			t.Fatalf("mutation report=%+v err=%v", r.Steps, err)
		}
	}
	_, err := reuseInputs(context.Background(), root, []string{"../outside"})
	if err == nil || !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("escape err=%v", err)
	}
}

func TestReuseToolHashesMiseSelectedExecutable(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "bin")
	if err := os.Mkdir(bin, 0o700); err != nil {
		t.Fatal(err)
	}
	first := filepath.Join(dir, "first")
	second := filepath.Join(dir, "second")
	for path, body := range map[string]string{first: "#!/bin/sh\necho first\n", second: "#!/bin/sh\necho second\n"} {
		if err := os.WriteFile(path, []byte(body), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	mise := filepath.Join(bin, "mise")
	if err := os.WriteFile(mise, []byte("#!/bin/sh\nprintf '%s\\n' \"$DVA_REUSE_SELECTED\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("mise", filepath.Join(bin, "go")); err != nil {
		t.Fatal(err)
	}
	base := "PATH=" + bin
	// macOS temporary paths may be exposed through /var while EvalSymlinks
	// reports /private/var, as production reuseTool does.
	first, _ = filepath.EvalSymlinks(first)
	second, _ = filepath.EvalSymlinks(second)
	one, err := reuseTool(context.Background(), dir, "go", []string{base, "DVA_REUSE_SELECTED=" + first})
	if err != nil {
		t.Fatal(err)
	}
	two, err := reuseTool(context.Background(), dir, "go", []string{base, "DVA_REUSE_SELECTED=" + second})
	if err != nil {
		t.Fatal(err)
	}
	if one == two || !strings.HasPrefix(one, first+"\x00") || !strings.HasPrefix(two, second+"\x00") {
		t.Fatalf("mise selection was not fingerprinted: %q %q", one, two)
	}
}

func TestReuseProbeCountsAgainstStepTimeout(t *testing.T) {
	root, step := reuseFixture(t)
	state := t.TempDir()
	step.Timeout = "50ms"
	step.Reuse.Probes = []string{"sleep 2"}
	dependent := config.CIStep{Name: "dependent", Run: "touch should-not-run", DependsOn: []string{step.Name}}
	p := profile(step, dependent)
	no := false
	p.FailFast = &no
	p.MaxParallel = 1
	started := time.Now()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", Profile: p, StateDir: state, lockDirectory: state, Env: os.Environ()})
	if err == nil || len(r.Steps) != 2 || r.Steps[0].Status != "timed_out" || r.Steps[1].Status != "skipped" {
		t.Fatalf("probe timeout report=%+v err=%v", r.Steps, err)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("probe ignored step timeout: %s", elapsed)
	}
	if _, err := os.Stat(filepath.Join(root, "should-not-run")); !os.IsNotExist(err) {
		t.Fatalf("dependent ran after probe timeout: %v", err)
	}
}
