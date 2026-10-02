package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
)

var composeCmd = &cobra.Command{
	Use:   "compose [ENTRY] [ARGS...]",
	Short: "Execute raw Docker Compose commands",
	Long: `Execute raw Docker Compose commands against a stack entry.

This is a low-level debugging escape hatch. Use 'dva up <plan>' for normal,
validated lifecycle execution.

If only one compose entry exists, the entry name can be omitted.
If multiple compose entries exist, the first argument must be the entry name.`,
	Example: `  dva compose ps                    # Single compose entry
  dva compose main-db ps            # Multiple entries: specify name
  dva compose main-db logs -f api   # Passthrough with entry name`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if helpRequested(args) {
			return cmd.Help()
		}
		c := mustLoadConfig()
		e, envReport := loadEnv(c)
		if err := envReport.Err(); err != nil {
			return err
		}

		// Same leak as `dva stack log` (TASK-092), and worse-positioned: these args are
		// appended before the compose subcommand, so `dva --debug --json compose logs`
		// produced `docker compose -f … --debug --json logs`, offering both to `docker
		// compose` itself rather than to `logs`.
		var err error
		if args, err = consumeRootPersistentFlags(args); err != nil {
			return err
		}

		composeEntries := c.ComposeEntries()
		if len(composeEntries) == 0 {
			return fmt.Errorf("no compose entries in stack")
		}

		if len(composeEntries) == 1 {
			// Single entry: name can be omitted. Consume it when callers use the same
			// explicit form as the multiple-entry path, while preserving all other args.
			entryArgs := args
			if len(entryArgs) > 0 && entryArgs[0] == composeEntries[0].Name {
				entryArgs = entryArgs[1:]
			}
			return execComposePassthroughForEntry(e, c, composeEntries[0], nil /* no stack profiles */, entryArgs)
		}

		// Multiple entries: first arg must be entry name
		if len(args) > 0 {
			if entry := c.FindStackEntry(args[0]); entry != nil && entry.ComposeConfig() != nil {
				return execComposePassthroughForEntry(e, c, entry, nil /* no stack profiles */, args[1:])
			}
		}

		// No valid entry name provided — show available entries
		var names []string
		for _, entry := range composeEntries {
			names = append(names, entry.Name)
		}
		return fmt.Errorf("multiple compose entries: %s\nSpecify one: dva compose <name> [args...]",
			strings.Join(names, ", "))
	},
}

