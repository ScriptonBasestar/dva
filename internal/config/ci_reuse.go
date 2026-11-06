package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// CIFreshEnv is supervisor-owned. Projects use it to bypass language test caches.
const CIFreshEnv = "DVA_CI_FRESH"

// CIReuse declares the complete inputs of a deterministic native check. Omission
// always executes the check. Artifacts must still exist and match Outputs on a hit.
type CIReuse struct {
	Inputs            []string `yaml:"inputs" json:"inputs"`
	Tools             []string `yaml:"tools" json:"tools"`
	Environment       []string `yaml:"environment,omitempty" json:"environment,omitempty"`
	Outputs           []string `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	MaxAge            string   `yaml:"max_age" json:"max_age"`
	Probes            []string `yaml:"probes,omitempty" json:"probes,omitempty"`
	UnlessEnvironment []string `yaml:"unless_environment,omitempty" json:"unless_environment,omitempty"`
}

var ciEnvironmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func validateCIReuse(step CIStep) error {
	r := step.Reuse
	if r == nil {
		return nil
	}
	if step.OCI != nil || step.ChildCI != "" {
		return fmt.Errorf("reuse is supported only on native steps without child_ci")
	}
	if len(r.Inputs) == 0 || len(r.Tools) == 0 {
		return fmt.Errorf("reuse requires nonempty inputs and tools")
	}
	if age, err := time.ParseDuration(r.MaxAge); err != nil || age <= 0 {
		return fmt.Errorf("reuse.max_age must be a positive duration")
	}
	for _, path := range append(append([]string{}, r.Inputs...), r.Outputs...) {
		if strings.TrimSpace(path) == "" || filepath.IsAbs(path) || strings.Contains(path, "\\") {
			return fmt.Errorf("reuse paths must be nonempty repository-relative paths")
		}
		for part := range strings.SplitSeq(path, "/") {
			if part == ".." || part == ".git" {
				return fmt.Errorf("reuse paths must not escape the repository or read .git")
			}
		}
		// Git magic can replace prefix/path semantics; ordinary Git globs remain supported.
		if strings.HasPrefix(path, ":") {
			return fmt.Errorf("reuse.inputs does not accept Git pathspec magic")
		}
	}
	for _, tool := range r.Tools {
		if tool == "" || strings.ContainsAny(tool, "/\\ \t\n") {
			return fmt.Errorf("reuse.tools requires executable names resolved through the step PATH")
		}
	}
	for _, key := range append(append([]string{}, r.Environment...), r.UnlessEnvironment...) {
		if !ciEnvironmentName.MatchString(key) || key == CIParentRunEnv || key == CICompositionEnv || key == CIFreshEnv {
			return fmt.Errorf("reuse.environment contains invalid or volatile key %q", key)
		}
	}
	for _, command := range r.Probes {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("reuse.probes must not contain empty commands")
		}
	}
	return nil
}
