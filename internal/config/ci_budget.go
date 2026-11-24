package config

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
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

var ciBudgetInitMutex sync.Mutex

// Each Config snapshots policy on its first resolution. Load validates profiles
// before returning, while programmatic Config values snapshot on first use.
func (c *Config) budgetPolicy() (*ciBudgetPolicy, error) {
	ciBudgetInitMutex.Lock()
	defer ciBudgetInitMutex.Unlock()
	if !c.ciBudgetInitialized {
		policy, err := loadCIBudgetPolicy()
		c.ciBudgetSnapshot, c.ciBudgetError = &policy, err
		c.ciBudgetInitialized = true
	}
	return c.ciBudgetSnapshot, c.ciBudgetError
}

func (c *Config) resolveCIBudget(profile CIProfile, family, name string) (CIProfile, error) {
	if profile.BudgetSources != nil {
		return profile, nil
	}
	policy, err := c.budgetPolicy()
	if err != nil {
		return profile, err
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
	kind := profile.Kind
	if kind == "" {
		switch name {
		case "dev", "unit", "integration", "test", "e2e", "full":
			kind = name
		case "audit":
			kind = "full"
		}
	}
	if family != "" {
		apply(policy.Families[family][kind], "family:"+family)
	}
	apply(policy.Global[kind], "global")

	builtins := map[string]ciBudget{
		"dev": {"2m", "1m"}, "unit": {"5m", "2m"},
		"integration": {"5m", "3m"}, "test": {"5m", "3m"},
		"e2e": {"5m", "3m"}, "full": {"15m", "10m"},
		"lint": {"30s", "15s"}, "compile": {"60s", "30s"},
		"build": {"5m", "3m"},
	}
	if kind != "" {
		budget, ok := builtins[kind]
		if ok {
			apply(budget, "builtin")
		} else if profile.Timeout == "" || profile.WarnAfter == "" {
			return profile, fmt.Errorf("custom CI budget kind %q needs timeout and warn_after from policy or project", kind)
		}
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
