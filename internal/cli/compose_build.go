package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
)

var buildCmd = &cobra.Command{
	Use:   "build [PLAN] [ENTRY] [OPTIONS] [SERVICE...]",
	Short: "Build a plan's entries (or compose services)",
	Long: `Build what a named plan runs, one of its entries, or compose services.

Plan usage:
  dva build <plan>           Build every entry of the plan that has something to build
  dva build <plan> <entry>   Build one entry of the plan

Compose entries run 'docker compose build'; native entries run their
'runners.native.build' command in the entry's directory, with the entry's
variables. Entries with neither are skipped, not reported as failures.

Everything after the plan and entry names is passed to whatever does the
building — '--no-cache', '--pull' and service names reach docker compose
unchanged. A native build command is run as written and takes no extra
arguments. With more than one entry to build there is nothing to pass them to,
so name the entry first.

Without plans, or when the first argument is not a plan name, this stays a
compose passthrough: 'dva build api' still means the 'api' service.`,
	Example: `  dva build                # Build default_plan, or the only plan, or every declared entry
  dva build local-dev      # Build every entry of a named plan that has something to build
  dva build local-dev api  # Build one entry of a plan
  dva build api --no-cache # Compose passthrough: build the api service`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if helpRequested(args) {
			return cmd.Help()
		}
		if err := rejectRemovedModeFlags(args); err != nil {
			return err
		}
		c := mustLoadConfig()
		el := rootEnvLoad(c)

		// remaining is docker's argv from here on — `dva build --no-cache` has to reach
		// docker, so nothing downstream can tell a malformed DVA flag from a valid docker
		// one. parseDvaFlags is the last code that can, and now does. TASK-172.
		envName, includeTags, excludeTags, remaining, err := parseDvaFlags(args)
		if err != nil {
			return err
		}
		// envName/includeTags/excludeTags used to go to _, _, _ here: parseDvaFlags still
		// consumed the three selectors off argv (that is why none of them ever leaked into
		// docker's), but nothing bound the values, so `dva build --exclude-tag app` and
		// `dva build --env prod` against a config with no environments: both answered exit 0
		// having done nothing with what they parsed — silence in both directions where the
		// stack path's `dva up --env prod` fails with "env 'prod' not found" (TASK-279 §3).
		// build's compose shortcut has no notion of an environment or a tag filter
		// to apply them to, so binding the values only to fail on them (rather than pretending
		// to honour a filter this route cannot act on) is the fix: the flag is rejected here
		// instead of being silently absorbed, matching how the plan path already answers an
		// unsupported plan flag.
		if bad := unsupportedBuildSelectors(envName, includeTags, excludeTags); bad != "" {
			return fmt.Errorf("'dva build' does not support %s; docker's own flags and service names still reach it unchanged", bad)
		}
		// Consume build's own leading separator before plan detection. The shared helper below
		// deliberately consumes only the plan-name slot; a second terminator remains backend
		// argv, so `build -- -- --` still reaches compose as `build -- --`.
		remaining = dropLeadingTerminator(remaining)

		// Plan routing reads what parseDvaFlags left, so --dry-run is already
		// claimed and the plan-name slot holds a name or a tool's flag. logsCmd calls
		// consumeRootPersistentFlags at this point instead; here parseDvaFlags has done that
		// job and more, and calling both would walk the same argv twice.
		if planName, extraArgs, ok := detectPlanRoute(c, remaining); ok {
			if isCompositionPlan(c, planName) {
				return runCompositionBuild(c, el, planName, extraArgs)
			}
			return runPlanBuild(c, el, planName, extraArgs)
		}
		// Root-owned route from here: no plan claimed the invocation, so the root's own
		// env-input report governs and this fails closed before any hook or backend child.
		if err := el.report.Err(); err != nil {
			return err
		}
		e := el.env
		if err := requirePlanSelection(c, "build", remaining); err != nil {
			return err
		}
		if err := rejectSuppressedDefaultPlan(c, "build", remaining); err != nil {
			return err
		}
		// No rejectUnknownPlanArg, as in logs: `dva build api` naming a compose service
		// predates plans and is still what most of that argument's uses mean.

		return execComposePassthrough(e, c, append([]string{"build"}, remaining...))
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs [PLAN] [ENTRY] [OPTIONS] [SERVICE...]",
	Short: "View output from a plan's entries (or from compose services)",
	Long: `Show logs for a named plan, one of its entries, or compose services.

Plan usage:
  dva logs <plan>           Logs for the plan's only log-producing entry
  dva logs <plan> <entry>   Logs for one entry of the plan

Everything after the plan and entry names is passed to whatever owns the logs —
'-f', '--tail 50' and service names reach docker compose unchanged. Entries that
run as a process or a script are read from their log file instead, which cannot
follow, so those take no extra arguments.

Without plans, or when the first argument is not a plan name, this stays a
compose passthrough: 'dva logs api' still means the 'api' service.`,
	Example: `  dva logs                # Logs for default_plan, or the only plan, or the whole stack
  dva logs local-dev      # Logs for a named plan's log-producing entry
  dva logs local-dev api  # Logs for one entry of a plan
  dva logs api -f         # Compose passthrough: follow the api service's logs`,
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if helpRequested(args) {
			return cmd.Help()
		}
		c := mustLoadConfig()
		el := rootEnvLoad(c)
		// TASK-092, third site: `dva --debug logs` sent --debug on to docker as a flag of
		// `compose logs`.
		args, err := consumeRootPersistentFlags(args)
		if err != nil {
			return err
		}
		if planName, extraArgs, ok := detectPlanRoute(c, args); ok {
			if isCompositionPlan(c, planName) {
				return runCompositionLogs(c, el, planName, extraArgs)
			}
			return runPlanLogs(c, el, planName, extraArgs)
		}
		// Observation, not execution: nothing reaches stdout and no compose child or
		// log file is read. The diagnostic target is the literal word `stack` whatever
		// trailing argv follows, because DVA does not guess a service name out of raw
		// passthrough argv (TASK-247 §5).
		if el.report.Incomplete() {
			return fmt.Errorf("logs not queried for stack: environment inputs are incomplete")
		}
		e := el.env
		if err := requirePlanSelection(c, "logs", args); err != nil {
			return err
		}
		if err := rejectSuppressedDefaultPlan(c, "logs", args); err != nil {
			return err
		}
		// No rejectUnknownPlanArg here, unlike up/down/stop/restart. Their positional slot
		// means a plan and nothing else, so an unmatched name is a typo. This one has a
		// second legitimate occupant — `dva logs api` naming a compose service predates
		// plans and still works — and rejecting it would break that to catch a misspelling.
		return execComposePassthrough(e, c, append([]string{config.LogsDirName}, args...))
	},
}

// unsupportedBuildSelectors names which of --env/--tag/--exclude-tag buildCmd was given, in a
// stable order, for the error that stops it discarding them at the parse call (TASK-279 §3).
// An empty result means no unsupported selector was supplied.
func unsupportedBuildSelectors(envName string, includeTags, excludeTags []string) string {
	var bad []string
	if envName != "" {
		bad = append(bad, "--env")
	}
	if len(includeTags) > 0 {
		bad = append(bad, "--tag")
	}
	if len(excludeTags) > 0 {
		bad = append(bad, "--exclude-tag")
	}
	return strings.Join(bad, ", ")
}
