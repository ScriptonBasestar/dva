// Package worktarget finds the work worktrees of a repository. It is the neutral home of the
// target providers that both `dva compare` (`worktree:<id>` variants) and the `dva target`
// surface (TASK-538) use: it looks targets up and starts nothing.
package worktarget

import (
	"bytes"
	"context"
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

// Worktree is one entry of `git worktree list --porcelain`.
type Worktree struct {
	Path     string
	Head     string
	Detached bool
	// Branch is the short branch name checked out there; empty when detached.
	Branch string
	Bare   bool
}

// List returns every worktree of the repository dir belongs to, the main one included.
func List(ctx context.Context, dir string) ([]Worktree, error) {
	out, err := git(ctx, dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var list []Worktree
	for block := range strings.SplitSeq(out, "\n\n") {
		var wt Worktree
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

// Canonical resolves symlinks so a path DVA built and the path git prints compare equal; on
// macOS a temporary directory under /var is reported by git as /private/var. For a path that
// no longer exists the nearest existing ancestor is resolved and the rest re-appended, so a
// registered but deleted worktree still matches what git prints.
func Canonical(path string) string {
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
