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
// a tool now; it is keyed by its state instead, so the key changes if that
// element later appears or changes kind.
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
		if pathInside(root, resolveMissingPath(filepath.Clean(dir))) {
			return "", errors.New("repository reuse forbids checkout-local PATH entries")
		}
		return "absent", nil
	}
	if err != nil {
		return "", fmt.Errorf("repository reuse PATH %q: %w", dir, err)
	}
	if pathInside(root, resolved) {
		return "", errors.New("repository reuse forbids checkout-local PATH entries")
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

// resolveMissingPath canonicalizes the nearest existing ancestor of path and
// appends the missing remainder, so symlinked parents are seen.
func resolveMissingPath(path string) string {
	rest := ""
	for current := path; ; current = filepath.Dir(current) {
		if resolved, err := filepath.EvalSymlinks(current); err == nil {
			return filepath.Join(resolved, rest)
		}
		if filepath.Dir(current) == current {
			return path
		}
		rest = filepath.Join(filepath.Base(current), rest)
	}
}

func pathInside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