var upCmd = &cobra.Command{
	Use:   "up [PLAN] [OPTIONS]",
	Short: "Start services (a named plan, or the whole stack)",
	Long: `Start a named plan when plans are configured.
Without a plan name it uses default_plan, or the only plan when exactly one
is declared. With several plans and no default_plan it refuses and asks you
to name one. With no plans configured it starts every declared stack entry.

A leading -- is a separator, never an argument: "dva up --" does exactly what a
bare "dva up" does in that config, and "dva up -- X" is read exactly as
"dva up X" -- including when that is a refusal. So a wrapper written as
'dva up -- "$@"' is safe when "$@" is empty. Only the LEADING one is consumed:
"dva up -- --" is still an error, and a -- written after something else is left
alone.

Plan usage:
  dva up <plan>           Start the selected plan
  --force                 Compose only: pass --force-recreate (other plugins ignore)
  --no-wait               Return without waiting for readiness
  --var KEY=VAL           Override a plan variable
  --dry-run               Print the variable resolution and the actions, without executing

Stack flags:
  --force                   Compose only: pass --force-recreate (other plugins ignore)
  --no-wait                 Start services and return immediately without waiting

Whole-stack-path flags (rejected, not ignored, once a plan is named):
  --mode, -M MODE           Use a named mode from dva.yml modes section
  --env, -E ENV             Use a named environment from dva.yml environments section
  --tag, -T TAG[,TAG]       Include only lifecycle entries matching any of the given tags
  --exclude-tag TAG[,TAG]   Exclude lifecycle entries matching any of the given tags

Plan-path flags (only when a plan is being run, e.g. 'dva up <plan>'):
  --var KEY=VAL             Override a plan variable. Ignored off the plan path.`,
	Example: `  dva up                            # Start default_plan, or the only plan, or the whole stack
  dva up local-dev                  # Start a named plan
  dva up local-dev --force          # Force-recreate a named plan
  dva up local-dev --var PORT=8080  # Override a plan variable
  dva up --dry-run                  # Print what would run without executing`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if helpRequested(args) {
			return cmd.Help()
		}
		c := mustLoadConfig()
		el := rootEnvLoad(c)
		if planName, extraArgs, ok := detectPlanRoute(c, args); ok {
			if isCompositionPlan(c, planName) {
				return runCompositionUp(c, el, planName, extraArgs)
			}
			return runPlanUp(c, el, planName, extraArgs)
		}
		// Root-owned route from here: no plan claimed the invocation, so the root's own
		// env-input report governs and this fails closed before any hook or backend child.
		if err := el.report.Err(); err != nil {
			return err
		}
		e := el.env
		// A LEADING `--` is a separator: `dva up --` ≡ `dva up`, and `dva up -- X` ≡ `dva up X`.
		// TASK-216 widened TASK-207's restart-local identity to up/down/stop; the argument, and
		// what keeping it restart-local cost, is written at dropFlagTerminator (selectors.go).
		//
		// "Leading" and not "the first one anywhere", which is load-bearing in both directions.
		// `dva up --debug --` keeps its terminator and is still refused — that one is not at
		// args[0]. And the identity is false at X = `--`: `dva up -- --` is `unknown flag "--"`
		// while `dva up --` starts the stack, because exactly one is consumed and `up` takes no
		// arguments. That is not an exception to the rule; it is what a separator means, and
		// `restart` behaves the same way (`dva restart -- -- s1` refuses).
		//
		// It has to go HERE — above requirePlanSelection — rather than beside the flag guard
		// that used to refuse the token. detectPlanRoute returns ok=false for "no plans" before
		// it reaches its own dropLeadingTerminator, and for "several plans, none named" as well,
		// so the terminator survives into both, and requirePlanSelection is the first guard to
		// read it: it counts one surviving token as "the user named something" and lets the
		// several-plans config through, where a bare `dva up` is refused. Dropping any later
		// leaves that row diverging.
		//
		// What the guards see is unchanged: `dva up -- X` reaches rejectSuppressedDefaultPlan,
		// rejectUpPositionalArg and rejectUnknownFlags with exactly the args `dva up X` reaches
		// them with. What that does NOT mean is "only the empty case moved" — an earlier draft
		// of this comment said so and the review refuted it. `dva up -- -` went rc=1
		// (`unknown flag "--"`) to rc=0 starting the whole stack, and so did `dva up -- --debug`,
		// because `dva up -` and `dva up --debug` were ALREADY rc=0: rejectUnknownFlags is
		// reached only after parseDvaFlags has consumed the token. So the identity is applied
		// faithfully and it inherits whatever `dva up X` does, including where that is wrong.
		// `dva up -` accepted a bare dash when this was written, so the identity inherited that
		// too. TASK-218 has since settled it the other way — isFlagToken reads a lone `-` as a
		// name and rejectUpPositionalArg reports it — and both spellings now refuse together.
		// The identity held across that change without this line being touched, which is what
		// it is for. down/stop reach `-` by another route: teardownCommon still refuses it as an
		// unknown FLAG, wording TASK-218 deliberately left alone.
		args = dropLeadingTerminator(args)
		if err := requirePlanSelection(c, "up", args); err != nil {
			return err
		}
		if err := rejectSuppressedDefaultPlan(c, "up", args); err != nil {
			return err
		}
		if err := rejectUpPositionalArg(c, args); err != nil {
			return err
		}

		mode, envName, includeTags, excludeTags, args, err := parseDvaFlags(args)
		if err != nil {
			return err
		}
		mode, isDefault := applyDefaultMode(c, mode)

		force := false
		noWait := false
		var leftover []string
		for i := 0; i < len(args); i++ {
			a := args[i]
			switch {
			case a == "--force":
				force = true
			case a == "--no-wait":
				noWait = true

			// --var belongs to the plan path (runPlanUp consumes it before this loop is
			// reached) and upCmd.Long documents it as "Ignored off the plan path". So it is a
			// known flag here, not an unknown one, and the guard below must not reject it —
			// TestUpWithoutPlansGuardOnlyInspectsPlanNameSlot pins that. Its VALUE has to be
			// consumed with it or `--var FOO=bare` would leave FOO=bare behind, which carries
			// no leading dash and would then be rejected as a stray positional argument.
			//
			// Ignoring it silently is the same shape as the defect this task is about, so it
			// says so on stderr. The exit code and the documented semantics are unchanged.
			case a == "--var":
				if i+1 < len(args) {
					i++
				}
				fmt.Fprintln(os.Stderr, "[warn] --var applies only when running a plan ('dva up <plan>'); ignored here")
			case strings.HasPrefix(a, "--var="):
				fmt.Fprintln(os.Stderr, "[warn] --var applies only when running a plan ('dva up <plan>'); ignored here")

			default:
				leftover = append(leftover, a)
			}
		}
		// Before TASK-113 this switch had no default and nothing followed it, so every
		// unrecognised token was discarded: `dva up --force=true` and `dva up --forse` both
		// ran as if no flag had been given and exited 0.
		//
		// Two guards, because `dva up` accepts no positional names either. rejectUpPositionalArg
		// already ran above, but only on args[0] — with a flag in front of it, as in
		// `dva up --force nosuchthing`, it returns nil at its leading-dash check and the name
		// reached here to be dropped. Measured: exit 0, the whole stack started, the argument
		// silently gone. (The example read `--dev` until that flag was removed with
		// `applications:` — it is an accepted flag that has to lead, and --force is the
		// nearest surviving one.)
		if err := rejectUnknownFlags("up", "", leftover, withSelectors([]string{"--force", "--no-wait", "--var"}, stackSelectorFlags), nil); err != nil {
			return err
		}
		if err := rejectUpPositionalArg(c, leftover); err != nil {
			return err
		}

		if err := applyEnv(e, c, envName); err != nil {
			return err
		}

		rm, err := resolveMode(c, mode)
		if err != nil {
			return err
		}
		if rm.Mode != nil {
			if isDefault {
				fmt.Fprintf(os.Stderr, "[mode: %s (default)] %s\n", mode, rm.Mode.Description)
			} else {
				fmt.Fprintf(os.Stderr, "[mode: %s] %s\n", mode, rm.Mode.Description)
			}
			if len(rm.Mode.Environment) > 0 {
				e.MergeVars(rm.Mode.Environment)
			}
			if rm.Mode.Provision != "" {
				suggestProvision(c, rm.Mode.Provision)
			}
		}

		// Phase 1: Stack up (infrastructure)
		orch := lifecycle.NewOrchestrator(c, e)
		upErr := orch.Up(context.Background(), lifecycle.UpOptions{
			DryRun:      dryRun,
			Force:       force,
			Wait:        !noWait,
			IncludeTags: includeTags,
			ExcludeTags: excludeTags,
			Mode:        mode,
			Env:         envName,
		})

		// Print status summary and endpoints regardless of up errors,
		// so users can see connection info for services that did start.
		fmt.Fprintln(os.Stderr)
		status, statusErr := orch.Status(context.Background())
		if statusErr == nil {
			lifecycle.PrintStatus(status, c.FileDir())
		}
		if len(c.Endpoints) > 0 {
			allHC := checkEndpointHealth(c.Endpoints)
			var epTags []string
			if rm.Mode != nil {
				epTags = rm.Mode.EndpointTags
			}
			printEndpointTable(c.Endpoints, epTags, allHC)
		}

		// Returned here rather than at the call site so the status and endpoint tables
		// above still print first. A user whose stack half-failed wants the connection
		// details of what did come up.
		//
		// This was errors.Join(upErr, appErr) until `applications:` was removed. The app
		// half was the reason the join existed: appErr had been swallowed into
		// "[warn] app start: %v", so TASK-117's readiness fix inside StartApps could not
		// reach the exit code and `dva up && next-step` chained on a failed start. Only
		// upErr remains, and orch.Up already returns it unswallowed.
		return upErr
	},
}

