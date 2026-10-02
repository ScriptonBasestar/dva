package config

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// ValidateWarnings runs semantic warning checks and returns human-readable messages.
// These are non-fatal issues that should be surfaced by `dva config validate`.
func (c *Config) ValidateWarnings() []string {
	var warnings []string
	warnings = append(warnings, c.warnLegacyModes()...)
	warnings = append(warnings, c.warnLegacyStackOrder()...)
	warnings = append(warnings, c.warnLegacyEnvironmentFields()...)
	warnings = append(warnings, c.warnNoPlansHint()...)
	warnings = append(warnings, c.warnHealthCheckRedundancy()...)
	warnings = append(warnings, c.warnUnreachableHealthChecks()...)
	warnings = append(warnings, c.warnDuplicateParentSubcommand()...)
	warnings = append(warnings, c.warnDuplicateStackOrder()...)
	warnings = append(warnings, c.warnMultiStackComposeSplit()...)
	warnings = append(warnings, c.warnMissingDefaultMode()...)
	warnings = append(warnings, c.warnDefaultModeHeavyInfra()...)
	warnings = append(warnings, c.warnChildOverridesParentCritical()...)
	warnings = append(warnings, c.warnDeepSubcommandNesting()...)
	warnings = append(warnings, c.warnLiteralKeyShadowsSubproject()...)
	warnings = append(warnings, c.warnUnreachableCommands()...)
	warnings = append(warnings, c.warnInertProvisionSteps()...)
	warnings = append(warnings, c.warnIgnoredParallelSteps()...)
	warnings = append(warnings, c.warnIgnoredPlanFilters()...)
	warnings = append(warnings, c.warnDuplicatePlanDeclarations()...)
	warnings = append(warnings, c.warnMultiplePlansWithoutDefault()...)
	warnings = append(warnings, c.warnPlanServicesNotDeclared()...)
	warnings = append(warnings, c.warnPlanProfilesNotDefined()...)
	warnings = append(warnings, c.warnUnreferencedEnvironmentsAndSites()...)
	warnings = append(warnings, c.warnNoOpEntryOverrides()...)
	warnings = append(warnings, c.warnEmptyInteractionCommands()...)
	warnings = append(warnings, c.warnRemovedCLIReferences()...)
	warnings = append(warnings, c.warnEquivalentReplaceHooks()...)
	warnings = append(warnings, c.warnOrphanHealthChecks()...)
	warnings = append(warnings, c.warnMultiplePrimaryCompose()...)

	// Build a contextual environment for accurate interpolation checks.
	//
	// TASK-248: validation performs no env-file I/O. Structural validation has to
	// keep working when those files are missing or unreadable — availability is
	// doctor's and the runtime's to report, not the schema validator's. The cost
	// is that a reference satisfied only by an env file is no longer decidable
	// here, so the unresolved-variable check defers rather than reporting a
	// variable it can no longer see. It does not become a new warning category
	// and it does not claim success.
	env := NewEnvironment(c.Environment, c.FileDir(), c.FileDir())

	warnings = append(warnings, c.warnUnresolvedEnvVars(env, c.EnvFile != nil)...)
	warnings = append(warnings, c.warnSuspiciousEnvPatterns()...)

	if c.filePath != "" {
		warnings = append(warnings, validateCanonicalOrder(c.filePath)...)
	}
	return warnings
}

// warnEquivalentReplaceHooks finds a deliberately narrow candidate for human review.
// A replace hook can change lifecycle semantics in ways config validation cannot prove,
// so this never declares it equivalent to a built-in or tells an author to delete it.
func (c *Config) warnEquivalentReplaceHooks() []string {
	declaredFiles := make(map[string]bool)
	for _, entry := range c.ComposeEntries() {
		for _, file := range entry.ComposeConfig().Files {
			declaredFiles[filepath.Clean(file)] = true
		}
	}
	if len(declaredFiles) == 0 {
		return nil
	}

	var warnings []string
	for builtin, command := range c.Interaction {
		if (builtin != "build" && builtin != "logs") || command == nil || len(command.Replace) != 1 {
			continue
		}
		if replaceHookIsComposeCandidate(command.Replace[0], builtin, declaredFiles) {
			warnings = append(warnings, fmt.Sprintf(
				"interaction.%s.replace: may duplicate the configured compose %s invocation; review whether this hook is still needed alongside `dva %s <plan>`",
				builtin, builtin, builtin))
		}
	}
	sort.Strings(warnings)
	return warnings
}

