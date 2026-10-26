package cirun

import (
	"runtime"
	"strconv"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// StepWorkers exposes the same bounded worker evidence as a step receipt.
type StepWorkers struct {
	Step         string        `json:"step"`
	WorkerBudget *WorkerBudget `json:"worker_budget,omitempty"`
}

func availableCPUs() int { return runtime.NumCPU() }

func toolWorkerBudget(p config.CIProfile, cpu int) (int, string) {
	if p.ToolJobs != nil {
		return *p.ToolJobs, "profile.tool_jobs"
	}
	s := max(1, min(max(1, p.MaxParallel), len(p.Steps)))
	return max(1, min(4, max(1, cpu))/s), "computed(cpu/max_parallel)"
}

func workerBudget(env []string, step map[string]string, inherited []string, jobs int, source string) *WorkerBudget {
	values := map[string]int{}
	for _, key := range []string{"DVA_CI_JOBS", "CARGO_BUILD_JOBS", "RUST_TEST_THREADS"} {
		value, err := strconv.Atoi(environmentValue(env, key))
		if err != nil || value < 1 {
			return nil
		}
		values[key] = value
	}
	origin := func(key, fallback string) string {
		if _, ok := step[key]; ok {
			return "step.environment"
		}
		if environmentValue(inherited, key) != "" {
			return "inherited.environment"
		}
		return fallback
	}
	return &WorkerBudget{
		ToolJobs:              jobs,
		Source:                source,
		DVAJobs:               values["DVA_CI_JOBS"],
		CargoBuildJobs:        values["CARGO_BUILD_JOBS"],
		CargoBuildJobsSource:  origin("CARGO_BUILD_JOBS", source),
		RustTestThreads:       values["RUST_TEST_THREADS"],
		RustTestThreadsSource: origin("RUST_TEST_THREADS", "default"),
	}
}

// WorkerPlan uses the same owning environment and worker resolver as execution.
func WorkerPlan(p config.CIProfile, inherited []string) []StepWorkers {
	jobs, source := toolWorkerBudget(p, availableCPUs())
	result := make([]StepWorkers, 0, len(p.Steps))
	for _, step := range p.Steps {
		if step.OCI != nil {
			result = append(result, StepWorkers{Step: step.Name})
			continue
		}
		env := ciEnv(inherited, step.Environment, jobs)
		result = append(result, StepWorkers{Step: step.Name, WorkerBudget: workerBudget(env, step.Environment, inherited, jobs, source)})
	}
	return result
}
