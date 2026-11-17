package cirun

import (
	"runtime"
	"strconv"
	"strings"

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
	observed := func(value, source string) *ObservedWorker {
		n, err := strconv.Atoi(value)
		if err != nil || n < 1 {
			return nil
		}
		return &ObservedWorker{Value: n, Source: source}
	}
	rawFlags := environmentValue(env, "GOFLAGS")
	goFlags := strings.Fields(rawFlags)
	if strings.ContainsAny(rawFlags, "'\"\\") {
		goFlags = nil // Go's quoted flag syntax needs its own parser; do not guess.
	}
	var goJobs *ObservedWorker
	goSource := origin("GOFLAGS", "default")
	if _, explicit := step["GOFLAGS"]; !explicit && !strings.Contains(" "+environmentValue(inherited, "GOFLAGS")+" ", " -p=") {
		goSource = "default"
	}
	for i := 0; i < len(goFlags); i++ {
		if value, ok := strings.CutPrefix(goFlags[i], "-p="); ok {
			goJobs = observed(value, goSource)
		} else if goFlags[i] == "-p" && i+1 < len(goFlags) {
			i++
			goJobs = observed(goFlags[i], goSource)
		}
	}
	return &WorkerBudget{
		GoMaxProcs:            observed(environmentValue(env, "GOMAXPROCS"), origin("GOMAXPROCS", "default")),
		GoPackageJobs:         goJobs,
		PytestWorkers:         observed(environmentValue(env, "PYTEST_WORKERS"), origin("PYTEST_WORKERS", "unknown")),
		PytestMaxWorkers:      observed(environmentValue(env, "PYTEST_MAX_WORKERS"), origin("PYTEST_MAX_WORKERS", "unknown")),
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
	return workerPlan(p, inherited, jobs, source)
}

func workerPlan(p config.CIProfile, inherited []string, jobs int, source string) []StepWorkers {
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
