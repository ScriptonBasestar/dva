package lifecycle

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

type ExecutionPlan struct {
	Name            string
	EnvironmentName string
	SiteName        string
	EndpointTags    []string
	EnvVars         map[string]string
	Entries         []ResolvedEntry
	ResolutionTrace []string

	// Warnings carries the resolution facts a user needs even when they did not ask to
	// see the resolution. ResolutionTrace is the full narration and prints only under
	// --dry-run; anything in here prints on every path, so it stays short by policy —
	// an entry silently vanishing from the plan is the case it exists for (TASK-374).
	Warnings []string

	owner *config.Config
}

// OwnerConfig returns the configuration that supplied the declarations resolved
// into this plan. Plans constructed by callers before owner tracking retain the
// supplied fallback configuration.
func (p *ExecutionPlan) OwnerConfig(fallback *config.Config) *config.Config {
	if p != nil && p.owner != nil {
		return p.owner
	}
	return fallback
}

type ResolvedEntry struct {
	Name         string
	StackEntry   *config.LifecycleEntry
	Runner       string
	RunnerConfig any
	Order        int
	DependsOn    []string
	Profiles     []string
	Services     []string
	Wave         int
	WorkingDir   string
	Vars         map[string]string
}

type ResolveError struct {
	PlanName string
	Step     string
	Cause    error
}

func (e *ResolveError) Error() string {
	return fmt.Sprintf("plan %q: %s: %v", e.PlanName, e.Step, e.Cause)
}

func (e *ResolveError) Unwrap() error {
	return e.Cause
}

func ResolvePlanName(cfg *config.Config, name string) (planName string, plan *config.PlanConfig, err error) {
	if cfg == nil {
		return "", nil, fmt.Errorf("nil config")
	}
	if len(cfg.Plans) == 0 {
		return "", nil, fmt.Errorf("no plans configured")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return "", nil, fmt.Errorf("plan name is empty")
	}
	resolved, ok := cfg.Plans[name]
	if !ok || resolved == nil {
		return "", nil, fmt.Errorf("plan %q not found", name)
	}
	return name, resolved, nil
}

// ResolvePlanAlias resolves an alias chain, returning the final concrete plan.
// It detects and reports cycles, self-references, and chain length violations.
// The returned plan is guaranteed to be a concrete plan (Alias == "").
func ResolvePlanAlias(cfg *config.Config, name string) (string, *config.PlanConfig, error) {
	const maxAliasDepth = 10
	visited := make(map[string]bool)
	current := name

	for depth := 0; depth <= maxAliasDepth; depth++ {
		if depth == maxAliasDepth {
			return "", nil, fmt.Errorf("plan %q: alias chain exceeds maximum depth of %d (possible cycle)", name, maxAliasDepth)
		}
		resolved, ok := cfg.Plans[current]
		if !ok || resolved == nil {
			return "", nil, fmt.Errorf("plan %q not found", current)
		}
		if resolved.Alias == "" {
			return current, resolved, nil
		}
		if resolved.Alias == current {
			return "", nil, fmt.Errorf("plan %q: alias cannot reference itself", current)
		}
		if visited[resolved.Alias] {
			return "", nil, fmt.Errorf("plan %q: alias cycle detected (via %q)", name, resolved.Alias)
		}
		visited[current] = true
		current = resolved.Alias
	}
	return "", nil, fmt.Errorf("plan %q: alias resolution failed", name)
}

