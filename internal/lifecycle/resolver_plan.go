package lifecycle

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

type ExecutionPlan struct {
	Name            string
	EnvironmentName string
	SiteName        string
	EndpointTags    []string
	EnvVars         map[string]string
	Entries         []ResolvedEntry
	ResolutionTrace []string

	// Warnings carries the resolution facts a user needs even when they did not ask to
	// see the resolution. ResolutionTrace is the full narration and prints only under
	// --dry-run; anything in here prints on every path, so it stays short by policy —
	// an entry silently vanishing from the plan is the case it exists for (TASK-374).
	Warnings []string

	owner *config.Config
}

// OwnerConfig returns the configuration that supplied the declarations resolved
// into this plan. Plans constructed by callers before owner tracking retain the
// supplied fallback configuration.
func (p *ExecutionPlan) OwnerConfig(fallback *config.Config) *config.Config {
	if p != nil && p.owner != nil {
		return p.owner
	}
	return fallback
}

type ResolvedEntry struct {
	Name         string
	StackEntry   *config.LifecycleEntry
	Runner       string
	RunnerConfig any
	Order        int
	DependsOn    []string
	Profiles     []string
	Services     []string
	Wave         int
	WorkingDir   string
	Vars         map[string]string
}

func ResolvePlanName(cfg *config.Config, name string) (planName string, plan *config.PlanConfig, err error) {
	if cfg == nil {
		return "", nil, fmt.Errorf("nil config")
	}
	if len(cfg.Plans) == 0 {
		return "", nil, fmt.Errorf("no plans configured")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("plan name is empty")
	}
	resolved, ok := cfg.Plans[name]
	if !ok || resolved == nil {
		return "", nil, fmt.Errorf("plan %q not found", name)
	}
	return name, resolved, nil
}

// ResolvePlanAlias resolves an alias chain, returning the final concrete plan.
// It detects and reports cycles, self-references, and chain length violations.
// The returned plan is guaranteed to be a concrete plan (Alias == "").
func ResolvePlanAlias(cfg *config.Config, name string) (string, *config.PlanConfig, error) {
	const maxAliasDepth = 10
	visited := make(map[string]bool)
	current := name

	for depth := 0; depth <= maxAliasDepth; depth++ {
		if depth == maxAliasDepth {
			return "", nil, fmt.Errorf("plan %q: alias chain exceeds maximum depth of %d (possible cycle)", name, maxAliasDepth)
		}
		resolved, ok := cfg.Plans[current]
		if !ok || resolved == nil {
			return "", nil, fmt.Errorf("plan %q not found", current)
		}
		if resolved.Alias == "" {
			return current, resolved, nil
		}
		if resolved.Alias == current {
			return "", nil, fmt.Errorf("plan %q: alias cannot reference itself", current)
		}
		if visited[resolved.Alias] {
			return "", nil, fmt.Errorf("plan %q: alias cycle detected (via %q)", name, resolved.Alias)
		}
		visited[current] = true
		current = resolved.Alias
	}
	return "", nil, fmt.Errorf("plan %q: alias resolution failed", name)
}

