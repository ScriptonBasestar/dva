package config

import (
	"fmt"
	"maps"
)

// mergeFrom merges another config into this one.
//
// Merge semantics (see docs/30-config-merge-semantics.md):
//   - map sections: key-level deep merge (entries are field-merged, not replaced)
//   - list fields: replace (later layer wins entirely)
//   - scalar fields: replace (non-zero later value wins)
//   - nil/absent: inherits from base; explicit empty clears
//
// Returns an error if a restricted field override is attempted.
func (c *Config) mergeFrom(other *Config) error {
	if err := c.mergeRemote(other); err != nil {
		return err
	}
	c.Vars = mergeStringMap(c.Vars, other.Vars)

	// environment: map merge (key-level)
	c.Environment = mergeStringMap(c.Environment, other.Environment)

	if other.Plans != nil {
		if c.Plans == nil {
			c.Plans = make(map[string]*PlanConfig)
		}
		for k, v := range other.Plans {
			if existing, ok := c.Plans[k]; ok {
				c.Plans[k] = mergePlanConfig(existing, v)
			} else {
				c.Plans[k] = v
			}
		}
	}

	if other.Sites != nil {
		if c.Sites == nil {
			c.Sites = make(map[string]*SiteConfig)
		}
		for k, v := range other.Sites {
			if existing, ok := c.Sites[k]; ok {
				c.Sites[k] = mergeSiteConfig(existing, v)
			} else {
				c.Sites[k] = v
			}
		}
	}

	// interaction: deep merge per entry
	if other.Interaction != nil {
		if c.Interaction == nil {
			c.Interaction = make(map[string]*InteractionCommand)
		}
		for k, v := range other.Interaction {
			if existing, ok := c.Interaction[k]; ok {
				merged, err := mergeInteractionCommand(existing, v)
				if err != nil {
					return fmt.Errorf("interaction %q: %w", k, err)
				}
				c.Interaction[k] = merged
			} else {
				c.Interaction[k] = v
			}
		}
	}

	// ci: profiles deep-merge by name; profile scalars replace when non-zero,
	// maps merge, and dependency/step lists replace as a whole.
	if other.CI != nil {
		if c.CI == nil {
			c.CI = &CIConfig{Profiles: make(map[string]CIProfile)}
		}
		if other.CI.Profiles != nil {
			if c.CI.Profiles == nil {
				c.CI.Profiles = make(map[string]CIProfile)
			}
			for name, profile := range other.CI.Profiles {
				if existing, ok := c.CI.Profiles[name]; ok {
					c.CI.Profiles[name] = mergeCIProfile(existing, profile)
				} else {
					c.CI.Profiles[name] = profile
				}
			}
		}
	}

	// provision: scalar replace + map key-level replace (profiles are step lists)
	if other.Provision.DefaultProfile != "" {
		c.Provision.DefaultProfile = other.Provision.DefaultProfile
	}
	if len(other.Provision.Profiles) > 0 {
		if c.Provision.Profiles == nil {
			c.Provision.Profiles = make(map[string][]ProvisionItem)
		}
		maps.Copy(c.Provision.Profiles, other.Provision.Profiles)
		// Ownership travels with the profile it belongs to. Modules are local files and
		// carry no owner today, but a copied profile that left its owner behind would
		// silently fall back to the parent — the exact defect TASK-264 repairs.
		for name := range other.Provision.Profiles {
			c.Provision.setProfileOwner(name, other.Provision.profileOwners[name])
			if other.Provision.profileSubprojects != nil {
				if sub := other.Provision.profileSubprojects[name]; sub != "" {
					if c.Provision.profileSubprojects == nil {
						c.Provision.profileSubprojects = make(map[string]string)
					}
					c.Provision.profileSubprojects[name] = sub
				}
			}
			if other.Provision.profileCanonicals != nil {
				if canon := other.Provision.profileCanonicals[name]; canon != "" {
					if c.Provision.profileCanonicals == nil {
						c.Provision.profileCanonicals = make(map[string]string)
					}
					c.Provision.profileCanonicals[name] = canon
				}
			}
		}
	}

	// health_checks: deep merge per entry (struct fields replace individually)
	if other.HealthChecks != nil {
		if c.HealthChecks == nil {
			c.HealthChecks = make(map[string]HealthCheckConfig)
		}
		for k, v := range other.HealthChecks {
			if existing, ok := c.HealthChecks[k]; ok {
				c.HealthChecks[k] = mergeHealthCheckConfig(existing, v)
			} else {
				c.HealthChecks[k] = v
			}
		}
	}

	// endpoints: deep merge per entry
	if other.Endpoints != nil {
		if c.Endpoints == nil {
			c.Endpoints = make(map[string]EndpointConfig)
		}
		for k, v := range other.Endpoints {
			if existing, ok := c.Endpoints[k]; ok {
				c.Endpoints[k] = mergeEndpointConfig(existing, v)
			} else {
				c.Endpoints[k] = v
			}
		}
	}

	// infra: key-level replace (simple struct)
	if other.Infra != nil {
		if c.Infra == nil {
			c.Infra = make(map[string]InfraConfig)
		}
		maps.Copy(c.Infra, other.Infra)
	}

	// default_mode: scalar replace
	if other.DefaultMode != "" {
		c.DefaultMode = other.DefaultMode
	}

	// default_plan: scalar replace
	if other.DefaultPlanName != "" {
		c.DefaultPlanName = other.DefaultPlanName
	}

	// modes: deep merge per entry
	if other.Modes != nil {
		if c.Modes == nil {
			c.Modes = make(map[string]ModeConfig)
		}
		for k, v := range other.Modes {
			if existing, ok := c.Modes[k]; ok {
				c.Modes[k] = mergeModeConfig(existing, v)
			} else {
				c.Modes[k] = v
			}
		}
	}

	// environments: deep merge per entry
	if other.Environments != nil {
		if c.Environments == nil {
			c.Environments = make(map[string]EnvironmentProfile)
		}
		for k, v := range other.Environments {
			if existing, ok := c.Environments[k]; ok {
				c.Environments[k] = mergeEnvironmentProfile(existing, v)
			} else {
				c.Environments[k] = v
			}
		}
	}

	// ssh: scalar replace
	if other.Ssh.AgentImage != "" {
		c.Ssh.AgentImage = other.Ssh.AgentImage
	}

	// suggestion_ignore: list replace
	if other.SuggestionIgnore != nil {
		c.SuggestionIgnore = other.SuggestionIgnore
	}

	// drift_ignore: list replace, same as suggestion_ignore. A module that declares the
	// key owns the whole list; appending would make the effective list depend on module
	// order, and an ignore list whose contents depend on load order is exactly the kind
	// of invisible suppression docs/56 §2 rules out.
	if other.DriftIgnore != nil {
		c.DriftIgnore = other.DriftIgnore
	}

	// suggestions: replace as a whole, for the same reason. The struct is small enough
	// that per-field merging would only buy the ambiguity of a half-overridden policy.
	if other.Suggestions != nil {
		c.Suggestions = other.Suggestions
	}

	// env_file: replace as a whole
	if other.EnvFile != nil {
		c.EnvFile = other.EnvFile
	}

	if other.Subprojects != nil {
		if c.Subprojects == nil {
			c.Subprojects = make(map[string]SubprojectConfig)
		}
		for k, v := range other.Subprojects {
			if existing, ok := c.Subprojects[k]; ok {
				c.Subprojects[k] = mergeSubprojectConfig(existing, v)
			} else {
				c.Subprojects[k] = v
			}
		}
	}

	// stack: deep merge per entry
	if len(other.Stack) > 0 {
		if c.Stack == nil {
			c.Stack = make(map[string]*LifecycleEntry)
		}
		for k, v := range other.Stack {
			// Resolve deferred plugin from entry name before merge
			v.Name = k
			if err := v.ResolvePluginFromName(); err != nil {
				return err
			}
			if existing, ok := c.Stack[k]; ok {
				merged, err := MergeLifecycleEntry(existing, v)
				if err != nil {
					return err
				}
				c.Stack[k] = merged
			} else {
				c.Stack[k] = v
			}
		}
	}

	// doctor checks: append (existing behavior preserved)
	if len(other.DoctorChecks) > 0 {
		c.DoctorChecks = append(c.DoctorChecks, other.DoctorChecks...)
	}

	// devcontainer: replace as a whole
	if other.Devcontainer != nil {
		c.Devcontainer = other.Devcontainer
	}

	return nil
}
