package config

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// CompareConfig declares one concurrent variant comparison (docs/75). It names a plan and
// splits that plan's entries into those started once from the calling checkout (Shared) and
// those started once per variant checkout (PerVariant). Ports hands variant i the i-th value
// of each list as a plan var. Variants is the optional default set used when `dva compare up`
// names none: variant name → source (a ref, or a provider work ID such as worktree:TASK-12).
//
// This card (TASK-589) only parses and validates the declaration; running it belongs to the
// compare command cards that follow.
type CompareConfig struct {
	Plan       string            `yaml:"plan"`
	Shared     []string          `yaml:"shared,omitempty"`
	PerVariant []string          `yaml:"per_variant,omitempty"`
	Ports      map[string][]int  `yaml:"ports,omitempty"`
	Variants   map[string]string `yaml:"variants,omitempty"`
}

// compareNamePattern keeps comparison and variant names usable as directory and compose
// project name segments, since each variant later gets its own checkout.
var compareNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// portVarPattern keeps port keys usable as plan var names.
var portVarPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateCompare checks every compare declaration before any process starts. Each problem
// names the comparison and the field so the user can find it without reading the code.
func (c *Config) validateCompare() []error {
	names := make([]string, 0, len(c.Compare))
	for name := range c.Compare {
		names = append(names, name)
	}
	sort.Strings(names)

	var errs []error
	for _, name := range names {
		errs = append(errs, c.validateCompareEntry(name, c.Compare[name])...)
	}
	return errs
}

func (c *Config) validateCompareEntry(name string, cmp *CompareConfig) []error {
	prefix := "compare." + name
	if !compareNamePattern.MatchString(name) {
		return []error{fmt.Errorf("%s: name must match %s", prefix, compareNamePattern)}
	}
	if cmp == nil {
		return []error{fmt.Errorf("%s: declaration is empty", prefix)}
	}

	var errs []error
	errs = append(errs, validateComparePorts(prefix, cmp)...)
	errs = append(errs, validateCompareVariants(prefix, cmp)...)

	if cmp.Plan == "" {
		return append(errs, fmt.Errorf("%s.plan: required", prefix))
	}
	plan, ok := c.Plans[cmp.Plan]
	if !ok || plan == nil {
		return append(errs, fmt.Errorf("%s.plan: plan %q not found", prefix, cmp.Plan))
	}

	entries, ok := c.comparePlanEntryNames(cmp.Plan)
	if !ok {
		// A broken alias/extends chain or a composition plan. Composition is rejected here;
		// chain errors are already reported by validatePlanAliasExtends.
		if c.planIsComposition(cmp.Plan) {
			errs = append(errs, fmt.Errorf("%s.plan: %q is a composition plan; compare needs a plan with entries", prefix, cmp.Plan))
		}
		return errs
	}

	errs = append(errs, validateCompareSplit(prefix, cmp, entries)...)

	for _, entryName := range cmp.PerVariant {
		if cc := c.Stack[entryName].ComposeConfig(); cc != nil && cc.ProjectName != "" {
			errs = append(errs, fmt.Errorf(
				"%s.per_variant: stack entry %q sets project_name %q, so every variant would share one compose project; remove project_name or move the entry to shared",
				prefix, entryName, cc.ProjectName))
		}
	}
	return errs
}

// validateCompareSplit requires shared and per_variant to be disjoint and together to cover
// the plan's entries exactly.
func validateCompareSplit(prefix string, cmp *CompareConfig, entries []string) []error {
	var errs []error
	seen := map[string]string{}
	for _, field := range []struct {
		name  string
		items []string
	}{{"shared", cmp.Shared}, {"per_variant", cmp.PerVariant}} {
		for _, item := range field.items {
			if prev, dup := seen[item]; dup {
				if prev == field.name {
					errs = append(errs, fmt.Errorf("%s.%s: entry %q is listed twice", prefix, field.name, item))
				} else {
					errs = append(errs, fmt.Errorf("%s: entry %q is in both shared and per_variant", prefix, item))
				}
				continue
			}
			seen[item] = field.name
			if !slices.Contains(entries, item) {
				errs = append(errs, fmt.Errorf("%s.%s: entry %q is not an entry of plan %q (entries: %s)",
					prefix, field.name, item, cmp.Plan, strings.Join(entries, ", ")))
			}
		}
	}
	var missing []string
	for _, e := range entries {
		if _, ok := seen[e]; !ok {
			missing = append(missing, e)
		}
	}
	if len(missing) > 0 {
		errs = append(errs, fmt.Errorf("%s: plan %q entries %s are in neither shared nor per_variant",
			prefix, cmp.Plan, strings.Join(missing, ", ")))
	}
	return errs
}

