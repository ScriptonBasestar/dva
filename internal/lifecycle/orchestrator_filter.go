package lifecycle

import (
	"fmt"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// partitionPlanServices splits a full project service list into the plan-selected
// subset and other services that are still running (or starting/restarting).
// Selected services missing from the project list are reported as "not found".
func partitionPlanServices(all []ServiceStatus, selected []string) (inPlan []ServiceStatus, outOfPlanRunning []ServiceStatus) {
	sel := make(map[string]struct{}, len(selected))
	for _, name := range selected {
		sel[name] = struct{}{}
	}
	seen := make(map[string]struct{}, len(selected))
	for _, s := range all {
		if _, ok := sel[s.Name]; ok {
			inPlan = append(inPlan, s)
			seen[s.Name] = struct{}{}
			continue
		}
		if serviceLooksRunning(s.State) {
			outOfPlanRunning = append(outOfPlanRunning, s)
		}
	}
	for _, name := range selected {
		if _, ok := seen[name]; ok {
			continue
		}
		inPlan = append(inPlan, ServiceStatus{Name: name, State: "not found"})
	}
	return inPlan, outOfPlanRunning
}

func serviceLooksRunning(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "running", "up", "healthy", "restarting", "starting":
		return true
	default:
		// docker compose often reports "running" only; accept prefix "up "
		// (e.g. "Up 2 days") from non-JSON fallbacks.
		low := strings.ToLower(state)
		return strings.HasPrefix(low, "up ") || strings.HasPrefix(low, "running")
	}
}

// filterEntries returns lifecycle entries matching the given name, tag, mode, and env filters.
// It also applies StackOverrides for the given environment if configured.
func (o *Orchestrator) filterEntries(names, includeTags, excludeTags []string, mode, env string) ([]config.LifecycleEntry, error) {
	if err := validateDeclaredTags(o.entries, includeTags); err != nil {
		return nil, err
	}
	if err := validateDeclaredTags(o.entries, excludeTags); err != nil {
		return nil, err
	}

	entries := o.entries

	// Filter by explicit entry names
	if len(names) > 0 {
		entries = filterByNames(entries, names)
	}

	// Filter by env (stack entry names)
	if env != "" {
		if ep, ok := o.cfg.Environments[env]; ok && len(ep.StackEntries()) > 0 {
			entries = filterByNames(entries, ep.StackEntries())
		}
	}

	// Filter by mode (stack entry names) — narrows further if both env and mode specify
	if mode != "" {
		if m, ok := o.cfg.Modes[mode]; ok && len(m.StackEntries()) > 0 {
			entries = filterByNames(entries, m.StackEntries())
		}
	}

	// Filter by include tags
	if len(includeTags) > 0 {
		entries = filterByTags(entries, includeTags, false)
	}

	// Filter by exclude tags
	if len(excludeTags) > 0 {
		entries = filterByTags(entries, excludeTags, true)
	}

	// Apply overrides after filtering is complete
	if env != "" {
		if ep, ok := o.cfg.Environments[env]; ok && len(ep.StackOverrides) > 0 {
			for i := range entries {
				if override, exists := ep.StackOverrides[entries[i].Name]; exists {
					merged, err := config.MergeLifecycleEntry(&entries[i], override)
					if err != nil {
						return nil, fmt.Errorf("applying env %q stack_override for %q: %w", env, entries[i].Name, err)
					}
					entries[i] = *merged
				}
			}
		}
	}

	return entries, nil
}

// validateDeclaredTags rejects a selector whose name no stack entry declares.
// It checks the full declaration set, before name, environment, or mode filters
// narrow the run, so a valid tag remains valid even when another selector removes
// its entry from this invocation.
func validateDeclaredTags(entries []config.LifecycleEntry, tags []string) error {
	declared := make(map[string]struct{})
	for _, entry := range entries {
		for _, tag := range entry.Tags {
			declared[tag] = struct{}{}
		}
	}
	for _, tag := range tags {
		if _, ok := declared[tag]; !ok {
			return fmt.Errorf("no entry declares tag %q", tag)
		}
	}
	return nil
}

// filterByNames retains only the entries whose names exist in targetNames.
func filterByNames(entries []config.LifecycleEntry, targetNames []string) []config.LifecycleEntry {
	nameSet := make(map[string]bool, len(targetNames))
	for _, n := range targetNames {
		nameSet[n] = true
	}
	var filtered []config.LifecycleEntry
	for _, e := range entries {
		if nameSet[e.Name] {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// filterByTags retains entries based on tag matching. If exclude is true, matching entries are excluded.
func filterByTags(entries []config.LifecycleEntry, tags []string, exclude bool) []config.LifecycleEntry {
	tagSet := make(map[string]bool, len(tags))
	for _, t := range tags {
		tagSet[t] = true
	}
	var filtered []config.LifecycleEntry
	for _, e := range entries {
		hasMatch := hasAnyTag(e.Tags, tagSet)
		if (exclude && !hasMatch) || (!exclude && hasMatch) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// hasAnyTag returns true if any of the entry's tags exist in the tag set.
func hasAnyTag(tags []string, tagSet map[string]bool) bool {
	for _, t := range tags {
		if tagSet[t] {
			return true
		}
	}
	return false
}
