package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/runner"
)

func printConfigSuggestionWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(w, "[warn] config suggestion: %s\n", warning)
	}
}

func detectConfigSuggestionWarnings(c *config.Config) []string {
	warnings, _ := detectConfigSuggestionWarningsWithSuppressions(c)
	return warnings
}

// detectConfigSuggestionWarningsWithSuppressions is detectConfigSuggestionWarnings plus
// what `suggestion_ignore` and `suggestions:` hid, and which ignore patterns hid nothing.
// See detectConfigDriftWarningsWithSuppressions for why the plain wrapper stays.
func detectConfigSuggestionWarningsWithSuppressions(c *config.Config) ([]string, *suppressionSummary) {
	suppressed := &suppressionSummary{}
	allCommands := runner.NewInteractionTree(c.Interaction).List()

	// Build subcommand coverage set: for "app:build ce" → also match "build-ce"
	// This detects when a Makefile target like "build-ce" is already covered by a
	// DVA interaction subcommand under a different parent name.
	commandSet := map[string]bool{}
	subcommandCoverage := map[string]bool{}
	importedLeafSources := map[string]map[string]bool{}
	for name, cmd := range allCommands {
		commandSet[name] = true

		// Walk the segments the tree recorded instead of re-splitting name on spaces:
		// the join is one-way once a segment contains a space, and a consumer that
		// re-splits gets the boundary wrong (TASK-097, expandInto's own comment).
		path := cmd.Path
		if len(path) == 0 {
			path = []string{name}
		}

		// Imported interactions are reachable through their canonical
		// "subproject/name" address (or an explicit as: alias), never through their
		// leaf name alone. Keep the leaf source so an otherwise-valid suggestion can
		// point to the alias that creates a real root route (TASK-352).
		coverageHead := path[0]
		head := coverageHead
		if idx := strings.LastIndex(head, "/"); idx >= 0 {
			leaf := head[idx+1:]
			if len(path) == 1 && cmd.SubprojectName != "" && name == cmd.CanonicalAddress {
				if importedLeafSources[leaf] == nil {
					importedLeafSources[leaf] = map[string]bool{}
				}
				importedLeafSources[leaf][cmd.SubprojectName] = true
			}
			head = leaf
		}

		if len(path) < 2 {
			continue
		}
		// Preserve the canonical imported spelling as well as the leaf spelling.
		// `dva frontend/test e2e` is a real route for `frontend/test-e2e`, while the
		// leaf form keeps TASK-320's coverage for established root Makefile targets.
		if idx := strings.LastIndex(coverageHead, ":"); idx >= 0 {
			coverageHead = coverageHead[idx+1:]
		}
		subcommandCoverage[strings.Join(append([]string{coverageHead}, path[1:]...), "-")] = true
		// "app:build ce" → "build-ce", "test all" → "test-all"
		subParts := append([]string{head}, path[1:]...)
		subcommandCoverage[strings.Join(subParts, "-")] = true
	}

	candidates := map[string]string{}
	for _, target := range extractDocumentedMakefileTargetNamesInDir(c.FileDir()) {
		candidates[target] = "Makefile"
	}
	for _, script := range extractPackageScriptNamesInDir(c.FileDir()) {
		if _, exists := candidates[script]; !exists {
			candidates[script] = "package.json"
		}
	}

	var names []string
	for name := range candidates {
		names = append(names, name)
	}
	sort.Strings(names)

	wrapped := wrappedWorkflowTargets(c)

	var warnings []string
	for _, name := range names {
		if commandSet[name] {
			continue
		}
		if subcommandCoverage[name] {
			continue
		}
		// docs/56 §3 C-1. Not a suppression: the target is already wrapped, so there is
		// nothing to suggest and nothing for the operator to review.
		if wrapped[name] {
			continue
		}
		if kind := suggestionSourceKind(candidates[name]); !c.Suggestions.Enabled(kind) {
			suppressed.add(suppressedSuggestion, name, fmt.Sprintf("suggestions.%s: false", kind))
			continue
		}
		if pattern, ok := matchIgnorePattern(name, c.SuggestionIgnore); ok {
			suppressed.add(suppressedSuggestion, name, fmt.Sprintf("suggestion_ignore: %s", pattern))
			continue
		}
		suppressed.suggested = append(suppressed.suggested, name)
		if sources := importedLeafSources[name]; len(sources) > 0 {
			warnings = append(warnings,
				fmt.Sprintf("%s defines %q but it is only imported from %s; add `as: %s` to that interaction import to create a runnable root route",
					candidates[name], name, formatList(sortedSuggestionSources(sources)), name))
			continue
		}
		warnings = append(warnings,
			fmt.Sprintf("%s defines %q but no DVA interaction with the same name exists; consider adding a direct mapping if it is part of the developer workflow",
				candidates[name], name))
	}

	// The universe is every documented target and script before the built-in exclusions,
	// not the candidate list above: see allDocumentedMakefileTargetNamesInDir.
	//
	// An empty universe means nothing was examined — no Makefile and no package.json, or
	// ones that declare no documented target and no script — not that every pattern went
	// unused. Condemning the whole list there would be a verdict drawn from an empty
	// sample, the same reason the drift side skips its stale check when the
	// unregistered-file scan does not run.
	universe := append(allDocumentedMakefileTargetNamesInDir(c.FileDir()), allPackageScriptNamesInDir(c.FileDir())...)
	if len(universe) > 0 {
		suppressed.stale = append(suppressed.stale,
			staleIgnoreWarnings("suggestion_ignore", c.SuggestionIgnore, universe, "Makefile target or package.json script")...)
	}

	return warnings, suppressed
}

