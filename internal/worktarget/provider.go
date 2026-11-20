package worktarget

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Provider finds work worktrees. It is the seam the later providers (worktrunk, CE) plug into
// without their callers importing them: it knows how to look a target up, nothing about
// starting it.
type Provider interface {
	// Resolve returns the worktree the target ID names right now, or an error. An unknown ID,
	// several candidates and a failed lookup are all errors; there is no fallback checkout.
	Resolve(ctx context.Context, repoDir, id string) (Target, error)
}

// Target is one worktree a provider found.
type Target struct {
	ID     string // the provider's ID for it
	Path   string // worktree root
	Branch string // branch checked out there, empty when detached
	Head   string // commit checked out there now
}

// Git reads `git worktree list`; the target ID is the branch checked out in the worktree.
type Git struct{}

// Resolve implements Provider.
func (Git) Resolve(ctx context.Context, repoDir, id string) (Target, error) {
	list, err := List(ctx, repoDir)
	if err != nil {
		return Target{}, fmt.Errorf("look up worktree %q: %w", id, err)
	}
	var found []Worktree
	var branches []string
	for _, wt := range list {
		if wt.Bare || wt.Branch == "" {
			continue // a detached or bare entry has no target ID
		}
		branches = append(branches, wt.Branch)
		if wt.Branch == id {
			found = append(found, wt)
		}
	}
	switch len(found) {
	case 0:
		sort.Strings(branches)
		return Target{}, fmt.Errorf("no worktree has branch %q checked out; worktrees by branch: %s",
			id, strings.Join(branches, ", "))
	case 1:
		wt := found[0]
		if info, err := os.Stat(wt.Path); err != nil || !info.IsDir() {
			return Target{}, fmt.Errorf("worktree of branch %q is registered at %s but missing on "+
				"disk; `git worktree prune` removes the stale entry", id, wt.Path)
		}
		return Target{ID: id, Path: wt.Path, Branch: wt.Branch, Head: wt.Head}, nil
	}
	paths := make([]string, len(found))
	for i, wt := range found {
		paths[i] = wt.Path
	}
	return Target{}, fmt.Errorf("branch %q is checked out in several worktrees (%s); dva does not "+
		"choose one", id, strings.Join(paths, ", "))
}
