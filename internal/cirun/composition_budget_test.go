package cirun

import (
	"math"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestCompositionWorkerCeiling(t *testing.T) {
	profile := func() config.CIProfile {
		return config.CIProfile{MaxParallel: 2, Steps: []config.CIStep{{Name: "one", Run: "true"}, {Name: "two", Run: "true"}}}
	}
	t.Run("defaults fit", func(t *testing.T) {
		p := profile()
		resolved, err := constrainCompositionWorkers(p, nil, 2, 64)
		if err != nil || resolved.jobs != 1 || resolved.source != "computed(cpu/max_parallel,composition)" || p.ToolJobs != nil {
			t.Fatalf("workers=%+v profile=%+v err=%v", resolved, p, err)
		}
		for _, step := range p.Steps {
			budget := workerBudget(ciEnv(nil, step.Environment, resolved.jobs), step.Environment, nil, resolved.jobs, resolved.source)
			if budget == nil || budget.Source != resolved.source || budget.ToolJobs*2 > 2 || budget.CargoBuildJobs*2 > 2 || budget.RustTestThreads*2 > 2 {
				t.Fatalf("budget escaped ceiling: %+v", budget)
			}
		}
	})
	t.Run("explicit source preserved", func(t *testing.T) {
		p := profile()
		jobs := 1
		p.ToolJobs = &jobs
		got, err := constrainCompositionWorkers(p, nil, 2, 64)
		if err != nil || got.source != "profile.tool_jobs" || got.jobs != 1 {
			t.Fatalf("workers=%+v err=%v", got, err)
		}
	})
	for _, tc := range []struct {
		name    string
		ceiling int
		jobs    int
		env     []string
		step    map[string]string
	}{
		{name: "zero", ceiling: 0},
		{name: "parallel", ceiling: 1},
		{name: "tool explicit", ceiling: 2, jobs: 2},
		{name: "tool overflow", ceiling: math.MaxInt, jobs: math.MaxInt},
		{name: "cargo inherited", ceiling: 2, env: []string{"CARGO_BUILD_JOBS=2"}},
		{name: "rust explicit", ceiling: 2, step: map[string]string{"RUST_TEST_THREADS": "2"}},
		{name: "invalid cargo", ceiling: 2, env: []string{"CARGO_BUILD_JOBS=auto"}},
		{name: "empty explicit", ceiling: 2, step: map[string]string{"CARGO_BUILD_JOBS": ""}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := profile()
			if tc.jobs != 0 {
				p.ToolJobs = &tc.jobs
			}
			p.Steps[0].Environment = tc.step
			if _, err := constrainCompositionWorkers(p, tc.env, tc.ceiling, 64); err == nil {
				t.Fatal("accepted excess or malformed budget")
			}
		})
	}
	t.Run("step override owns inherited value", func(t *testing.T) {
		p := profile()
		for i := range p.Steps {
			p.Steps[i].Environment = map[string]string{"CARGO_BUILD_JOBS": "1"}
		}
		if _, err := constrainCompositionWorkers(p, []string{"CARGO_BUILD_JOBS=99"}, 2, 64); err != nil {
			t.Fatal(err)
		}
	})
}

func TestCompositionWorkerDefaultProvenance(t *testing.T) {
	grant := compositionGrant{WorkerDefaults: map[string]string{"CARGO_BUILD_JOBS": "4", "RUST_TEST_THREADS": "1"}}
	inherited := []string{"CARGO_BUILD_JOBS=4", "RUST_TEST_THREADS=3", "OTHER=untouched"}
	got := compositionWorkerEnvironment(inherited, grant, nil)
	if environmentValue(got, "CARGO_BUILD_JOBS") != "" || environmentValue(got, "RUST_TEST_THREADS") != "3" || environmentValue(got, "OTHER") != "untouched" {
		t.Fatalf("incorrect default filtering: %v", got)
	}
	got = compositionWorkerEnvironment(inherited, grant, map[string]bool{"CARGO_BUILD_JOBS": true})
	if environmentValue(got, "CARGO_BUILD_JOBS") != "4" {
		t.Fatal("explicit same-value root declaration lost")
	}
}
