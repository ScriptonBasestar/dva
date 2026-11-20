package compare

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/worktarget"
)

// git runs one git command in dir and returns its trimmed stdout. A failure carries git's
// stderr, because that is where git says which path or ref it refused.
func git(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// findWorktree returns the worktree registered at path, or nil.
func findWorktree(list []worktarget.Worktree, path string) *worktarget.Worktree {
	want := canonical(path)
	for i := range list {
		if canonical(list[i].Path) == want {
			return &list[i]
		}
	}
	return nil
}

// resolveCommit pins ref to the full commit it names. `--end-of-options` keeps a ref that
// starts with "-" from being read as a flag.
func resolveCommit(ctx context.Context, dir, ref string) (string, error) {
	if strings.TrimSpace(ref) == "" {
		return "", errors.New("source ref is empty")
	}
	commit, err := git(ctx, dir, "rev-parse", "--verify", "--quiet", "--end-of-options", ref+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("ref %q does not name a commit in %s", ref, dir)
	}
	return commit, nil
}

// isIgnored reports whether git ignores path in the checkout at dir. check-ignore exits 1 for a
// path it does not ignore and 128 for a real failure; only the latter is an error.
func isIgnored(ctx context.Context, dir, path string) (bool, error) {
	_, err := git(ctx, dir, "check-ignore", "--quiet", "--", path)
	if err == nil {
		return true, nil
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok && exitErr.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// canonical is worktarget.Canonical: a path DVA built and the path git prints compare equal.
func canonical(path string) string { return worktarget.Canonical(path) }