func sortedSuggestionSources(sources map[string]bool) []string {
	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func detectComposeFilesInDir(dir string) []string {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	var found []string
	for _, name := range candidates {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			found = append(found, name)
		}
	}

	for _, name := range []string{"docker-compose.override.yml", "docker-compose.override.yaml"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			found = append(found, name)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return deduplicateComposeFiles(dir, found)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if hasComposeFileNamePrefix(name) &&
			(strings.HasSuffix(name, ".yml") || strings.HasSuffix(name, ".yaml")) &&
			!contains(found, name) {
			found = append(found, name)
		}
	}

	if len(found) > 1 {
		primary := []string{}
		rest := []string{}
		for _, file := range found {
			switch filepath.Base(file) {
			case "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml":
				primary = append(primary, file)
			default:
				rest = append(rest, file)
			}
		}
		found = append(primary, rest...)
	}

	return deduplicateComposeFiles(dir, found)
}

// configuredComposeServices returns the services declared in the configured compose files,
// following compose `include:` (composeServiceCollector.collect recurses into it, cycle-safe
// via a visited-path set — see TestDetectConfigDriftWarnings_InteractionServiceFromIncludedComposeMatches).
// TASK-068 originally left `include:` unresolved; TASK-316 Finding 1 found it had since been
// fixed and only this comment still described the old gap.
//
// A file built entirely out of `include:` with no top-level `services:` of its own still
// contributes nothing on its own line, but its included files are visited and do contribute.
// What keeps an incomplete configured corpus from becoming a false positive is the empty-set
// early return in detectConfigDriftWarnings, not anything here: when the configured files
// yield no services at all, this returns an empty map and the interaction-service comparison
// is skipped wholesale. Stated because that is a load-bearing dependency between two functions
// that neither of them declared, and it survives only as long as nobody refactors either half.
func configuredComposeServices(c *config.Config) (map[string]bool, bool) {
	services := map[string]bool{}
	configured, _, complete := configuredComposeFiles(c)
	for _, file := range configured {
		if !composeFileReadable(file.path) {
			complete = false
			continue
		}
		for _, service := range extractComposeServices(file.path) {
			services[service] = true
		}
	}
	return services, complete
}

func composeFileReadable(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	return f.Close() == nil
}

func sortedSetKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	return strings.Join(items, ", ")
}

func extractDocumentedMakefileTargetNamesInDir(dir string) []string {
	var kept []string
	for _, target := range allDocumentedMakefileTargetNamesInDir(dir) {
		if shouldIgnoreMakefileTarget(target) {
			continue
		}
		kept = append(kept, target)
	}
	return kept
}

// allDocumentedMakefileTargetNamesInDir is the same walk without the built-in exclusions
// shouldIgnoreMakefileTarget applies. It exists for the stale-ignore check: a
// `suggestion_ignore: [docker-*]` entry names targets that do exist, and judging its
// staleness against the already-narrowed list would report it as pointing at nothing the
// moment the built-in rules learned to exclude the same names (docs/56 §6-5).
func allDocumentedMakefileTargetNamesInDir(dir string) []string {
	targets := extractDocumentedTargetNamesFromMakefiles(filepath.Join(dir, "Makefile"))
	sort.Strings(targets)
	return targets
}

// extractDocumentedTargetNamesFromMakefiles follows include directives and
// extracts target names (without descriptions) from documented targets.
func extractDocumentedTargetNamesFromMakefiles(path string) []string {
	seen := map[string]bool{}
	var targets []string
	collectDocumentedTargetNames(path, seen, &targets)
	return targets
}