// MergePlanExtends merges a child plan into its parent (extends).
// The parent must be a concrete plan (no alias, no composes).
// Returns the merged plan (a new PlanConfig) or an error.
func MergePlanExtends(cfg *config.Config, childName string, child *config.PlanConfig) (*config.PlanConfig, error) {
	if child.Extends == "" {
		return child, nil
	}
	if child.Alias != "" {
		return nil, fmt.Errorf("plan %q: cannot have both alias and extends", childName)
	}
	if len(child.Composes) > 0 {
		return nil, fmt.Errorf("plan %q: extends is mutually exclusive with composes", childName)
	}

	const maxExtendsDepth = 3
	parentName := child.Extends
	if parentName == childName {
		return nil, fmt.Errorf("plan %q: extends cannot reference itself", childName)
	}

	// Resolve parent (following any aliases)
	parentName, parent, err := ResolvePlanAlias(cfg, parentName)
	if err != nil {
		return nil, fmt.Errorf("plan %q: %w", childName, err)
	}

	if parent.Alias != "" {
		return nil, fmt.Errorf("plan %q: extends target %q must be a concrete plan (not an alias)", childName, parentName)
	}
	if len(parent.Composes) > 0 {
		return nil, fmt.Errorf("plan %q: extends target %q must be a concrete plan (not a composition plan)", childName, parentName)
	}

	// Check extends depth by walking up the chain
	depth := 1
	current := parent
	visited := map[string]bool{childName: true, parentName: true}
	for current.Extends != "" && depth < maxExtendsDepth {
		currentName := current.Extends
		if visited[currentName] {
			return nil, fmt.Errorf("plan %q: extends cycle detected at %q", childName, currentName)
		}
		visited[currentName] = true
		var err error
		currentName, current, err = ResolvePlanAlias(cfg, currentName)
		if err != nil {
			return nil, fmt.Errorf("plan %q: %w", childName, err)
		}
		if current.Alias != "" || len(current.Composes) > 0 {
			return nil, fmt.Errorf("plan %q: extends chain contains non-concrete plan %q", childName, currentName)
		}
		depth++
	}
	if depth >= maxExtendsDepth && current.Extends != "" {
		return nil, fmt.Errorf("plan %q: extends chain exceeds maximum depth of %d", childName, maxExtendsDepth)
	}

	// Deep clone parent and merge child into it
	merged := clonePlanConfig(parent)
	merged = mergePlanConfigs(merged, child)
	return merged, nil
}

// clonePlanConfig creates a deep copy of a PlanConfig.
func clonePlanConfig(p *config.PlanConfig) *config.PlanConfig {
	if p == nil {
		return nil
	}
	cloned := *p
	if p.Vars != nil {
		cloned.Vars = maps.Clone(p.Vars)
	}
	if p.Entries != nil {
		cloned.Entries = make([]config.PlanEntry, len(p.Entries))
		copy(cloned.Entries, p.Entries)
	}
	if p.Composes != nil {
		cloned.Composes = make([]config.CompositionEntry, len(p.Composes))
		copy(cloned.Composes, p.Composes)
	}
	if p.EndpointTags != nil {
		cloned.EndpointTags = slices.Clone(p.EndpointTags)
	}
	return &cloned
}

// mergePlanConfigs merges child into parent (parent is base, child overrides).
// Scalar fields: child wins if non-empty.
// Vars: key-merged (child keys override parent).
// Entries: matched by Name — child entry with same Name replaces parent entry entirely.
// New entries (Names not in parent) are appended.
func mergePlanConfigs(parent, child *config.PlanConfig) *config.PlanConfig {
	if child.Description != "" {
		parent.Description = child.Description
	}
	if child.Environment != "" {
		parent.Environment = child.Environment
	}
	if child.Site != "" {
		parent.Site = child.Site
	}
	if len(child.EndpointTags) > 0 {
		parent.EndpointTags = slices.Clone(child.EndpointTags)
	}
	if len(child.Vars) > 0 {
		if parent.Vars == nil {
			parent.Vars = make(map[string]string)
		}
		maps.Copy(parent.Vars, child.Vars)
	}

	// Merge entries by Name
	if len(child.Entries) > 0 {
		if parent.Entries == nil {
			parent.Entries = make([]config.PlanEntry, 0, len(child.Entries))
		}
		parentByName := make(map[string]int, len(parent.Entries))
		for i, e := range parent.Entries {
			parentByName[e.Name] = i
		}
		for _, childEntry := range child.Entries {
			if idx, exists := parentByName[childEntry.Name]; exists {
				parent.Entries[idx] = childEntry
			} else {
				parent.Entries = append(parent.Entries, childEntry)
			}
		}
	}
	return parent
}

