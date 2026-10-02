package config

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// ComposeOptions holds per-command Docker Compose options.
type ComposeOptions struct {
	Method     string   `yaml:"method"`
	Profiles   []string `yaml:"profiles"`
	RunOptions []string `yaml:"run_options"`
}

// ProvisionConfig holds provision profiles with an optional default_profile alias.
type ProvisionConfig struct {
	DefaultProfile string                     `yaml:"-"`
	Profiles       map[string][]ProvisionItem `yaml:"-"`

	// profileOwners records the child configuration an imported profile executes
	// against, keyed by its registered name. Profiles is map[string][]ProvisionItem —
	// a value slice with nowhere to hang the pointer PlanConfig and InteractionCommand
	// carry inline — so ownership lives beside it rather than inside the item type,
	// which is also shared by interaction steps and hooks that have no owner of their
	// own. Unexported for the same reason as PlanConfig.owner: it holds local absolute
	// paths and must not serialize (TASK-264).
	profileOwners map[string]*Config

	profileSubprojects map[string]string
	profileCanonicals  map[string]string
}

// ProfileOwner returns the configuration a provision profile resolves against.
// Locally declared profiles have no recorded owner, so fallback preserves their
// historical behavior.
func (pc *ProvisionConfig) ProfileOwner(name string, fallback *Config) *Config {
	if pc != nil && pc.profileOwners != nil {
		if owner := pc.profileOwners[name]; owner != nil {
			return owner
		}
	}
	return fallback
}

// setProfileOwner records the owner of a registered profile name. Canonical and alias
// registrations of one import must be passed the same *Config so they cannot drift.
func (pc *ProvisionConfig) setProfileOwner(name string, owner *Config) {
	if owner == nil {
		return
	}
	if pc.profileOwners == nil {
		pc.profileOwners = make(map[string]*Config)
	}
	pc.profileOwners[name] = owner
}

// SetProfileIdentity records the owner and identity metadata of a registered profile name.
func (pc *ProvisionConfig) SetProfileIdentity(name, subprojectName, canonicalAddress string, owner *Config) {
	if owner != nil {
		pc.setProfileOwner(name, owner)
	}
	if subprojectName != "" {
		if pc.profileSubprojects == nil {
			pc.profileSubprojects = make(map[string]string)
		}
		pc.profileSubprojects[name] = subprojectName
	}
	if canonicalAddress != "" {
		if pc.profileCanonicals == nil {
			pc.profileCanonicals = make(map[string]string)
		}
		pc.profileCanonicals[name] = canonicalAddress
	}
}

// ProfileSubproject returns the subproject name this profile was imported from,
// or "" for a locally declared profile.
func (pc *ProvisionConfig) ProfileSubproject(name string) string {
	if pc != nil && pc.profileSubprojects != nil {
		return pc.profileSubprojects[name]
	}
	return ""
}

// ProfileCanonicalAddress returns the canonical address of an imported profile,
// or "" for a locally declared profile.
func (pc *ProvisionConfig) ProfileCanonicalAddress(name string) string {
	if pc != nil && pc.profileCanonicals != nil {
		return pc.profileCanonicals[name]
	}
	return ""
}

// MarshalYAML restores the schema shape consumed by UnmarshalYAML.
func (pc ProvisionConfig) MarshalYAML() (any, error) {
	provision := make(map[string]any, len(pc.Profiles)+1)
	if pc.DefaultProfile != "" {
		provision["default_profile"] = pc.DefaultProfile
	}
	for name, items := range pc.Profiles {
		provision[name] = items
	}
	return provision, nil
}

// UnmarshalYAML handles the mixed-type provision mapping:
// "default_profile" key is extracted as a string; all other keys are profiles.
func (pc *ProvisionConfig) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("provision: expected mapping, got kind %d", node.Kind)
	}

	pc.Profiles = make(map[string][]ProvisionItem)

	for i := 0; i < len(node.Content)-1; i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]

		if key == "default_profile" {
			pc.DefaultProfile = val.Value
			continue
		}

		var items []ProvisionItem
		if err := val.Decode(&items); err != nil {
			return fmt.Errorf("provision profile '%s': %w", key, err)
		}
		pc.Profiles[key] = items
	}

	return nil
}

