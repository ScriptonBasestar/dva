package cli

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

var downCmd = &cobra.Command{
	Use:   "down [PLAN] [OPTIONS]",
	Short: "Stop and remove services (a named plan, or the whole stack)",
	Long: `Stop and remove a named plan.
Without a plan name it uses default_plan, or the only plan when exactly one
is declared. With several plans and no default_plan it refuses and asks you
to name one. With no plans configured it tears down every declared stack entry.

A leading -- is a separator, never an argument: "dva down --" does exactly what a
bare "dva down" does in that config, and "dva down -- X" is read exactly as
"dva down X" -- including when that is a refusal. So a wrapper written as
'dva down -- "$@"' is safe when "$@" is empty. Only the LEADING one is consumed:
"dva down -- --" is still an error, and a -- written after something else is left
alone.

Plan usage:
  dva down <plan>         Tear down the selected plan
  --var KEY=VAL           Override a plan variable
  --volumes, -v           Also remove volumes
  --purge                 Also remove volumes, locally built images and provision markers.
                          Asks for confirmation first; --force answers it.
  --dry-run               Print the variable resolution and the actions, without executing

Whole-stack-path flags (rejected, not ignored, once a plan is named):
  --mode, -M MODE           Use a named mode from dva.yml modes section
  --env, -E ENV             Use a named environment from dva.yml environments section
  --tag, -T TAG[,TAG]       Include only lifecycle entries matching any of the given tags
  --exclude-tag TAG[,TAG]   Exclude lifecycle entries matching any of the given tags

Plan-path flags (only when a plan is being run, e.g. 'dva down <plan>'):
  --var KEY=VAL             Override a plan variable. Ignored off the plan path.
  --volumes, -v             Also remove volumes. Rejected off the plan path.
  --purge                   Also remove volumes, locally built images and provision
                            markers. Rejected off the plan path.`,
	Example: `  dva down                            # Tear down default_plan, or the only plan, or the whole stack
  dva down local-dev                  # Tear down a named plan
  dva down local-dev -v               # Also remove volumes
  dva down local-dev --purge --force  # Remove volumes, images and provision markers; skip the prompt`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if helpRequested(args) {
			return cmd.Help()
		}
		c := mustLoadConfig()
		el := rootEnvLoad(c)
		if planName, extraArgs, ok := detectPlanRoute(c, args); ok {
			if isCompositionPlan(c, planName) {
				return runCompositionDown(c, el, planName, extraArgs)
			}
			return runPlanDown(c, el, planName, extraArgs)
		}
		// `dva down --` ≡ `dva down`. TASK-216; the ruling and its cost are at
		// dropFlagTerminator (selectors.go), the placement argument at `up` above.
		//
		// down refuses a surviving terminator through a different door than up — teardownCommon's
		// own dash test, not rejectUnknownFlags — and the fix is deliberately not applied at
		// either door. Consuming the separator once, before the guards, is what stops the table
		// looking arbitrary; a third classifier that knows about `--` is what TASK-216 asked not
		// to add. A token after the terminator still reaches teardownCommon and is still refused
		// there, flag-shaped or name-shaped alike.
		args = dropLeadingTerminator(args)
		if err := requirePlanSelection(c, "down", args); err != nil {
			return err
		}
		if err := rejectSuppressedDefaultPlan(c, "down", args); err != nil {
			return err
		}

		c, e, mode, includeTags, excludeTags, err := teardownCommon(args, "down")
		if err != nil {
			return err
		}
		_, envName, _, _, _, err := parseDvaFlags(args)
		if err != nil {
			return err
		}

		orch := lifecycle.NewOrchestrator(c, e)
		return orch.Down(context.Background(), lifecycle.DownOptions{
			DryRun:      dryRun,
			IncludeTags: includeTags,
			ExcludeTags: excludeTags,
			Mode:        mode,
			Env:         envName,
		})
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop [PLAN] [OPTIONS]",
	Short: "Stop services without removing them (a named plan, or the whole stack)",
	Long: `Stop a named plan without removing its resources.
Without a plan name it uses default_plan, or the only plan when exactly one
is declared. With several plans and no default_plan it refuses and asks you
to name one. With no plans configured it stops every declared stack entry.

A leading -- is a separator, never an argument: "dva stop --" does exactly what a
bare "dva stop" does in that config, and "dva stop -- X" is read exactly as
"dva stop X" -- including when that is a refusal. So a wrapper written as
'dva stop -- "$@"' is safe when "$@" is empty. Only the LEADING one is consumed:
"dva stop -- --" is still an error, and a -- written after something else is left
alone.

Plan usage:
  dva stop <plan>         Stop the selected plan without removing resources
  --var KEY=VAL           Override a plan variable
  --dry-run               Print the variable resolution and the actions, without executing

Whole-stack-path flags (rejected, not ignored, once a plan is named):
  --mode, -M MODE           Use a named mode from dva.yml modes section
  --env, -E ENV             Use a named environment from dva.yml environments section
  --tag, -T TAG[,TAG]       Include only lifecycle entries matching any of the given tags
  --exclude-tag TAG[,TAG]   Exclude lifecycle entries matching any of the given tags

Plan-path flags (only when a plan is being run, e.g. 'dva stop <plan>'):
  --var KEY=VAL             Override a plan variable. Ignored off the plan path.`,
	Example: `  dva stop                      # Stop default_plan, or the only plan, or the whole stack
  dva stop local-dev            # Stop a named plan without removing resources
  dva stop local-dev --dry-run  # Print what would stop, without executing`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if helpRequested(args) {
			return cmd.Help()
		}
		c := mustLoadConfig()
		el := rootEnvLoad(c)
		if planName, extraArgs, ok := detectPlanRoute(c, args); ok {
			if isCompositionPlan(c, planName) {
				return runCompositionStop(c, el, planName, extraArgs)
			}
			return runPlanStop(c, el, planName, extraArgs)
		}
		// `dva stop --` ≡ `dva stop`. TASK-216; same ruling, same placement argument, and the
		// same teardownCommon door as `down` above — stop and down share that helper, but not
		// this line, because each RunE reads the raw args through its own plan guards first.
		args = dropLeadingTerminator(args)
		if err := requirePlanSelection(c, "stop", args); err != nil {
			return err
		}
		if err := rejectSuppressedDefaultPlan(c, "stop", args); err != nil {
			return err
		}

		c, e, mode, includeTags, excludeTags, err := teardownCommon(args, "stop")
		if err != nil {
			return err
		}
		_, envName, _, _, _, err := parseDvaFlags(args)
		if err != nil {
			return err
		}

		orch := lifecycle.NewOrchestrator(c, e)
		return orch.Stop(context.Background(), lifecycle.StopOptions{
			DryRun:      dryRun,
			IncludeTags: includeTags,
			ExcludeTags: excludeTags,
			Mode:        mode,
			Env:         envName,
		})
	},
}