func ResolvePlan(cfg *config.Config, planName string, cliVars map[string]string) (*ExecutionPlan, error) {
	concreteName, concretePlan, err := ResolvePlanAlias(cfg, planName)
	if err != nil {
		return nil, &ResolveError{PlanName: planName, Step: "alias", Cause: err}
	}

	finalPlan, err := MergePlanExtends(cfg, concreteName, concretePlan)
	if err != nil {
		return nil, &ResolveError{PlanName: planName, Step: "extends", Cause: err}
	}

	owner := finalPlan.OwnerConfig(cfg)
	if owner == nil {
		return nil, &ResolveError{PlanName: concreteName, Step: "owner", Cause: fmt.Errorf("plan owner config is nil")}
	}

	resolved := &ExecutionPlan{
		Name:            planName,
		EnvironmentName: finalPlan.Environment,
		SiteName:        finalPlan.Site,
		EndpointTags:    copyStringSlice(finalPlan.EndpointTags),
		EnvVars:         make(map[string]string),
		Entries:         make([]ResolvedEntry, 0, len(finalPlan.Entries)),
		ResolutionTrace: make([]string, 0, 16),
		owner:           owner,
	}

	if concreteName != planName {
		resolved.trace("plan: %q is an alias for %q", planName, concreteName)
	}
	if finalPlan.Extends != "" {
		resolved.trace("plan: %q extends %q", concreteName, finalPlan.Extends)
	}
	resolved.trace("plan: resolved %q", concreteName)

	// Layers 1-2 of the documented precedence chain are applied by the config loader
	// (loadEnv, internal/cli/root.go), not here — see appendUpstreamVarTrace.
	appendUpstreamVarTrace(resolved, owner)

	// Layers 3-6. Each merge is recorded whether or not it contributed: a layer the config
	// does not declare is the answer to "why is my variable not set", so it is stated rather
	// than skipped (the reasoning TASK-082 settled for the dogfood loop applies to the
	// precedence chain too — an absence is information).
	resolved.traceLayer("vars: global vars", owner.Vars, "not declared")
	mergeStringMap(resolved.EnvVars, owner.Vars)

	var envProfile *config.EnvironmentProfile
	if finalPlan.Environment != "" {
		p, ok := owner.Environments[finalPlan.Environment]
		if !ok {
			return nil, &ResolveError{
				PlanName: concreteName,
				Step:     "environment",
				Cause:    fmt.Errorf("environment %q not found", finalPlan.Environment),
			}
		}
		envProfile = &p
		resolved.traceLayer(fmt.Sprintf("vars: environments.%q", finalPlan.Environment), envProfile.Environment, "selected, declares no vars")
		mergeStringMap(resolved.EnvVars, envProfile.Environment)
	} else {
		resolved.trace("vars: environments — none selected by this plan")
	}

	var site *config.SiteConfig
	if finalPlan.Site != "" {
		s, ok := owner.Sites[finalPlan.Site]
		if !ok || s == nil {
			return nil, &ResolveError{
				PlanName: concreteName,
				Step:     "site",
				Cause:    fmt.Errorf("site %q not found", finalPlan.Site),
			}
		}
		site = s
		resolved.traceLayer(fmt.Sprintf("vars: sites.%q.vars", finalPlan.Site), site.Vars, "selected, declares no vars")
		mergeStringMap(resolved.EnvVars, site.Vars)
	} else {
		resolved.trace("vars: sites — none selected by this plan")
	}

	resolved.traceLayer(fmt.Sprintf("vars: plans.%q.vars", concreteName), finalPlan.Vars, "not declared")
	mergeStringMap(resolved.EnvVars, finalPlan.Vars)

	resolved.traceLayer("vars: cli --var", cliVars, "none passed")
	mergeStringMap(resolved.EnvVars, cliVars)

	// Layer 7, and the reason the chain cannot be read as "last writer wins": Environment
	// .MergeVars gives a key already present in the OS environment priority over every
	// value above, so this is the last word on any variable the shell already exports.
	resolved.trace("vars: OS environment overrides every layer above")

	if len(finalPlan.Entries) == 0 {
		resolved.trace("entries: none declared by this plan")
		return resolved, nil
	}

	if len(owner.Stack) == 0 {
		return nil, &ResolveError{PlanName: concreteName, Step: "stack_ref", Cause: fmt.Errorf("stack is empty")}
	}

	for i := range finalPlan.Entries {
		planEntry := finalPlan.Entries[i]
		entryName := strings.TrimSpace(planEntry.Name)
		if entryName == "" {
			return nil, &ResolveError{PlanName: concreteName, Step: "stack_ref", Cause: fmt.Errorf("entry[%d] has empty name", i)}
		}

		stackEntry, ok := owner.Stack[entryName]
		if !ok || stackEntry == nil {
			return nil, &ResolveError{
				PlanName: concreteName,
				Step:     "stack_ref",
				Cause:    fmt.Errorf("stack entry %q not found", entryName),
			}
		}

		finalRunner := normalizeRunnerName(stackEntry.DefaultRunner)
		entryOverride := (*config.SiteEntryOverride)(nil)
		if site != nil && site.EntryOverrides != nil {
			entryOverride = site.EntryOverrides[entryName]
			if entryOverride != nil && strings.TrimSpace(entryOverride.Runner) != "" {
				finalRunner = normalizeRunnerName(entryOverride.Runner)
			}
		}
		if strings.TrimSpace(planEntry.Runner) != "" {
			finalRunner = normalizeRunnerName(planEntry.Runner)
		}

		if finalRunner == "" {
			if len(stackEntry.Runners) == 1 {
				for k := range stackEntry.Runners {
					finalRunner = normalizeRunnerName(k)
					break
				}
			} else {
				finalRunner = normalizeRunnerName(stackEntry.DetectPlugin())
			}
		}

		if finalRunner == "" {
			return nil, &ResolveError{
				PlanName: concreteName,
				Step:     "runner",
				Cause:    fmt.Errorf("entry %q has no resolvable runner", entryName),
			}
		}

		if len(stackEntry.Runners) > 0 && !runnerDeclared(stackEntry.Runners, finalRunner) {
			return nil, &ResolveError{
				PlanName: concreteName,
				Step:     "runner",
				Cause:    fmt.Errorf("entry %q runner %q is not declared in stack.runners", entryName, finalRunner),
			}
		}

		runnerConfig, err := stackEntry.GetRunnerConfig(finalRunner)
		if err != nil {
			return nil, &ResolveError{PlanName: concreteName, Step: "runner_config", Cause: fmt.Errorf("entry %q: %w", entryName, err)}
		}

		// Optional directory check: if the entry is marked optional and the directory it
		// declares does not exist, drop the entry from the plan instead of failing the
		// whole plan (TASK-319).
		//
		// This sits after the runner is chosen, not before it, because the directory that
		// decides the entry's fate belongs to the runner that will actually run it. An
		// entry declaring runners.native (with a dir) alongside runners.compose (without
		// one) must not be skipped on native's directory when the plan picked compose
		// (TASK-374). Consulting the entry's declarations in isolation could only ever
		// guess, and picked the wrong one whenever the plan disagreed with the guess.
		//
		// Placing it here also means a malformed optional entry — an undeclared runner, an
		// unresolvable one — now reports that error instead of being silently dropped.
		// optional: promises tolerance for a directory that is not checked out, not for a
		// declaration that does not parse, and the two failures want opposite responses.
		if stackEntry.Optional {
			if dir := optionalSkipDir(stackEntry, runnerConfig); dir != "" {
				resolvedDir := EntryDir(owner.FileDir(), dir)
				if _, err := os.Stat(resolvedDir); err != nil && os.IsNotExist(err) {
					resolved.warn("entry: %s (optional) — skipped, directory %q not found", entryName, resolvedDir)
					continue
				}
			}
		}

		resolvedEntry := ResolvedEntry{
			Name:         entryName,
			StackEntry:   stackEntry,
			Runner:       finalRunner,
			RunnerConfig: runnerConfig,
			Order:        planEntry.Order,
			DependsOn:    copyStringSlice(planEntry.DependsOn),
			Profiles:     copyStringSlice(planEntry.Profiles),
			Services:     copyStringSlice(planEntry.Services),
			Wave:         0,
			Vars:         make(map[string]string),
		}

		mergeStringMap(resolvedEntry.Vars, resolved.EnvVars)
		mergeStringMap(resolvedEntry.Vars, stackEntry.Vars)
		// runners.<name>.env sits just inside the entry's own vars: it is still a declaration,
		// so the override and plan layers below overrule it, but it is scoped to one runner
		// choice and so beats vars that apply to the entry whichever runner is picked.
		//
		// Merged into Vars rather than carried on the runner config because native is desugared
		// to the process plugin (applyRunnerConfig) and ProcessPluginConfig has no Env field.
		// Vars already reach the command — the orchestrator merges them into the entry
		// Environment and startLocalProcess passes Env.EnvSlice() to it — so this reuses a
		// delivered path instead of opening a second one. Until this, schema.json advertised
		// native_runner_config.env and decodeRunnerNode decoded it, and nothing ever read it.
		if native, ok := runnerConfig.(*config.NativeRunnerConfig); ok {
			mergeStringMap(resolvedEntry.Vars, native.Env)
		}
		if entryOverride != nil {
			mergeStringMap(resolvedEntry.Vars, entryOverride.Vars)
		}
		mergeStringMap(resolvedEntry.Vars, planEntry.Vars)

		switch rc := runnerConfig.(type) {
		case *config.ProcessPluginConfig:
			resolvedEntry.WorkingDir = rc.Dir
		case *config.NativeRunnerConfig:
			resolvedEntry.WorkingDir = rc.Dir
		}

		resolved.Entries = append(resolved.Entries, resolvedEntry)
		resolved.trace("entry: %s -> runner=%s order=%d deps=%d", entryName, finalRunner, resolvedEntry.Order, len(resolvedEntry.DependsOn))
	}

	if err := CalculateWaves(resolved.Entries); err != nil {
		return nil, &ResolveError{PlanName: concreteName, Step: "waves", Cause: err}
	}

	sort.Slice(resolved.Entries, func(i, j int) bool {
		a := resolved.Entries[i]
		b := resolved.Entries[j]
		if a.Wave != b.Wave {
			return a.Wave < b.Wave
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		return a.Name < b.Name
	})

	resolved.trace("waves: calculated and entries sorted")

	return resolved, nil
}

func CalculateWaves(entries []ResolvedEntry) error {
	if len(entries) == 0 {
		return nil
	}

	nameToIdx := make(map[string]int, len(entries))
	for i := range entries {
		name := strings.TrimSpace(entries[i].Name)
		if name == "" {
			return fmt.Errorf("entry[%d] has empty name", i)
		}
		if _, exists := nameToIdx[name]; exists {
			return fmt.Errorf("duplicate entry name %q", name)
		}
		nameToIdx[name] = i
	}

	inDegree := make([]int, len(entries))
	outgoing := make([][]int, len(entries))
	for i := range entries {
		for _, depName := range entries[i].DependsOn {
			depName = strings.TrimSpace(depName)
			if depName == "" {
				continue
			}
			depIdx, ok := nameToIdx[depName]
			if !ok {
				return fmt.Errorf("entry %q depends_on unknown entry %q", entries[i].Name, depName)
			}
			outgoing[depIdx] = append(outgoing[depIdx], i)
			inDegree[i]++
		}
	}

	queue := make([]int, 0, len(entries))
	for i := range entries {
		if inDegree[i] == 0 {
			entries[i].Wave = 0
			queue = append(queue, i)
		}
	}

	processed := 0
	for len(queue) > 0 {
		idx := queue[0]
		queue = queue[1:]
		processed++

		for _, next := range outgoing[idx] {
			if entries[next].Wave < entries[idx].Wave+1 {
				entries[next].Wave = entries[idx].Wave + 1
			}
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if processed != len(entries) {
		return fmt.Errorf("cycle detected in depends_on graph")
	}

	return nil
}

// trace appends one resolution step. Everything the user reads under 'Resolution:' in
// 'dva up <plan> --dry-run' comes from here, so a step must state what happened, not what
// the code intended to do.
func (p *ExecutionPlan) trace(format string, args ...any) {
	p.ResolutionTrace = append(p.ResolutionTrace, fmt.Sprintf(format, args...))
}

// warn records a fact that has to reach the user on the execution path, and traces it as
// well so the dry-run narration stays a complete account of the resolution rather than a
// parallel one that happens to omit the parts that matter most.
func (p *ExecutionPlan) warn(format string, args ...any) {
	p.Warnings = append(p.Warnings, fmt.Sprintf(format, args...))
	p.trace(format, args...)
}

// traceLayer records one layer of the variable precedence chain and how much it carried.
// An empty layer is reported instead of omitted: "why did my variable never get set" is
// answered by the layer that had nothing in it, so the layer has to appear either way.
//
// absent phrases the empty case for this particular layer, because the layers are not the
// same kind of thing — a config section is declared or not, a --var flag is passed or not,
// and saying "not declared" about a command-line flag would be wrong.
func (p *ExecutionPlan) traceLayer(label string, vars map[string]string, absent string) {
	if len(vars) == 0 {
		p.trace("%s — %s", label, absent)
		return
	}
	p.trace("%s — merged (%s)", label, pluralKeys(len(vars)))
}

// appendUpstreamVarTrace records the two layers that sit below everything ResolvePlan
// merges but are applied by a different stage: the top-level `environment:` map and
// `env_file:`, both loaded by loadEnv (internal/cli/root.go) into the Environment that
// this plan's EnvVars are later merged onto (e.MergeVars(plan.EnvVars), plan_lifecycle.go).
//
// They are named here rather than merged here on purpose. loadEnv is the config-load stage
// for every command — the legacy stack path has no plan to resolve and still needs env_file
// — so re-reading those files during plan resolution would read the same paths twice per
// invocation and give the resolver file I/O it otherwise does not do. What the trace owes
// the user is the layer's position in the chain, which is knowable from the declaration
// alone; the values themselves are already in the Environment by the time this runs.
//
// Verified against the real binary: an env_file key reaches the process, and a global
// `vars:` key of the same name wins over it — exactly the documented order.
func appendUpstreamVarTrace(resolved *ExecutionPlan, cfg *config.Config) {
	// "declared", not "applied": an optional env_file that does not exist is skipped by
	// config.LoadEnvFile without error, so naming the declared set is the strongest claim
	// this function can make without opening the files itself.
	if files := cfg.AllEnvFiles(); len(files) > 0 {
		resolved.trace("vars: env_file — declared [%s], applied at config load below every layer here", strings.Join(files, ", "))
	} else {
		resolved.trace("vars: env_file — not declared")
	}

	if len(cfg.Environment) > 0 {
		resolved.trace("vars: environment: — applied at config load, below every layer here (%s)", pluralKeys(len(cfg.Environment)))
	} else {
		resolved.trace("vars: environment: — not declared")
	}
}

func pluralKeys(n int) string {
	if n == 1 {
		return "1 key"
	}
	return fmt.Sprintf("%d keys", n)
}

func mergeStringMap(dst map[string]string, src map[string]string) {
	if len(src) == 0 {
		return
	}
	maps.Copy(dst, src)
}

func copyStringSlice(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, len(in))
	copy(out, in)
	return out
}

func normalizeRunnerName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return ""
	}
	return strings.ReplaceAll(name, "_", "-")
}

