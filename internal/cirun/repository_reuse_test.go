package cirun

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func repositoryFixture(t *testing.T) (string, string, config.CIStep, Options) {
	t.Helper()
	root, step := reuseFixture(t)
	if err := os.Chmod(filepath.Join(root, "source", "a"), 0o644); err != nil {
		t.Fatal(err)
	}
	other, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	other = filepath.Join(other, "worktree")
	if err := runGit(root, "worktree", "add", "--detach", other, "HEAD"); err != nil {
		t.Fatal(err)
	}
	step.Reuse.Scope = "repository"
	state := t.TempDir()
	opts := Options{Root: root, ProfileName: "full", Profile: profile(step), StateDir: state, lockDirectory: state, Env: []string{"PATH=/usr/bin:/bin", "DVA_TEST_VALUE=one"}}
	return root, other, step, opts
}

func TestRepositoryReuseAndLookupWorktrees(t *testing.T) {
	root, other, step, opts := repositoryFixture(t)
	step.Reuse.Environment = []string{"DVA_TEST_VALUE"}
	opts.Profile = profile(step)
	first, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(other, "artifact"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts.Root = other
	look, err := Lookup(context.Background(), opts)
	if err != nil || !look.Covered || look.Steps[0].OriginalRunID != first.ID {
		t.Fatalf("lookup: %+v %v", look, err)
	}
	second, err := Run(context.Background(), opts)
	if err != nil || second.Steps[0].Reuse == nil || second.Steps[0].Reuse.Disposition != "reused" {
		t.Fatalf("reuse: %+v %v", second, err)
	}
	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T)
	}{
		{"content", func(t *testing.T) { t.Helper(); writeTestFile(t, filepath.Join(other, "source/a"), "changed") }},
		{"addition", func(t *testing.T) { t.Helper(); writeTestFile(t, filepath.Join(other, "source/b"), "new") }},
		{"deletion", func(t *testing.T) {
			t.Helper()
			if err := os.Remove(filepath.Join(other, "source/a")); err != nil {
				t.Fatal(err)
			}
		}},
		{"output", func(t *testing.T) {
			t.Helper()
			if err := os.Remove(filepath.Join(other, "artifact")); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Cleanup(func() {
				writeTestFile(t, filepath.Join(other, "source/a"), "one")
				_ = os.Remove(filepath.Join(other, "source/b"))
				writeTestFile(t, filepath.Join(other, "artifact"), "ok")
			})
			tc.mutate(t)
			r, err := Lookup(context.Background(), opts)
			if err != nil || r.Covered || r.Steps[0].State != "stale" {
				t.Fatalf("mutation covered: %+v %v", r, err)
			}
		})
	}
	opts.Env = []string{"PATH=/usr/bin:/bin", "DVA_TEST_VALUE=changed"}
	if r, err := Lookup(context.Background(), opts); err != nil || r.Covered {
		t.Fatalf("environment: %+v %v", r, err)
	}
	opts.Env = []string{"PATH=/usr/bin:/bin", "DVA_TEST_VALUE=one"}
	opts.Fresh = true
	if r, err := Lookup(context.Background(), opts); err != nil || r.Covered || r.Steps[0].Reason != "fresh" {
		t.Fatalf("fresh: %+v %v", r, err)
	}
	if fresh, err := Run(context.Background(), opts); err != nil || fresh.Steps[0].Reuse == nil || fresh.Steps[0].Reuse.Disposition != "bypassed" {
		t.Fatalf("fresh execution reused result: %+v %v", fresh, err)
	}
	_ = root
}

