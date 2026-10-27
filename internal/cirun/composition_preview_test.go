//go:build darwin || linux || freebsd || netbsd || openbsd

package cirun

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestCompositionPreviewWorkersWithoutAdmission(t *testing.T) {
	// This unit fixture supplies its own live ancestry/store, independent of the
	// enclosing CI run. Testing restores the real ambient token on completion.
	t.Setenv(parentRunEnv, "")
	if err := os.Unsetenv(parentRunEnv); err != nil {
		t.Fatal(err)
	}
	scope, grant := compositionScopeFixture(t)
	grant.WorkerDefaults = map[string]string{"CARGO_BUILD_JOBS": "2", "RUST_TEST_THREADS": "1"}
	if err := os.WriteFile(filepath.Join(scope.dir, "grant"), []byte(compositionTestJSON(t, grant)), 0o600); err != nil {
		t.Fatal(err)
	}
	root := compositionTestDir(t)
	profile := compositionProcessProfile("true", "")
	profile.MaxParallel = 2
	profile.Steps = append(profile.Steps, config.CIStep{Name: "second", Run: "true"})
	opts := Options{Root: root, ProfileName: "full", Profile: profile, lockDirectory: grant.LockState, Env: []string{
		parentRunEnv + "=" + compositionTestParent(t, grant), config.CICompositionEnv + "=" + compositionTestJSON(t, grant), "CARGO_BUILD_JOBS=2", "RUST_TEST_THREADS=1",
	}}
	workers, err := PreviewWorkers(opts)
	if err != nil || len(workers) != 2 {
		t.Fatalf("workers=%+v err=%v", workers, err)
	}
	for _, worker := range workers {
		if worker.WorkerBudget == nil || worker.WorkerBudget.ToolJobs != 1 || worker.WorkerBudget.CargoBuildJobs != 1 || worker.WorkerBudget.Source != "computed(cpu/max_parallel,composition)" {
			t.Fatalf("preview exceeds actual execution budget: %+v", worker)
		}
	}
	children, err := scope.children()
	if err != nil || len(children) != 0 {
		t.Fatalf("preview registered an execution: %+v %v", children, err)
	}
	owner, err := lockOwner(grant.LockState, "root", root)
	if err != nil || owner != "" {
		t.Fatalf("preview acquired root: %s %v", owner, err)
	}
	opts.WorkerEnvironment = map[string]bool{"CARGO_BUILD_JOBS": true}
	if _, err := PreviewWorkers(opts); err == nil {
		t.Fatal("preview silently reduced explicit same-value root declaration")
	}
	opts.WorkerEnvironment = nil
	if err := scope.close(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := PreviewWorkers(opts); err == nil {
		t.Fatal("preview accepted closed grant")
	}
}

func TestCompositionPreviewPreservesLegacyWorkers(t *testing.T) {
	profile := compositionProcessProfile("true", "")
	opts := Options{ProfileName: "full", Profile: profile, Env: []string{"CARGO_BUILD_JOBS=3"}}
	got, err := PreviewWorkers(opts)
	if err != nil || !reflect.DeepEqual(got, WorkerPlan(profile, opts.Env)) {
		t.Fatalf("legacy preview changed: %+v %v", got, err)
	}
	opts.Env = append(opts.Env, config.CICompositionEnv+"=")
	opts.Root = compositionTestDir(t)
	if _, err := PreviewWorkers(opts); err == nil {
		t.Fatal("explicit empty grant became omission")
	}
}