// replaceHookIsComposeCandidate recognizes only an exact, one-command compose invocation:
// docker compose -f FILE [-f FILE ...] VERB, or docker-compose -f FILE ... VERB.
// The -f set must exactly match all declared compose files. A quoted path, a newline, or a
// shell separator is outside this recognizer: valid shell syntax is not a proof that the
// hook and lifecycle path have the same behaviour.
func replaceHookIsComposeCandidate(step ProvisionItem, builtin string, declaredFiles map[string]bool) bool {
	commands := step.RunCommands()
	if len(commands) != 1 {
		return false
	}
	command := commands[0]
	if strings.ContainsAny(command, "\r\n;|&") {
		return false
	}
	fields := strings.Fields(command)
	if len(fields) < 4 {
		return false
	}

	i := 0
	switch {
	case fields[0] == "docker-compose":
		i = 1
	case len(fields) >= 2 && fields[0] == "docker" && fields[1] == "compose":
		i = 2
	default:
		return false
	}

	files := make(map[string]bool)
	for i+1 < len(fields) && fields[i] == "-f" {
		file := filepath.Clean(fields[i+1])
		if !declaredFiles[file] {
			return false
		}
		files[file] = true
		i += 2
	}
	// Without -f, compose's default-file lookup depends on the invocation directory.
	// A subset also cannot establish even a useful candidate across a multi-file stack.
	return len(files) > 0 && sameStringSet(files, declaredFiles) && i+1 == len(fields) && fields[i] == builtin
}

func sameStringSet(left, right map[string]bool) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if !right[value] {
			return false
		}
	}
	return true
}

// warnLegacyModes warns when legacy `modes` are present and suggests migration.
func (c *Config) warnLegacyModes() []string {
	if len(c.Modes) == 0 {
		return nil
	}

	return []string{
		fmt.Sprintf("⚠ 'modes' section detected — consider migrating to 'plans' + 'environments' + 'sites'\n  Migration guide: %s\n  Hint: modes will continue to work but are deprecated in favor of the new plans model", migrationGuideURL),
	}
}

// warnLegacyStackOrder warns when `stack.*.order` is used and suggests moving order to plan entries.
func (c *Config) warnLegacyStackOrder() []string {
	if len(c.Stack) == 0 {
		return nil
	}

	var affected []string
	for name, entry := range c.Stack {
		if entry != nil && entry.Order > 0 {
			affected = append(affected, name)
		}
	}
	if len(affected) == 0 {
		return nil
	}

	sort.Strings(affected)
	return []string{
		fmt.Sprintf("⚠ 'stack.*.order' detected — execution order must move to 'plans.*.entries[].order'\n  Run: dva config migrate (preview) / dva config migrate --write (apply)\n  Migration guide: %s\n  Affected entries: %s\n  Hint: a plan entry's order is the one that runs, with no fallback to the declaration — an order left here is not read on the plan path", migrationGuideURL, strings.Join(affected, ", ")),
	}
}

// warnLegacyEnvironmentFields warns when deprecated environment stack fields are used.
func (c *Config) warnLegacyEnvironmentFields() []string {
	if len(c.Environments) == 0 {
		return nil
	}

	var affected []string
	for envName, profile := range c.Environments {
		if len(profile.Stack) > 0 || len(profile.StackOverrides) > 0 {
			affected = append(affected, envName)
		}
	}
	if len(affected) == 0 {
		return nil
	}

	sort.Strings(affected)
	return []string{
		fmt.Sprintf("⚠ 'environments.*.stack/stack_overrides' detected — these fields are deprecated\n  Migration guide: %s\n  Affected environments: %s\n  Hint: an environment carries 'environment:' (its variables) only; stack selection belongs in plans", migrationGuideURL, strings.Join(affected, ", ")),
	}
}

// warnNoPlansHint emits a migration hint when stack exists but no plans are defined.
func (c *Config) warnNoPlansHint() []string {
	if len(c.Stack) == 0 || len(c.Plans) > 0 {
		return nil
	}

	return []string{
		fmt.Sprintf("ℹ No 'plans' defined — consider adding execution plans for 'dva up <name>' support\n  Migration guide: %s\n  Hint: plans combine stack entries, environments, and sites into named execution targets\n  Example:\n    plans:\n      local-dev:\n        environment: dev\n        site: local\n        entries:\n          - name: <stack-entry>\n            runner: <runner-name>\n            order: 10", migrationGuideURL),
	}
}