// MergePlanExtends merges a child plan into its parent (extends).
// The parent must be a concrete plan (no alias, no composes).
// Returns the merged plan (a new PlanConfig) or an error.
func MergePlanExtends(cfg *config.Config, childName string, child *config.PlanConfig) (*config.PlanConfig, error) {
	if child.Extends == "" {
		return child, nil
	}
	if child.Alias != "" {
		return nil, fmt.Errorf("plan %q: cannot have both alias and extends", childName)
	}
	if len(child.Composes) > 0 {
		return nil, fmt.Errorf("plan %q: extends is mutually exclusive with composes", childName)
	}

	const maxExtendsDepth = 3
	parentName := child.Extends
	if parentName == childName {
		return nil, fmt.Errorf("plan %q: extends cannot reference itself", childName)
	}

	// Resolve parent (following any aliases)
	parentName, parent, err := ResolvePlanAlias(cfg, parentName)
	if err != nil {
		return nil, fmt.Errorf("plan %q: %w", childName, err)
	}

	if parent.Alias != "" {
		return nil, fmt.Errorf("plan %q: extends target %q must be a concrete plan (not an alias)", childName, parentName)
	}
	if len(parent.Composes) > 0 {
		return nil, fmt.Errorf("plan %q: extends target %q must be a concrete plan (not a composition plan)", childName, parentName)
	}

	// Check extends depth by walking up the chain
	depth := 1
	current := parent
	visited := map[string]bool{childName: true, parentName: true}
	for current.Extends != "" && depth < maxExtendsDepth {
		currentName := current.Extends
		if visited[currentName] {
			return nil, fmt.Errorf("plan %q: extends cycle detected at %q", childName, currentName)
		}
		visited[currentName] = true
		var err error
		currentName, current, err = ResolvePlanAlias(cfg, currentName)
		if err != nil {
			return nil, fmt.Errorf("plan %q: %w", childName, err)
		}
		if current.Alias != "" || len(current.Composes) > 0 {
			return nil, fmt.Errorf("plan %q: extends chain contains non-concrete plan %q", childName, currentName)
		}
		depth++
	}
	if depth >= maxExtendsDepth && current.Extends != "" {
		return nil, fmt.Errorf("plan %q: extends chain exceeds maximum depth of %d", childName, maxExtendsDepth)
	}

	// Deep clone parent and merge child into it
	merged := clonePlanConfig(parent)
	merged = mergePlanConfigs(merged, child)
	return merged, nil
}

// clonePlanConfig creates a deep copy of a PlanConfig.
func clonePlanConfig(p *config.PlanConfig) *config.PlanConfig {
	if p == nil {
		return nil
	}
	cloned := *p
	if p.Vars != nil {
		cloned.Vars = maps.Clone(p.Vars)
	}
	if p.Entries != nil {
		cloned.Entries = make([]config.PlanEntry, len(p.Entries))
		copy(cloned.Entries, p.Entries)
	}
	if p.Composes != nil {
		cloned.Composes = make([]config.CompositionEntry, len(p.Composes))
		copy(cloned.Composes, p.Composes)
	}
	if p.EndpointTags != nil {
		cloned.EndpointTags = slices.Clone(p.EndpointTags)
	}
	return &cloned
}

// mergePlanConfigs merges child into parent (parent is base, child overrides).
// Scalar fields: child wins if non-empty.
// Vars: key-merged (child keys override parent).
// Entries: matched by Name — child entry with same Name replaces parent entry entirely.
// New entries (Names not in parent) are appended.
func mergePlanConfigs(parent, child *config.PlanConfig) *config.PlanConfig {
	if child.Description != "" {
		parent.Description = child.Description
	}
	if child.Environment != "" {
		parent.Environment = child.Environment
	}
	if child.Site != "" {
		parent.Site = child.Site
	}
	if len(child.EndpointTags) > 0 {
		parent.EndpointTags = slices.Clone(child.EndpointTags)
	}
	if len(child.Vars) > 0 {
		if parent.Vars == nil {
			parent.Vars = make(map[string]string)
		}
		maps.Copy(parent.Vars, child.Vars)
	}

	// Merge entries by Name
	if len(child.Entries) > 0 {
		if parent.Entries == nil {
			parent.Entries = make([]config.PlanEntry, 0, len(child.Entries))
		}
		parentByName := make(map[string]int, len(parent.Entries))
		for i, e := range parent.Entries {
			parentByName[e.Name] = i
		}
		for _, childEntry := range child.Entries {
			if idx, exists := parentByName[childEntry.Name]; exists {
				parent.Entries[idx] = childEntry
			} else {
				parent.Entries = append(parent.Entries, childEntry)
			}
		}
	}
	return parent
}
