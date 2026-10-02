package config

import (
	"fmt"
	"sort"
	"strings"
)

// hasExecutionTarget reports whether this node itself supplies something to execute, ignoring
// anything it would inherit.
//
// `default_args` counts because it executes on its own, which is not obvious from the name.
// exec.buildCommandLine appends the args to the command and, in shell mode — the default —
// hands the joined string to `sh -c`, so an empty command with args leaves the args as the
// whole shell line. Measured: a node with only `default_args: "echo reached"` and nothing else
// prints `reached` and exits 0. Omitting it here made the predicate disagree with the runtime
// in the one direction that produces a false warning — telling an author a node cannot run
// while it runs. TASK-165.
func (c *InteractionCommand) hasExecutionTarget() bool {
	return c.Command != "" || len(c.CommandLines) > 0 || c.HasScript() ||
		c.ScriptFile != "" || c.HasSteps() || c.HasHooks() || c.Compose != nil ||
		c.Runner != "" || c.Service != "" || c.Pod != "" || c.DefaultArgs != ""
}

// warnDuplicateParentSubcommand warns when an interaction command has the same command value
// as one of its subcommands, at any depth.
//
// Results are sorted because both this tree and each node's subcommands are maps: without it
// the same dva.yml prints its warnings in a different order on consecutive runs, which is the
// defect TASK-107 closed for command suggestions.
func (c *Config) warnDuplicateParentSubcommand() []string {
	var warnings []string

	eachInteractionNode(c.Interaction, func(path string, cmd *InteractionCommand, _ inheritedExec) {
		if cmd.Command == "" {
			return
		}
		for subName, sub := range cmd.Subcommands {
			if sub.Command == cmd.Command {
				warnings = append(warnings,
					fmt.Sprintf("%s.subcommands.%s: command %q is identical to parent; subcommand is redundant",
						path, subName, cmd.Command))
			}
		}
	})

	sort.Strings(warnings)
	return warnings
}

// warnChildOverridesParentCritical warns when a child overrides its parent's runner or pod,
// potentially altering the backend unexpectedly.
//
// Severity: Semantic Warning
func (c *Config) warnChildOverridesParentCritical() []string {
	var warnings []string

	eachInteractionNode(c.Interaction, func(path string, cmd *InteractionCommand, inherited inheritedExec) {
		// The effective values, not the raw ones. A middle node that sets no runner still
		// hands its parent's down, so comparing against cmd.Runner made the warning fire
		// only when an author happened to restate a value they would have inherited anyway
		// — silent on the identical config written without the redundant line. TASK-128.
		parentRunner := firstNonEmptyStr(cmd.Runner, inherited.runner)
		parentPod := firstNonEmptyStr(cmd.Pod, inherited.pod)

		for subName, sub := range cmd.Subcommands {
			if parentRunner != "" && sub.Runner != "" && parentRunner != sub.Runner {
				warnings = append(warnings,
					fmt.Sprintf("%s.subcommands.%s: overrides parent runner (%s → %s); this may change execution backend unexpectedly",
						path, subName, parentRunner, sub.Runner))
			}
			if parentPod != "" && sub.Pod != "" && parentPod != sub.Pod {
				warnings = append(warnings,
					fmt.Sprintf("%s.subcommands.%s: overrides parent pod (%s → %s); this may change execution backend unexpectedly",
						path, subName, parentPod, sub.Pod))
			}
		}
	})

	sort.Strings(warnings)
	return warnings
}

const MaxSubcommandDepth = 5

// warnDeepSubcommandNesting warns when nested subcommands exceed a specific depth,
// signifying overly complex DSL structure.
//
// Severity: Semantic Warning
func (c *Config) warnDeepSubcommandNesting() []string {
	var warnings []string

	for name, cmd := range c.Interaction {
		depth := calculateSubcommandDepth(cmd, 0)
		if depth > MaxSubcommandDepth {
			warnings = append(warnings,
				fmt.Sprintf("interaction.%s: nested %d levels deep (max %d); consider flattening the command structure",
					name, depth, MaxSubcommandDepth))
		}
	}

	// c.Interaction is a map, so the order here is whatever Go's randomized range hands back.
	// TestFlatMapWarningsAreOrderStable diverges on the first repeat with this removed. TASK-128.
	sort.Strings(warnings)
	return warnings
}

func calculateSubcommandDepth(cmd *InteractionCommand, current int) int {
	if len(cmd.Subcommands) == 0 {
		return current
	}

	maxDepth := current + 1
	for _, sub := range cmd.Subcommands {
		depth := calculateSubcommandDepth(sub, current+1)
		if depth > maxDepth {
			maxDepth = depth
		}
	}

	return maxDepth
}

// warnUnreachableCommands warns when an interaction node supplies no execution context
// (e.g. no command, no service, no compose) and inherits none, in either of the two shapes
// that produces: a parent that cannot be called directly, and a leaf that cannot be called
// at all.
//
// The leaf is the worse of the two and used to be the one nobody reported. Until TASK-165
// this check returned early on `len(cmd.Subcommands) == 0`, so it fired only where there was
// still something to route to. On the fixture below, `grp` — which at least reaches its
// children — got the warning, and `grp.leaf`, the node that can never do anything, got
// nothing from the validator and exit 0 from the runtime:
//
//	interaction:
//	  grp:
//	    subcommands:
//	      leaf: {description: does nothing at all}
//
// Severity: Semantic Warning
func (c *Config) warnUnreachableCommands() []string {
	var warnings []string

	eachInteractionNode(c.Interaction, func(path string, cmd *InteractionCommand, inherited inheritedExec) {
		// Inheritance is the whole point of asking both: examples/full-stack.yml's
		// `rails db` sets nothing itself, yet `dva run rails db` executes
		// `bundle exec rails` inherited from `rails`. Testing the raw node alone reported
		// that shipped, working config as unreachable. TASK-128.
		if cmd.hasExecutionTarget() || inherited.callable {
			return
		}
		if len(cmd.Subcommands) > 0 {
			warnings = append(warnings,
				fmt.Sprintf("%s: has subcommands but is not directly callable; add an execution target or remove subcommands",
					path))
			return
		}
		warnings = append(warnings,
			fmt.Sprintf("%s: has no execution target and no subcommands, so running it does nothing; add a command, script, steps or service — or remove the entry",
				path))
	})

	sort.Strings(warnings)
	return warnings
}

// hasUnsupportedBracedOperator reports whether v contains a `${NAME<op>...}` form whose
// operator the expander does not handle, i.e. anything after the name other than `}`,
// `:-` or `-`.
func hasUnsupportedBracedOperator(v string) bool {
	for i := 0; i+1 < len(v); i++ {
		if v[i] != '$' || v[i+1] != '{' {
			continue
		}
		name := scanVarName(v[i+2:])
		rest := v[i+2+len(name):]
		if name == "" || rest == "" {
			continue
		}
		if rest[0] == '}' || rest[0] == '-' || strings.HasPrefix(rest, ":-") {
			continue
		}
		return true
	}
	return false
}
