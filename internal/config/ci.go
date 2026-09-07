package config

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"
)

const (
	defaultCIProfileName     = "commit"
	defaultCICommitTimeout   = 10 * time.Minute
	defaultCICommitWarnAfter = 5 * time.Minute
	defaultCIMaxParallel     = 1
	maxCIMaxParallel         = 32
)

var ciNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// CIConfig declares named CI gate profiles.
type CIConfig struct {
	Profiles map[string]CIProfile `yaml:"profiles" json:"profiles"`
}

// CIProfile declares the ordered gates and shared budget for a CI invocation.
type CIProfile struct {
	Description string   `yaml:"description" json:"description"`
	Timeout     string   `yaml:"timeout" json:"timeout"`
	WarnAfter   string   `yaml:"warn_after" json:"warn_after"`
	MaxParallel int      `yaml:"max_parallel" json:"max_parallel"`
	Steps       []CIStep `yaml:"steps" json:"steps"`
}

// CIStep is one named CI gate.
type CIStep struct {
	Name        string            `yaml:"name" json:"name"`
	Run         string            `yaml:"run" json:"run"`
	DependsOn   []string          `yaml:"depends_on" json:"depends_on"`
	Workdir     string            `yaml:"workdir" json:"workdir"`
	Environment map[string]string `yaml:"environment" json:"environment"`
	Timeout     string            `yaml:"timeout" json:"timeout"`
}

// ResolveCIProfile returns an executable CI profile. An empty name selects commit.
// It never synthesizes steps: callers must explicitly declare every gate.
func (c *Config) ResolveCIProfile(name string) (CIProfile, error) {
	if name == "" {
		name = defaultCIProfileName
	}
	if !validCIName(name) {
		return CIProfile{}, fmt.Errorf("ci profile %q: name must match %s", name, ciNamePattern.String())
	}
	if name == "status" || name == "logs" {
		return CIProfile{}, fmt.Errorf("ci profile %q: name is reserved", name)
	}
	if c == nil || c.CI == nil {
		return CIProfile{}, fmt.Errorf("ci profile %q is not defined", name)
	}
	profile, ok := c.CI.Profiles[name]
	if !ok {
		return CIProfile{}, fmt.Errorf("ci profile %q is not defined", name)
	}
	if name == defaultCIProfileName {
		if profile.Timeout == "" {
			profile.Timeout = defaultCICommitTimeout.String()
		}
		if profile.WarnAfter == "" {
			profile.WarnAfter = defaultCICommitWarnAfter.String()
		}
	}
	if profile.MaxParallel == 0 {
		profile.MaxParallel = defaultCIMaxParallel
	}
	if err := validateCIProfile(name, profile); err != nil {
		return CIProfile{}, err
	}
	return profile, nil
}

func (c *Config) validateCIProfiles() error {
	if c == nil || c.CI == nil {
		return nil
	}
	names := make([]string, 0, len(c.CI.Profiles))
	for name := range c.CI.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if _, err := c.ResolveCIProfile(name); err != nil {
			return err
		}
	}
	return nil
}

