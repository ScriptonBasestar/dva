package cirun

import (
	"errors"
	"testing"
	"time"
)

type failingSummaryCloser struct{}

func (failingSummaryCloser) Close() error { return errors.New("fixture close failed") }

func TestSummaryRejectedClosePreservesBothErrors(t *testing.T) {
	cause := errors.New("fixture rejection")
	err := closeSummaryRejected(failingSummaryCloser{}, cause)
	if !errors.Is(err, cause) || !errors.Is(err, errSummaryClose) {
		t.Fatalf("close error lost: %v", err)
	}
}

func TestSummaryKeepsUnknownConditionsAndReuseSeparate(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	o := &Observability{Host: "host-a", OS: "darwin", Arch: "arm64", ConfigurationShapeFingerprint: "same", CacheCondition: "unknown", RepositoryLocalID: "repo"}
	base := Report{ID: "one", Root: "/a", Profile: "full", Status: "succeeded", StartedAt: now.Add(-time.Hour), FinishedAt: now, Observability: o, Steps: []StepReport{
		{Name: "ran", Status: "succeeded", StartedAt: now.Add(-time.Second), FinishedAt: now, Duration: time.Second, Reuse: &ReuseReport{Disposition: "executed"}},
		{Name: "cached", Status: "succeeded", FinishedAt: now, Duration: 2 * time.Second, Reuse: &ReuseReport{Disposition: "reused"}},
		{Name: "legacy", Status: "succeeded", FinishedAt: now, Duration: 3 * time.Second},
	}}
	other := base
	other.ID = "two"
	other.Status = "failed"
	copyObs := *o
	copyObs.Host = "host-b"
	copyObs.CacheCondition = "warm"
	other.Observability = &copyObs
	future := base
	future.ID = "three"
	future.FinishedAt = now.Add(time.Hour)
	future.Observability = nil
	s := SummarizeHistory([]Report{base, other, future}, "/a", "", "full", now, nil)
	if s.SelectionCount != 3 || s.StatusCounts["failed"] != 1 || s.EligibleSampleCount != 0 || len(s.Cohorts) != 0 || !s.Sparse {
		t.Fatalf("summary %+v", s)
	}
	if s.DispositionCounts["executed"] != 3 || s.DispositionCounts["reused"] != 3 || s.DispositionCounts["unknown"] != 3 {
		t.Fatalf("dispositions %+v", s.DispositionCounts)
	}
	if s.ExcludedByReason["cache_unknown"] != 2 || s.ExcludedByReason["host_missing"] != 1 || s.ExcludedByReason["invalid_or_future_time"] != 1 || s.ExcludedByReason["not_succeeded"] != 1 || s.ExcludedByReason["complete_provenance_unavailable"] != 3 {
		t.Fatalf("exclusions %+v", s.ExcludedByReason)
	}
	if len(s.CurrentLongestSteps[0].Steps) != 3 || s.CurrentLongestSteps[0].Steps[0].Name != "legacy" || s.CurrentLongestSteps[0].Steps[0].ReuseDisposition != "unknown" || s.CurrentLongestSteps[0].Steps[1].ReuseDisposition != "reused" {
		t.Fatalf("ranking %+v", s.CurrentLongestSteps)
	}
	if s.Conditions[0].Host == s.Conditions[1].Host {
		t.Fatalf("host identities mixed: %+v", s.Conditions)
	}
	empty := SummarizeHistory([]Report{base}, "/other", "", "full", now, nil)
	if empty.SelectionCount != 0 || !empty.Sparse || len(empty.Cohorts) != 0 {
		t.Fatalf("empty %+v", empty)
	}
	byID := SummarizeHistory([]Report{base, future}, "", "repo", "full", now, nil)
	if byID.SelectionCount != 1 || byID.ExcludedByReason["repository_id_missing"] != 1 {
		t.Fatalf("identity %+v", byID)
	}
}
