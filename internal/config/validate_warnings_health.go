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
// and/or start_hint. Top-level checks are no longer an execution surface: put lifecycle
// readiness and native start commands on the stack entry that owns them.
//
// Severity: Semantic Warning
func (c *Config) warnUnreachableHealthChecks() []string {
	var warnings []string

	for name, hc := range c.HealthChecks {
		if hc.Start == "" && hc.StartHint == "" {
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
			"health_checks.%s: declares %s but top-level health_checks are not run by plans; declare a native stack entry for startup and move the check under stack.<entry>.health_checks, or remove the field",
			name, strings.Join(fields, " and ")))
	}

	sort.Strings(warnings)
	return warnings
}
