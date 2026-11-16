package cirun

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Repository scope is an owner assertion, not a proof that shell commands are
// independent of their working directory or undeclared external state.
func validateRepositoryReuse(root string, step config.CIStep, env []string) error {
	if filepath.IsAbs(step.Workdir) || strings.Contains(step.Workdir, root) || strings.Contains(step.Run, root) {
		return errors.New("repository reuse forbids absolute workdir and checkout-root paths")
	}
	if len(step.Reuse.Probes) != 0 {
		return errors.New("repository reuse does not support probes")
	}
	path := environmentMap(env)["PATH"]
	if path == "" {
		return errors.New("repository reuse requires an absolute PATH")
	}
	for _, dir := range filepath.SplitList(path) {
		if !filepath.IsAbs(dir) {
			return errors.New("repository reuse forbids relative or empty PATH entries")
		}
		resolved, err := filepath.EvalSymlinks(dir)
		if err != nil {
			return fmt.Errorf("repository reuse PATH %q: %w", dir, err)
		}
		if pathInside(root, resolved) {
			return errors.New("repository reuse forbids checkout-local PATH entries")
		}
	}
	return nil
}

func pathInside(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
