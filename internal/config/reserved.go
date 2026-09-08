package config

import (
	"fmt"
	"log/slog"
	"maps"
	"sort"
	"strings"
)

// reservedCommands is the canonical set of built-in DVA command names.
// Use IsReservedCommand or ReservedCommandNames for read-only access.
var reservedCommands = map[string]bool{
	"help": true, "version": true, "ls": true, "compose": true,
	"up": true, "stop": true, "down": true, "build": true,
	"run": true, "provision": true, "validate": true, "manifest": true,
	"ktl": true, "kubectl": true, "ssh": true, "console": true,
	"completion": true, "init": true, "status": true, "config": true,
	"logs": true, "restart": true, "show": true, "doctor": true,
	"skill": true, "agent-deny": true, "ci": true,
}

// hookableCommands is the subset of reserved commands that support
// before/replace/after hooks via the interaction section.
//
// `clean` left this set with the command: teardown is `dva down <plan> --purge`, and a flag
// has no interaction key to hang a hook on. It is the one removal that could go quiet —
// `stack`/`app`/`infra` were never hookable, so nobody had a working hook under those names
// to lose — so validateHookPlacement carries a message naming the removal by name.
var hookableCommands = map[string]bool{
	"up": true, "down": true, "stop": true,
	"restart": true, "build": true,
	"logs": true,
}

// IsHookableCommand reports whether name is a built-in command that
// supports before/replace/after hooks.
func IsHookableCommand(name string) bool {
	return hookableCommands[name]
}