func runnerDeclared(runners map[string]any, runner string) bool {
	runner = normalizeRunnerName(runner)
	for k := range runners {
		if normalizeRunnerName(k) == runner {
			return true
		}
	}
	return false
}

// runnerConfigDir reports the working directory one runner config declares. It returns ""
// both when the config is absent and when that runner shape has no directory of its own —
// compose, helm, script, docker and friends locate their work by file, not by directory —
// and the caller treats "" as "nothing to check" rather than as the config directory, so an
// entry that never named a directory is not skipped on the strength of an unrelated path.
//
// A typed nil pointer stored in an any is not a nil interface, so every case has to guard
// its own pointer: the flat-field callers below pass fields that are usually nil.
func runnerConfigDir(cfg any) string {
	switch c := cfg.(type) {
	case *config.NativeRunnerConfig:
		if c != nil {
			return c.Dir
		}
	case *config.ProcessPluginConfig:
		if c != nil {
			return c.Dir
		}
	case *config.KustomizePluginConfig:
		if c != nil {
			return c.Dir
		}
	case *config.TiltPluginConfig:
		if c != nil {
			return c.Dir
		}
	case *config.VagrantPluginConfig:
		if c != nil {
			return c.Dir
		}
	case *config.ServerlessPluginConfig:
		if c != nil {
			return c.Dir
		}
	}
	return ""
}

// optionalSkipDir picks the directory whose existence decides whether an optional entry
// survives plan resolution, or "" when nothing about the entry names one.
//
// The runner config the plan settled on is asked first and answers on its own whenever it
// declares a directory. Only when it declares none — compose, helm, script, docker and
// friends locate their work by file, not by directory — does source.path stand in, because
// a source: entry that was never checked out is exactly the case optional: was added for.
//
// "" means "nothing to check", not "the config directory": an entry that named no directory
// must not be skipped on the strength of an unrelated path.
func optionalSkipDir(e *config.LifecycleEntry, runnerConfig any) string {
	if dir := runnerConfigDir(runnerConfig); dir != "" {
		return dir
	}
	if e.Source != nil && e.Source.Path != "" {
		return e.Source.Path
	}
	return ""
}