// teardownCommon resolves mode, applies env, and returns the parsed flags
// for both down and stop commands. verb is "down" or "stop" for error messages.
func teardownCommon(args []string, verb string) (*config.Config, *config.Environment, string, []string, []string, error) {
	c := mustLoadConfig()
	e, envReport := loadEnv(c)

	// Teardown fails closed exactly like startup. Tearing down with the wrong
	// environment resolves the wrong resource identity, and cleaning up someone
	// else's resources is worse than refusing to clean up at all (TASK-247 §4).
	// This precedes every hook, marker removal and backend child below.
	if err := envReport.Err(); err != nil {
		return nil, nil, "", nil, nil, err
	}

	mode, envName, includeTags, excludeTags, remaining, err := parseDvaFlags(args)
	if err != nil {
		return nil, nil, "", nil, nil, err
	}
	mode, isDefault := applyDefaultMode(c, mode)

	if len(remaining) > 0 {
		// A leftover that starts with "-" is a flag, not a service name, and quoting it into
		// the suggestion produced advice that cannot work: `dva down --bogus` used to answer
		// "Use 'dva stack down --bogus'", which failed the same way for the same reason. Only
		// the name-shaped case gets the selective-teardown hint. TASK-172.
		//
		// The hint's destination changed with `dva stack`: selective teardown is naming a plan,
		// which detectPlanRoute would already have taken if the argument were one. Reaching
		// here means it is not, so the message says what to name rather than quoting the word
		// back — a plan name is not derivable from a service name.
		if strings.HasPrefix(remaining[0], "-") {
			return nil, nil, "", nil, nil, fmt.Errorf("unknown flag %q for \"dva %s\"\n       → 'dva %s' takes no service names or flags of its own; it %ss everything declared",
				remaining[0], verb, verb, verb)
		}
		return nil, nil, "", nil, nil, fmt.Errorf("'dva %s' %ss all declared entries. Name a plan instead — 'dva %s <plan>' %ss just that plan's entries, and 'dva ls' lists the plans this config declares",
			verb, verb, verb, verb)
	}

	if err := applyEnv(e, c, envName); err != nil {
		return nil, nil, "", nil, nil, err
	}

	rm, err := resolveMode(c, mode)
	if err != nil {
		return nil, nil, "", nil, nil, err
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

	return c, e, mode, includeTags, excludeTags, nil
}
