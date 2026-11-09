package compare

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ModeWorktree marks a variant whose checkout is an existing work worktree that a provider
// named. dva neither creates nor removes it: it only starts and stops processes there.
const ModeWorktree = "worktree"

// TargetPrefix opens a worktree source: `worktree:<target-id>`. The target ID is defined by the
// provider; the git provider uses the branch name (the CE provider will use a task ID and the
// worktrunk provider a branch).
const TargetPrefix = "worktree:"

// Provider finds work worktrees. It is the seam the later providers (worktrunk, CE) plug into
// without compare importing them: it knows how to look a target up, nothing about starting it.
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

// targetProvider serves every `worktree:` source. Only the git provider exists in this build;
// this is the one place another provider is wired in.
var targetProvider Provider = gitProvider{}

// IsTargetSource reports whether source names a provider target rather than a git ref. A git
// ref cannot contain ":", so the two never collide.
func IsTargetSource(source string) bool { return strings.Contains(source, ":") }

// TargetID returns the ID in a `worktree:<id>` source.
func TargetID(source string) (string, error) {
	id, ok := strings.CutPrefix(source, TargetPrefix)
	if !ok {
		return "", fmt.Errorf("source %q names a provider target dva does not know; this build "+
			"knows %q only", source, TargetPrefix+"<branch>")
	}
	if strings.TrimSpace(id) == "" {
		return "", fmt.Errorf("source %q has no target ID after %q", source, TargetPrefix)
	}
	return id, nil
}

// ResolveTarget looks up source (`worktree:<id>`) in the repository holding configDir and
// returns it as a Variant (name unset): path, config directory and head now. It creates nothing.
func ResolveTarget(ctx context.Context, configDir, source string) (Variant, error) {
	id, err := TargetID(source)
	if err != nil {
		return Variant{}, err
	}
	t, err := targetProvider.Resolve(ctx, configDir, id)
	if err != nil {
		return Variant{}, err
	}
	rel, err := configRel(ctx, configDir)
	if err != nil {
		return Variant{}, err
	}
	return Variant{
		Mode: ModeWorktree, Source: source, Commit: t.Head,
		Path: t.Path, ConfigDir: filepath.Join(t.Path, rel),
	}, nil
}

// configRel is configDir's path below the top of its repository, so the same relative config
// directory can be found inside another worktree of that repository.
func configRel(ctx context.Context, configDir string) (string, error) {
	top, err := git(ctx, configDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return filepath.Rel(canonical(top), canonical(configDir))
}

// targetDrift reports how a recorded worktree variant no longer matches what its provider says.
// A moved head is not drift: a work worktree is being edited and committed to.
func targetDrift(ctx context.Context, configDir string, v Variant) []string {
	now, err := ResolveTarget(ctx, configDir, v.Source)
	if err != nil {
		return []string{fmt.Sprintf("provider no longer resolves %s: %v", v.Source, err)}
	}
	if canonical(now.Path) != canonical(v.Path) {
		return []string{fmt.Sprintf("%s now resolves to %s, record says %s", v.Source, now.Path, v.Path)}
	}
	return nil
}

// gitProvider reads `git worktree list`; the target ID is the branch checked out in the worktree.
type gitProvider struct{}

func (gitProvider) Resolve(ctx context.Context, repoDir, id string) (Target, error) {
	list, err := listWorktrees(ctx, repoDir)
	if err != nil {
		return Target{}, fmt.Errorf("look up worktree %q: %w", id, err)
	}
	var found []worktree
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