func writeTestFile(t *testing.T, path, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryReuseRejectsPathDependentInputs(t *testing.T) {
	root, _, step, opts := repositoryFixture(t)
	for _, tc := range []struct {
		name string
		edit func(*config.CIStep, *[]string)
	}{
		{"relative-path", func(_ *config.CIStep, env *[]string) { *env = []string{"PATH=.:/bin"} }},
		{"root-path", func(_ *config.CIStep, env *[]string) { *env = []string{"PATH=" + root + ":/bin"} }},
		{"absolute-workdir", func(s *config.CIStep, _ *[]string) { s.Workdir = root }},
		{"root-command", func(s *config.CIStep, _ *[]string) { s.Run = "echo " + root }},
		{"probe", func(s *config.CIStep, _ *[]string) { s.Reuse.Probes = []string{"true"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			copyStep := step
			copyReuse := *step.Reuse
			copyStep.Reuse = &copyReuse
			env := append([]string{}, opts.Env...)
			tc.edit(&copyStep, &env)
			if _, err := reuseFor(context.Background(), root, copyStep, env); err == nil {
				t.Fatal("unsafe repository reuse accepted")
			}
		})
	}
}

func TestCheckoutScopeIsStillIsolated(t *testing.T) {
	root, other, step, opts := repositoryFixture(t)
	step.Reuse.Scope = ""
	opts.Profile = profile(step)
	if _, err := Run(context.Background(), opts); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(other, "artifact"), "ok")
	opts.Root = other
	r, err := Lookup(context.Background(), opts)
	if err != nil || r.Covered || r.Steps[0].State != "absent" {
		t.Fatalf("checkout reuse leaked: %+v %v", r, err)
	}
	_ = root
}

func TestLookupNoCommandsOrStateWrites(t *testing.T) {
	root, step := reuseFixture(t)
	step.Run = "touch ran"
	step.Reuse.Probes = []string{"touch probed"}
	state := filepath.Join(t.TempDir(), "missing")
	opts := Options{Root: root, ProfileName: "full", Profile: profile(step), StateDir: state, Env: []string{"PATH=/usr/bin:/bin"}}
	r, err := Lookup(context.Background(), opts)
	if err != nil || r.Covered || r.Steps[0].State != "not-reusable" || !strings.Contains(r.Steps[0].Reason, "probe") {
		t.Fatalf("lookup: %+v %v", r, err)
	}
	for _, path := range []string{filepath.Join(root, "ran"), filepath.Join(root, "probed"), state, filepath.Join(root, ".git", "dva-ci-cache")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("lookup created %s: %v", path, err)
		}
	}
}

func TestLookupParentRunIsReadOnlyAndCompositionIsRejected(t *testing.T) {
	root, step := reuseFixture(t)
	writeTestFile(t, filepath.Join(root, "artifact"), "ok")
	step.Run = "touch must-not-run"
	jobs := 1
	p := profile(step)
	p.ToolJobs = &jobs
	env := []string{"PATH=/usr/bin:/bin", parentRunEnv + "=parent-marker"}
	inherited := setCIEnvironment(subprojectEnv(env, ""), config.CIFreshEnv, "0")
	plan, err := reuseFor(context.Background(), root, step, ciEnv(inherited, nil, jobs))
	if err != nil {
		t.Fatal(err)
	}
	if err := saveReuse(plan, root, "original", time.Now()); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(plan.cache)
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(t.TempDir(), "absent")
	opts := Options{Root: root, ProfileName: "full", Profile: p, Env: env, StateDir: state}
	result, err := Lookup(context.Background(), opts)
	if err != nil || !result.Covered || result.Steps[0].OriginalRunID != "original" {
		t.Fatalf("parent-only lookup: %+v %v", result, err)
	}
	after, err := os.ReadFile(plan.cache)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("lookup changed cache: %v", err)
	}
	for _, path := range []string{filepath.Join(root, "must-not-run"), state} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("lookup created %s: %v", path, err)
		}
	}
	opts.Env = append(opts.Env, config.CICompositionEnv+"=grant-marker")
	result, err = Lookup(context.Background(), opts)
	if err != nil || result.Covered || result.Steps[0].State != "not-reusable" || result.Steps[0].Reason != "composition context" {
		t.Fatalf("composition lookup: %+v %v", result, err)
	}
}

