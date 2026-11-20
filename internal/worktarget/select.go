package worktarget

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// gitPrefix is the provider prefix of a git target ID (`git:<branch>`). The git provider is the
// only one today, so an unprefixed ID is read as git; any other prefix names a provider dva does
// not have and is refused rather than guessed (TASK-538 decision: prefixed IDs select the
// provider).
const gitPrefix = "git:"

// Selection is a target chosen by ID from the candidates of the caller's repository.
type Selection struct {
	Repository string // canonical common Git directory of the caller's repository
	Candidate
}

// Select resolves id against the candidates of the repository dir belongs to. Only an exact ID
// selects: an unknown ID, an unknown provider prefix or a failed lookup selects nothing, and the
// caller's own checkout is never a fallback. The default checkout is selectable because it was
// named.
func Select(ctx context.Context, dir, id string) (Selection, error) {
	branch, err := gitID(id)
	if err != nil {
		return Selection{}, err
	}
	repo, list, err := Candidates(ctx, dir)
	if err != nil {
		return Selection{}, err
	}
	var hits []Candidate
	var ids []string
	for _, c := range list {
		if c.ID == "" {
			continue
		}
		ids = append(ids, c.ID)
		if c.ID == branch {
			hits = append(hits, c)
		}
	}
	switch len(hits) {
	case 1:
		return Selection{Repository: repo, Candidate: hits[0]}, nil
	case 0:
		sort.Strings(ids)
		return Selection{}, fmt.Errorf("no target %q in this repository; targets: %s (see `dva target`)",
			id, strings.Join(ids, ", "))
	default:
		paths := make([]string, len(hits))
		for i, h := range hits {
			paths[i] = h.Path
		}
		return Selection{}, fmt.Errorf("target %q matches several worktrees (%s); dva does not pick one",
			id, strings.Join(paths, ", "))
	}
}

func gitID(id string) (string, error) {
	if rest, ok := strings.CutPrefix(id, gitPrefix); ok {
		id = rest
	} else if p, _, ok := strings.Cut(id, ":"); ok && !strings.Contains(p, "/") {
		return "", fmt.Errorf("target %q: unknown provider %q (known: git)", id, p)
	}
	if id == "" {
		return "", fmt.Errorf("empty target ID")
	}
	return id, nil
}

// Verify re-checks a selection immediately before anything runs in it, because the worktree can
// be removed, replaced or switched to another branch between selection and execution. It fails
// unless the same path is still a registered, existing worktree of the same repository with the
// same branch checked out. A new commit or uncommitted change is not a failure: the target is
// being worked on.
func Verify(ctx context.Context, s Selection) error {
	info, err := os.Stat(s.Path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return fmt.Errorf("target %s: worktree %s is gone", s.ID, s.Path)
	case err != nil:
		return fmt.Errorf("target %s: %w", s.ID, err)
	case !info.IsDir():
		return fmt.Errorf("target %s: %s is no longer a directory", s.ID, s.Path)
	}
	common, err := git(ctx, s.Path, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return fmt.Errorf("target %s: %w", s.ID, err)
	}
	if Canonical(common) != s.Repository {
		return fmt.Errorf("target %s: %s now belongs to another repository (%s, expected %s)",
			s.ID, s.Path, Canonical(common), s.Repository)
	}
	list, err := List(ctx, s.Path)
	if err != nil {
		return fmt.Errorf("target %s: %w", s.ID, err)
	}
	want := Canonical(s.Path)
	for _, wt := range list {
		if Canonical(wt.Path) != want {
			continue
		}
		if wt.Prunable {
			return fmt.Errorf("target %s: worktree %s is prunable", s.ID, s.Path)
		}
		if wt.Detached || wt.Branch != s.Branch {
			now := wt.Branch
			if now == "" {
				now = "a detached HEAD"
			}
			return fmt.Errorf("target %s: worktree %s now has %s checked out, not %s", s.ID, s.Path, now, s.Branch)
		}
		return nil
	}
	return fmt.Errorf("target %s: %s is no longer a worktree of this repository", s.ID, s.Path)
}

// ConfigDir maps the caller's directory into the target: the same path relative to the worktree
// root, so a dva.yml in a subdirectory is found at the same place in the target. It fails when
// that directory does not exist there.
func ConfigDir(ctx context.Context, dir string, s Selection) (string, error) {
	prefix, err := git(ctx, dir, "rev-parse", "--show-prefix")
	if err != nil {
		return "", err
	}
	out := filepath.Join(s.Path, filepath.FromSlash(prefix))
	if info, err := os.Stat(out); err != nil || !info.IsDir() {
		return "", fmt.Errorf("target %s has no %s (the directory this command was run from)", s.ID, prefix)
	}
	return out, nil
}
