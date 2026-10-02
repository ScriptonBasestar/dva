package cli

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
	dvaexec "github.com/ScriptonBasestar/dva/internal/exec"
	"github.com/ScriptonBasestar/dva/internal/runner"
)

// detectInteractionCollisionWarnings reports command names that more than one declaration
// produces.
//
// Two different YAML keys can flatten to one command name — a subcommand literally named "b c"
// under `a`, and `a` → `b` → `c` — because the tree joins path segments with a space and
// schema.json admits a space inside a key. The expansion keeps the first and drops the second,
// which used to be decided by Go's map seed and is now decided by the sorted walk. Deterministic
// is not the same as visible, and losing a declared command without a word is the part that had
// to stop. TASK-104.
//
// It asks the tree rather than re-walking c.Interaction here: the paths are the tree's to
// construct, and a second walk that agreed today would drift the first time expansion changed.
func detectInteractionCollisionWarnings(c *config.Config) []string {
	_, collisions := runner.NewInteractionTree(c.Interaction).ListWithCollisions()

	warnings := make([]string, 0, len(collisions))
	for _, col := range collisions {
		winner, loser := describeInteractionPath(col.Winner), describeInteractionPath(col.Loser)
		// Two shapes collide, and only one makes the loser unreachable. When both declarations
		// live under the SAME top-level key (intra-entry), the entry's own expansion drops the
		// loser and Find cannot reach it — "only the first is reachable" is true. When they live
		// under DIFFERENT top-level keys (cross-entry, e.g. interaction.rails.subcommands.console
		// vs interaction."rails console"), each is still reached by invoking its own key (the
		// literal one by quoting it); what the loser lost is the `dva ls` listing, not reachability.
		// Telling an author the cross-entry loser is unreachable sends them to delete a declaration
		// that is still running for every user who types the quoted form (TASK-152).
		if col.Winner[0] != col.Loser[0] {
			warnings = append(warnings, fmt.Sprintf(
				"%s and %s both resolve to the command %q; both still run (each by its own spelling), but only the first is listed in `dva ls` — rename one so both are visible",
				winner, loser, col.Key))
		} else {
			warnings = append(warnings, fmt.Sprintf(
				"%s and %s both resolve to the command %q; only the first is reachable — rename one",
				winner, loser, col.Key))
		}
	}
	return warnings
}

// describeInteractionPath renders a command path as the dva.yml location that declares it, so the
// warning points at a line the author can edit rather than at the flattened name they never wrote.
// A segment is quoted only when it contains whitespace — which is exactly the segment that caused
// the collision, so the quoting doubles as the explanation.
func describeInteractionPath(path []string) string {
	var b strings.Builder
	b.WriteString("interaction")
	for i, seg := range path {
		if i > 0 {
			b.WriteString(".subcommands")
		}
		b.WriteString(".")
		if strings.ContainsAny(seg, " \t\n") {
			b.WriteString(strconv.Quote(seg))
		} else {
			b.WriteString(seg)
		}
	}
	return b.String()
}

// detectUnrunnableComposeCommands reports stack entries whose compose `command:` is
// non-empty but contains no command word: three spaces, a lone tab, or a pair of
// single quotes around nothing.
//
// This cannot live in config.Validate(): that is JSON-schema-only, and to the schema
// "   " is a perfectly good string. It cannot live in the config package at all, since
// internal/exec imports config and the answer has to come from SplitCommand — the very
// function the runners use. That is the point of asking it here rather than
// approximating it with a trimmed-quotes predicate: an approximation would disagree with
// the runners on some input, and two conditions that disagree is the shape of the bug
// this check exists to catch. TASK-115.
func detectUnrunnableComposeCommands(c *config.Config) []string {
	names := make([]string, 0, len(c.Stack))
	for name := range c.Stack {
		names = append(names, name)
	}
	sort.Strings(names)

	var problems []string
	for _, name := range names {
		cc := c.Stack[name].ComposeConfig()
		if cc == nil || cc.Command == "" {
			continue
		}
		if len(dvaexec.SplitCommand(cc.Command)) == 0 {
			problems = append(problems, fmt.Sprintf(
				"stack.%s.runners.compose.command: %q contains no command word", name, cc.Command))
		}
	}
	return problems
}
