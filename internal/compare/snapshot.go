package compare

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// SnapshotPath is where DVA checks out variant of comparison compareName, under configDir:
// `.sb/dva/variants/<compare>/<variant>/`.
func SnapshotPath(configDir, compareName, variant string) (string, error) {
	if !namePattern.MatchString(compareName) {
		return "", fmt.Errorf("compare name %q must match %s", compareName, namePattern)
	}
	if !namePattern.MatchString(variant) {
		return "", fmt.Errorf("variant name %q must match %s", variant, namePattern)
	}
	return filepath.Join(configDir, config.DotDirName, config.VariantsDirName, compareName, variant), nil
}

// EnsureSnapshot gives variant of compareName a detached checkout of ref in the repository that
// holds configDir. ref is pinned to a commit first, and no branch is created.
//
// prev is the variant's entry in the existing record, or nil. When prev names the same source
// and commit and that checkout is still registered at the same commit, it is reused and
// reused is true. Any other content already at the snapshot path is refused rather than
// replaced: DVA does not overwrite a checkout it cannot show it created for this source.
func EnsureSnapshot(ctx context.Context, configDir, compareName, variant, ref string, prev *Variant) (v Variant, reused bool, err error) {
	path, err := SnapshotPath(configDir, compareName, variant)
	if err != nil {
		return Variant{}, false, err
	}
	commit, err := resolveCommit(ctx, configDir, ref)
	if err != nil {
		return Variant{}, false, err
	}
	top, err := git(ctx, configDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return Variant{}, false, err
	}
	rel, err := filepath.Rel(canonical(top), canonical(configDir))
	if err != nil {
		return Variant{}, false, err
	}
	v = Variant{
		Name: variant, Mode: ModeSnapshot, Source: ref, Commit: commit,
		Path: path, ConfigDir: filepath.Join(path, rel),
	}

	// A snapshot the caller's checkout does not ignore shows up in its `git status` as an
	// untracked directory, and a later `git add -A` there would commit it as an embedded
	// repository. Refuse before creating anything.
	ignored, err := isIgnored(ctx, configDir, path)
	if err != nil {
		return Variant{}, false, err
	}
	if !ignored {
		return Variant{}, false, fmt.Errorf("snapshot path %s is not ignored by git in %s; "+
			"add the rule %q to .gitignore (dva doctor reports the full rule set)",
			path, configDir, config.DotDirName+"/*")
	}

	list, err := listWorktrees(ctx, configDir)
	if err != nil {
		return Variant{}, false, err
	}
	registered := findWorktree(list, path)
	_, statErr := os.Stat(path)
	exists := statErr == nil
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return Variant{}, false, statErr
	}

	if exists || registered != nil {
		switch {
		case registered == nil:
			return Variant{}, false, fmt.Errorf("snapshot path %s exists but is not a git worktree of "+
				"this repository; dva does not overwrite it", path)
		case !exists:
			return Variant{}, false, fmt.Errorf("snapshot path %s is registered as a git worktree but "+
				"missing on disk; inspect it with `git worktree list` before creating it again", path)
		case prev == nil || prev.Mode != ModeSnapshot || prev.Source != ref:
			return Variant{}, false, fmt.Errorf("snapshot path %s already holds a checkout this compare "+
				"record does not attribute to source %q; run `dva compare down %s %s` first",
				path, ref, compareName, variant)
		case prev.Commit != commit || registered.Head != commit || !registered.Detached:
			return Variant{}, false, fmt.Errorf("snapshot %s is at %s but %q now resolves to %s; run "+
				"`dva compare down %s %s` first to re-pin it", path, registered.Head, ref, commit,
				compareName, variant)
		}
		return v, true, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Variant{}, false, fmt.Errorf("create snapshot parent: %w", err)
	}
	if _, err := git(ctx, configDir, "worktree", "add", "--detach", "--", path, commit); err != nil {
		return Variant{}, false, err
	}
	return v, false, nil
}

// RemoveSnapshot reclaims the snapshot checkout of variant in rec, the record of the comparison
// configDir started. It removes only a snapshot the record names at DVA's own snapshot path,
// only through `git worktree remove` without force, and only when the checkout is clean. A
// refusal names the path so a person can inspect it. The record itself is the caller's to
// update.
func RemoveSnapshot(ctx context.Context, configDir string, rec *Record, variant string) error {
	v := rec.Variant(variant)
	if v == nil {
		return fmt.Errorf("variant %q is not in compare record %q; dva removes only checkouts it "+
			"recorded", variant, rec.Name)
	}
	if v.Mode != ModeSnapshot {
		return fmt.Errorf("variant %q of %q is a %s checkout at %s; dva removes only snapshots it "+
			"created", variant, rec.Name, v.Mode, v.Path)
	}
	want, err := SnapshotPath(configDir, rec.Name, variant)
	if err != nil {
		return err
	}
	if canonical(v.Path) != canonical(want) {
		return fmt.Errorf("compare record %q places variant %q at %s, not at dva's snapshot path %s; "+
			"refusing to remove it", rec.Name, variant, v.Path, want)
	}

	list, err := listWorktrees(ctx, configDir)
	if err != nil {
		return err
	}
	if findWorktree(list, want) == nil {
		return fmt.Errorf("snapshot path %s is not a git worktree of this repository; refusing to "+
			"remove it", want)
	}
	status, err := git(ctx, want, "status", "--porcelain")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("snapshot %s has uncommitted or untracked changes; dva does not remove "+
			"it (inspect with `git -C %s status`)", want, want)
	}
	if _, err := git(ctx, configDir, "worktree", "remove", "--", want); err != nil {
		return err
	}
	return nil
}