func collectDocumentedTargetNames(path string, seen map[string]bool, targets *[]string) {
	absPath, _ := filepath.Abs(path)
	if seen[absPath] {
		return
	}
	seen[absPath] = true

	data, err := os.ReadFile(path)
	if err != nil {
		matches, globErr := filepath.Glob(path)
		if globErr != nil || len(matches) == 0 {
			return
		}
		for _, m := range matches {
			collectDocumentedTargetNames(m, seen, targets)
		}
		return
	}

	dir := filepath.Dir(path)
	lines := strings.SplitSeq(string(data), "\n")
	for line := range lines {
		trimmed := strings.TrimSpace(line)

		// Follow include/-include directives
		if strings.HasPrefix(trimmed, "include ") || strings.HasPrefix(trimmed, "-include ") {
			includePath := strings.TrimPrefix(trimmed, "-include ")
			includePath = strings.TrimPrefix(includePath, "include ")
			includePath = strings.TrimSpace(includePath)
			if !filepath.IsAbs(includePath) {
				includePath = filepath.Join(dir, includePath)
			}
			matches, globErr := filepath.Glob(includePath)
			if globErr == nil && len(matches) > 0 {
				for _, m := range matches {
					collectDocumentedTargetNames(m, seen, targets)
				}
			} else {
				collectDocumentedTargetNames(includePath, seen, targets)
			}
			continue
		}

		// Extract target: ## description lines
		if strings.Contains(line, "##") && !strings.HasPrefix(line, "#") &&
			!strings.HasPrefix(line, "\t") && !strings.HasPrefix(line, " ") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			// `export NAME := value` contains a colon but is a variable assignment,
			// not a target declaration. Do this before splitting its left side into
			// tokens so neither `export` nor NAME becomes a suggestion candidate.
			if strings.HasPrefix(strings.TrimSpace(parts[1]), "=") {
				continue
			}
			// One recipe may serve several targets: `a b: ## desc` declares both, and
			// Make separates them by whitespace. Taking parts[0] whole invented a single
			// target spelled "a b", which no interaction name can ever match, so the
			// suggestion warned about a target that does not exist and stayed silent
			// about the two that do — flow-pipechain's
			// "log-search-bench perf-log-search:" (TASK-320).
			for target := range strings.FieldsSeq(parts[0]) {
				if strings.HasPrefix(target, ".") || strings.Contains(target, "$(") || strings.Contains(target, "%") {
					continue
				}
				*targets = append(*targets, target)
			}
		}
	}
}

// shouldIgnoreMakefileTarget returns true for Makefile targets that are meta/infra
// targets unlikely to be useful as DVA interactions.
func shouldIgnoreMakefileTarget(name string) bool {
	ignoredTargets := map[string]bool{
		// Meta targets
		"help": true, "all": true, "default": true,
		// DVA reserved commands — overlap with built-in DVA commands
		"stop": true, "up": true, "down": true, "restart": true,
		"run": true, config.LogsDirName: true, "build": true,
		// Not DVA commands, kept on their own merit rather than by overlap. `ps` never was
		// one — it is reached as `dva compose ps`. `clean` was, until the command surface
		// was restructured (docs/43); teardown is `dva down <plan> --purge` now, so there
		// is no built-in left for a `make clean` suggestion to collide with. Both stay
		// ignored because this list feeds a suggestion of project commands for
		// `interaction:`, and these two are the build system's own housekeeping.
		"ps": true, "clean": true,
		// Generic infra targets that overlap with DVA modes/stack
		"infra-up": true, "infra-down": true, "infra-start": true, "infra-stop": true,
		// Generic setup/dependency targets handled by provision
		"deps": true, "install": true, "prepare": true, "setup": true,
		"install-hooks": true,
		// Documentation targets
		"docs": true, "docs-build": true, "docs-serve": true,
	}
	if ignoredTargets[name] {
		return true
	}

	// Compose lifecycle suffixes: e.g., dev-full-up, e2e-down, app-logs
	// DVA handles these natively via modes and `dva up/down/logs` commands
	for _, suffix := range []string{"-up", "-down", "-stop", "-restart", "-logs", "-ps", "-build"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}

	// Target families DVA replaces outright (docs/56 §3 C-2). A `docker-shell` or
	// `k8s-apply` target is the orchestration dva.yml took over, not a project command
	// waiting to be wrapped in an interaction — and these prefixes were the bulk of the
	// dogfood suggestion_ignore lists. They are excluded by the rules rather than offered
	// as a `suggestions:` category on purpose (§6-2): a category would make every project
	// re-declare the same judgement, and the two copies drift.
	for _, prefix := range []string{"docker-", "compose-", "k8s-", "helm-"} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	// A leading underscore is the make convention for an internal helper target — one the
	// author already marked as not part of the developer-facing workflow.
	return strings.HasPrefix(name, "_")
}

func extractPackageScriptNamesInDir(dir string) []string {
	return packageScriptNamesInDir(dir, false)
}

// allPackageScriptNamesInDir is extractPackageScriptNamesInDir without the built-in
// exclusions, for the same reason allDocumentedMakefileTargetNamesInDir exists.
func allPackageScriptNamesInDir(dir string) []string {
	return packageScriptNamesInDir(dir, true)
}

func packageScriptNamesInDir(dir string, all bool) []string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}

	var pkg struct {
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	var scripts []string
	for name := range pkg.Scripts {
		if !all && shouldIgnorePackageScript(name) {
			continue
		}
		scripts = append(scripts, name)
	}
	sort.Strings(scripts)
	return scripts
}

func shouldIgnorePackageScript(name string) bool {
	if name == "" {
		return true
	}
	// Same convention as the Makefile side: `_build` is an internal helper the author
	// already flagged as not developer-facing.
	if strings.HasPrefix(name, "_") {
		return true
	}
	if strings.HasPrefix(name, "pre") && len(name) > 3 {
		return true
	}
	if strings.HasPrefix(name, "post") && len(name) > 4 {
		return true
	}
	switch name {
	case "prepare":
		return true
	default:
		return false
	}
}