// No warning exists for a config version below the running binary, and none should.
// `version:` is the minimum DVA a config requires (USAGE.md), so config < binary is
// the correct, portable state: the binary satisfies the floor. The removed
// warnVersionOutdated advised raising the floor to match the running binary, which
// would strand every user on an older DVA and ratchet upward on every release.
// config > binary is the only real failure and Load() already rejects it.

// warnInertProvisionSteps flags provision and hook items that carry a label and no payload.
//
// A warning rather than an error, deliberately. Such an item is always a mistake — `note:`
// is what an author uses to print a message — but rejecting it would turn a config that
// validates today into one that fails, and the item has been quietly doing nothing since
// long before this check existed. The runtime notice (InertStepMessage, printed by every
// step runner) is the part that reaches the author at the moment the hook misbehaves;
// `validate` is not what anyone runs when a build silently produces nothing.
//
// Sorted, because both sources are maps and an unsorted result would reorder between runs.
func (c *Config) warnInertProvisionSteps() []string {
	var warnings []string

	collect := func(path string, items []ProvisionItem) {
		for i, item := range items {
			if !item.IsInert() {
				continue
			}
			label := item.Step
			if label == "" {
				label = fmt.Sprintf("step %d", i+1)
			}
			warnings = append(warnings, fmt.Sprintf("%s[%d] %q: %s", path, i, label, InertStepMessage))
		}
	}

	// Recursive, because hooks nest: `interaction.db.subcommands.migrate.before` is as real
	// a place to write an inert step as the top level, and a check that stopped at depth 1
	// would report the shallow mistake and stay silent on the identical deep one.
	//
	// Uses the shared walker rather than its own. It previously joined segments with a bare
	// dot, producing `interaction.db.migrate.before[0]` — a path that does not exist in the
	// document, since `migrate` lives under `db.subcommands`. A user searching their file for
	// it finds nothing. TASK-128.
	eachInteractionNode(c.Interaction, func(path string, cmd *InteractionCommand, _ inheritedExec) {
		collect(path+".steps", cmd.Steps)
		collect(path+".before", cmd.Before)
		collect(path+".replace", cmd.Replace)
		collect(path+".after", cmd.After)
	})

	for profile, items := range c.Provision.Profiles {
		collect(fmt.Sprintf("provision.%s", profile), items)
	}

	sort.Strings(warnings)
	return warnings
}

// warnIgnoredParallelSteps flags interaction steps that ask for concurrency the interaction
// path does not implement.
//
// `parallel:` reaches `interaction.*.steps` only because that field and `provision.*` share
// the ProvisionItem type, and schema.json documents the type. `Parallel` appears zero times
// in non-test code under internal/runner/, so runStepLoop has no batching: the key parses,
// validates, and is dropped. Measured — two `sleep 1` steps both marked parallel take 2.02s
// under `dva run` and 1.01s under `dva provision`, off one config.
//
// A warning rather than an error, for the reason warnInertProvisionSteps records: the key has
// been quietly doing nothing since it existed, and rejecting it would fail configs that
// validate today. The difference from an inert step is why the runtime notice matters more
// here — an inert step betrays itself by producing nothing, while this one produces exactly
// the right output and merely takes twice as long, so `validate` alone would reach nobody who
// was not already suspicious.
//
// Provision items are deliberately not walked: there the key works.
func (c *Config) warnIgnoredParallelSteps() []string {
	var warnings []string

	collect := func(path string, items []ProvisionItem) {
		for i, item := range items {
			if !item.Parallel {
				continue
			}
			label := item.Step
			if label == "" {
				label = fmt.Sprintf("step %d", i+1)
			}
			warnings = append(warnings, fmt.Sprintf("%s[%d] %q: %s", path, i, label, IgnoredParallelMessage))
		}
	}

	// Hooks as well as steps, and recursively: `interaction.db.subcommands.migrate.before` runs
	// through the same loop with the same absent scheduler, so a check that stopped at
	// `.steps` would report the shallow case and stay silent on the identical deep one.
	eachInteractionNode(c.Interaction, func(path string, cmd *InteractionCommand, _ inheritedExec) {
		collect(path+".steps", cmd.Steps)
		collect(path+".before", cmd.Before)
		collect(path+".replace", cmd.Replace)
		collect(path+".after", cmd.After)
	})

	sort.Strings(warnings)
	return warnings
}

