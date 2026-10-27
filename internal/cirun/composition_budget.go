package cirun

import (
	"fmt"
	"strconv"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// constrainCompositionWorkers resolves omitted defaults inside the inherited
// ceiling. Explicit settings are rejected, never silently reduced. Division
// avoids overflowing when a caller supplies a very large worker count.
type compositionWorkers struct {
	jobs   int
	source string
}

func constrainCompositionWorkers(p config.CIProfile, inherited []string, ceiling, cpu int) (compositionWorkers, error) {
	var result compositionWorkers
	if ceiling < 1 {
		return result, compositionFailure("jobs", "jobs limit must be positive")
	}
	parallel := max(1, min(max(1, p.MaxParallel), len(p.Steps)))
	if parallel > ceiling {
		return result, compositionFailure("jobs", "max_parallel exceeds inherited jobs limit")
	}
	perStep := ceiling / parallel
	jobs, source := toolWorkerBudget(p, cpu)
	if p.ToolJobs != nil && (jobs < 1 || jobs > perStep) {
		return result, compositionFailure("jobs", "tool_jobs is outside inherited jobs limit")
	}
	jobs = min(jobs, perStep)
	if p.ToolJobs == nil {
		source = "computed(cpu/max_parallel,composition)"
	}
	result = compositionWorkers{jobs: jobs, source: source}
	for _, step := range p.Steps {
		for _, key := range []string{"CARGO_BUILD_JOBS", "RUST_TEST_THREADS"} {
			value, explicit := step.Environment[key]
			if !explicit {
				value = environmentValue(inherited, key)
			}
			if value == "" && !explicit {
				continue
			}
			n, err := strconv.Atoi(value)
			if err != nil || n < 1 || n > perStep {
				return result, compositionFailure("jobs", fmt.Sprintf("step %q %s must be a positive integer within %d", step.Name, key, perStep))
			}
		}
	}
	return result, nil
}
