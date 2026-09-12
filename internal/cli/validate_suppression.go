package cli

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/runner"
)

// This file implements docs/56 §6's single load-bearing rule: an ignored finding is a
// declaration, never silence. Every suppression funnels through suppressionSummary, whose
// count always reaches the summary line and whose entries `--show-ignored` can expand, and
// every ignore pattern that hides nothing comes back as a stale-pattern warning. The
// ignore surfaces themselves (`suggestion_ignore`, `suggestions:`, `drift_ignore`) are only
// safe because that accounting cannot be turned off (§6-4).

// Suppression kinds. They are also the words the summary line uses, so a new kind changes
// one place, not two.
const (
	suppressedSuggestion = "suggestion"
	suppressedDrift      = "drift file"
)

type suppressedItem struct {
	kind string
	name string
	// reason names the dva.yml surface that hid this item, spelled the way it appears in
	// the file ("drift_ignore: compose.ci.yaml", "suggestions.makefile: false"), so
	// --show-ignored points at the line to edit rather than at a concept.
	reason string
}

type suppressionSummary struct {
	items []suppressedItem
	stale []string
	// suggested holds the candidate names that survived every filter and produced a
	// warning. --suggest-ignore renders exactly this list, so the block it prints can
	// never name something the run did not actually warn about.
	suggested []string
}

func (s *suppressionSummary) add(kind, name, reason string) {
	s.items = append(s.items, suppressedItem{kind: kind, name: name, reason: reason})
}

func (s *suppressionSummary) merge(other *suppressionSummary) {
	if other == nil {
		return
	}
	s.items = append(s.items, other.items...)
	s.stale = append(s.stale, other.stale...)
	s.suggested = append(s.suggested, other.suggested...)
}

func (s *suppressionSummary) count(kind string) int {
	n := 0
	for _, item := range s.items {
		if item.kind == kind {
			n++
		}
	}
	return n
}

// summarySuffix is what gets appended to "✅ dva.yml is valid". It is empty when nothing
// was suppressed, so a config with no ignore lists reads exactly as it did before.
func (s *suppressionSummary) summarySuffix() string {
	var parts []string
	for _, kind := range []string{suppressedSuggestion, suppressedDrift} {
		if n := s.count(kind); n > 0 {
			parts = append(parts, pluralizeCount(n, kind))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf(" (%s ignored by dva.yml)", strings.Join(parts, ", "))
}

func pluralizeCount(n int, noun string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, noun)
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

// sortedItems orders the items so two runs over the same config produce byte-identical
// output (TASK-128), for both the text listing and the JSON document.
func (s *suppressionSummary) sortedItems() []suppressedItem {
	sorted := make([]suppressedItem, len(s.items))
	copy(sorted, s.items)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].kind != sorted[j].kind {
			return sorted[i].kind < sorted[j].kind
		}
		if sorted[i].name != sorted[j].name {
			return sorted[i].name < sorted[j].name
		}
		return sorted[i].reason < sorted[j].reason
	})
	return sorted
}

func (s *suppressionSummary) showIgnoredLines() []string {
	sorted := s.sortedItems()
	lines := make([]string, 0, len(sorted))
	for _, item := range sorted {
		lines = append(lines, fmt.Sprintf("%s %q — %s", item.kind, item.name, item.reason))
	}
	return lines
}

func printSuppressedItems(w io.Writer, summary *suppressionSummary) {
	for _, line := range summary.showIgnoredLines() {
		_, _ = fmt.Fprintf(w, "[info] ignored: %s\n", line)
	}
}

func printStaleIgnoreWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(w, "[warn] stale ignore: %s\n", warning)
	}
}

// matchIgnorePattern returns the first pattern that matches name, and whether one did.
// Returning the pattern rather than a bool is what lets --show-ignored name the line
// responsible; a bool would leave the operator grepping a twenty-entry list by hand.
func matchIgnorePattern(name string, patterns []string) (string, bool) {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return pattern, true
		}
	}
	return "", false
}

// staleIgnoreWarnings reports patterns that match nothing in universe.
//
// A pattern that hides nothing is not merely dead weight: it is a claim to the next reader
// that this project has a target or compose file that in fact no longer exists (docs/56
// §6-5). universe must be the FULL candidate set before any other filter — including the
// suggestion rules' own built-in exclusions — or narrowing those rules would silently
// convert correct ignore entries into stale ones.
func staleIgnoreWarnings(key string, patterns, universe []string, nothing string) []string {
	var warnings []string
	for i, pattern := range patterns {
		matched := false
		for _, name := range universe {
			if ok, _ := filepath.Match(pattern, name); ok {
				matched = true
				break
			}
		}
		if !matched {
			warnings = append(warnings, fmt.Sprintf("%s[%d] %q matches no %s; remove it", key, i, pattern, nothing))
		}
	}
	return warnings
}

// suggestionSourceKind maps the file a candidate name was read from to its `suggestions:`
// key. The mapping lives here rather than at the extraction sites so the schema, the
// opt-out check, and the --show-ignored reason can never disagree about the spelling.
func suggestionSourceKind(source string) string {
	switch source {
	case "Makefile":
		return config.SuggestionSourceMakefile
	case "package.json":
		return config.SuggestionSourcePackageJSON
	}
	return ""
}