// warnIgnoredPlanFilters flags a `plans:` filter where nothing routes a plan to compare it
// against.
//
// `plans:` reaches `provision:` and `interaction.*.steps` only because those share the
// ProvisionItem type with hooks, exactly as `parallel:` reaches hooks from the other
// direction. Neither of those two paths has a routed plan: `dva provision` and `dva run` take
// a command name, not a plan name, so the filter parses, validates, and is dropped.
//
// A warning rather than an error, following warnIgnoredParallelSteps: the key does nothing
// here, and refusing it would fail configs over a line that changes no behaviour. It differs
// from the parallel case in which direction the silence runs — a dropped `parallel:` produces
// the right output more slowly, while a dropped `plans:` produces the output of a step the
// author believed was filtered out. That is the worse of the two, which is why the message
// says what the key did rather than only that it was ignored.
//
// Hook phases are deliberately not walked: there the key works. A hook filter naming a
// missing plan is a validateHookPlanFilters error, not a warning here.
func (c *Config) warnIgnoredPlanFilters() []string {
	var warnings []string

	collect := func(path string, items []ProvisionItem) {
		for i, item := range items {
			if len(item.Plans) == 0 {
				continue
			}
			label := item.Step
			if label == "" {
				label = fmt.Sprintf("step %d", i+1)
			}
			warnings = append(warnings, fmt.Sprintf("%s[%d] %q: %s", path, i, label, IgnoredPlanFilterMessage))
		}
	}

	eachInteractionNode(c.Interaction, func(path string, cmd *InteractionCommand, _ inheritedExec) {
		collect(path+".steps", cmd.Steps)
	})

	// c.Provision.Profiles is a map (TASK-128): walk it in name order so two configs with
	// the same defect report it identically on every run.
	profileNames := make([]string, 0, len(c.Provision.Profiles))
	for name := range c.Provision.Profiles {
		profileNames = append(profileNames, name)
	}
	sort.Strings(profileNames)
	for _, name := range profileNames {
		collect("provision."+name, c.Provision.Profiles[name])
	}

	sort.Strings(warnings)
	return warnings
}

// warnDuplicatePlanDeclarations warns when two plans declared in the same partition
// (see below) carry equal declaration fields: `environment`, `site`, `vars`,
// `endpoint_tags`, and per-entry `name`, `runner`, `order`, `depends_on`, `services`,
// `vars` (TASK-244 / PLAN-002's frozen D6 contract).
//
// This is deliberately narrower than "the plans are equivalent at runtime" — it does
// not resolve site overrides, environment profiles, or stack entries, so the message
// below states only that the compared declaration fields match. It also never
// recommends a canonical name or deletion: two authors may have equal declarations on
// purpose (e.g. one mid-migration to a renamed plan), and this check's job is to
// surface the coincidence, not to resolve it.
func (c *Config) warnDuplicatePlanDeclarations() []string {
	if len(c.Plans) < 2 {
		return nil
	}

	names := make([]string, 0, len(c.Plans))
	for name := range c.Plans {
		names = append(names, name)
	}
	sort.Strings(names)

	var warnings []string
	for i, nameA := range names {
		planA := c.Plans[nameA]
		if planA == nil {
			continue
		}
		// Skip alias plans — they intentionally duplicate their target
		if planA.Alias != "" {
			continue
		}
		for _, nameB := range names[i+1:] {
			planB := c.Plans[nameB]
			if planB == nil {
				continue
			}
			// Skip alias plans
			if planB.Alias != "" {
				continue
			}
			// Same PlanConfig pointer (imported plan with `as:` alias)
			if planA == planB {
				continue
			}
			// Partition by SubprojectPath
			if planA.SubprojectPath != planB.SubprojectPath {
				continue
			}
			if !plansHaveEqualDeclaration(planA, planB) {
				continue
			}
			warnings = append(warnings, fmt.Sprintf(
				"plans %q and %q declare equal environment, site, vars, endpoint_tags, entries, and composes — review whether both are intentional; consider using alias: { alias: %q } for one",
				nameA, nameB, nameA,
			))
		}
	}
	return warnings
}