func TestLookupExpiredCacheIsStale(t *testing.T) {
	root, step := reuseFixture(t)
	writeTestFile(t, filepath.Join(root, "artifact"), "ok")
	env := ciEnv([]string{"PATH=/usr/bin:/bin"}, nil, 1)
	plan, err := reuseFor(context.Background(), root, step, env)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveReuse(plan, root, "old", time.Now().Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	jobs := 1
	p := profile(step)
	p.ToolJobs = &jobs
	r, err := Lookup(context.Background(), Options{Root: root, ProfileName: "full", Profile: p, Env: []string{"PATH=/usr/bin:/bin"}})
	if err != nil || r.Steps[0].State != "stale" || r.Steps[0].OriginalRunID != "old" {
		t.Fatalf("expired: %+v %v", r, err)
	}
}

func TestRepositoryReuseToolMutationAndReadOnlyHit(t *testing.T) {
	root, other, step, opts := repositoryFixture(t)
	tools, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	tool := filepath.Join(tools, "fixture-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	step.Reuse.Tools = []string{"fixture-tool"}
	opts.Env = []string{"PATH=" + tools + ":/usr/bin:/bin"}
	opts.Profile = profile(step)
	first, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(other, "artifact"), "ok")
	jobs, _ := toolWorkerBudget(opts.Profile, availableCPUs())
	plan, err := reuseFor(context.Background(), root, step, ciEnv(opts.Env, nil, jobs))
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(plan.cache)
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(plan.cache)
	if err != nil {
		t.Fatal(err)
	}
	opts.Root = other
	r, err := Lookup(context.Background(), opts)
	if err != nil || !r.Covered || r.Steps[0].OriginalRunID != first.ID {
		t.Fatalf("covered: %+v %v", r, err)
	}
	after, err := os.ReadFile(plan.cache)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatalf("lookup changed cache contents: %v", err)
	}
	afterInfo, err := os.Stat(plan.cache)
	if err != nil || !info.ModTime().Equal(afterInfo.ModTime()) {
		t.Fatalf("lookup changed cache timestamp: %v", err)
	}
	if err := os.WriteFile(tool, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r, err = Lookup(context.Background(), opts)
	if err != nil || r.Covered || r.Steps[0].State != "stale" {
		t.Fatalf("changed tool: %+v %v", r, err)
	}
}

func TestLookupRevalidatesEarlierCoveredStep(t *testing.T) {
	for _, mutation := range []string{"input", "output"} {
		t.Run(mutation, func(t *testing.T) {
			root, step := reuseFixture(t)
			writeTestFile(t, filepath.Join(root, ".gitignore"), "artifact\nsecond-artifact\n")
			step.Name = "first"
			second := config.CIStep{Name: "second", Run: "printf ok > second-artifact", Reuse: &config.CIReuse{Inputs: []string{"source"}, Tools: []string{"sh"}, Outputs: []string{"second-artifact"}, MaxAge: "1h"}}
			state := t.TempDir()
			opts := Options{Root: root, ProfileName: "full", Profile: profile(step, second), StateDir: state, lockDirectory: state, Env: []string{"PATH=/usr/bin:/bin"}}
			initial, err := Run(context.Background(), opts)
			if err != nil {
				t.Fatal(err)
			}
			observed := false
			r, err := lookup(context.Background(), opts, func(i int) {
				if i != 0 {
					return
				}
				observed = true
				path := "source/a"
				if mutation == "output" {
					path = "artifact"
				}
				writeTestFile(t, filepath.Join(root, path), "changed")
			})
			if err != nil || !observed || r.Covered || r.Steps[0].State != "stale" || r.Steps[0].OriginalRunID != initial.ID {
				t.Fatalf("mixed lookup accepted: %+v observed=%v err=%v", r, observed, err)
			}
		})
	}
}

func TestLookupFreshDoesNotReadMalformedCache(t *testing.T) {
	root, step := reuseFixture(t)
	jobs := 1
	p := profile(step)
	p.ToolJobs = &jobs
	env := []string{"PATH=/usr/bin:/bin"}
	plan, err := reuseFor(context.Background(), root, step, ciEnv(env, nil, jobs))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(plan.cache), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plan.cache, []byte("invalid"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := Options{Root: root, ProfileName: "full", Profile: p, Env: env}
	r, err := Lookup(context.Background(), opts)
	if err != nil || r.Steps[0].State != "stale" || r.Steps[0].Reason != "malformed cache" {
		t.Fatalf("malformed: %+v %v", r, err)
	}
	for _, profileFresh := range []bool{false, true} {
		opts.Fresh = !profileFresh
		opts.Profile.Fresh = profileFresh
		r, err = Lookup(context.Background(), opts)
		if err != nil || r.Steps[0].State != "not-reusable" || r.Steps[0].Reason != "fresh" {
			t.Fatalf("fresh=%v: %+v %v", profileFresh, r, err)
		}
	}
	writeTestFile(t, filepath.Join(root, "artifact"), "ok")
	if err := saveReuse(plan, root, "future", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	opts.Fresh, opts.Profile.Fresh = false, false
	r, err = Lookup(context.Background(), opts)
	if err != nil || r.Steps[0].State != "stale" || r.Steps[0].Reason != "future cache timestamp" || r.Steps[0].OriginalRunID != "future" {
		t.Fatalf("future: %+v %v", r, err)
	}
}

// macOS ships PATH entries that do not exist (/etc/paths.d/10-cryptex). Such an
// entry holds no tool, so repository scope keys its state instead of failing,
// and the key moves when the entry appears or changes kind.
func TestRepositoryReuseKeysMissingAndNonDirectoryPathEntries(t *testing.T) {
	root, other, step, opts := repositoryFixture(t)
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(base, "missing", "bin")
	file := filepath.Join(base, "file")
	writeTestFile(t, file, "x")
	underFile := filepath.Join(file, "bin")
	opts.Env = []string{"PATH=" + missing + ":" + underFile + ":" + file + ":/usr/bin:/bin"}
	opts.Profile = profile(step)
	jobs, _ := toolWorkerBudget(opts.Profile, availableCPUs())
	keyOf := func(env []string) string {
		t.Helper()
		plan, err := reuseFor(context.Background(), root, step, ciEnv(env, nil, jobs))
		if err != nil {
			t.Fatalf("repository reuse rejected PATH: %v", err)
		}
		return plan.key
	}
	first, err := Run(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(other, "artifact"), "ok")
	crossOpts := opts
	crossOpts.Root = other
	r, err := Lookup(context.Background(), crossOpts)
	if err != nil || !r.Covered || r.Steps[0].OriginalRunID != first.ID {
		t.Fatalf("missing PATH entry blocked cross-worktree reuse: %+v %v", r, err)
	}
	original := keyOf(opts.Env)
	if err := os.MkdirAll(missing, 0o755); err != nil {
		t.Fatal(err)
	}
	appeared := keyOf(opts.Env)
	if appeared == original {
		t.Fatal("key ignored a PATH entry that appeared")
	}
	if err := os.RemoveAll(filepath.Join(base, "missing")); err != nil {
		t.Fatal(err)
	}
	if keyOf(opts.Env) != original {
		t.Fatal("key is not a function of PATH entry state")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(file, 0o755); err != nil {
		t.Fatal(err)
	}
	if keyOf(opts.Env) == original {
		t.Fatal("key ignored a PATH entry that changed from file to directory")
	}
}

func TestRepositoryReuseRejectsMissingCheckoutLocalPathEntries(t *testing.T) {
	root, _, step, opts := repositoryFixture(t)
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	for name, dir := range map[string]string{
		"lexical":   filepath.Join(root, "not-yet", "bin"),
		"symlinked": filepath.Join(link, "not-yet", "bin"),
	} {
		t.Run(name, func(t *testing.T) {
			env := append([]string{}, opts.Env...)
			env[0] = "PATH=" + dir + ":/usr/bin:/bin"
			if _, err := reuseFor(context.Background(), root, step, env); err == nil || !strings.Contains(err.Error(), "checkout-local") {
				t.Fatalf("missing checkout-local PATH entry accepted: %v", err)
			}
		})
	}
}
