package cirun

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const maxCompositionDepth = 8

type compositionFrame struct {
	RunID string `json:"run_id"`
	Root  string `json:"root"`
	Step  string `json:"step"`
}

type compositionGrant struct {
	Version        int                `json:"version"`
	Ancestry       []compositionFrame `json:"ancestry"`
	Deadline       string             `json:"deadline"`
	JobsLimit      int                `json:"jobs_limit"`
	LockState      string             `json:"lock_state"`
	WorkerDefaults map[string]string  `json:"worker_defaults,omitempty"`
}

func compositionFailure(kind, detail string) error {
	return fmt.Errorf("ci composition %s: %s", kind, detail)
}

func canonicalCompositionDir(value string) error {
	if !filepath.IsAbs(value) || filepath.Clean(value) != value {
		return fmt.Errorf("expected a canonical absolute directory")
	}
	canonical, err := filepath.EvalSymlinks(value)
	if err != nil {
		return err
	}
	if canonical != value {
		return fmt.Errorf("directory is not canonical")
	}
	info, err := os.Stat(value)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("expected a directory")
	}
	return nil
}

func compositionDeadline(value string) (time.Time, error) {
	deadline, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, err
	}
	if !strings.HasSuffix(value, "Z") || deadline.UTC().Format(time.RFC3339Nano) != value {
		return time.Time{}, fmt.Errorf("deadline must be canonical UTC RFC3339Nano")
	}
	return deadline, nil
}

// Admission validates the entire grant before any child command or lock acquisition.
// The caller must still register lifecycle ownership before issuing child work.
func admitComposition(state, childRoot, parentValue, grantValue string, now time.Time,
	probe func(string, string, string) (string, error),
) (*compositionGrant, error) {
	var grant compositionGrant
	if err := decodeCompositionJSON(grantValue, &grant); err != nil {
		return nil, compositionFailure("invalid", err.Error())
	}
	if grant.Version != 1 || len(grant.Ancestry) == 0 || len(grant.Ancestry) > maxCompositionDepth || grant.JobsLimit < 1 {
		return nil, compositionFailure("invalid", "unsupported version, depth or jobs limit")
	}
	for key, value := range grant.WorkerDefaults {
		n, err := strconv.Atoi(value)
		if (key != "CARGO_BUILD_JOBS" && key != "RUST_TEST_THREADS") || err != nil || n < 1 || n > grant.JobsLimit || strconv.Itoa(n) != value {
			return nil, compositionFailure("jobs", "invalid runtime worker default provenance")
		}
	}
	deadline, err := compositionDeadline(grant.Deadline)
	if err != nil {
		return nil, compositionFailure("invalid", err.Error())
	}
	if !now.Before(deadline) {
		return nil, compositionFailure("expired", "parent step deadline has elapsed")
	}
	if err := canonicalCompositionDir(state); err != nil {
		return nil, compositionFailure("state", err.Error())
	}
	if grant.LockState != state {
		return nil, compositionFailure("state", "child and grant lock stores differ")
	}
	if err := canonicalCompositionDir(childRoot); err != nil {
		return nil, compositionFailure("root", err.Error())
	}
	var parent parentRun
	if err := decodeCompositionJSON(parentValue, &parent); err != nil {
		return nil, compositionFailure("parent", err.Error())
	}
	last := grant.Ancestry[len(grant.Ancestry)-1]
	if parent.ID != last.RunID || parent.Root != last.Root {
		return nil, compositionFailure("parent", "immediate parent does not match final ancestry frame")
	}
	seen := map[string]bool{}
	for _, frame := range grant.Ancestry {
		if !safeID(frame.RunID) || !config.IsCIName(frame.Step) {
			return nil, compositionFailure("identity", "invalid ancestor run or step ID")
		}
		if err := canonicalCompositionDir(frame.Root); err != nil {
			return nil, compositionFailure("root", err.Error())
		}
		if seen[frame.Root] {
			return nil, compositionFailure("invalid", "duplicate ancestry root")
		}
		seen[frame.Root] = true
	}
	if probe == nil {
		probe = lockOwner
	}
	for _, frame := range grant.Ancestry {
		owner, err := probe(state, "root", frame.Root)
		if err != nil {
			return nil, compositionFailure("ancestor", err.Error())
		}
		if owner != frame.RunID {
			return nil, compositionFailure("ancestor", "ancestor lock is stale or belongs to a different run")
		}
	}
	for _, frame := range grant.Ancestry {
		if frame.Root == childRoot {
			return nil, &BusyError{ActiveRunID: frame.RunID, Conflict: Conflict{
				Kind: "composition_cycle", Key: frame.Root, RunID: frame.RunID,
			}}
		}
	}
	return &grant, nil
}
