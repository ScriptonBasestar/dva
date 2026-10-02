package config

import (
	"fmt"
	"sort"
	"strings"
)

// warnHealthCheckRedundancy warns when both start and start_hint are set on a health check.
func (c *Config) warnHealthCheckRedundancy() []string {
	var warnings []string

	// Top-level health checks
	for name, hc := range c.HealthChecks {
		if hc.Start != "" && hc.StartHint != "" {
			warnings = append(warnings,
				fmt.Sprintf("health_checks.%s: 'start_hint' is redundant when 'start' is set (auto-start takes priority)", name))
		}
	}

	// Stack-nested health checks
	for entryName, entry := range c.Stack {
		for name, hc := range entry.HealthChecks {
			if hc.Start != "" && hc.StartHint != "" {
				warnings = append(warnings,
					fmt.Sprintf("stack.%s.health_checks.%s: 'start_hint' is redundant when 'start' is set (auto-start takes priority)", entryName, name))
			}
		}
	}

	// Both sources are maps, so without this the same dva.yml prints these warnings in a
	// different order between runs — 4 distinct orderings over 15 runs of the real binary with
	// the sort removed, and that count is itself a sample, not a constant. TASK-125 sorted
	// three interaction checks and left this one. TASK-128.
	sort.Strings(warnings)
	return warnings
}

// warnUnreachableHealthChecks warns when a top-level health_checks entry declares start
// and/or start_hint but no modes.*.health_checks list references it.
//
// Top-level start is executed through exactly one path — Orchestrator.startModeProcesses —
// and that path returns early unless opts.Mode is set, the mode exists, and the mode's
// health_checks list names the entry (signalModeProcesses gates the down/stop half the
// same way). Without a mode reference, start is dead config and start_hint never surfaces
// through that path either. Nested stack.<entry>.health_checks is a different field on a
// different code path (un-gated readiness/wait); this pass does not touch it. TASK-179.
//
// The same pass also names modes.*.health_checks entries that point at a missing
// health_checks name: startModeProcesses silently continues on !ok, so a typo used to
// validate clean and do nothing at runtime.
//
// Severity: Semantic Warning
func (c *Config) warnUnreachableHealthChecks() []string {
	referenced := make(map[string]bool)
	for _, mode := range c.Modes {
		for _, name := range mode.HealthChecks {
			referenced[name] = true
		}
	}

	var warnings []string

	for name, hc := range c.HealthChecks {
		if hc.Start == "" && hc.StartHint == "" {
			continue
		}
		if referenced[name] {
			continue
		}
		var fields []string
		if hc.Start != "" {
			fields = append(fields, "start")
		}
		if hc.StartHint != "" {
			fields = append(fields, "start_hint")
		}
		warnings = append(warnings, fmt.Sprintf(
			"health_checks.%s: declares %s but no modes.*.health_checks entry references it, so auto-start/hint will never run; add it to a mode's health_checks list, move the check under stack.<entry>.health_checks, or remove the field",
			name, strings.Join(fields, " and ")))
	}

	// Dangling mode references: sort mode names so map iteration cannot reorder warnings.
	modeNames := make([]string, 0, len(c.Modes))
	for modeName := range c.Modes {
		modeNames = append(modeNames, modeName)
	}
	sort.Strings(modeNames)
	for _, modeName := range modeNames {
		mode := c.Modes[modeName]
		for _, hcName := range mode.HealthChecks {
			if _, ok := c.HealthChecks[hcName]; ok {
				continue
			}
			warnings = append(warnings, fmt.Sprintf(
				"modes.%s.health_checks: references %q which is not defined under top-level health_checks; startModeProcesses skips missing names silently",
				modeName, hcName))
		}
	}

	sort.Strings(warnings)
	return warnings
}