func validateComparePorts(prefix string, cmp *CompareConfig) []error {
	keys := make([]string, 0, len(cmp.Ports))
	for k := range cmp.Ports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var errs []error
	used := map[int]string{}
	for _, key := range keys {
		field := prefix + ".ports." + key
		if !portVarPattern.MatchString(key) {
			errs = append(errs, fmt.Errorf("%s: var name must match %s", field, portVarPattern))
		}
		if len(cmp.Ports[key]) == 0 {
			errs = append(errs, fmt.Errorf("%s: list is empty", field))
		}
		for _, p := range cmp.Ports[key] {
			if p < 1 || p > 65535 {
				errs = append(errs, fmt.Errorf("%s: port %d is outside 1-65535", field, p))
				continue
			}
			if prev, dup := used[p]; dup {
				errs = append(errs, fmt.Errorf("%s: port %d is already assigned under ports.%s", field, p, prev))
				continue
			}
			used[p] = key
		}
	}
	if n := len(cmp.Variants); n > 0 {
		if err := cmp.CheckPortCapacity(n); err != nil {
			errs = append(errs, fmt.Errorf("%s.variants: %w", prefix, err))
		}
	}
	return errs
}

func validateCompareVariants(prefix string, cmp *CompareConfig) []error {
	keys := make([]string, 0, len(cmp.Variants))
	for k := range cmp.Variants {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var errs []error
	for _, key := range keys {
		if !compareNamePattern.MatchString(key) {
			errs = append(errs, fmt.Errorf("%s.variants.%s: name must match %s", prefix, key, compareNamePattern))
		}
		if strings.TrimSpace(cmp.Variants[key]) == "" {
			errs = append(errs, fmt.Errorf("%s.variants.%s: source is empty", prefix, key))
		}
	}
	return errs
}

// CheckPortCapacity fails when any port list is shorter than the requested variant count.
// Variant i receives the i-th port of every list; a short list must fail before anything
// starts instead of moving a variant to an unlisted port. The compare command (TASK-591)
// calls this with the number of variants the user asked for.
func (cmp *CompareConfig) CheckPortCapacity(variants int) error {
	keys := make([]string, 0, len(cmp.Ports))
	for k := range cmp.Ports {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var short []string
	for _, k := range keys {
		if len(cmp.Ports[k]) < variants {
			short = append(short, fmt.Sprintf("%s has %d", k, len(cmp.Ports[k])))
		}
	}
	if len(short) > 0 {
		return fmt.Errorf("%d variants need %d ports per list, but %s", variants, variants, strings.Join(short, ", "))
	}
	return nil
}

// comparePlanEntryNames returns the sorted entry names a plan runs after following its alias
// chain and unioning its extends chain. It reports false when the chain is broken or cyclic,
// or the plan is a composition, because those plans have no single entry set to split.
func (c *Config) comparePlanEntryNames(name string) ([]string, bool) {
	const maxDepth = 16
	current := name
	visited := map[string]bool{}
	set := map[string]bool{}
	for range maxDepth {
		if visited[current] {
			return nil, false
		}
		visited[current] = true
		plan, ok := c.Plans[current]
		if !ok || plan == nil || len(plan.Composes) > 0 {
			return nil, false
		}
		if plan.Alias != "" {
			current = plan.Alias
			continue
		}
		for _, e := range plan.Entries {
			set[e.Name] = true
		}
		if plan.Extends == "" {
			names := make([]string, 0, len(set))
			for n := range set {
				names = append(names, n)
			}
			sort.Strings(names)
			return names, true
		}
		current = plan.Extends
	}
	return nil, false
}

// planIsComposition reports whether name (after aliases) is a composition plan.
func (c *Config) planIsComposition(name string) bool {
	current := name
	for range 16 {
		plan, ok := c.Plans[current]
		if !ok || plan == nil {
			return false
		}
		if len(plan.Composes) > 0 {
			return true
		}
		if plan.Alias == "" {
			return false
		}
		current = plan.Alias
	}
	return false
}
