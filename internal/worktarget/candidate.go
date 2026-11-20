package worktarget

import (
	"context"
	"fmt"
	"os"
)

// Candidate kinds. The main checkout is listed as default and is only run when its ID is named
// (TASK-538 decision); a detached worktree is a snapshot and has no target ID.
const (
	KindDefault  = "default"
	KindTask     = "task"
	KindSnapshot = "snapshot"
)

// Candidate is one worktree `dva target` can show. It is read from Git each time and never
// stored: there is no target registry and no remembered selection.
type Candidate struct {
	// ID is the git provider's target ID (the branch); empty for a snapshot, which cannot be
	// selected by ID.
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Branch string `json:"branch,omitempty"`
	Head   string `json:"head"`
	Dirty  bool   `json:"dirty"`
	Locked bool   `json:"locked"`
}

// Candidates lists the worktrees of the repository dir belongs to, together with that
// repository's identity (its canonical common Git directory). A bare entry, and a registered
// worktree whose directory is gone (prunable or missing), is left out: it cannot run anything.
// A dirty worktree stays, because previewing uncommitted work is the point. Any Git failure
// is returned; a partial list is never presented as complete.
func Candidates(ctx context.Context, dir string) (repo string, list []Candidate, err error) {
	common, err := git(ctx, dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", nil, fmt.Errorf("identify repository: %w", err)
	}
	wts, err := List(ctx, dir)
	if err != nil {
		return "", nil, fmt.Errorf("list worktrees: %w", err)
	}
	list = []Candidate{}
	for i, wt := range wts {
		if wt.Bare || wt.Prunable {
			continue
		}
		if info, statErr := os.Stat(wt.Path); statErr != nil || !info.IsDir() {
			continue
		}
		status, err := git(ctx, wt.Path, "status", "--porcelain", "--untracked-files=normal")
		if err != nil {
			return "", nil, fmt.Errorf("read status of %s: %w", wt.Path, err)
		}
		c := Candidate{
			ID: wt.Branch, Path: wt.Path, Branch: wt.Branch, Head: wt.Head,
			Dirty: status != "", Locked: wt.Locked,
		}
		switch {
		case i == 0: // git lists the main worktree first
			c.Kind = KindDefault
		case wt.Detached || wt.Branch == "":
			c.Kind = KindSnapshot
		default:
			c.Kind = KindTask
		}
		list = append(list, c)
	}
	return Canonical(common), list, nil
}
