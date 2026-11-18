package cirun

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func reuseFixture(t *testing.T) (string, config.CIStep) {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := runGit(root, "init"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "source"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "source", "a"), []byte("one"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("artifact\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	commitAll(t, root)
	return root, config.CIStep{Name: "unit", Run: "printf ok > artifact", Reuse: &config.CIReuse{Inputs: []string{"source"}, Tools: []string{"sh"}, Outputs: []string{"artifact"}, MaxAge: "1h"}}
}

func TestReuseKeyMutationMatrix(t *testing.T) {
	// The former "workers" row asserted that a DVA_CI_JOBS change alone
	// differentiated the key. TASK-622 removed that entry from the key after
	// measuring that the value is DVA's own computed budget, ambient-proof and
	// behavior-neutral; TestReuseKeyIgnoresDVAJobs now asserts the replacement
	// contract and TestReuseKeyStillDifferentiatesToolEnvironment keeps the
	// discriminating control on the environment the command obeys.
	for _, kind := range []string{"content", "mode", "addition", "deletion", "empty-environment", "probe"} {
		t.Run(kind, func(t *testing.T) {
			root, step := reuseFixture(t)
			env := os.Environ()
			step.Reuse.Environment = []string{"DVA_REUSE_FEATURE"}
			env = withoutEnv(env, "DVA_REUSE_FEATURE")
			step.Reuse.Probes = []string{"printf one"}
			before, err := reuseFor(context.Background(), root, step, env)
			if err != nil {
				t.Fatal(err)
			}
			switch kind {
			case "content":
				err = os.WriteFile(filepath.Join(root, "source/a"), []byte("two"), 0o600)
			case "mode":
				err = os.Chmod(filepath.Join(root, "source/a"), 0o700)
			case "addition":
				err = os.WriteFile(filepath.Join(root, "source/b"), []byte("new"), 0o600)
			case "deletion":
				err = os.Remove(filepath.Join(root, "source/a"))
				if err == nil {
					err = os.Remove(filepath.Join(root, "source"))
				}
			case "empty-environment":
				env = append(env, "DVA_REUSE_FEATURE=")
			case "probe":
				step.Reuse.Probes = []string{"printf two"}
			}
			if err != nil {
				t.Fatal(err)
			}
			after, err := reuseFor(context.Background(), root, step, env)
			if err != nil {
				t.Fatal(err)
			}
			if before.key == after.key {
				t.Fatal("mutation reused identical key")
			}
		})
	}
}

func TestReuseOutputAndRecordIntegrity(t *testing.T) {
	root, step := reuseFixture(t)
	if err := os.WriteFile(filepath.Join(root, "artifact"), []byte("ok"), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := reuseFor(context.Background(), root, step, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := saveReuse(plan, root, "original", now); err != nil {
		t.Fatal(err)
	}
	if hit, _ := loadReuse(plan, time.Hour, now, root); hit == nil {
		t.Fatal("initial cache missed")
	}
	if hit, _ := loadReuse(plan, time.Hour, now.Add(2*time.Hour), root); hit != nil {
		t.Fatal("expired cache passed")
	}
	if hit, _ := loadReuse(plan, time.Hour, now.Add(-time.Hour), root); hit != nil {
		t.Fatal("future cache passed")
	}
	if err := os.Chmod(filepath.Join(root, "artifact"), 0o700); err != nil {
		t.Fatal(err)
	}
	if hit, _ := loadReuse(plan, time.Hour, now, root); hit != nil {
		t.Fatal("changed output mode reused")
	}
	if err := os.WriteFile(plan.cache, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if hit, _ := loadReuse(plan, time.Hour, now, root); hit != nil {
		t.Fatal("corrupt cache passed")
	}
	out := t.TempDir()
	if err := os.WriteFile(filepath.Join(out, "file"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(out, filepath.Join(root, "escape")); err != nil {
		t.Fatal(err)
	}
	if _, err := reuseOutputState(root, []string{"escape/file"}); err == nil {
		t.Fatal("output parent symlink escaped root")
	}
}

func TestReusableStepFailureAndFreshPreconditions(t *testing.T) {
	root, step := reuseFixture(t)
	state := t.TempDir()
	run := func(ctx context.Context, p config.CIProfile, fresh bool) (Report, error) {
		return Run(ctx, Options{Root: root, ProfileName: "full", Profile: p, StateDir: state, lockDirectory: state, Env: os.Environ(), Fresh: fresh})
	}
	step.Reuse.Probes = []string{"exit 99"}
	step.Run = `test "$DVA_CI_FRESH" = 1; printf ok > artifact`
	if _, err := run(context.Background(), profile(step), true); err != nil {
		t.Fatalf("fresh ran failing probe: %v", err)
	}
	step.Reuse.Probes = nil
	step.Run = "printf changed > source/a"
	dependent := config.CIStep{Name: "dependent", Run: "touch dependent", DependsOn: []string{"unit"}}
	independent := config.CIStep{Name: "independent", Run: "touch independent"}
	p := profile(step, dependent, independent)
	no := false
	p.FailFast = &no
	p.MaxParallel = 1
	r, err := run(context.Background(), p, false)
	if err == nil || r.Steps[0].Status != "failed" || r.Steps[1].Status != "skipped" || r.Steps[2].Status != "skipped" {
		t.Fatalf("integrity failure did not stop all work: %+v %v", r.Steps, err)
	}
	step.Run = "exit 1"
	step.Reuse.Probes = nil
	if _, err := run(context.Background(), profile(step), false); err == nil {
		t.Fatal("failed command passed")
	}
	plan, err := reuseFor(context.Background(), root, step, ciEnv(os.Environ(), nil, 1))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(plan.cache); !os.IsNotExist(err) {
		t.Fatal("failed command stored PASS")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := run(ctx, profile(step), false); err == nil {
		t.Fatal("canceled check passed")
	}
}

// The reuse key must not discriminate on DVA_CI_JOBS. The value a step sees is
// DVA's own computed worker budget — ciEnv overwrites any ambient value before
// execution and keying — the tool knobs it feeds (CARGO_BUILD_JOBS default,
// GOMAXPROCS, GOFLAGS, RUST_TEST_THREADS) are separately keyed, and no command
// sizes its behavior from the informational variable itself. Two runs whose
// only difference is the computed budget therefore share one cache entry.
func TestReuseKeyIgnoresDVAJobs(t *testing.T) {
	root, step := reuseFixture(t)
	base := []string{
		"PATH=" + os.Getenv("PATH"),
		// Pin the tool knob so the two prepared envs differ in DVA_CI_JOBS
		// alone; ciEnv would otherwise derive CARGO_BUILD_JOBS from the same
		// budget and the key comparison would not isolate the fix.
		"CARGO_BUILD_JOBS=4",
	}
	four := ciEnv(base, step.Environment, 4)
	two := ciEnv(base, step.Environment, 2)
	if got := environmentValue(four, "DVA_CI_JOBS"); got != "4" {
		t.Fatalf("prepared env DVA_CI_JOBS=%q, want 4", got)
	}
	if got := environmentValue(two, "DVA_CI_JOBS"); got != "2" {
		t.Fatalf("prepared env DVA_CI_JOBS=%q, want 2", got)
	}
	planFour, err := reuseFor(context.Background(), root, step, four)
	if err != nil {
		t.Fatal(err)
	}
	planTwo, err := reuseFor(context.Background(), root, step, two)
	if err != nil {
		t.Fatal(err)
	}
	if planFour.key != planTwo.key {
		t.Fatal("reuse key changed with the computed DVA_CI_JOBS budget")
	}
}

// Dropping DVA_CI_JOBS from the key must not stop the key from discriminating
// the environment the step command actually obeys.
func TestReuseKeyStillDifferentiatesToolEnvironment(t *testing.T) {
	root, step := reuseFixture(t)
	prepared := func(flags string) []string {
		base := []string{"PATH=" + os.Getenv("PATH"), "CARGO_BUILD_JOBS=4", "GOFLAGS=" + flags}
		return ciEnv(base, step.Environment, 2)
	}
	one, err := reuseFor(context.Background(), root, step, prepared("-p=1"))
	if err != nil {
		t.Fatal(err)
	}
	four, err := reuseFor(context.Background(), root, step, prepared("-p=4"))
	if err != nil {
		t.Fatal(err)
	}
	if one.key == four.key {
		t.Fatal("GOFLAGS -p change did not differentiate the reuse key")
	}
}

// The admission cap creates PYTEST_MAX_WORKERS from nothing —
// CapWorkerEnvironment sets an absent entry to the limit — so an unadmitted
// run (variable absent) and an admitted run (variable at the limit) are two
// different executions. The key must discriminate them: the counterpoint to
// DVA_CI_JOBS, which DVA computes itself and no consumer sizes behavior from.
func TestReuseKeyTracksAdmissionCreatedPytestWorkers(t *testing.T) {
	root, step := reuseFixture(t)
	base := func() []string {
		// Pin the keyed worker knobs at the limit so the two prepared envs
		// differ in PYTEST_MAX_WORKERS alone; an unpinned knob would let the
		// cap differentiate the key for the wrong reason.
		return []string{
			"PATH=" + os.Getenv("PATH"),
			"GOMAXPROCS=2",
			"CARGO_BUILD_JOBS=2",
			"RUST_TEST_THREADS=2",
		}
	}
	unadmitted := ciEnv(base(), step.Environment, 2)
	admitted := ciEnv(append(base(), "DVA_RESOURCE_WORKER_LIMIT=2"), step.Environment, 2)
	if got := environmentValue(unadmitted, "PYTEST_MAX_WORKERS"); got != "" {
		t.Fatalf("unadmitted env PYTEST_MAX_WORKERS=%q, want absent", got)
	}
	if got := environmentValue(admitted, "PYTEST_MAX_WORKERS"); got != "2" {
		t.Fatalf("admitted env PYTEST_MAX_WORKERS=%q, want the cap-created limit 2", got)
	}
	planUnadmitted, err := reuseFor(context.Background(), root, step, unadmitted)
	if err != nil {
		t.Fatal(err)
	}
	planAdmitted, err := reuseFor(context.Background(), root, step, admitted)
	if err != nil {
		t.Fatal(err)
	}
	if planUnadmitted.key == planAdmitted.key {
		t.Fatal("reuse key ignored the admission-created PYTEST_MAX_WORKERS")
	}
}

func withoutEnv(env []string, key string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if len(entry) <= len(key) || entry[:len(key)+1] != key+"=" {
			out = append(out, entry)
		}
	}
	return out
}
