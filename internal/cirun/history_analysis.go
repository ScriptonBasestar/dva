package cirun

import (
	"sort"
	"time"
)

// HistoryAnalysis reports individual receipts without asserting that unknown
// host, configuration, or cache conditions make them comparable.
type HistoryAnalysis struct {
	Root          string         `json:"root"`
	Profile       string         `json:"profile"`
	Comparability string         `json:"comparability"`
	StatusCounts  map[string]int `json:"status_counts"`
	Runs          []HistoryRun   `json:"runs"`
}

type HistoryRun struct {
	ID                      string        `json:"id"`
	Status                  string        `json:"status"`
	StartedAt               time.Time     `json:"started_at"`
	Duration                *string       `json:"duration"`
	LogPath                 string        `json:"log_path"`
	Steps                   []HistoryStep `json:"steps"`
	Slowest                 []HistoryStep `json:"slowest_steps"`
	RecordedDurationRanking []HistoryStep `json:"recorded_duration_ranking"`
}

type HistoryStep struct {
	Name             string  `json:"name"`
	Status           string  `json:"status"`
	Duration         *string `json:"duration"`
	ReuseDisposition string  `json:"reuse_disposition"`
	OriginalRunID    string  `json:"original_run_id,omitempty"`
}

// AnalyzeHistory selects the exact recorded root and profile; an absent host,
// cache or configuration identity must never be treated as a common cohort.
func AnalyzeHistory(reports []Report, root, profile string) HistoryAnalysis {
	a := HistoryAnalysis{Root: root, Profile: profile, Comparability: "unknown: host, configuration and cache identity are not recorded", StatusCounts: map[string]int{}, Runs: []HistoryRun{}}
	for _, r := range reports {
		if r.Root != root || r.Profile != profile {
			continue
		}
		a.StatusCounts[r.Status]++
		run := HistoryRun{ID: r.ID, Status: r.Status, StartedAt: r.StartedAt, LogPath: r.LogPath, Steps: []HistoryStep{}, Slowest: []HistoryStep{}, RecordedDurationRanking: []HistoryStep{}}
		if !r.FinishedAt.IsZero() {
			d := r.Duration.String()
			run.Duration = &d
		}
		var completed []StepReport
		var recorded []struct {
			step     HistoryStep
			duration time.Duration
		}
		for _, s := range r.Steps {
			step := HistoryStep{Name: s.Name, Status: s.Status, ReuseDisposition: "unknown"}
			if s.Reuse != nil {
				step.ReuseDisposition = s.Reuse.Disposition
				step.OriginalRunID = s.Reuse.OriginalRunID
				if step.ReuseDisposition == "" {
					step.ReuseDisposition = "unknown"
				}
			}
			if !s.FinishedAt.IsZero() && (s.Status == "succeeded" || s.Status == "failed" || s.Status == "timed_out" || s.Status == "canceled") {
				d := s.Duration.String()
				step.Duration = &d
				recorded = append(recorded, struct {
					step     HistoryStep
					duration time.Duration
				}{step, s.Duration})
				if step.ReuseDisposition == "executed" || step.ReuseDisposition == "bypassed" {
					completed = append(completed, s)
				}
			}
			run.Steps = append(run.Steps, step)
		}
		sort.SliceStable(recorded, func(i, j int) bool { return recorded[i].duration > recorded[j].duration })
		for i, item := range recorded {
			if i == 5 {
				break
			}
			run.RecordedDurationRanking = append(run.RecordedDurationRanking, item.step)
		}
		sort.SliceStable(completed, func(i, j int) bool { return completed[i].Duration > completed[j].Duration })
		for i, s := range completed {
			if i == 5 {
				break
			}
			d := s.Duration.String()
			run.Slowest = append(run.Slowest, HistoryStep{Name: s.Name, Status: s.Status, Duration: &d, ReuseDisposition: s.Reuse.Disposition, OriginalRunID: s.Reuse.OriginalRunID})
		}
		a.Runs = append(a.Runs, run)
	}
	return a
}
