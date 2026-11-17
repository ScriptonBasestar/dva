package cirun

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

var errSummaryClose = errors.New("close CI history input")

func closeSummaryRejected(closer io.Closer, cause error) error {
	if err := closer.Close(); err != nil {
		return errors.Join(cause, fmt.Errorf("%w: %w", errSummaryClose, err))
	}
	return cause
}

// HistorySummary is descriptive; this receipt schema cannot prove comparable
// complete command/configuration and cache provenance.
type HistorySummary struct {
	Schema              string             `json:"schema"`
	Root                string             `json:"root,omitempty"`
	RepositoryLocalID   string             `json:"repository_local_id,omitempty"`
	Profile             string             `json:"profile"`
	SelectionCount      int                `json:"selection_count"`
	StatusCounts        map[string]int     `json:"status_counts"`
	DispositionCounts   map[string]int     `json:"disposition_counts"`
	ExcludedByReason    map[string]int     `json:"excluded_by_reason"`
	ReadExclusions      map[string]int     `json:"read_exclusions"`
	Conditions          []SummaryCondition `json:"conditions"`
	CurrentLongestSteps []SummaryRun       `json:"current_longest_steps"`
	EligibleSampleCount int                `json:"eligible_sample_count"`
	Cohorts             []any              `json:"cohorts"`
	Sparse              bool               `json:"sparse"`
}

type SummaryCondition struct {
	RunID                 string   `json:"run_id"`
	Host                  string   `json:"host"`
	OS                    string   `json:"os"`
	Arch                  string   `json:"arch"`
	Profile               string   `json:"profile"`
	ConfigurationShape    string   `json:"configuration_shape"`
	ConfigurationComplete bool     `json:"configuration_complete"`
	CacheCondition        string   `json:"cache_condition"`
	Exclusions            []string `json:"exclusions"`
}

type SummaryRun struct {
	RunID  string        `json:"run_id"`
	Status string        `json:"status"`
	Steps  []HistoryStep `json:"steps"`
}

// ReadSummaryReceipts bounds untrusted individual receipts and directory size.
// It never follows symlinks or reads logs/events, and reports skipped inputs.
func ReadSummaryReceipts(dir string) ([]Report, map[string]int, error) {
	state, err := stateDirectory(dir)
	if err != nil {
		return nil, nil, err
	}
	f, err := openSummaryState(state)
	if os.IsNotExist(err) {
		return nil, map[string]int{}, nil
	}
	if err != nil {
		if errors.Is(err, errSummaryClose) {
			return nil, nil, err
		}
		return nil, map[string]int{"state_directory_unavailable": 1}, nil
	}
	reports, excluded, readErr := readSummaryFromOpenedDir(f)
	if closeErr := f.Close(); closeErr != nil {
		return nil, nil, errors.Join(readErr, fmt.Errorf("%w: %w", errSummaryClose, closeErr))
	}
	return reports, excluded, readErr
}

func readSummaryFromOpenedDir(f *os.File) ([]Report, map[string]int, error) {
	entries, err := f.ReadDir(10001)
	if err != nil && err != io.EOF {
		return nil, nil, err
	}
	if len(entries) > 10000 {
		return nil, nil, fmt.Errorf("CI history exceeds 10000 directory entries")
	}
	var reports []Report
	excluded := map[string]int{}
	totalBytes := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") || !safeID(strings.TrimSuffix(name, ".json")) {
			continue
		}
		file, reason, e := openSummaryReceipt(f, name)
		if e != nil {
			excluded[reason]++
			if errors.Is(e, errSummaryClose) {
				excluded["close_failed"]++
			}
			continue
		}
		data, e := io.ReadAll(io.LimitReader(file, (4<<20)+1))
		closeErr := file.Close()
		if e != nil || closeErr != nil {
			excluded["unreadable"]++
			if closeErr != nil {
				excluded["close_failed"]++
			}
			continue
		}
		if len(data) > 4<<20 {
			excluded["oversized"]++
			continue
		}
		totalBytes += len(data)
		if totalBytes > 64<<20 {
			return nil, nil, fmt.Errorf("CI history receipt data exceeds 64 MiB")
		}
		var report Report
		if json.Unmarshal(data, &report) != nil || report.ID != strings.TrimSuffix(name, ".json") {
			excluded["invalid_receipt"]++
			continue
		}
		reports = append(reports, report)
	}
	return reports, excluded, nil
}

