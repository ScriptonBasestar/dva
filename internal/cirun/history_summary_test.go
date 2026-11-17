package cirun

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSummaryKeepsUnknownConditionsAndReuseSeparate(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	o := &Observability{Host: "host-a", OS: "darwin", Arch: "arm64", ConfigurationShapeFingerprint: "same", CacheCondition: "unknown", RepositoryLocalID: "repo"}
	base := Report{ID: "one", Root: "/a", Profile: "full", Status: "succeeded", StartedAt: now.Add(-time.Hour), FinishedAt: now, Observability: o, Steps: []StepReport{
		{Name: "ran", Status: "succeeded", StartedAt: now.Add(-time.Second), FinishedAt: now, Duration: time.Second, Reuse: &ReuseReport{Disposition: "executed"}},
		{Name: "cached", Reuse: &ReuseReport{Disposition: "reused"}}, {Name: "legacy"},
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
	if len(s.CurrentLongestSteps[0].Steps) != 1 || s.CurrentLongestSteps[0].Steps[0].Name != "ran" {
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

func TestReadSummaryReceiptsBoundsAndRejectsUnsafeInputs(t *testing.T) {
	dir := t.TempDir()
	id := strings.Repeat("a", 32)
	data, err := json.Marshal(Report{ID: id, Root: "/a", Profile: "full", Status: "succeeded", Duration: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, strings.Repeat("b", 32)+".json"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, strings.Repeat("c", 32)+".json"), make([]byte, (4<<20)+1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, id+".json"), filepath.Join(dir, strings.Repeat("d", 32)+".json")); err != nil {
		t.Fatal(err)
	}
	reports, excluded, err := ReadSummaryReceipts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || excluded["invalid_receipt"] != 1 || excluded["oversized"] != 1 || excluded["non_regular"] != 1 {
		t.Fatalf("reports %v excluded %v", reports, excluded)
	}
}
