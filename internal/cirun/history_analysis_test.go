package cirun

import (
	"testing"
	"time"
)

func TestAnalyzeHistoryKeepsUnknownConditionsAndStatusesSeparate(t *testing.T) {
	r := []Report{{ID: "a", Root: "/a", Profile: "full", Status: "succeeded", Duration: time.Second}, {ID: "b", Root: "/a", Profile: "full", Status: "timed_out"}, {ID: "c", Root: "/b", Profile: "full", Status: "succeeded"}}
	a := AnalyzeHistory(r, "/a", "full")
	if len(a.Runs) != 2 || a.StatusCounts["succeeded"] != 1 || a.StatusCounts["timed_out"] != 1 || a.Comparability == "" {
		t.Fatalf("unexpected analysis: %+v", a)
	}
}

func TestRecordedRankingLabelsUnknownAndReuse(t *testing.T) {
	now := time.Now()
	r := Report{Root: "/a", Profile: "full", Steps: []StepReport{{Name: "short", Status: "succeeded", FinishedAt: now, Duration: 2 * time.Second}, {Name: "long", Status: "succeeded", FinishedAt: now, Duration: 3 * time.Second}, {Name: "cached", Status: "succeeded", FinishedAt: now, Duration: time.Second, Reuse: &ReuseReport{Disposition: "reused", OriginalRunID: "old"}}, {Name: "pending", Status: "pending"}}}
	run := AnalyzeHistory([]Report{r}, "/a", "full").Runs[0]
	if len(run.Slowest) != 0 || len(run.RecordedDurationRanking) != 3 || run.RecordedDurationRanking[0].Name != "long" || run.RecordedDurationRanking[1].Name != "short" || run.RecordedDurationRanking[0].ReuseDisposition != "unknown" || run.RecordedDurationRanking[2].ReuseDisposition != "reused" || run.RecordedDurationRanking[2].OriginalRunID != "old" || run.Steps[3].Duration != nil {
		t.Fatalf("bad ranking: %+v", run)
	}
}

func TestAnalyzeHistoryDoesNotInventUnfinishedDurations(t *testing.T) {
	now := time.Now()
	r := Report{ID: "stale", Root: "/a", Profile: "full", Status: "stale", Steps: []StepReport{{Name: "waiting", Status: "pending"}, {Name: "done", Status: "succeeded", FinishedAt: now, Duration: 2 * time.Second, Reuse: &ReuseReport{Disposition: "executed"}}, {Name: "slow", Status: "failed", FinishedAt: now, Duration: 3 * time.Second, Reuse: &ReuseReport{Disposition: "bypassed"}}, {Name: "cached", Status: "succeeded", FinishedAt: now, Duration: time.Second, Reuse: &ReuseReport{Disposition: "reused", OriginalRunID: "prior"}}, {Name: "legacy", Status: "succeeded", FinishedAt: now, Duration: 4 * time.Second}}}
	a := AnalyzeHistory([]Report{r}, "/a", "full")
	if a.Runs[0].Duration != nil || a.Runs[0].Steps[0].Duration != nil || len(a.Runs[0].Slowest) != 2 || a.Runs[0].Slowest[0].Name != "slow" {
		t.Fatalf("unexpected history: %+v", a.Runs[0])
	}
	if a.Runs[0].Steps[3].ReuseDisposition != "reused" || a.Runs[0].Steps[3].OriginalRunID != "prior" || a.Runs[0].Steps[4].ReuseDisposition != "unknown" {
		t.Fatalf("reuse lost: %+v", a.Runs[0].Steps)
	}
}

func TestRecordedRankingLimitsAndPreservesTieOrder(t *testing.T) {
	now := time.Now()
	r := Report{Root: "/a", Profile: "full"}
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		r.Steps = append(r.Steps, StepReport{Name: name, Status: "succeeded", FinishedAt: now, Duration: time.Second})
	}
	ranking := AnalyzeHistory([]Report{r}, "/a", "full").Runs[0].RecordedDurationRanking
	if len(ranking) != 5 || ranking[0].Name != "a" || ranking[4].Name != "e" {
		t.Fatalf("bad ranking: %+v", ranking)
	}
}
