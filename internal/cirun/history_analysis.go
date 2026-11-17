package cirun

import "time"

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
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	StartedAt time.Time `json:"started_at"`
	Duration  string    `json:"duration"`
	LogPath   string    `json:"log_path"`
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
		a.Runs = append(a.Runs, HistoryRun{r.ID, r.Status, r.StartedAt, r.Duration.String(), r.LogPath})
	}
	return a
}
