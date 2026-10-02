package config

import (
	"fmt"
	"maps"
	"os"
)

// mergeStringMap merges src into dst (key-level merge).
// Existing keys in dst are overwritten by src.
func mergeStringMap(dst, src map[string]string) map[string]string {
	if src == nil {
		return dst
	}
	if dst == nil {
		dst = make(map[string]string, len(src))
	}
	maps.Copy(dst, src)
	return dst
}

// mergeHealthCheckConfig deep-merges other into base (scalar replace).
func mergeHealthCheckConfig(base, other HealthCheckConfig) HealthCheckConfig {
	if other.Type != "" {
		base.Type = other.Type
	}
	if other.URL != "" {
		base.URL = other.URL
	}
	if other.Address != "" {
		base.Address = other.Address
	}
	if other.Command != "" {
		base.Command = other.Command
	}
	if other.Start != "" {
		base.Start = other.Start
	}
	if other.StartHint != "" {
		base.StartHint = other.StartHint
	}
	if other.Timeout != 0 {
		base.Timeout = other.Timeout
	}
	if other.ReadyTimeout != 0 {
		base.ReadyTimeout = other.ReadyTimeout
	}
	return base
}

// mergeEndpointConfig deep-merges other into base.
func mergeEndpointConfig(base, other EndpointConfig) EndpointConfig {
	if other.URL != "" {
		base.URL = other.URL
	}
	if other.Label != "" {
		base.Label = other.Label
	}
	if other.Source != "" {
		base.Source = other.Source
	}
	// List replace
	if other.Tags != nil {
		base.Tags = other.Tags
	}
	// Map merge
	if other.Paths != nil {
		if base.Paths == nil {
			base.Paths = make(map[string]string)
		}
		maps.Copy(base.Paths, other.Paths)
	}
	return base
}

// mergeInteractionCommand deep-merges other into base.
// Returns an error if runner (restricted field) is changed.
func mergeInteractionCommand(base, other *InteractionCommand) (*InteractionCommand, error) {
	if base == nil {
		return other, nil
	}
	if other == nil {
		return base, nil
	}

	// Restricted field: runner must not change
	if other.Runner != "" && base.Runner != "" && other.Runner != base.Runner {
		return nil, fmt.Errorf("cannot override runner for interaction command: %q → %q (restricted field)", base.Runner, other.Runner)
	}

	// Scalar replace
	if other.Description != "" {
		base.Description = other.Description
	}
	if other.Service != "" {
		base.Service = other.Service
	}
	if other.Command != "" {
		base.Command = other.Command
		// When command is overridden as a scalar, clear multi-command list
		base.CommandLines = nil
	}
	if other.CommandLines != nil {
		base.CommandLines = other.CommandLines
		if len(other.CommandLines) > 0 {
			base.Command = other.CommandLines[0]
		}
	}
	if other.Script != "" {
		base.Script = other.Script
	}
	if other.ScriptFile != "" {
		base.ScriptFile = other.ScriptFile
	}
	if other.Steps != nil {
		base.Steps = other.Steps
	}
	if other.Workdir != "" {
		base.Workdir = other.Workdir
	}
	if other.User != "" {
		base.User = other.User
	}
	if other.DefaultArgs != "" {
		base.DefaultArgs = other.DefaultArgs
	}
	if other.Entrypoint != "" {
		base.Entrypoint = other.Entrypoint
	}
	if other.Pod != "" {
		base.Pod = other.Pod
	}
	if other.Runner != "" {
		base.Runner = other.Runner
	}
	if other.Shell != nil {
		base.Shell = other.Shell
	}
	if other.Destructive != nil {
		base.Destructive = other.Destructive
	}
	if other.Compose != nil {
		base.Compose = other.Compose
	}

	// List replace
	if other.Tags != nil {
		base.Tags = other.Tags
	}

	// Map merge
	base.Environment = mergeStringMap(base.Environment, other.Environment)

	// Subcommands map merge (deep)
	if other.Subcommands != nil {
		if base.Subcommands == nil {
			base.Subcommands = make(map[string]*InteractionCommand)
		}
		for k, v := range other.Subcommands {
			if existing, ok := base.Subcommands[k]; ok {
				merged, err := mergeInteractionCommand(existing, v)
				if err != nil {
					return nil, fmt.Errorf("subcommand %q: %w", k, err)
				}
				base.Subcommands[k] = merged
			} else {
				base.Subcommands[k] = v
			}
		}
	}

	return base, nil
}

