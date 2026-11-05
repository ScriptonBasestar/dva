package config

import (
	"fmt"
	"sort"
	"strings"
)

// warnDuplicateStackOrder warns when multiple stack entries share the same order value.
//
// Entries a plan names are exempt too, and that exemption is the point of TASK-084. It used to be
// `len(c.Plans) > 0` — any plan anywhere silenced everything — on the premise that "plan order owns
// sequencing when plans exist". Too coarse in one direction: it hid entries no plan mentions. Too
// coarse in the other: dropping it entirely warned about the shape `docs/40-declarative-stack-and-
// plans.md` prescribes, where order belongs to the plan layer, so three shipped examples started
// being told they had not chosen a sequence when their plan declares infra→api→worker→web.
//
// What is checkable is whether a plan names the entry, not whether some plan exists. An entry a
// plan names has a declared position; the gap is only that plan-less `dva up` does not read plans
// (its help says so), and that is a property of the command rather than of the config.
func (c *Config) warnDuplicateStackOrder() []string {
	if len(c.Stack) < 2 {
		return nil
	}
	planned := c.entriesNamedByPlans()

	orderMap := make(map[int][]string)
	for name, entry := range c.Stack {
		orderMap[entry.Order] = append(orderMap[entry.Order], name)
	}

	var warnings []string
	// Sort by order value for deterministic output
	var orders []int
	for order := range orderMap {
		orders = append(orders, order)
	}
	sort.Ints(orders)

	for _, order := range orders {
		names := orderMap[order]
		if len(names) < 2 {
			continue
		}
		// Report only the entries no plan names. Dropping the covered ones rather than the whole
		// group is what keeps the warning useful on a plan that names two entries of five: the
		// other three still have no declared position anywhere, and they are the ones to say so
		// about. Below two, there is no pair left to sequence.
		unplanned := make([]string, 0, len(names))
		for _, name := range names {
			if !planned[name] {
				unplanned = append(unplanned, name)
			}
		}
		if len(unplanned) < 2 {
			continue
		}
		sort.Strings(unplanned)
		// Not "undefined": since TASK-084 half 1 the sequence is (Order, Name), so equal orders
		// resolve alphabetically and repeat run to run. What is left to report is not a hazard but
		// a position never declared — for these entries, in the plan layer or anywhere else.
		var msg string
		if order == 0 {
			msg = fmt.Sprintf("stack: entries %s are at the default order and no plan names them, so `dva up` starts them in name order rather than one you chose",
				strings.Join(unplanned, ", "))
		} else {
			msg = fmt.Sprintf("stack: entries %s share order value %d and no plan names them, so `dva up` starts them in name order rather than one you chose",
				strings.Join(unplanned, ", "), order)
		}
		// Named only when plans exist, because that is the reader who would otherwise assume the
		// plan already settled this. Saying it unconditionally would advertise plans to configs
		// that have none.
		if len(c.Plans) > 0 {
			msg += "; plan entry order governs `dva up <plan>` only"
		}
		warnings = append(warnings, msg)
	}

	return warnings
}

// entriesNamedByPlans is the set of stack entries some plan gives a position to. Membership is
// what warnDuplicateStackOrder treats as "the author declared where this runs" — a plan walks its
// entries in declaration order, so being listed is a position even with no explicit `order:`.
func (c *Config) entriesNamedByPlans() map[string]bool {
	planned := make(map[string]bool)
	for _, plan := range c.Plans {
		if plan == nil {
			continue
		}
		for _, e := range plan.Entries {
			planned[e.Name] = true
		}
	}
	return planned
}