var restartCmd = &cobra.Command{
	Use:   "restart [PLAN | ENTRY...] [OPTIONS]",
	Short: "Restart services (stop + start)",
	Long: `Restart a named plan (stop followed by start).
Without a plan name it uses default_plan, or the only plan when exactly one
is declared. With several plans and no default_plan it refuses and asks you
to name one. With no plans configured it restarts every declared stack entry.

The first argument is read as a plan name when it names a plan, and as a stack
entry name otherwise; the two cannot be combined.

Plan usage:
  dva restart <plan>      Restart the selected plan
  --var KEY=VAL           Override a plan variable (plan only)
  --no-wait               Return without waiting for readiness (plan only)
  --dry-run               Print the variable resolution and the actions, without executing

--var and --no-wait apply only when a plan is being restarted. On the stack path
they are rejected rather than accepted and ignored: it always waits, and there
are no plan variables to override.

Stack usage:
  dva restart <entry>     Restart only the named stack entries (works in any config)
  dva restart -- <name>   Read what follows as names, never as flags

A name matching no declared stack entry is an error. That rule is restart's own,
not a parity with its siblings: restart is the only lifecycle verb taking stack
entry names at all, up reads a positional as a plan name and rejects a real entry
with "plan not found", and down and stop refuse positionals outright. After --
every argument is a name whatever it spells, so a flag written there is reported
as an unknown name rather than silently dropped.

A bare "dva restart --" means "no names given" and does whatever a bare
"dva restart" does in that config, with no exceptions: it restarts every declared
entry where a bare restart does, it runs the default plan where a bare restart
runs it, and it refuses to guess where several plans are configured. Wrapper
scripts should know that "dva restart -- $@" with an empty "$@" restarts
everything in a plan-less config; it was a no-op before TASK-207. A config with a
resolvable default plan — an explicit default_plan, or a lone plan, which counts
as one — was the last exception, and TASK-210 removed it: the terminator is a
separator, so what follows it is classified, never the separator itself.

Whole-stack-path flags (rejected, not ignored, once a plan is named):
  --mode, -M MODE           Use a named mode from dva.yml modes section
  --env, -E ENV             Use a named environment from dva.yml environments section
  --tag, -T TAG[,TAG]       Include only lifecycle entries matching any of the given tags
  --exclude-tag TAG[,TAG]   Exclude lifecycle entries matching any of the given tags`,
	Example: `  dva restart              # Restart default_plan, or the only plan, or the whole stack
  dva restart local-dev    # Restart a named plan
  dva restart api web      # Restart only these stack entries (no plan route)
  dva restart -- --oddname # Everything after -- is read as a name, never a flag`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if helpRequested(args) {
			return cmd.Help()
		}
		c := mustLoadConfig()
		el := rootEnvLoad(c)
		if planName, extraArgs, ok := detectPlanRoute(c, args); ok {
			if isCompositionPlan(c, planName) {
				return runCompositionRestart(c, el, planName, extraArgs)
			}
			return runPlanRestart(c, el, planName, extraArgs)
		}
		// Root-owned route from here: no plan claimed the invocation, so the root's own
		// env-input report governs and this fails closed before any hook or backend child.
		if err := el.report.Err(); err != nil {
			return err
		}
		e := el.env
		if err := requirePlanSelection(c, "restart", args); err != nil {
			return err
		}
		if err := rejectSuppressedDefaultPlan(c, "restart", args); err != nil {
			return err
		}

		mode, envName, includeTags, excludeTags, names, err := parseDvaFlags(args)
		if err != nil {
			return err
		}
		// Everything parseDvaFlags did not recognise arrives here as a service name, and
		// DisableFlagParsing means cobra never vets it. Measured: `dva restart --no-wat`
		// matched no entry, restarted nothing and exited 0, reporting only `[warn] no
		// lifecycle entries matched filters` — the same shape TASK-113 closed for `up`.
		// The other three lifecycle verbs already reject a leftover flag; restart is the
		// only one taking flags AND positional names, so it needs up's allowlist form
		// (:168) rather than teardownCommon's wholesale refusal (:261). TASK-198.
		//
		// The advertised list is stackSelectorFlags alone. --var and --no-wait appear in
		// this command's help under "Plan usage" and runPlanRestart consumes them there.
		// Here they are unknown, and the reason is the path rather than the config: this
		// line is reached whenever no plan was selected, and then Wait is hardcoded true
		// below and there is no plan whose variables --var could override. Naming them in
		// "accepted here" would advertise a flag this path then ignores, which is what
		// rejectUnknownFlags' contract forbids. They are rejected with the rest instead of
		// silently swallowed; whether they should warn and continue, as `dva up` does for
		// --var, is a separate ruling this card does not make.
		//
		// Do NOT restate that as "the config declares no plans at all" — measured false.
		// requirePlanSelection returns nil as soon as planRoutingArgs leaves anything
		// behind, and planRoutingArgs strips only --debug and --json, so a leading FLAG
		// counts as something left behind. With two plans and no default_plan,
		// `dva restart --no-wait` and `dva restart s1 --zzznonsense` both land here with
		// plans configured. The second is the worse half of the defect this guard closes:
		// before it, that invocation restarted s1 and exited 0 having silently discarded
		// the argument, rather than merely doing nothing.
		//
		// The `--` terminator is exempt from the FLAG check, and it is the one place restart
		// must not copy up. parseDvaFlags keeps the terminator deliberately so each caller can
		// rule on it, and up rejects a stray one because it takes no positional names at all.
		// restart does take them, so `dva restart -- s1` is the ordinary way to say s1 is a
		// name and not a flag — measured working before this guard, and rc=1 `unknown flag
		// "--"` with an unconditional check, which the card's "no change to which flags restart
		// accepts" forbids. Only what precedes the terminator is flag-checked.
		guarded := names
		if i := slices.Index(names, "--"); i >= 0 {
			guarded = names[:i]
		}
		if err := rejectUnknownFlags("restart", "a stack entry name", guarded, stackSelectorFlags, []string{"--no-wait", "--var"}); err != nil {
			return err
		}

		// TASK-207 — the unmatchable-name ruling TASK-198 deferred to here, for `--`, `-`, a
		// bad flag after the terminator, and a typo'd entry name alike. 198 left all four
		// exiting 0 on purpose: deciding them inside a flag-rejection card would have settled
		// four cases by accident. They are one class, because after `--` every token is a name
		// whatever it spells, so a single name-shaped check answers all four.
		//
		// Consuming the terminator has to come first, and it is what makes rejecting the rest
		// safe. 198's comment argued the token had to stay in names, reasoning that an empty
		// Names means "every entry" to lifecycle, so dropping it would escalate `dva restart --`
		// from a no-op into a stop+start of the whole stack. The premise is right and the
		// conclusion does not survive the next line: with unknown names rejected, keeping it
		// makes `dva restart --` exit 1 instead — and `dva restart -- "$@"` with an empty "$@"
		// is the exact idiom `--` exists for, whose meaning is "no names given", i.e. exactly
		// what a bare `dva restart` does. Dropping it is what preserves that; the escalation
		// 198 measured is the correct behaviour once the token means "separator" here.
		//
		// Emptying the name list re-opens a gate that has already run. requirePlanSelection is
		// called above against the RAW args, where ["--"] counts as "a token was given", so a
		// config with several plans and no default_plan lets it through — and dropping the token
		// then leaves zero names, which lifecycle reads as "every entry". Measured, not reasoned
		// about: `dva restart --` stopped and started the whole stack in a config whose bare
		// `dva restart` is refused as too ambiguous to act on. That is 198's escalation arriving
		// from the other side, so the gate is re-applied. Found by review; the test above
		// asserted the divergence as correct.
		//
		// It is re-applied on the RAW args, not on the empty name list, and the difference is
		// the whole guard. An empty `names` is not the terminator's signature: `dva restart
		// --tag web` empties it too, and so does every other stack selector. A first draft
		// gated on `len(names) == 0` alone and refused all of them — measured, `restart --tag
		// web` went from rc=0 bouncing s1 on master to rc=1 "multiple plans configured", while
		// `up --tag web` and `stop --tag web` kept working, making restart the one lifecycle
		// verb whose tag filter needs a plan name. `restart --mode dev` stopped reporting the
		// unknown mode and reported the plan gate instead. The raw-args gate above has already
		// ruled on flag-only invocations — any surviving token means "do not ask for a plan" —
		// and that ruling stands. Only an invocation whose every token was the terminator means
		// "no names given", so that is what this asks: strip --debug/--json exactly as the gate
		// itself does, drop the terminator, and require nothing else to be left.
		//
		// The identity this restores is with the STACK route. The PLAN route was decided one
		// card later: a config with a default_plan refused `dva restart --` earlier still, from
		// rejectSuppressedDefaultPlan, because args[0] starts with "-". TASK-210 consumed the
		// leading terminator in detectPlanRoute and in that helper instead, which is why this
		// gate now sees an empty list here rather than being unreachable in that shape.
		//
		// Checked against SortedStack, the DECLARED entries, not the post-filter selection —
		// a name that exists but is excluded by --tag selects nothing legitimately and keeps
		// its warning. See rejectUnknownEntryNames.
		names = dropFlagTerminator(names)
		if len(dropFlagTerminator(planRoutingArgs(args))) == 0 {
			if err := requirePlanSelection(c, "restart", nil); err != nil {
				return err
			}
		}
		declared := make([]string, 0, len(c.Stack))
		for _, entry := range c.SortedStack() {
			declared = append(declared, entry.Name)
		}
		if err := rejectUnknownEntryNames("restart", "stack entry", names, declared); err != nil {
			return err
		}
		mode, isDefault := applyDefaultMode(c, mode)

		if err := applyEnv(e, c, envName); err != nil {
			return err
		}

		rm, err := resolveMode(c, mode)
		if err != nil {
			return err
		}
		if rm.Mode != nil {
			if isDefault {
				fmt.Fprintf(os.Stderr, "[mode: %s (default)]\n", mode)
			} else {
				fmt.Fprintf(os.Stderr, "[mode: %s]\n", mode)
			}
			if len(rm.Mode.Environment) > 0 {
				e.MergeVars(rm.Mode.Environment)
			}
		}

		orch := lifecycle.NewOrchestrator(c, e)
		return orch.Restart(context.Background(), lifecycle.UpOptions{
			DryRun:      dryRun,
			Force:       true,
			Wait:        true,
			Names:       names,
			IncludeTags: includeTags,
			ExcludeTags: excludeTags,
			Mode:        mode,
			Env:         envName,
		})
	},
}