func SummarizeHistory(reports []Report, root, repositoryID, profile string, now time.Time, readExclusions map[string]int) HistorySummary {
	s := HistorySummary{Schema: "dva-ci-history-summary/1", Root: root, RepositoryLocalID: repositoryID, Profile: profile,
		StatusCounts: map[string]int{}, DispositionCounts: map[string]int{"executed": 0, "reused": 0, "unknown": 0},
		ExcludedByReason: map[string]int{}, ReadExclusions: readExclusions, Conditions: []SummaryCondition{},
		CurrentLongestSteps: []SummaryRun{}, Cohorts: []any{}, Sparse: true}
	if s.ReadExclusions == nil {
		s.ReadExclusions = map[string]int{}
	}
	var selected []Report
	for _, r := range reports {
		if r.Profile != profile {
			continue
		}
		if repositoryID != "" {
			if r.Observability == nil || r.Observability.RepositoryLocalID == "" {
				s.ExcludedByReason["repository_id_missing"]++
				continue
			}
			if r.Observability.RepositoryLocalID != repositoryID {
				continue
			}
		} else if r.Root != root {
			continue
		}
		selected = append(selected, r)
	}
	sort.SliceStable(selected, func(i, j int) bool { return selected[i].StartedAt.After(selected[j].StartedAt) })
	for _, r := range selected {
		s.SelectionCount++
		s.StatusCounts[r.Status]++
		c := SummaryCondition{RunID: r.ID, Profile: r.Profile, Host: "unknown", OS: "unknown", Arch: "unknown", ConfigurationShape: "unknown", CacheCondition: "unknown", Exclusions: []string{}}
		if o := r.Observability; o != nil {
			if o.Host != "" {
				c.Host = o.Host
			}
			if o.OS != "" {
				c.OS = o.OS
			}
			if o.Arch != "" {
				c.Arch = o.Arch
			}
			if o.ConfigurationShapeFingerprint != "" {
				c.ConfigurationShape = o.ConfigurationShapeFingerprint
			}
			c.ConfigurationComplete = o.ConfigurationFingerprintComplete
			if o.CacheCondition != "" {
				c.CacheCondition = o.CacheCondition
			}
		}
		if c.Host == "unknown" || c.OS == "unknown" || c.Arch == "unknown" {
			c.Exclusions = append(c.Exclusions, "host_missing")
		}
		if !c.ConfigurationComplete {
			c.Exclusions = append(c.Exclusions, "configuration_incomplete")
		}
		if c.CacheCondition == "unknown" {
			c.Exclusions = append(c.Exclusions, "cache_unknown")
		}
		if r.Status != "succeeded" {
			c.Exclusions = append(c.Exclusions, "not_succeeded")
		}
		if r.StartedAt.IsZero() || r.FinishedAt.IsZero() || r.FinishedAt.Before(r.StartedAt) || r.FinishedAt.After(now) || r.StartedAt.After(now) {
			c.Exclusions = append(c.Exclusions, "invalid_or_future_time")
		}
		// A shape fingerprint does not prove exact command/configuration or cache
		// provenance, even if a producer sets the completeness boolean.
		c.Exclusions = append(c.Exclusions, "complete_provenance_unavailable")
		for _, reason := range c.Exclusions {
			s.ExcludedByReason[reason]++
		}
		for _, step := range r.Steps {
			disposition := "unknown"
			if step.Reuse != nil {
				switch step.Reuse.Disposition {
				case "executed", "bypassed":
					disposition = "executed"
				case "reused":
					disposition = "reused"
				}
			}
			s.DispositionCounts[disposition]++
		}
		s.Conditions = append(s.Conditions, c)
		analysis := AnalyzeHistory([]Report{r}, r.Root, r.Profile)
		if len(analysis.Runs) > 0 {
			s.CurrentLongestSteps = append(s.CurrentLongestSteps, SummaryRun{RunID: r.ID, Status: r.Status, Steps: analysis.Runs[0].RecordedDurationRanking})
		}
	}
	return s
}