// ProvisionItem represents a single item in a provision profile.
type ProvisionItem struct {
	// Step-based format
	Step     string `yaml:"step"`
	Run      any    `yaml:"run"`
	Note     string `yaml:"note"`
	Parallel bool   `yaml:"parallel"` // Run concurrently with consecutive parallel steps

	// Plans restricts a hook step to the named plans. Empty means every plan, which is
	// what every pre-TASK-331 config says by omission. Honoured under
	// interaction.<hookable>.before/replace/after only; `provision:` and
	// interaction.*.steps have no routed plan to compare against, and validate warns
	// when the key appears there (docs/64 §4, mirroring `parallel:`).
	Plans []string `yaml:"plans"`

	// Compose-aware commands (inherit compose.files and compose.project_name)
	ComposeUp   []string `yaml:"compose_up"`   // Services to start: [postgres, minio, redis]
	ComposeExec string   `yaml:"compose_exec"` // Command in service: "pg_isready -U ndstack"
	ComposeRun  string   `yaml:"compose_run"`  // One-off command in service

	// Legacy structured format
	Echo string `yaml:"echo"`
	Cmd  string `yaml:"cmd"`

	// Raw string format (set during custom unmarshal)
	Raw string `yaml:"-"`
}

// UnmarshalYAML handles both string and object provision items.
func (p *ProvisionItem) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind == yaml.ScalarNode {
		p.Raw = node.Value
		return nil
	}
	// unmarshal the object fields manually to avoid recursion
	type plain ProvisionItem
	return node.Decode((*plain)(p))
}

// RunCommands extracts the run commands from a ProvisionItem.
func (p *ProvisionItem) RunCommands() []string {
	if p.Raw != "" {
		return []string{p.Raw}
	}
	if p.Run == nil {
		return nil
	}
	switch v := p.Run.(type) {
	case string:
		return []string{v}
	case []any:
		cmds := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				cmds = append(cmds, s)
			}
		}
		return cmds
	}
	return nil
}

// InertStepMessage is what every step runner prints in place of the work an inert item
// implied. It lives beside IsInert because the loops that print it are seven call sites
// across three packages; while the wording lived nowhere, they could not even agree on
// whether to print anything.
const InertStepMessage = "nothing ran — this item is a label with no 'run:'. Add 'run:' to execute a command, or 'note:' if it is a message."

// IgnoredParallelMessage is what the interaction step loop prints when a step asks for
// concurrency it will not get. `parallel:` is honoured on the provision path only; the
// interaction path has no scheduler, so the key parses, validates, and changes nothing.
//
// It is worded as a timing claim because that is the only symptom. An inert step announces
// itself by producing no output, which is why InertStepMessage can afford to be a
// description; a step that runs sequentially instead of concurrently produces byte-identical
// output and is simply slower, so nothing but this line tells the author the key was read
// and dropped. TASK-140.
const IgnoredParallelMessage = "'parallel:' is ignored here — interaction steps always run sequentially. It is honoured under 'provision:'."

// IgnoredPlanFilterMessage is what validate prints where `plans:` cannot filter anything.
//
// It names the consequence rather than only the fact, because the two failure directions are
// not symmetric with IgnoredParallelMessage above: a dropped `parallel:` still does the right
// work, while a dropped `plans:` runs a step the author believed was excluded. TASK-331.
const IgnoredPlanFilterMessage = "'plans:' is ignored here and this step runs unfiltered — only interaction hooks (before/replace/after) route a plan to filter on. Move the step to a hook, or drop the key."

