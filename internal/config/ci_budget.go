package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type ciBudget struct {
	Timeout   string `yaml:"timeout"`
	WarnAfter string `yaml:"warn_after"`
}

type ciBudgetPolicy struct {
	Global   map[string]ciBudget            `yaml:"global"`
	Families map[string]map[string]ciBudget `yaml:"families"`
}

func loadCIBudgetPolicy() (ciBudgetPolicy, error) {
	var policy ciBudgetPolicy
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		var err error
		dir, err = os.UserConfigDir()
		if err != nil {
			return policy, err
		}
	}
	path := filepath.Join(dir, "dva", "ci-budgets.yml")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return policy, nil
	}
	if err != nil {
		return policy, fmt.Errorf("read CI budget policy: %w", err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&policy); err != nil {
		return policy, fmt.Errorf("parse CI budget policy: %w", err)
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); err != io.EOF {
		return policy, fmt.Errorf("CI budget policy must contain one YAML document")
	}
	validate := func(label string, budgets map[string]ciBudget) error {
		for kind, budget := range budgets {
			if !validCIName(kind) {
				return fmt.Errorf("invalid budget kind %q", kind)
			}
			for field, value := range map[string]string{"timeout": budget.Timeout, "warn_after": budget.WarnAfter} {
				if value != "" {
					if _, err := parsePositiveCIDuration(label+" "+kind, field, value); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	if err := validate("global", policy.Global); err != nil {
		return policy, err
	}
	for family, budgets := range policy.Families {
		if !validCIName(family) {
			return policy, fmt.Errorf("invalid CI family %q", family)
		}
		if err := validate("family "+family, budgets); err != nil {
			return policy, err
		}
	}
	return policy, nil
}

func resolveCIBudget(profile CIProfile, family, name string) (CIProfile, error) {
	if profile.BudgetSources != nil {
		return profile, nil
	}
	policy, err := loadCIBudgetPolicy()
	if err != nil {
		return profile, err
	}
	if family != "" {
		if _, ok := policy.Families[family]; !ok {
			return profile, fmt.Errorf("CI budget family %q is not configured", family)
		}
	}
	sources := map[string]string{}
	if profile.Timeout != "" {
		sources["timeout"] = "project"
	}
	if profile.WarnAfter != "" {
		sources["warn_after"] = "project"
	}
	apply := func(b ciBudget, source string) {
		if profile.Timeout == "" && b.Timeout != "" {
			profile.Timeout = b.Timeout
			sources["timeout"] = source
		}
		if profile.WarnAfter == "" && b.WarnAfter != "" {
			profile.WarnAfter = b.WarnAfter
			sources["warn_after"] = source
		}
	}
	if family != "" {
		apply(policy.Families[family][profile.Kind], "family:"+family)
	}
	apply(policy.Global[profile.Kind], "global")
	builtins := map[string]ciBudget{
		"dev": {"2m", "1m"}, "unit": {"5m", "2m"},
		"integration": {"5m", "3m"}, "test": {"5m", "3m"},
		"e2e": {"5m", "3m"}, "full": {"15m", "10m"},
		"lint": {"30s", "15s"}, "compile": {"60s", "30s"},
		"build": {"5m", "3m"},
	}
	if profile.Kind != "" {
		budget, ok := builtins[profile.Kind]
		if !ok {
			return profile, fmt.Errorf("unknown CI budget kind %q", profile.Kind)
		}
		apply(budget, "builtin")
	}
	if name == defaultCIProfileName {
		apply(ciBudget{Timeout: "10m0s", WarnAfter: "5m0s"}, "builtin")
	}
	if sources["warn_after"] != "project" && profile.WarnAfter != "" && profile.Timeout != "" {
		warn, werr := time.ParseDuration(profile.WarnAfter)
		limit, lerr := time.ParseDuration(profile.Timeout)
		if werr == nil && lerr == nil && limit > 0 && warn >= limit {
			profile.WarnAfter = (limit - limit/5).String()
			sources["warn_after"] += ":capped"
		}
	}
	profile.BudgetSources = sources
	return profile, nil
}