func validateCIProfile(name string, profile CIProfile) error {
	if !validCIName(name) {
		return fmt.Errorf("ci profile %q: name must match %s", name, ciNamePattern.String())
	}
	if name == "status" || name == "logs" {
		return fmt.Errorf("ci profile %q: name is reserved", name)
	}
	if profile.MaxParallel < 1 || profile.MaxParallel > maxCIMaxParallel {
		return fmt.Errorf("ci profile %q: max_parallel must be between 1 and %d", name, maxCIMaxParallel)
	}
	profileTimeout, err := parsePositiveCIDuration(name, "timeout", profile.Timeout)
	if err != nil {
		return err
	}
	if name == defaultCIProfileName && profileTimeout > defaultCICommitTimeout {
		return fmt.Errorf("ci profile %q: timeout must not exceed %s", name, defaultCICommitTimeout)
	}
	if profile.WarnAfter != "" {
		warnAfter, err := parsePositiveCIDuration(name, "warn_after", profile.WarnAfter)
		if err != nil {
			return err
		}
		if warnAfter > profileTimeout {
			return fmt.Errorf("ci profile %q: warn_after must not exceed timeout", name)
		}
		if name == defaultCIProfileName && warnAfter > defaultCICommitWarnAfter {
			return fmt.Errorf("ci profile %q: warn_after must not exceed %s", name, defaultCICommitWarnAfter)
		}
	}
	if len(profile.Steps) == 0 {
		return fmt.Errorf("ci profile %q: steps must not be empty", name)
	}

	steps := make(map[string]CIStep, len(profile.Steps))
	for _, step := range profile.Steps {
		if !validCIName(step.Name) {
			return fmt.Errorf("ci profile %q: step name %q must match %s", name, step.Name, ciNamePattern.String())
		}
		if _, exists := steps[step.Name]; exists {
			return fmt.Errorf("ci profile %q: duplicate step name %q", name, step.Name)
		}
		if strings.TrimSpace(step.Run) == "" {
			return fmt.Errorf("ci profile %q step %q: run must not be empty", name, step.Name)
		}
		if step.Timeout != "" {
			stepTimeout, err := parsePositiveCIDuration(name+" step "+fmt.Sprintf("%q", step.Name), "timeout", step.Timeout)
			if err != nil {
				return err
			}
			if stepTimeout > profileTimeout {
				return fmt.Errorf("ci profile %q step %q: timeout must not exceed profile timeout", name, step.Name)
			}
		}
		steps[step.Name] = step
	}
	for stepName, step := range steps {
		seenDependencies := make(map[string]struct{}, len(step.DependsOn))
		for _, dependency := range step.DependsOn {
			if !validCIName(dependency) {
				return fmt.Errorf("ci profile %q step %q: dependency %q must match %s", name, stepName, dependency, ciNamePattern.String())
			}
			if _, duplicate := seenDependencies[dependency]; duplicate {
				return fmt.Errorf("ci profile %q step %q: duplicate dependency %q", name, stepName, dependency)
			}
			seenDependencies[dependency] = struct{}{}
			if _, exists := steps[dependency]; !exists {
				return fmt.Errorf("ci profile %q step %q: dependency %q is not defined", name, stepName, dependency)
			}
		}
	}
	if hasCIStepCycle(steps) {
		return fmt.Errorf("ci profile %q: step dependencies contain a cycle", name)
	}
	return nil
}

func parsePositiveCIDuration(subject, field, value string) (time.Duration, error) {
	duration, err := time.ParseDuration(value)
	if err != nil || duration <= 0 {
		return 0, fmt.Errorf("ci %s: %s must be a positive duration", subject, field)
	}
	return duration, nil
}

func validCIName(name string) bool { return ciNamePattern.MatchString(name) }

func hasCIStepCycle(steps map[string]CIStep) bool {
	const (
		unseen = iota
		visiting
		visited
	)
	states := make(map[string]int, len(steps))
	var visit func(string) bool
	visit = func(name string) bool {
		switch states[name] {
		case visiting:
			return true
		case visited:
			return false
		}
		states[name] = visiting
		if slices.ContainsFunc(steps[name].DependsOn, visit) {
			return true
		}
		states[name] = visited
		return false
	}
	for name := range steps {
		if states[name] == unseen && visit(name) {
			return true
		}
	}
	return false
}

func mergeCIProfile(base, other CIProfile) CIProfile {
	if other.Description != "" {
		base.Description = other.Description
	}
	if other.Timeout != "" {
		base.Timeout = other.Timeout
	}
	if other.WarnAfter != "" {
		base.WarnAfter = other.WarnAfter
	}
	if other.MaxParallel != 0 {
		base.MaxParallel = other.MaxParallel
	}
	if other.Steps != nil {
		base.Steps = other.Steps
	}
	return base
}