// StepsIgnoreParallel reports whether a step list asks for concurrency the executor will not
// give it, so both executors decide to warn from one place.
//
// Two of them exist. `steps:` runs through runner.runStepLoop; `before:`/`replace:`/`after:`
// run through cli.runHookSteps, a separate loop in a package internal/runner cannot import.
// The first cut of TASK-140 put the check inline in runStepLoop only, and `dva up` with a
// parallel-marked before-hook stayed silent while `validate` warned — the exact split this
// change exists to close, since validate is the surface an author may never visit. A
// predicate rather than a bool field because the answer is derived, and rather than a copied
// three-line loop because runStepLoop's own header records what copied loops cost here.
func StepsIgnoreParallel(steps []ProvisionItem) bool {
	for _, s := range steps {
		if s.Parallel {
			return true
		}
	}
	return false
}

// AppliesToPlan reports whether this hook step runs for the plan a lifecycle command
// routed to. planName is "" when nothing routed — no argument, no `default_plan`, or a
// dva.yml with no plans at all.
//
// Omitting `plans:` means every plan, so the configs that existed before the key did
// keep running exactly as they did. A filter that names plans never matches "" : the
// author said which plans the step belongs to, and "no plan" is not one of them. docs/64 §3
// carries the full table.
func (p *ProvisionItem) AppliesToPlan(planName string) bool {
	if len(p.Plans) == 0 {
		return true
	}
	return slices.Contains(p.Plans, planName)
}

// StepsForPlan returns the steps of a hook phase that apply to planName, and the labels of
// those it filtered out.
//
// The skipped labels are returned rather than discarded because the caller has to announce
// them (docs/64 §3): a declared step that does not run and says nothing leaves "why did my
// hook not fire" unanswerable from the output. Labels are resolved here, against the
// pre-filter index, so a skipped step's synthesised "step N" matches the position the author
// counts in dva.yml rather than its position in the surviving slice.
//
// Returning a filtered slice rather than filtering at the call site is what lets
// wrapWithHooks ask `len(replace) > 0` *after* filtering. Asking before would let a
// `replace:` list that this plan filters away empty out into a no-op that also suppresses the
// built-in — the command would do nothing at all, where the declaration means "on this plan,
// use the built-in".
func StepsForPlan(steps []ProvisionItem, planName string) (kept []ProvisionItem, skipped []string) {
	for i, s := range steps {
		if s.AppliesToPlan(planName) {
			kept = append(kept, s)
			continue
		}
		label := s.Step
		if label == "" {
			label = fmt.Sprintf("step %d", i+1)
		}
		skipped = append(skipped, fmt.Sprintf("%s — skipped: plans: [%s], running plan is %s",
			label, strings.Join(s.Plans, ", "), describeRoutedPlan(planName)))
	}
	return kept, skipped
}

// describeRoutedPlan renders the routed plan for the skip line. "" is not a plan name and
// quoting it as one yields an empty pair of quotes, which reads like a plan whose name
// really is the empty string.
func describeRoutedPlan(planName string) string {
	if planName == "" {
		return "none"
	}
	return "'" + planName + "'"
}

// IsInert reports whether this item carries no payload at all: nothing to run, nothing to
// print.
//
// `step:` is a label. Every examples/*.yml uses it that way and the runners synthesise
// "step N" when it is missing, so an item holding a label and nothing else announces work
// and performs none. Measured on 0.1.44, `- step: "make build"` in a directory with no
// Makefile printed `[hook:replace:build] [1/1] make build` and exited 0 having run nothing —
// the only signal being the absence of the `$` line that the executing form prints.
//
// Every payload field counts, not just Run. An item with compose_up, echo or cmd does
// something, and reporting it as inert would be a false positive on a working config. Raw
// needs no test of its own: RunCommands returns it.
func (p *ProvisionItem) IsInert() bool {
	return len(p.RunCommands()) == 0 &&
		p.Note == "" &&
		len(p.ComposeUp) == 0 &&
		p.ComposeExec == "" &&
		p.ComposeRun == "" &&
		p.Echo == "" &&
		p.Cmd == ""
}
