package compare

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

// VariantState is Check's verdict on one recorded variant. A stale variant carries every
// reason found, not just the first, so one report explains the whole mismatch.
type VariantState struct {
	Name    string   `json:"name"`
	Stale   bool     `json:"stale"`
	Reasons []string `json:"reasons,omitempty"`
}

// Check re-verifies every variant of rec against what exists now: the checkout path, its
// registration in `git worktree list` of the repository holding configDir, the commit it is
// at, and the pids the variant's perVariant entries left in its own `.sb/dva/pids/`.
//
// It only reads. A stale variant is reported, never repaired or dropped from the record,
// because which side is wrong — the record or the checkout — is not something Check can know.
// An entry without a pid file is not stale: compose and other runners record no pid.
func Check(ctx context.Context, configDir string, rec *Record, perVariant []string) ([]VariantState, error) {
	list, err := listWorktrees(ctx, configDir)
	if err != nil {
		return nil, err
	}
	states := make([]VariantState, 0, len(rec.Variants))
	for _, v := range rec.Variants {
		var reasons []string
		info, statErr := os.Stat(v.Path)
		switch {
		case errors.Is(statErr, fs.ErrNotExist):
			reasons = append(reasons, fmt.Sprintf("checkout %s does not exist", v.Path))
		case statErr != nil:
			reasons = append(reasons, fmt.Sprintf("checkout %s cannot be read: %v", v.Path, statErr))
		case !info.IsDir():
			reasons = append(reasons, fmt.Sprintf("checkout %s is not a directory", v.Path))
		}
		if v.Mode == ModeWorktree {
			// The provider is asked again: a work worktree is not dva's, so its path or branch
			// may have changed since up.
			reasons = append(reasons, targetDrift(ctx, configDir, v)...)
		} else if wt := findWorktree(list, v.Path); wt == nil {
			reasons = append(reasons, fmt.Sprintf("%s is not a git worktree of this repository", v.Path))
		} else if wt.Head != v.Commit {
			reasons = append(reasons, fmt.Sprintf("checkout is at %s, record says %s", wt.Head, v.Commit))
		}
		reasons = append(reasons, deadPids(v.ConfigDir, perVariant)...)
		states = append(states, VariantState{Name: v.Name, Stale: len(reasons) > 0, Reasons: reasons})
	}
	return states, nil
}

// deadPids reports each entry whose pid file names a process that is not running, or cannot be
// read as a pid.
func deadPids(configDir string, entries []string) []string {
	var reasons []string
	for _, entry := range entries {
		pidFile := filepath.Join(configDir, config.DotDirName, config.PidsDirName, entry+".pid")
		data, err := os.ReadFile(pidFile)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			reasons = append(reasons, fmt.Sprintf("entry %q pid file %s cannot be read: %v", entry, pidFile, err))
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil || pid <= 0 {
			reasons = append(reasons, fmt.Sprintf("entry %q pid file %s does not hold a pid", entry, pidFile))
			continue
		}
		if !lifecycle.IsProcessRunning(pid) {
			reasons = append(reasons, fmt.Sprintf("entry %q pid %d is not running", entry, pid))
		}
	}
	return reasons
}