// warnMultiStackComposeSplit warns when compose entries that can run together were
// split across stack entries. Each entry is its own `docker compose` invocation, so
// an overlay entry cannot patch a base entry's service definitions.
//
// Named plans are the authoritative lifecycle surface. Only two non-empty,
// fully literal project_name values that differ are provably independent Compose
// projects. Empty or interpolation-bearing names remain unknown: plan, site and
// entry variables can change their runtime value, which validation cannot prove.
func (c *Config) warnMultiStackComposeSplit() []string {
	byProjectName := make(map[string]map[string]bool)
	unknown := make(map[string]bool)
	for name, entry := range c.Stack {
		// ComposeConfig(), not entry.Compose: the supported shape stores compose under
		// runners, so reading the legacy field alone made this warning unreachable for
		// every config that follows the current schema.
		if entry.ComposeConfig() != nil {
			projectName, literal := literalComposeProjectName(entry.ComposeConfig().ProjectName)
			if !literal {
				unknown[name] = true
				continue
			}
			if byProjectName[projectName] == nil {
				byProjectName[projectName] = make(map[string]bool)
			}
			byProjectName[projectName][name] = true
		}
	}

	projectNames := make([]string, 0, len(byProjectName))
	for projectName := range byProjectName {
		projectNames = append(projectNames, projectName)
	}
	sort.Strings(projectNames)

	var warnings []string
	for _, projectName := range projectNames {
		entries := make(map[string]bool, len(byProjectName[projectName])+len(unknown))
		for name := range byProjectName[projectName] {
			entries[name] = true
		}
		for name := range unknown {
			entries[name] = true
		}
		if len(entries) < 2 || c.composeEntriesAreIsolated(entries) {
			continue
		}
		warnings = append(warnings, c.composeSplitWarning(entries))
	}
	if len(projectNames) == 0 && len(unknown) > 1 && !c.composeEntriesAreIsolated(unknown) {
		warnings = append(warnings, c.composeSplitWarning(unknown))
	}
	return warnings
}

func literalComposeProjectName(projectName string) (string, bool) {
	projectName = strings.TrimSpace(projectName)
	return projectName, projectName != "" && !strings.Contains(projectName, "$")
}

func (c *Config) composeEntriesAreIsolated(entries map[string]bool) bool {
	if len(c.Plans) > 0 {
		return c.plansIsolateEntries(entries) || c.plansPartitionComposeServices(entries)
	}
	return false
}

// plansPartitionComposeServices reports whether, in every plan that selects two or
// more of the given compose entries together as compose invocations, each of those
// entries restricts itself to a disjoint, non-empty services: subset. That is the
// supported "compose service subsets" shape (examples/service-orchestration.yml's
// infra-compose/frontend pair): each entry is still its own 'docker compose'
// invocation, but the invocations start different services from the same file
// rather than one entry's file list patching another's service definitions.
// warnMultiStackComposeSplit's overlay concern — an entry cannot patch another
// entry's services because they never run in the same invocation — does not apply
// when the invocations share nothing to begin with.
//
// A plan entry that resolves to a non-compose runner (api/worker's runner: native
// in that same example, falling back to their default_runner otherwise) never
// issues the 'docker compose' call this warning is about, so it is excluded before
// counting co-occurrence — it is not a participant to partition against. Site
// entry_overrides are not consulted: like the rest of this file's plan-shape
// warnings, the check reasons about the plan's own declaration, not a runtime
// site selection.
//
// An empty services: (nil, meaning "all services in the file") or a service name
// repeated across two co-selected entries falls through to the real overlay warning,
// because either shape can still start the same service from two separate invocations.
//
// Gated on DefaultPlan() != "" exactly like plansIsolateEntries: partitioning inside
// every named plan says nothing about the unnamed lifecycle path (a bare `dva up`),
// which is unsafe on its own whenever no default_plan bounds it.
func (c *Config) plansPartitionComposeServices(entries map[string]bool) bool {
	return c.DefaultPlan() != "" && c.plansWouldPartitionComposeServices(entries)
}

// plansWouldPartitionComposeServices is plansPartitionComposeServices without the
// default_plan gate, split out so callers can ask the same co-occurrence question
// without repeating the gate inline.
func (c *Config) plansWouldPartitionComposeServices(entries map[string]bool) bool {
	if len(c.Plans) == 0 {
		return false
	}
	claimed := make(map[string]bool, len(entries))
	for _, plan := range c.Plans {
		if plan == nil {
			return false
		}
		var matched []PlanEntry
		for _, entry := range plan.Entries {
			if !entries[entry.Name] {
				continue
			}
			claimed[entry.Name] = true
			if c.planEntryRunner(entry) != "compose" {
				continue
			}
			matched = append(matched, entry)
		}
		if len(matched) < 2 {
			continue
		}
		seen := make(map[string]bool, len(matched))
		for _, entry := range matched {
			if len(entry.Services) == 0 {
				return false
			}
			for _, svc := range entry.Services {
				if seen[svc] {
					return false
				}
				seen[svc] = true
			}
		}
	}
	// An entry no plan mentions at all is reachable through an entry-oriented
	// lifecycle command with no plan-declared services: restriction in play,
	// same hazard plansWouldIsolateEntries's own claimed-length check guards.
	return len(claimed) == len(entries)
}