// plansHaveEqualDeclaration compares exactly the fields TASK-244's D6 contract
// freezes. Map fields (Vars) are order-insensitive by construction: maps.Equal
// compares key/value pairs, not iteration order, and treats a nil map as equal to
// an empty one because both have length zero. Slice fields (EndpointTags, and
// per-entry DependsOn/Services) are order-sensitive: slices.Equal compares
// position by position, so a reordered list is NOT a duplicate, and it likewise
// treats nil and empty as equal by length. Description is intentionally excluded —
// it is prose, not a declared executable field, and the card does not list it.
func plansHaveEqualDeclaration(a, b *PlanConfig) bool {
	if a.Environment != b.Environment || a.Site != b.Site {
		return false
	}
	if !maps.Equal(a.Vars, b.Vars) {
		return false
	}
	if !slices.Equal(a.EndpointTags, b.EndpointTags) {
		return false
	}
	if len(a.Entries) != len(b.Entries) {
		return false
	}
	for i := range a.Entries {
		if !planEntriesEqual(a.Entries[i], b.Entries[i]) {
			return false
		}
	}
	// Composition plans (TASK-260) carry their whole declaration in Composes and have
	// no Entries, so without this comparison any two of them looked equal (TASK-324).
	if len(a.Composes) != len(b.Composes) {
		return false
	}
	for i := range a.Composes {
		if !compositionEntriesEqual(a.Composes[i], b.Composes[i]) {
			return false
		}
	}
	return true
}

// compositionEntriesEqual is planEntriesEqual for CompositionEntry: Plan, Order,
// DependsOn, Vars, compared positionally like Entries.
func compositionEntriesEqual(a, b CompositionEntry) bool {
	return a.Plan == b.Plan &&
		a.Order == b.Order &&
		slices.Equal(a.DependsOn, b.DependsOn) &&
		maps.Equal(a.Vars, b.Vars)
}

// planEntriesEqual compares one PlanEntry pair on the fields D6 freezes: Name,
// Runner, Order, DependsOn, Profiles, Services, Vars. Entries are compared
// positionally by plansHaveEqualDeclaration (list order in Entries is itself part of the
// declaration), so this only judges whether the entries at matching positions
// agree.
func planEntriesEqual(a, b PlanEntry) bool {
	return a.Name == b.Name &&
		a.Runner == b.Runner &&
		a.Order == b.Order &&
		slices.Equal(a.DependsOn, b.DependsOn) &&
		slices.Equal(a.Profiles, b.Profiles) &&
		slices.Equal(a.Services, b.Services) &&
		maps.Equal(a.Vars, b.Vars)
}

// warnMultiplePlansWithoutDefault warns when two or more plans are declared but
// `default_plan` is not set, leaving bare lifecycle commands (e.g. `dva up`) with no
// bounded plan selection — plan_lifecycle.go's guard already refuses those at
// runtime ("multiple plans configured; specify one: ..."), so this surfaces the same
// condition earlier, at `validate` time.
//
// Checks c.DefaultPlanName == "" rather than c.DefaultPlan() == "" (or
// c.DefaultPlanSource() == "none"). Those two also resolve to "" / "none" when
// default_plan IS declared but names a plan that does not exist — and
// validate.go's Validate() already rejects exactly that as a hard error
// ("default_plan '%s' not found in plans"). Testing the resolved value here would
// re-warn on top of that hard error and would contradict this warning's own
// wording ("default_plan is not set"), which is only true when the key is absent,
// not when it is present but wrong.
//
// len(c.Plans) >= 2 excludes the single-plan implicit-default contract: DefaultPlan()
// already treats a lone plan as the default with no declaration required, so a
// single-plan config is not ambiguous and must not warn here.
func (c *Config) warnMultiplePlansWithoutDefault() []string {
	if len(c.Plans) < 2 || c.DefaultPlanName != "" {
		return nil
	}

	names := make([]string, 0, len(c.Plans))
	for name := range c.Plans {
		names = append(names, name)
	}
	sort.Strings(names)

	return []string{fmt.Sprintf(
		"%d plans are defined (%s) but default_plan is not set — bare lifecycle commands (e.g. 'dva up') require naming a plan explicitly; set default_plan to one of them",
		len(names), strings.Join(names, ", "),
	)}
}

