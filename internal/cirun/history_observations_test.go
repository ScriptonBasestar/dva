package cirun

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestHistoryObservationProjectionAndLimit(t *testing.T) {
	now := time.Now()
	admission := &AdmissionObservation{Class: "test", Outcome: "acquired"}
	output := &OutputObservation{Scope: "native_leaf_child_output", StartedAt: now}
	steps := make([]StepReport, maxHistoryStepObservations)
	steps[0] = StepReport{Name: "executed", Status: "succeeded", Admission: admission, OutputObservation: output, Workdir: "/safe", WorkerBudget: &WorkerBudget{DVAJobs: 2}}
	steps[1] = StepReport{Name: "reused", Status: "succeeded", Reuse: &ReuseReport{Disposition: "reused"}}
	runs := []Report{{ID: "new", Root: "/r", Profile: "p", StartedAt: now, Admission: admission, Steps: steps}, {ID: "old", Root: "/r", Profile: "p", Steps: []StepReport{{Name: "legacy", Status: "unknown"}}}, {ID: "other", Root: "/other", Profile: "p"}}
	a := AnalyzeHistory(runs, "/r", "p")
	s := SummarizeHistory(runs, "/r", "", "p", now, nil)
	for _, observations := range [][]HistoryObservation{a.Observations, s.Observations} {
		if len(observations) != 2 || len(observations[0].Steps) != maxHistoryStepObservations || observations[1].OmittedSteps != 1 || observations[1].Admission != nil {
			t.Fatalf("projection bounds: %+v", observations)
		}
		first := observations[0].Steps[0]
		if first.Admission != admission || first.OutputObservation != output || first.Workdir != "/safe" || first.WorkerBudget.DVAJobs != 2 || observations[0].Steps[1].OutputObservation != nil {
			t.Fatalf("projection values: %+v", first)
		}
	}
	if s.EligibleSampleCount != 0 || len(s.Cohorts) != 0 {
		t.Fatal("observations changed eligibility")
	}
	data, err := json.Marshal(a.Observations[0].Steps[1])
	if err != nil || !strings.Contains(string(data), `"admission":null`) || !strings.Contains(string(data), `"output_observation":null`) {
		t.Fatalf("missing nulls: %s %v", data, err)
	}
}
