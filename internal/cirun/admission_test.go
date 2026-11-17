package cirun

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

func TestAdmissionPerStepWorkerCeiling(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	jobs := 4
	p := config.CIProfile{MaxParallel: 2, ToolJobs: &jobs, Steps: []config.CIStep{{Name: "one", Run: "true", Environment: map[string]string{"CARGO_BUILD_JOBS": "4"}}, {Name: "two", Run: "true"}}}
	r, err := execute(context.Background(), root, p, []string{"DVA_RESOURCE_WORKER_LIMIT=2", "CARGO_BUILD_JOBS=4", "RUST_TEST_THREADS=4"}, io.Discard, func([]StepReport) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, step := range r {
		if step.WorkerBudget == nil || step.WorkerBudget.DVAJobs != 2 || step.WorkerBudget.CargoBuildJobs != 2 || step.WorkerBudget.RustTestThreads != 2 || step.WorkerBudget.CargoBuildJobsSource != "admission" {
			t.Fatalf("not bounded: %+v", step.WorkerBudget)
		}
	}
}

func TestAdmissionProfileSharesWorkersAndReleasesLease(t *testing.T) {
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	root, state, resource := t.TempDir(), t.TempDir(), filepath.Join(t.TempDir(), "resource")
	policy := filepath.Join(root, "policy.json")
	data := `{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m","worker_limits":{"heavy":4,"light":1}}`
	if err := os.WriteFile(policy, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	jobs := 4
	p := config.CIProfile{Timeout: "5s", MaxParallel: 2, ToolJobs: &jobs, Admission: &config.CIAdmission{Class: "heavy", Priority: "normal", WaitTimeout: "1s"}, Steps: []config.CIStep{{Name: "one", Run: "true"}, {Name: "two", Run: "true"}}}
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + policy, "CARGO_BUILD_JOBS=4", "RUST_TEST_THREADS=4"}, Profile: p})
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range r.Steps {
		if s.WorkerBudget == nil || s.WorkerBudget.DVAJobs != 2 || s.WorkerBudget.CargoBuildJobs != 2 || s.WorkerBudget.RustTestThreads != 2 {
			t.Fatalf("bad budget %+v", s.WorkerBudget)
		}
	}
	_, digest, err := resourceadmission.ReadPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := resourceadmission.Status(context.Background(), resource, digest)
	if err != nil || len(entries) != 0 {
		t.Fatalf("lease not released: %+v %v", entries, err)
	}
}