// HookableCommandList renders the hookable set as the sentence fragment the two hook
// placement errors both end in.
//
// Derived rather than written out. The literal list said "up, down, stop, restart, build,
// clean, logs" in both messages, so removing `clean` from the set above would have left two
// messages advertising a command the same file no longer accepts — the reader is told to
// use a name that fails the check that printed the advice.
func HookableCommandList() string {
	names := make([]string, 0, len(hookableCommands))
	for name := range hookableCommands {
		names = append(names, name)
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// HookableCommands returns a copy of the hookable command set.
func HookableCommands() map[string]bool {
	cp := make(map[string]bool, len(hookableCommands))
	maps.Copy(cp, hookableCommands)
	return cp
}

// ReservedCommands returns a copy of the built-in DVA command set.
// A custom interaction declared under one of these names is rejected by Validate; nothing about
// it is silent. See ShadowedByBuiltin for what the bare form actually runs, and ConflictAdvice
// for the way out, which differs by the kind of collision.
func ReservedCommands() map[string]bool {
	cp := make(map[string]bool, len(reservedCommands))
	maps.Copy(cp, reservedCommands)
	return cp
}

// IsReservedCommand reports whether name is a built-in DVA command.
func IsReservedCommand(name string) bool {
	return reservedCommands[name]
}

// ShadowedByBuiltin reports whether the bare `dva <name>` form runs a built-in command
// instead of the interaction declared under name.
//
// cli.Execute only rewrites `dva <name>` to `dva run <name>` when name is not a built-in, so a
// reserved name never reaches its interaction that way. It stays reachable as `dva run <name>`,
// which is why the condition is shadowing and not the "ignored" this once claimed to be.
//
// A hookable built-in that declares before/replace/after is not shadowed: the built-in runs the
// hook, so the bare form does reach what the author declared. A namespaced name like `compose:ps`
// is not shadowed either — the colon keeps it out of the built-in set, so dynamic routing sends
// it to `run` — even though ValidateReservedCommands still rejects the prefix.
func ShadowedByBuiltin(name string, cmd *InteractionCommand) bool {
	if !IsReservedCommand(name) {
		return false
	}
	if IsHookableCommand(name) && cmd != nil && cmd.HasHooks() {
		return false
	}
	return true
}

// UnroutableNamespacePrefix returns the reserved built-in named by a namespaced
// interaction key's prefix, or "" when the key carries no such prefix.
//
// This is a third state, distinct from ShadowedByBuiltin. A shadowed key still runs —
// `dva run <name>` reaches it — so the surfaces can name a working invocation. A key
// like `compose:ps` is reached by nothing: the colon keeps it out of the built-in set,
// and the run form reads `compose:` as a subproject reference. ConflictAdvice carries the
// full reason and the way out; this reports only whether the condition holds, so the
// machine-readable surfaces and the validator cannot disagree about which keys it
// covers.
//
// Scope is deliberately narrow: a *reserved* prefix, and since TASK-167 that is the whole
// unroutable class rather than a subset of it. A free prefix (`mytool:fast`) used to fail
// the same way — run.go split every key on ':' — and TASK-167 answered that as the routing
// question it was: LiteralKeyWins now routes such a key to the command the author declared.
// The reserved prefix is the one case it excepts, so this predicate still describes exactly
// the keys nothing can reach.
func UnroutableNamespacePrefix(name string) string {
	idx := strings.Index(name, ":")
	if idx <= 0 {
		return ""
	}
	if prefix := name[:idx]; IsReservedCommand(prefix) {
		return prefix
	}
	return ""
}

// LiteralKeyWins reports whether a command name should be looked up as an interaction key
// as written, rather than split on ':' and read as `<subproject>:<command>`.
//
// run.go used to split unconditionally, before asking whether the literal key existed. That
// made every colon key whose prefix was not a declared subproject reachable by nothing:
// `dva mytool:fast` exited 1 with "subproject `mytool` not found" while `dva config validate`
// said the file was valid and `dva manifest` advertised `usage_example: dva mytool:fast`.
// The colon was never reserved for subprojects by the schema, the docs, or the config model
// — only by that one SplitN — so this makes routing agree with the four surfaces that
// already treated the key as an ordinary command.
//
// An exact match beats an inference: a declared key is what the author wrote, a subproject
// reference is what the shape suggests. A parent CAN declare a literal `engine:test` key of
// its own alongside a same-named `engine` subproject — nothing rejects that config — and when
// it does, this function is exactly why the literal key wins and the subproject's `test` of
// the same spelling becomes unreachable through the colon form. warnLiteralKeyShadowsSubproject
// (validate_warnings.go) exists precisely for that shape: it fires on load and names the
// `dva run --project` form that still reaches the shadowed child command.
//
// The reserved prefix is excepted so `compose:ps` keeps failing. Validate rejects such a
// config outright (rc 1, reserved conflict), and routing it here would ship a file that one
// surface calls a hard error while another runs it happily — the disagreement TASK-137 was
// about. It also keeps `unroutable_reason` ("prefix is a reserved DVA command") a true and
// complete account of what is left unroutable. TASK-167.
//
// The exception is keyed off the live reserved set, so removing a built-in moves keys out of
// it: `app:build` was unroutable while `app` was a command and is an ordinary interaction
// now. That is the intended consequence — the prefix names nothing DVA owns any more — and
// it is pinned by a test rather than left to be discovered.
func LiteralKeyWins(c *Config, name string) bool {
	if !strings.Contains(name, ":") {
		return false
	}
	if UnroutableNamespacePrefix(name) != "" {
		return false
	}
	_, declared := c.Interaction[name]
	return declared
}

// RenameSuggestion returns the name a colon-carrying key should be renamed to.
//
// Every colon goes, not just the first. `compose:sub:cmd` → `compose-sub:cmd` still carries
// a colon, so run.go still splits it and it still fails — and because `compose-sub` is not a
// reserved command, UnroutableNamespacePrefix no longer covers it, so validate reports
// the config clean and ls shows the entry unmarked. Following the advice literally turned
// a loud error into the silent one TASK-167 describes; measured before the fix on `dva
// app-sub:cmd` (back when `app` was a built-in), which exited 1 with subproject `app-sub`
// not found while validate said valid.
//
// One function because the advice string and the ls mark both need it and had drifted
// into two copies of the same expression.
func RenameSuggestion(name string) string {
	return strings.ReplaceAll(name, ":", "-")
}

// ReservedCommandConflict represents a conflict between an interaction
// command name and a reserved built-in command.
type ReservedCommandConflict struct {
	Name   string
	Source string // "interaction", "module:<name>", "override"
}

// ValidateReservedCommands checks if any interaction command names
// conflict with reserved built-in command names.
// Commands that define hook fields (before/replace/after) on hookable
// built-in commands are NOT treated as conflicts.
// Namespaced names like "compose:ps" are also rejected when the prefix
// before ':' is a reserved command (e.g., "compose", "run").
func ValidateReservedCommands(interaction map[string]*InteractionCommand) []ReservedCommandConflict {
	var conflicts []ReservedCommandConflict
	for name, cmd := range interaction {
		if IsReservedCommand(name) {
			// ShadowedByBuiltin carries the hook exemption, so the surfaces that mark a
			// conflict and the validator that rejects one cannot disagree about which
			// names are conflicts.
			if ShadowedByBuiltin(name, cmd) {
				conflicts = append(conflicts, ReservedCommandConflict{
					Name:   name,
					Source: "interaction",
				})
			}
			continue
		}
		// Check namespace prefix: "compose:ps" conflicts if "compose" is reserved. Shared with
		// the manifest/ls mark, so a key the surfaces call unroutable is exactly a key the
		// validator rejects.
		if UnroutableNamespacePrefix(name) != "" {
			conflicts = append(conflicts, ReservedCommandConflict{
				Name:   name,
				Source: "interaction",
			})
		}
	}
	return conflicts
}

// ConflictAdvice returns what happens to an interaction declared under name, and the way out.
//
// One function because the warning on every config load and the error from validate describe the
// same condition to the same reader. They disagreed before: the warning said the interaction was
// discarded while the error said the config was fatal, and neither named the invocation that works.
//
// Every invocation named here was executed against the binary. That is the bar, because advice
// that names a command which refuses is worse than no advice — the reader trusts it and stops.
func ConflictAdvice(name string) string {
	// Namespaced keys first: this is the one case reachable by no invocation at all, so it is
	// also the one case where "rename" is the whole answer rather than a preference.
	//
	// "No invocation reaches this key" is scoped to configs that pass `dva config validate`,
	// and what makes it true there is ValidateReservedCommands rejecting this interaction key
	// itself — not ReservedSubprojectNames, which runs in the same Validate and never on the
	// routing path. An unvalidated config still routes: measured at 1e73a99 against a parent
	// declaring both subproject `up` and interaction `up:web`, `dva up:web` logged this very
	// advice and then printed the child's CHILD-WEB, rc 0. The sentence is honest about the
	// key it is handed and says nothing about where that spelling lands in a config nobody
	// validated, which is the most a load-time warning can promise.
	if idx := strings.Index(name, ":"); idx > 0 {
		// The cause is spelled out but no failing invocation is written in full: this text is
		// read by machines (it reaches the reader through validate's stderr and the load-time
		// warning), and a consumer scanning for a `dva run …` form would lift it out of the
		// negation and execute the one command that cannot work.
		return fmt.Sprintf(
			"namespace prefix '%s' is a reserved DVA command — no invocation reaches this key: "+
				"the bare form is not a built-in, and the run form reads '%s:' as a subproject "+
				"reference, so it fails with subproject '%s' not found. Use a different "+
				"separator (e.g., '%s')",
			name[:idx], name[:idx], name[:idx], RenameSuggestion(name),
		)
	}
	if IsHookableCommand(name) {
		// The hook route is listed first because it is the only one that gets the short form
		// working; `dva run` is the fallback for an author who wants a separate command.
		return fmt.Sprintf(
			"'%s' is a reserved DVA command — declare before/replace/after to extend the "+
				"built-in, or reach this as 'dva run %s'",
			name, name,
		)
	}
	// The interaction is not discarded: measured, `dva run status` still executes it. Only the
	// bare `dva status` form is lost, to the built-in. Telling the reader it was ignored sends
	// them looking for a command that never ran, when what they need is the form that reaches it.
	//
	// The old wording is deliberately not quoted anywhere in this file, comments included: a
	// grep for it is one of TASK-076's checks, and a comment containing the phrase would satisfy
	// the grep while the message regressed — the check has to fail for exactly one reason.
	return fmt.Sprintf(
		"'%s' is a reserved DVA command — the built-in runs for 'dva %s', so this is reachable "+
			"only as 'dva run %s'. Rename to get the short form back (e.g., 'my-%s')",
		name, name, name, name,
	)
}

// FormatConflictWarnings formats conflict list as warning messages.
func FormatConflictWarnings(conflicts []ReservedCommandConflict) string {
	if len(conflicts) == 0 {
		return ""
	}

	// Sorted, and one clause per conflict: conflicts arrive in map iteration order, so a message
	// that detailed only conflicts[0] named a different command on every run — and the three
	// kinds of conflict do not share advice, so one clause cannot stand in for the others.
	names := make([]string, 0, len(conflicts))
	for _, c := range conflicts {
		names = append(names, c.Name)
	}
	sort.Strings(names)

	clauses := make([]string, 0, len(names))
	for _, name := range names {
		clauses = append(clauses, fmt.Sprintf("interaction command %s", ConflictAdvice(name)))
	}
	return strings.Join(clauses, "; ")
}

// WarnReservedCommandConflicts logs warnings for any conflicts found.
func WarnReservedCommandConflicts(interaction map[string]*InteractionCommand) []ReservedCommandConflict {
	conflicts := ValidateReservedCommands(interaction)
	if len(conflicts) > 0 {
		slog.Warn(FormatConflictWarnings(conflicts))
	}
	return conflicts
}

// ReservedSubprojectNames returns, sorted, the subproject names that collide with a
// reserved built-in command.
//
// TASK-263 §3 decision (a). It reads as mere consistency with the interaction namespace,
// but it closes a measured divergence. ConflictAdvice tells the author of an interaction
// key spelled `up:web` that "no invocation reaches this key ... the run form reads 'up:'
// as a subproject reference, so it fails with subproject 'up' not found". Measured on
// v0.1.48 against a parent declaring subproject `up` and interaction `up:web`, `dva up:web`
// printed the child's CHILD-WEB while that same run logged that advice — the invocation the
// advice calls impossible reached a different command in another project.
//
// The subproject name is the side that gives. The interaction key genuinely is unroutable
// whenever its prefix is reserved (UnroutableNamespacePrefix says so, and Validate rejects
// it); it is the subproject that quietly claims a spelling the reserved set already owns.
//
// What this rule buys is narrower than "it makes UnroutableNamespacePrefix's account true".
// That account was already true of every validating config, and vacuously so: the interaction
// key `up:web` is rejected by ValidateReservedCommands whether or not a subproject named `up`
// exists (measured — a config with only the interaction key exits 1). What this adds is that
// a config which validates can no longer hold the *declaration* that makes one spelling mean
// two things at once.
//
// It does not close the divergence on the routing path, and by TASK-263 §3 (a) it is not
// meant to: the decision put the rule in Validate and froze routing. So the two surfaces
// still disagree on a config nobody validated — measured at 1e73a99, a parent declaring
// subproject `compose` and a child `ps` runs `dva compose:ps` → CHILD-PS, rc 0, while
// `dva config validate` on that same file exits 1. Closing that would mean moving the check
// onto the routing path, which is a separate decision from the one this implements.
// RootOwnerName is the value `dva manifest` and `dva ls --json` report in `owner` for an
// item this dva.yml declares itself, rather than imports (TASK-333). It lives here, beside
// the reserved-name rules, because validate.go refuses a subproject spelled this way and
// internal/cli reads the same constant for the field it writes — one literal, so the
// sentinel and the rule that protects it cannot drift apart.
//
// Deliberately NOT in reservedCommands: that set also bans a name as an interaction key and
// as a `p:k` prefix, and nothing about `dva run root` is ambiguous. The narrower problem
// gets the narrower rule.
const RootOwnerName = "root"

func ReservedSubprojectNames(subprojects map[string]SubprojectConfig) []string {
	var names []string
	for name := range subprojects {
		if IsReservedCommand(name) {
			names = append(names, name)
		}
	}
	// Subprojects is a map: sort so two collisions are reported in the same order on every
	// run, matching FormatConflictWarnings' reason for doing the same.
	sort.Strings(names)
	return names
}

// SubprojectConflictAdvice returns what a reserved subproject name breaks, and the way out.
//
// Separate from ConflictAdvice because the two conditions have different consequences and
// different fixes: a reserved interaction key loses only its bare form and stays reachable
// through `dva run`, while a reserved subproject name makes one spelling mean two things.
// No invocation is written out in full here, for ConflictAdvice's reason — this text reaches
// machines through validate's stderr, and a consumer scanning for a runnable `dva ...` form
// would lift it out of the sentence that says it must not exist.
func SubprojectConflictAdvice(name string) string {
	return fmt.Sprintf(
		"'%s' is a reserved DVA command — an interaction key spelled '%s:<command>' is "+
			"rejected as reachable by nothing, while the same spelling on the command line "+
			"routes to this subproject's '<command>'. One spelling cannot mean both. Rename "+
			"the subproject (e.g., '%s-project')",
		name, name, name,
	)
}

// RejectsInteractionKey reports whether c's own `dva config validate` rejects the
// interaction key name as a reserved-command conflict, and returns the reserved built-in it
// collides with plus the advice that says why.
//
// Per key, not per config. The rule TASK-263 §3 decision (b) froze is about a key being
// addressable, so a child with an unrelated conflict elsewhere keeps its healthy keys
// importable and runnable from the parent.
//
// It delegates to ValidateReservedCommands rather than re-deriving the condition, so the
// set of keys the parent routes refuse is by construction the set the child's own validator
// rejects. Re-deriving it is how the two would drift the next time the hook exemption or the
// reserved set moves.
//
// builtin is returned from here rather than derived by the caller for the same reason. The
// listing surfaces name it in their unroutable mark, and a caller that recomputed it would
// be free to print a built-in name for a key this call reported as fine — the two answers
// have to come out of one predicate. It is the prefix for a namespaced key (`compose:ps` →
// `compose`) and the key itself for a plain reserved one (`status` → `status`); those are
// the only two shapes ValidateReservedCommands rejects.
func (c *Config) RejectsInteractionKey(name string) (rejected bool, builtin, advice string) {
	cmd, declared := c.Interaction[name]
	if !declared {
		return false, "", ""
	}
	if len(ValidateReservedCommands(map[string]*InteractionCommand{name: cmd})) == 0 {
		return false, "", ""
	}
	if prefix := UnroutableNamespacePrefix(name); prefix != "" {
		return true, prefix, ConflictAdvice(name)
	}
	return true, name, ConflictAdvice(name)
}

// SubprojectKeyRejection is the error every parent route returns for a child interaction key
// the child's own validator rejects (TASK-263 §3 decision (b)).
//
// One constructor for all three address forms — `--project`, the `p:key` shorthand and a
// `p/key` import — because a reader who tries the next form after the first refuses must
// not be told a different story about why. It names the rule and the declaration that
// tripped it, and carries the child's own diagnosis, which is the part the reader cannot
// get otherwise: they are standing in the parent directory and have no reason to have run
// validate in the child.
//
// The forwarded diagnosis explains the cause; it is not a command to run from here. It was
// written for a reader inside the child and says so in the child's terms — for a reserved
// key `status` it reads "reachable only as 'dva run status'", and measured at 1e73a99 that
// invocation exits 1 with "command `status` not recognized" from the parent directory and
// prints CHILD-STATUS only after a `cd` into the child. Rewriting it to be runnable from
// here would mean re-deriving the child's advice in the parent's terms, which is the
// duplicate diagnosis this constructor exists to avoid.
func SubprojectKeyRejection(subproject, key, childAdvice string) error {
	return fmt.Errorf(
		"subprojects.%s: interaction `%s` is rejected by `dva config validate` inside "+
			"subproject `%s`, so no parent route addresses it — `--project`, the `%s:%s` "+
			"shorthand and an `%s/%s` import all refuse it. The subproject's own diagnosis: %s",
		subproject, key, subproject, subproject, key, subproject, key, childAdvice,
	)
}