// warnLiteralKeyShadowsSubproject warns when a declared colon key's prefix also names a
// subproject, so the literal key wins and the subproject's command of the same spelling
// becomes unreachable.
//
// This shape could not occur before TASK-167: run.go split every colon key, so the
// subproject always won and a literal `engine:test` in the parent was dead config. Routing
// the literal key first fixes the far larger silent-failure class that task measured, and
// creates exactly this one ambiguity in exchange — so the warning ships with the routing
// change rather than after it. Closing one silent shadowing by opening another would be
// TASK-137's silent-relocation shape, which is the thing that task exists to stop.
//
// A warning, not an error: both readings are legitimate config, the author may well mean the
// local one, and it stays runnable either way. The message names the form that lost, because
// knowing `engine:test` now runs the parent's command is only half of what a reader needs —
// the other half is how to reach the child's.
//
// The escape hatch is spelled `dva run --project`, with the verb, and it was executed against
// the binary before being written here — the bar ConflictAdvice sets. The shorter `dva
// --project engine test` reads better and does not work: --project is registered on runCmd, so
// it only parses after an explicit `run`, and the bare form's rewrite in cli.Execute does not
// look past a leading flag. Measured, it exits 1 with `unknown command "test" for "dva"`.
//
// Severity: Semantic Warning
func (c *Config) warnLiteralKeyShadowsSubproject() []string {
	var warnings []string

	for name := range c.Interaction {
		idx := strings.Index(name, ":")
		if idx <= 0 {
			continue
		}
		prefix, sub := name[:idx], name[idx+1:]
		if _, ok := c.Subprojects[prefix]; !ok {
			continue
		}
		// A reserved prefix is still unroutable, so nothing is shadowed — that config has a
		// hard error from ValidateReservedCommands already and does not need a second opinion.
		// Both sides of the shape are now hard errors: ValidateReservedCommands rejects the
		// key, and ReservedSubprojectNames rejects the subproject that gave the prefix its
		// second meaning.
		if IsReservedCommand(prefix) {
			continue
		}
		warnings = append(warnings, fmt.Sprintf(
			"interaction.%s: `dva %s` runs this key, not subproject `%s`'s `%s` — "+
				"the literal key takes precedence; use `dva run --project %s %s` to reach the subproject",
			name, name, prefix, sub, prefix, sub))
	}

	// c.Interaction is a map; see warnDeepSubcommandNesting. TASK-128.
	sort.Strings(warnings)
	return warnings
}

// warnUnresolvedEnvVars checks if any variables in the environment block remain unresolved
// after config and OS interpolation, indicating a possible typo or missing variable.
//
// Severity: Semantic Warning
func (c *Config) warnUnresolvedEnvVars(env *Environment, envFilesDeclared bool) []string {
	// An env_file declaration can define any of the names still unresolved here,
	// and validation does not open it. Warning anyway would report a defect that
	// does not exist for every project that keeps its variables in .env.
	if envFilesDeclared {
		return nil
	}

	var warnings []string

	for k, v := range c.Environment {
		finalVal := env.Interpolate(v)
		// Extract all remaining `${VAR}` or `$VAR` patterns
		matches := findVarRefs(finalVal)
		if len(matches) > 0 {
			warnings = append(warnings,
				fmt.Sprintf("environment.%s: contains unresolved variable reference %v; verify variable name",
					k, matches))
		}
	}

	sort.Strings(warnings)
	return warnings
}

// warnSuspiciousEnvPatterns warns when users try to use shell-specific interpolation semantics
// that the config expander does not implement: `$#`, `${VAR:=x}`, `${VAR:+x}`, `${VAR:?x}`
// and the like. `${VAR:-default}` and `${VAR-default}` are supported (TASK-303) and are
// not reported.
//
// Severity: Semantic Warning
func (c *Config) warnSuspiciousEnvPatterns() []string {
	var warnings []string

	for k, v := range c.Environment {
		if strings.Contains(v, "$#") || hasUnsupportedBracedOperator(v) {
			warnings = append(warnings,
				fmt.Sprintf("environment.%s: contains shell-specific syntax that is not supported; use $VAR, ${VAR} or ${VAR:-default}", k))
		}
	}

	sort.Strings(warnings)
	return warnings
}

// warnMultiplePrimaryCompose warns when multiple compose stack entries declare primary=true.
// Only one entry should be marked as primary; the first one wins but this indicates a config conflict.
func (c *Config) warnMultiplePrimaryCompose() []string {
	var primaries []string
	for name, entry := range c.Stack {
		if entry != nil && entry.ComposeConfig() != nil && entry.Primary {
			primaries = append(primaries, name)
		}
	}
	if len(primaries) <= 1 {
		return nil
	}
	sort.Strings(primaries)
	return []string{
		fmt.Sprintf("stack: multiple compose entries marked primary=true (%s); only one should be primary — the first alphabetically (%s) is used, review the others",
			strings.Join(primaries, ", "), primaries[0]),
	}
}
