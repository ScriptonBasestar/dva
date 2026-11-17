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
	ID        string        `json:"id"`
	Status    string        `json:"status"`
	StartedAt time.Time     `json:"started_at"`
	Duration  *string       `json:"duration"`
	LogPath   string        `json:"log_path"`
	Steps     []HistoryStep `json:"steps"`
	Slowest   []HistoryStep `json:"slowest_steps"`
}

type HistoryStep struct {
	Name     string  `json:"name"`
	Status   string  `json:"status"`
	Duration *string `json:"duration"`
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
		run := HistoryRun{ID: r.ID, Status: r.Status, StartedAt: r.StartedAt, LogPath: r.LogPath, Steps: []HistoryStep{}, Slowest: []HistoryStep{}}
		if !r.FinishedAt.IsZero() {
			d := r.Duration.String()
			run.Duration = &d
		}
		var completed []StepReport
		for _, s := range r.Steps {
			step := HistoryStep{Name: s.Name, Status: s.Status}
			if !s.FinishedAt.IsZero() && (s.Status == "succeeded" || s.Status == "failed" || s.Status == "timed_out" || s.Status == "canceled") {
				d := s.Duration.String()
				step.Duration = &d
				completed = append(completed, s)
			}
			run.Steps = append(run.Steps, step)
		}
		sort.SliceStable(completed, func(i, j int) bool { return completed[i].Duration > completed[j].Duration })
		for i, s := range completed {
			if i == 5 {
				break
			}
			d := s.Duration.String()
			run.Slowest = append(run.Slowest, HistoryStep{Name: s.Name, Status: s.Status, Duration: &d})
		}
		a.Runs = append(a.Runs, run)
	}
	return a
}
