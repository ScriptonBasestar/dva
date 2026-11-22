package cirun

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Repository scope is an owner assertion, not a proof that shell commands are
// independent of their working directory or undeclared external state.
//
// It returns one key entry per PATH element, in PATH order. The checkout-local
// rule needs each element canonicalized, but an element that does not exist
// (macOS ships such entries in /etc/paths.d) or is not a directory cannot hold
// a tool now; it is keyed by its state instead (an existing element also by
// its canonical path, an absent one by the marker alone), so the key changes
// if that element later appears or changes kind.
func validateRepositoryReuse(root string, step config.CIStep, env []string) ([]string, error) {
	if filepath.IsAbs(step.Workdir) || strings.Contains(step.Workdir, root) || strings.Contains(step.Run, root) {
		return nil, errors.New("repository reuse forbids absolute workdir and checkout-root paths")
	}
	if len(step.Reuse.Probes) != 0 {
		return nil, errors.New("repository reuse does not support probes")
	}
	path := environmentMap(env)["PATH"]
	if path == "" {
		return nil, errors.New("repository reuse requires an absolute PATH")
	}
	entries := []string{}
	for _, dir := range filepath.SplitList(path) {
		if !filepath.IsAbs(dir) {
			return nil, errors.New("repository reuse forbids relative or empty PATH entries")
		}
		state, err := repositoryPathEntry(root, dir)
		if err != nil {
			return nil, err
		}
		entries = append(entries, dir+"\x00"+state)
	}
	return entries, nil
}

func repositoryPathEntry(root, dir string) (string, error) {
	resolved, err := filepath.EvalSymlinks(dir)
	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		// A missing element may sit under a checkout-local symlink; refuse it
		// when its nearest existing ancestor resolves inside the checkout.
		// The kernel never walks past a missing component, so a "." or ".."
		// after one has no defined target; real PATH entries never need them.
		if hasDotComponent(dir) {
			return "", fmt.Errorf("repository reuse forbids missing PATH entry %q with . or .. components", dir)
		}
		if err := refuseCheckoutLocal(root, dir, "PATH entries"); err != nil {
			return "", err
		}
		return "absent", nil
	}
	if err != nil {
		return "", fmt.Errorf("repository reuse PATH %q: %w", dir, err)
	}
	if err := refuseCheckoutLocal(root, resolved, "PATH entries"); err != nil {
		return "", err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("repository reuse PATH %q: %w", dir, err)
	}
	if !info.IsDir() {
		return "not-directory\x00" + resolved, nil
	}
	return "directory\x00" + resolved, nil
}

// refuseCheckoutLocal refuses path when it lies inside root, and also when
// containment cannot be decided: an error is a refusal that names its cause.
func refuseCheckoutLocal(root, path, what string) error {
	inside, err := pathInside(root, path)
	if err != nil {
		return fmt.Errorf("repository reuse cannot decide whether %q is checkout-local: %w", path, err)
	}
	if inside {
		return fmt.Errorf("repository reuse forbids checkout-local %s", what)
	}
	return nil
}

// statPath is os.Stat; tests replace it to make identity checks fail.
var statPath = os.Stat

// splitExisting canonicalizes the deepest existing ancestor of an absolute
// path and returns it with the lexical remainder below it, walking plain
// parents. The "." / ".." policy is enforced once, upstream in
// repositoryPathEntry; a dotted missing path here is a violated precondition
// (filepath.Dir would fold it lexically), so it is an error, never a verdict.
func splitExisting(path string) (string, string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, "", nil
	}
	if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
		return "", "", err
	}
	if hasDotComponent(path) {
		return "", "", fmt.Errorf("missing path %q has . or .. components", path)
	}
	missing := ""
	for current := filepath.Clean(path); ; {
		parent := filepath.Dir(current)
		if parent == current {
			return "", "", err
		}
		missing = filepath.Join(filepath.Base(current), missing)
		current = parent
		resolved, err = filepath.EvalSymlinks(current)
		if err == nil {
			return resolved, missing, nil
		}
		if !errors.Is(err, fs.ErrNotExist) && !errors.Is(err, syscall.ENOTDIR) {
			return "", "", err
		}
	}
}

func hasDotComponent(path string) bool {
	for part := range strings.SplitSeq(path, string(filepath.Separator)) {
		if part == "." || part == ".." {
			return true
		}
	}
	return false
}

// pathInside reports whether path is root or lies below it, deciding by file
// identity rather than spelling: on a case-insensitive volume two spellings
// name one directory. The canonical existing ancestor of path is walked upward
// and each directory compared with root by os.SameFile; a missing remainder
// below that ancestor is lexical and cannot leave it. Any failure to decide
// is returned as an error; callers must treat it as a refusal.
func pathInside(root, path string) (bool, error) {
	rootInfo, err := statPath(root)
	if err != nil {
		return false, fmt.Errorf("stat checkout root: %w", err)
	}
	ancestor, _, err := splitExisting(path)
	if err != nil {
		return false, fmt.Errorf("resolve %q: %w", path, err)
	}
	for current := ancestor; ; {
		info, err := statPath(current)
		if err != nil {
			return false, fmt.Errorf("stat ancestor: %w", err)
		}
		if os.SameFile(info, rootInfo) {
			return true, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false, nil
		}
		current = parent
	}
}
