package compare

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
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

// worktree is one entry of `git worktree list --porcelain`.
type worktree struct {
	Path     string
	Head     string
	Detached bool
	// Branch is the short branch name checked out there; empty when detached.
	Branch string
	Bare   bool
}

// listWorktrees returns every worktree of the repository dir belongs to, the main one included.
func listWorktrees(ctx context.Context, dir string) ([]worktree, error) {
	out, err := git(ctx, dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []worktree
	for block := range strings.SplitSeq(out, "\n\n") {
		var wt worktree
		for line := range strings.SplitSeq(block, "\n") {
			switch {
			case strings.HasPrefix(line, "worktree "):
				wt.Path = strings.TrimPrefix(line, "worktree ")
			case strings.HasPrefix(line, "HEAD "):
				wt.Head = strings.TrimPrefix(line, "HEAD ")
			case line == "detached":
				wt.Detached = true
			case line == "bare":
				wt.Bare = true
			case strings.HasPrefix(line, "branch refs/heads/"):
				wt.Branch = strings.TrimPrefix(line, "branch refs/heads/")
			}
		}
		if wt.Path != "" {
			list = append(list, wt)
		}
	}
	return list, nil
}

// findWorktree returns the worktree registered at path, or nil.
func findWorktree(list []worktree, path string) *worktree {
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

// canonical resolves symlinks so a path DVA built and the path git prints compare equal; on
// macOS a temporary directory under /var is reported by git as /private/var. For a path that
// no longer exists the nearest existing ancestor is resolved and the rest re-appended, so a
// registered but deleted worktree still matches what git prints.
func canonical(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	rest := ""
	for dir := abs; ; dir = filepath.Dir(dir) {
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			return filepath.Join(resolved, rest)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return abs
		}
		rest = filepath.Join(filepath.Base(dir), rest)
	}
}