// mergeModeConfig deep-merges other into base.
func mergeModeConfig(base, other ModeConfig) ModeConfig {
	if other.Description != "" {
		base.Description = other.Description
	}
	if other.Build != "" {
		base.Build = other.Build
	}
	if other.Run != "" {
		base.Run = other.Run
	}
	if other.Provision != "" {
		base.Provision = other.Provision
	}

	// List replace
	if other.ComposeProfiles != nil {
		base.ComposeProfiles = other.ComposeProfiles
	}
	if other.ComposeServices != nil {
		base.ComposeServices = other.ComposeServices
	}
	if other.HealthChecks != nil {
		base.HealthChecks = other.HealthChecks
	}
	if other.EndpointTags != nil {
		base.EndpointTags = other.EndpointTags
	}
	if other.Stack != nil {
		base.Stack = other.Stack
	}

	// Map merge
	base.Environment = mergeStringMap(base.Environment, other.Environment)

	return base
}

// mergeEnvironmentProfile deep-merges other into base.
func mergeEnvironmentProfile(base, other EnvironmentProfile) EnvironmentProfile {
	if other.Description != "" {
		base.Description = other.Description
	}

	// List replace
	if other.Stack != nil {
		base.Stack = other.Stack
	}

	// Map merge
	base.Environment = mergeStringMap(base.Environment, other.Environment)

	// StackOverrides map merge (deep)
	if len(other.StackOverrides) > 0 {
		if base.StackOverrides == nil {
			base.StackOverrides = make(map[string]*LifecycleEntry)
		}
		for k, v := range other.StackOverrides {
			if existing, ok := base.StackOverrides[k]; ok {
				merged, err := MergeLifecycleEntry(existing, v)
				if err != nil {
					// plugin type conflict in override is a config error;
					// caller (mergeFrom) propagates it via mergeEnvironmentProfile
					// but this function cannot return errors. Log and skip.
					//
					// stderr, not stdout: this fires during config load, before any
					// command has produced output, so a stdout write would prepend a
					// non-JSON line to a --json document and break `dva … | jq`.
					// The stream invariant is enforced by TestStackOverrideWarnGoesToStderr
					// (and any new production [warn] on stdout fails that class of test),
					// not by a census in this comment — the count rots (TASK-155).
					fmt.Fprintf(os.Stderr, "[warn] stack_override %q: %v\n", k, err)
					continue
				}
				base.StackOverrides[k] = merged
			} else {
				base.StackOverrides[k] = v
			}
		}
	}

	return base
}

func mergePlanConfig(base, other *PlanConfig) *PlanConfig {
	if base == nil {
		return other
	}
	if other == nil {
		return base
	}
	if other.Description != "" {
		base.Description = other.Description
	}
	// Alias and extends are structural — later layer wins entirely
	if other.Alias != "" {
		base.Alias = other.Alias
	}
	if other.Extends != "" {
		base.Extends = other.Extends
	}
	if other.Environment != "" {
		base.Environment = other.Environment
	}
	if other.Site != "" {
		base.Site = other.Site
	}
	if other.EndpointTags != nil {
		base.EndpointTags = other.EndpointTags
	}
	base.Vars = mergeStringMap(base.Vars, other.Vars)
	if other.Entries != nil {
		base.Entries = other.Entries
	}
	if other.Composes != nil {
		base.Composes = other.Composes
	}
	return base
}

func mergeSiteConfig(base, other *SiteConfig) *SiteConfig {
	if base == nil {
		return other
	}
	if other == nil {
		return base
	}
	if other.Description != "" {
		base.Description = other.Description
	}
	base.Vars = mergeStringMap(base.Vars, other.Vars)
	if other.EntryOverrides != nil {
		if base.EntryOverrides == nil {
			base.EntryOverrides = make(map[string]*SiteEntryOverride)
		}
		for k, v := range other.EntryOverrides {
			if existing, ok := base.EntryOverrides[k]; ok {
				base.EntryOverrides[k] = mergeSiteEntryOverride(existing, v)
			} else {
				base.EntryOverrides[k] = v
			}
		}
	}
	return base
}

func mergeSiteEntryOverride(base, other *SiteEntryOverride) *SiteEntryOverride {
	if base == nil {
		return other
	}
	if other == nil {
		return base
	}
	if other.Runner != "" {
		base.Runner = other.Runner
	}
	base.Vars = mergeStringMap(base.Vars, other.Vars)
	return base
}

func mergeSubprojectConfig(base, other SubprojectConfig) SubprojectConfig {
	if other.Path != "" {
		base.Path = other.Path
	}
	if other.ExcludeTags != nil {
		base.ExcludeTags = other.ExcludeTags
	}
	if other.Import != nil {
		base.Import = other.Import
	}
	return base
}