// planEntryRunner returns the runner a plan entry resolves to: its own runner:
// override when set, else the stack entry's default_runner, else "compose" — the
// entries warnMultiStackComposeSplit ever calls this with all carry a compose:
// block by construction (that is how they entered its entries map), so absent an
// explicit override elsewhere, compose is what they resolve to; the same absence
// is what the pre-TASK-288 warning always assumed. It mirrors lifecycle.resolver's
// finalRunner precedence for those two inputs only — the site-override and
// plugin-detection fallbacks the resolver also applies are runtime concerns this
// validation-time check does not have enough context to reproduce. Only positive
// evidence of a non-compose runner (an explicit plan runner: or stack
// default_runner naming something else) excludes an entry from
// plansPartitionComposeServices's co-occurrence count — silence does not, because
// silently excluding an entry there makes the check MORE likely to call a plan
// isolated, which would suppress the real overlay warning rather than only widen
// when it fires.
func (c *Config) planEntryRunner(entry PlanEntry) string {
	if runner := normalizeRunnerName(entry.Runner); runner != "" {
		return runner
	}
	if stackEntry, ok := c.Stack[entry.Name]; ok {
		if runner := stackEntry.DefaultRunnerName(); runner != "" {
			return runner
		}
	}
	return "compose"
}

func (c *Config) composeSplitWarning(entries map[string]bool) string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	remedy := "give each its own named plan (plans.<name>.entries: [{name: entry}])"
	if len(c.Plans) > 0 && c.plansWouldIsolateEntries(entries) && c.DefaultPlan() == "" {
		remedy = "set default_plan to a minimal plan that selects one entry, so unnamed lifecycle commands have a bounded selection"
	} else if len(c.Plans) > 0 && c.anyPlanSelectsEntries(entries) {
		remedy = "merge them into one entry whose files: lists the overlays"
	}

	return fmt.Sprintf("stack: compose entries [%s] can run in the same invocation set — each is a separate 'docker compose' call, so an overlay entry cannot patch another entry's services; %s",
		strings.Join(names, ", "), remedy)
}

// plansIsolateEntries reports whether named plans make the compose entries
// mutually exclusive. A default plan is required before an unnamed lifecycle
// command has a bounded selection; explicit entry-oriented lifecycle commands
// are deliberate escape hatches and are not evidence that plans co-occur.
func (c *Config) plansIsolateEntries(entries map[string]bool) bool {
	return c.DefaultPlan() != "" && c.plansWouldIsolateEntries(entries)
}

// plansWouldIsolateEntries reports whether each named plan selects at most one
// entry and every entry is selected by a plan. It intentionally does not inspect
// explicit entry-oriented lifecycle commands, which bypass plan selection by
// design and should not make otherwise isolated plans warn.
func (c *Config) plansWouldIsolateEntries(entries map[string]bool) bool {
	if len(c.Plans) == 0 {
		return false
	}
	claimed := make(map[string]bool, len(entries))
	for _, plan := range c.Plans {
		if plan == nil {
			return false
		}
		hits := 0
		for _, entry := range plan.Entries {
			if entries[entry.Name] {
				hits++
				claimed[entry.Name] = true
			}
		}
		if hits > 1 {
			return false
		}
	}

	return len(claimed) == len(entries)
}

func (c *Config) anyPlanSelectsEntries(entries map[string]bool) bool {
	for _, plan := range c.Plans {
		if plan == nil {
			continue
		}
		hits := 0
		for _, entry := range plan.Entries {
			if entries[entry.Name] {
				hits++
			}
		}
		if hits > 1 {
			return true
		}
	}
	return false
}
