package config

import (
	"fmt"
	"maps"
)

// Authority-bearing lists (key selections and secret targets) replace as a
// whole. Inheriting extra keys while narrowing a binding would widen a push.
func (c *Config) mergeRemote(other *Config) error {
	if other.Secrets != nil {
		if c.Secrets == nil {
			c.Secrets = &SecretsConfig{}
		}
		if c.Secrets.Sources == nil {
			c.Secrets.Sources = map[string]SecretSource{}
		}
		if c.Secrets.Targets == nil {
			c.Secrets.Targets = map[string]SecretTarget{}
		}
		for _, name := range sortedKeysOf(other.Secrets.Sources) {
			source := other.Secrets.Sources[name]
			if source.Sops != "" {
				c.Secrets.Sources[name] = source
			}
		}
		for _, name := range sortedKeysOf(other.Secrets.Targets) {
			overlay := other.Secrets.Targets[name]
			base := c.Secrets.Targets[name]
			if err := mergeRemoteIdentity(&base.Provider, &base.Repository, overlay.Provider, overlay.Repository); err != nil {
				return fmt.Errorf("secret target %q: %w", name, err)
			}
			if overlay.Source != "" {
				base.Source = overlay.Source
			}
			if overlay.Keys != nil {
				base.Keys = overlay.Keys
			}
			c.Secrets.Targets[name] = base
		}
	}
	if other.Jobs != nil && c.Jobs == nil {
		c.Jobs = map[string]JobConfig{}
	}
	for _, name := range sortedKeysOf(other.Jobs) {
		overlay := other.Jobs[name]
		base := c.Jobs[name]
		if err := mergeRemoteIdentity(&base.Provider, &base.Repository, overlay.Provider, overlay.Repository); err != nil {
			return fmt.Errorf("job %q: %w", name, err)
		}
		if overlay.Ref != "" {
			base.Ref = overlay.Ref
		}
		if overlay.Timeout != "" {
			base.Timeout = overlay.Timeout
		}
		if overlay.Runs != nil {
			base.Runs = overlay.Runs
		}
		if overlay.SecretTargets != nil {
			base.SecretTargets = overlay.SecretTargets
		}
		if overlay.Inputs != nil && base.Inputs == nil {
			base.Inputs = map[string]JobInput{}
		}
		// Each input is a complete validation contract; defaults must never be
		// accidentally detached from an inherited allowlist or required bit.
		maps.Copy(base.Inputs, overlay.Inputs)
		c.Jobs[name] = base
	}
	return nil
}

func mergeRemoteIdentity(provider, repository *string, nextProvider, nextRepository string) error {
	if *provider != "" && nextProvider != "" && *provider != nextProvider {
		return fmt.Errorf("cannot override provider (restricted field)")
	}
	if *repository != "" && nextRepository != "" && *repository != nextRepository {
		return fmt.Errorf("cannot override repository (restricted field)")
	}
	if nextProvider != "" {
		*provider = nextProvider
	}
	if nextRepository != "" {
		*repository = nextRepository
	}
	return nil
}
