package cirun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

type LookupReport struct {
	Root    string       `json:"root"`
	Profile string       `json:"profile"`
	Steps   []LookupStep `json:"steps"`
	Covered bool         `json:"covered"`
}

type LookupStep struct {
	Name          string    `json:"name"`
	State         string    `json:"state"`
	Reason        string    `json:"reason,omitempty"`
	Key           string    `json:"key,omitempty"`
	OriginalRunID string    `json:"original_run_id,omitempty"`
	FinishedAt    time.Time `json:"finished_at,omitzero"`
}

// Lookup inspects cached evidence without running checks or writing CI state.
// A covered result is an observation, not an attestation of a CI run.
func Lookup(parent context.Context, opts Options) (LookupReport, error) {
	return lookup(parent, opts, nil)
}

func lookup(parent context.Context, opts Options, observed func(int)) (LookupReport, error) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()
	var result LookupReport
	if opts.ProfileName == "" || opts.Root == "" {
		return result, errors.New("ci root and profile name are required")
	}
	declared := &config.Config{CI: &config.CIConfig{Profiles: map[string]config.CIProfile{opts.ProfileName: opts.Profile}}}
	profile, err := declared.ResolveCIProfile(opts.ProfileName)
	if err != nil {
		return result, err
	}
	root, err := filepath.EvalSymlinks(opts.Root)
	if err != nil {
		return result, err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return result, err
	}
	result = LookupReport{Root: root, Profile: opts.ProfileName, Covered: true, Steps: make([]LookupStep, 0, len(profile.Steps))}
	jobs, _ := toolWorkerBudget(profile, availableCPUs())
	inherited := setCIEnvironment(subprojectEnv(opts.Env, opts.Subproject), config.CIFreshEnv, "0")
	_, composition := compositionGrantValue(opts.Env)
	for i, step := range profile.Steps {
		s := LookupStep{Name: step.Name, State: "not-reusable"}
		switch {
		case step.Condition != nil:
			s.Reason = "condition-not-evaluated"
		case opts.Fresh || profile.Fresh:
			s.Reason = "fresh"
		case composition:
			s.Reason = "composition context"
		case step.Reuse == nil || step.OCI != nil || step.ChildCI != "":
			s.Reason = "no native reuse declaration"
		case len(step.Reuse.Probes) > 0:
			s.Reason = "lookup does not execute probes"
		default:
			actualEnv := ciEnv(inherited, step.Environment, jobs)
			for _, key := range step.Reuse.UnlessEnvironment {
				if environmentMap(actualEnv)[key] != "" {
					s.Reason = "environment:" + key
					break
				}
			}
			if s.Reason == "" {
				s, err = lookupStep(ctx, root, step, actualEnv)
				if err != nil {
					return result, err
				}
			}
		}
		result.Steps = append(result.Steps, s)
		if observed != nil {
			observed(i)
		}
	}
	for i, step := range profile.Steps {
		s := &result.Steps[i]
		if s.State == "covered" {
			env := ciEnv(inherited, step.Environment, jobs)
			plan, err := reuseFor(ctx, root, step, env)
			if err != nil {
				if ctx.Err() != nil {
					return result, ctx.Err()
				}
				s.State, s.Reason = "stale", "inputs unavailable after lookup: "+err.Error()
			} else if plan.key != s.Key {
				s.State, s.Reason = "stale", "inputs changed during lookup"
			} else {
				age, _ := time.ParseDuration(step.Reuse.MaxAge)
				record, reason := loadReuse(plan, age, time.Now(), root)
				if record == nil {
					s.State, s.Reason = "stale", reason
				} else if record.RunID != s.OriginalRunID || !record.FinishedAt.Equal(s.FinishedAt) {
					s.State, s.Reason = "stale", "cache record changed during lookup"
				}
			}
		}
		result.Covered = result.Covered && s.State == "covered"
	}
	return result, nil
}

func lookupStep(ctx context.Context, root string, step config.CIStep, env []string) (LookupStep, error) {
	s := LookupStep{Name: step.Name}
	plan, err := reuseFor(ctx, root, step, env)
	if err != nil {
		return s, err
	}
	s.Key = plan.key
	maxAge, _ := time.ParseDuration(step.Reuse.MaxAge)
	record, reason := loadReuse(plan, maxAge, time.Now(), root)
	switch {
	case record != nil:
		s.State, s.OriginalRunID, s.FinishedAt = "covered", record.RunID, record.FinishedAt
	case reason != "miss":
		s.State, s.Reason = "stale", reason
		// Preserve provenance if the original entry can be parsed and identifies this key.
		if previous, err := readLookupRecord(plan.cache); err == nil && previous.Key == plan.key {
			s.OriginalRunID, s.FinishedAt = previous.RunID, previous.FinishedAt
		}
	default:
		s.State = "absent"
		entries, err := os.ReadDir(filepath.Dir(plan.cache))
		if err != nil && !os.IsNotExist(err) {
			return s, fmt.Errorf("read reuse cache: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			previous, err := readLookupRecord(filepath.Join(filepath.Dir(plan.cache), entry.Name()))
			if err != nil {
				return s, fmt.Errorf("read reuse cache entry %q: %w", entry.Name(), err)
			}
			if previous.StepName == step.Name && previous.FinishedAt.After(s.FinishedAt) {
				s.State, s.Reason = "stale", "changed key"
				s.OriginalRunID, s.FinishedAt = previous.RunID, previous.FinishedAt
			}
		}
	}
	again, err := reuseFor(ctx, root, step, env)
	if err != nil {
		return s, err
	}
	if again.key != plan.key {
		s.State, s.Reason = "stale", "inputs changed during lookup"
	}
	return s, nil
}

func readLookupRecord(path string) (reuseRecord, error) {
	var record reuseRecord
	b, err := os.ReadFile(path)
	if err != nil {
		return record, err
	}
	if err := json.Unmarshal(b, &record); err != nil || record.Schema != reuseSchema || record.Key == "" || record.RunID == "" || record.FinishedAt.IsZero() {
		return record, errors.New("malformed cache")
	}
	return record, nil
}