// wrappedWorkflowTargets returns the Makefile targets and package.json scripts that some
// interaction already runs.
//
// This is docs/56 §3 rule C-1, and it is the largest single source of the dogfood ignore
// lists: an interaction declared as `test: {command: make test}` is exactly the wrapping
// the suggestion asks for, and suggesting it again taught users that the suggestion channel
// is noise. Coverage by name (commandSet/subcommandCoverage) cannot see this case, because
// the interaction is free to be called something else entirely.
func wrappedWorkflowTargets(c *config.Config) map[string]bool {
	wrapped := map[string]bool{}
	for _, cmd := range runner.NewInteractionTree(c.Interaction).List() {
		// An imported command reached only through its canonical "<subproject>/<name>"
		// address wraps the target inside that subproject, but `dva <target>` still runs
		// nothing at the root — which is the whole point of TASK-352's alias hint. Counting
		// it as coverage here would delete that hint and leave the root route missing with
		// no warning anywhere. An `as:` alias produces a second, slash-free path head and
		// is counted.
		if cmd.SubprojectName != "" && len(cmd.Path) > 0 && strings.Contains(cmd.Path[0], "/") {
			continue
		}
		sources := []string{cmd.Command, cmd.Script}
		sources = append(sources, cmd.CommandLines...)
		for i := range cmd.Steps {
			sources = append(sources, cmd.Steps[i].RunCommands()...)
			sources = append(sources, cmd.Steps[i].Cmd)
		}
		for _, source := range sources {
			collectWrappedTargets(source, wrapped)
		}
	}
	return wrapped
}

// runnerTakingATargetName lists the tools whose first non-flag argument is the name this
// file compares against Makefile targets and package.json scripts.
var runnerTakingATargetName = map[string]bool{
	"make": true,
	// pnpm/npm/yarn are here even though docs/56 §3 names only `pnpm <script>`: the key
	// the decision turns on is "the script is already wrapped", and which package manager
	// spelled it is not part of that. Leaving the other two out would suppress the
	// suggestion for a pnpm project and keep emitting it, unfixably, for an npm one.
	"pnpm": true,
	"npm":  true,
	"yarn": true,
}

// makeFlagsTakingAValue are the flags whose value is a separate argument, which therefore
// must not be read as the target name.
var makeFlagsTakingAValue = map[string]bool{
	"-C": true, "--directory": true,
	"-f": true, "--file": true, "--makefile": true,
	"-o": true, "--old-file": true,
	"-W": true, "--what-if": true,
	"--prefix": true, "--filter": true, "-w": true,
}

// changesDirectory marks the flags that make this invocation talk to a different project's
// Makefile or workspace. A target found after one of those is not the root target the
// suggestion is about, so the invocation contributes no coverage at all rather than the
// wrong coverage.
func changesDirectory(flag string) bool {
	switch flag {
	case "-C", "--directory", "-f", "--file", "--makefile", "--prefix", "--filter", "-w":
		return true
	}
	return false
}

func collectWrappedTargets(command string, into map[string]bool) {
	if command == "" {
		return
	}
	// Shell operators are split on rather than parsed: this only needs to find the word
	// after a tool name, and a full parse would be a second, divergent implementation of
	// what the runners already do at execution time.
	normalized := strings.NewReplacer("&&", " ", "||", " ", ";", " ", "|", " ", "\n", " ", "(", " ", ")", " ").Replace(command)
	fields := strings.Fields(normalized)

	for i := range fields {
		if !runnerTakingATargetName[fields[i]] {
			continue
		}
		for j := i + 1; j < len(fields); j++ {
			word := fields[j]
			switch {
			case strings.HasPrefix(word, "-"):
				if changesDirectory(word) {
					j = len(fields) // a different project's targets; contribute nothing
					continue
				}
				if makeFlagsTakingAValue[word] {
					j++
				}
			case strings.Contains(word, "="):
				// `make FOO=bar test`: a variable assignment, not the target.
			case word == "run" || word == "exec":
				// `pnpm run build` — the script name is the next word.
			default:
				into[word] = true
				j = len(fields)
			}
		}
	}
}

// suggestIgnoreBlock renders the run's live suggestions as a pasteable dva.yml fragment
// (docs/56 §6-1, option A).
//
// The recommendation had argued against this helper as a one-keystroke path to silence.
// What makes it safe is that silence is no longer reachable: whatever is pasted here is
// counted in the summary line, expandable with --show-ignored, and reported as stale once
// the target it names disappears. What is left for the helper to remove is transcription,
// and transcription never stopped anyone from ignoring everything — the twenty-entry
// dogfood lists were typed by hand.
//
// It prints the block and stops there. Rewriting dva.yml would put the decision back
// outside the author's reading of it, which is the part worth keeping.
func suggestIgnoreBlock(names []string) string {
	if len(names) == 0 {
		return ""
	}
	sorted := make([]string, len(names))
	copy(sorted, names)
	sort.Strings(sorted)

	var b strings.Builder
	b.WriteString("suggestion_ignore:\n")
	for _, name := range sorted {
		fmt.Fprintf(&b, "  - %q\n", name)
	}
	return b.String()
}
