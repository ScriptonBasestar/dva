package compare

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/worktarget"
)

// ModeWorktree marks a variant whose checkout is an existing work worktree that a provider
// named. dva neither creates nor removes it: it only starts and stops processes there.
const ModeWorktree = "worktree"

// TargetPrefix opens a worktree source: `worktree:<target-id>`. The target ID is defined by the
// provider; the git provider uses the branch name (the CE provider will use a task ID and the
// worktrunk provider a branch).
const TargetPrefix = "worktree:"

// targetProvider serves every `worktree:` source. Only the git provider exists in this build;
// this is the one place another provider is wired in.
var targetProvider worktarget.Provider = worktarget.Git{}

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
