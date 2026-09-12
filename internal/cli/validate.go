package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ScriptonBasestar/dva/internal/config"
	dvaexec "github.com/ScriptonBasestar/dva/internal/exec"
	"github.com/ScriptonBasestar/dva/internal/output"
	"github.com/ScriptonBasestar/dva/internal/runner"
)

// validateNoticeWriter is where validate writes [warn]/[fixed]/[error] lines that
// accompany a successful (or soft-fail) pass.
//
// Rule (TASK-142): on the human path, notices that qualify the ✅ verdict share stdout
// with it so a reader of one stream sees both. On --json, stdout is reserved for the
// single document and notices stay on stderr so the document is not corrupted by prose.
// Errors that abort validation still use stderr directly where they are emitted.
func validateNoticeWriter() io.Writer {
	if jsonOutput {
		return os.Stderr
	}
	return os.Stdout
}

var validateStrict bool

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

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the syntax and schema of 'dva.yml'",
	// Set here in the struct literal, not from a later init(): validate_alias.go copies
	// this Long by value into the top-level 'dva validate' alias inside its own init(), so
	// assigning it afterward would leave that alias's Long empty while this command's own
	// Long looked fixed.
	Long: `Check dva.yml against its JSON schema, then run semantic checks: unrunnable
compose runner commands (hard failure), compose file project-name mismatches, interaction
name collisions, config drift, and other semantic warnings. Reached as both
'dva validate' and 'dva config validate'.

--fix rewrites compose file 'name:' mismatches (and creates a missing devcontainer.json
when a devcontainer: section is declared and enabled) instead of only reporting them.
--strict turns config-drift, semantic, and interaction-collision warnings into a failing
exit code; without it those are reported but do not fail validation.

See USAGE.md's "config validate" section for the full list of semantic checks.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := loadConfigForValidate()
		report := newValidateReport(c)

		// Hard errors are collected, not returned, so one malformed entry does not hide
		// the next one or the warnings below it. Every hard error is printed together at
		// the end (TASK-305); the exit code is unchanged.
		var hard []error
		for _, p := range c.LoadProblems() {
			hard = append(hard, errors.New(p))
		}

		if err := c.Validate(); err != nil {
			hard = append(hard, err)
		}

		// env_bridge's origin and version rules (TASK-281 §3-2) report only from
		// here and from `dva config env seal/show` — never from an ordinary
		// lifecycle command, so a policy declaration about the secret surface
		// cannot brick `dva up`.
		if err := checkEnvBridgeOriginAndVersion(c); err != nil {
			hard = append(hard, err)
		}

		// A hard failure rather than a warning. The schema accepts this config, and then
		// every compose runner rejects it at the moment it tries to run — so `dva validate`
		// exiting 0 here is the whole defect: a green check that is evidence about the
		// checker, not the config.
		if problems := detectUnrunnableComposeCommands(c); len(problems) > 0 {
			for _, p := range problems {
				fmt.Fprintf(os.Stderr, "[error] compose: %s\n", p)
			}
			hard = append(hard, fmt.Errorf("%d compose runner command(s) contain no command word", len(problems)))
		}

		notice := validateNoticeWriter()

		// Check compose file project name alignment
		warnings := c.ValidateComposeProjectNames()
		fix, _ := cmd.Flags().GetBool("fix")

		if fix {
			fixComposeNameWarnings(c, warnings)
		} else {
			printComposeNameWarnings(notice, warnings)
			// Only when they were reported: --fix rewrote the files, so the mismatch no
			// longer exists and putting it in the report would describe a fixed state as
			// an outstanding warning.
			for _, w := range warnings {
				report.addComposeNameWarning(w)
			}
		}

		// Semantic warnings (version, health checks, duplicate commands, etc.)
		semanticWarnings := c.ValidateWarnings()
		for _, w := range semanticWarnings {
			_, _ = fmt.Fprintf(notice, "[warn] semantic: %s\n", w)
		}
		report.add("semantic", semanticWarnings...)

		collisionWarnings := detectInteractionCollisionWarnings(c)
		for _, w := range collisionWarnings {
			_, _ = fmt.Fprintf(notice, "[warn] interaction: %s\n", w)
		}
		report.add("interaction_collision", collisionWarnings...)

		suppressed := &suppressionSummary{}

		driftWarnings, driftSuppressed := detectConfigDriftWarningsWithSuppressions(c)
		suppressed.merge(driftSuppressed)
		printConfigDriftWarnings(notice, driftWarnings)
		report.add("config_drift", driftWarnings...)

		suggestionWarnings, suggestionSuppressed := detectConfigSuggestionWarningsWithSuppressions(c)
		suppressed.merge(suggestionSuppressed)
		printConfigSuggestionWarnings(notice, suggestionWarnings)
		report.add("config_suggestion", suggestionWarnings...)

		// A pattern that hides nothing is reported whether or not --show-ignored was asked
		// for: it is a defect in dva.yml, not a detail about this run (docs/56 §6-5).
		printStaleIgnoreWarnings(notice, suppressed.stale)
		report.add("ignore_stale", suppressed.stale...)

		if showIgnored, _ := cmd.Flags().GetBool("show-ignored"); showIgnored {
			printSuppressedItems(notice, suppressed)
		}
		report.addSuppressions(suppressed)

		if err := failValidation(&report, dedupeErrors(hard)); err != nil {
			return err
		}

		if validateStrict && (len(driftWarnings) > 0 || len(semanticWarnings) > 0 || len(collisionWarnings) > 0) {
			return report.fail(fmt.Errorf("config warnings detected; review warnings above or run 'am run dva-improve'"))
		}

		// Check devcontainer sync
		if len(c.Devcontainer) > 0 && isDevcontainerEnabled(c.Devcontainer) {
			dcPath := filepath.Join(c.FileDir(), ".devcontainer", "devcontainer.json")
			if _, err := os.Stat(dcPath); os.IsNotExist(err) {
				if fix {
					if err := writeDevcontainerFiles(c.Devcontainer, c.AllComposeFiles(), c.FileDir()); err != nil {
						fmt.Fprintf(os.Stderr, "[error] devcontainer: %v\n", err)
					} else {
						_, _ = fmt.Fprintf(notice, "[fixed] created .devcontainer/devcontainer.json\n")
					}
				} else {
					_, _ = fmt.Fprintf(notice, "[warn] devcontainer section found but .devcontainer/devcontainer.json missing\n")
					_, _ = fmt.Fprintf(notice, "       → run: dva config validate --fix\n")
					report.add("devcontainer", "devcontainer section found but .devcontainer/devcontainer.json missing\n  → run: dva config validate --fix")
				}
			}
		}

		if suggestIgnore, _ := cmd.Flags().GetBool("suggest-ignore"); suggestIgnore && !jsonOutput {
			// Printed raw, and only on the human path: this block is meant to be copied
			// into dva.yml rather than read. On this path the notice writer is stdout
			// anyway; the guard is about --json, which reserves stdout for the single
			// document that a YAML fragment beside it would corrupt.
			fmt.Print(suggestIgnoreBlock(suppressed.suggested))
		}

		if jsonOutput {
			return output.PrintJSON(report)
		}
		// The suffix is not optional and has no flag to turn it off. It is what lets every
		// ignore surface above exist: an ignored finding stays counted, so "valid" never
		// silently means "valid once N findings were hidden" (docs/56 §6-4).
		fmt.Printf("✅ dva.yml is valid%s\n", suppressed.summarySuffix())
		return nil
	},
}

// loadConfigForValidate loads the config the way every other command does, and when that
// fails on a stack-entry shape problem, loads it again recording those problems instead
// (config.CollectEntryProblems) so the rest of the diagnostics still run. A file that
// cannot be parsed at all, or fails a check the lenient load does not defer, exits
// through mustLoadConfig's path exactly as before.
func loadConfigForValidate() *config.Config {
	c, err := loadConfig()
	if err == nil {
		return c
	}
	cfg = nil
	lenient, lenientErr := config.Load(".", config.CollectEntryProblems())
	if lenientErr != nil {
		return mustLoadConfig()
	}
	cfg = lenient
	checkGitignoreForWarning(lenient.FileDir())
	return lenient
}

// dedupeErrors flattens joined errors (config.Validate returns one ValidationErrors for
// all its findings) so each finding is its own numbered item, and drops repeated
// messages: the lenient load and Validate both run validateEntrySource, so a missing
// source would otherwise be listed twice.
func dedupeErrors(errs []error) []error {
	seen := make(map[string]bool, len(errs))
	var out []error
	var walk func([]error)
	walk = func(list []error) {
		for _, err := range list {
			if joined, ok := err.(interface{ Unwrap() []error }); ok {
				walk(joined.Unwrap())
				continue
			}
			if seen[err.Error()] {
				continue
			}
			seen[err.Error()] = true
			out = append(out, err)
		}
	}
	walk(errs)
	return out
}

// failValidation returns the failing error, or nil when there are none. Several hard
// errors come back as one numbered list (validationFailure) so root's single "ERROR:"
// line shows every problem after the warnings, and `--json` still gets each member
// through Unwrap.
func failValidation(report *validateReport, hard []error) error {
	switch len(hard) {
	case 0:
		return nil
	case 1:
		return report.fail(hard[0])
	}
	return report.fail(validationFailure(hard))
}

// validationFailure renders several hard errors as a numbered list with a count, so a
// twelve-line schema error and a one-line hook error next to it read as two items.
type validationFailure []error

func (v validationFailure) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d errors found in this config:", len(v))
	for i, err := range v {
		fmt.Fprintf(&b, "\n\n[%d] %s", i+1, err.Error())
	}
	return b.String()
}

func (v validationFailure) Unwrap() []error { return v }

func init() {
	addValidateFlags(validateCmd)
	configCmd.AddCommand(validateCmd)
}

// composeNameWarningLines renders one compose-name warning as its headline followed by
// its continuation lines, without any of the prefixes either consumer adds.
//
// It exists so the prose on stderr and the JSON in validateReport are the same sentences
// by construction rather than by two authors agreeing (TASK-088). The alternative — a
// second set of format strings in validate_json.go — is the shape that produced the
// note-printed-four-different-ways defect in TASK-086.
func composeNameWarningLines(w config.ComposeNameWarning) []string {
	if w.ComposeName == "" {
		return []string{
			fmt.Sprintf("%s: missing top-level 'name: %s'", w.File, w.DvaName),
			"Running 'docker compose up' directly will use the directory name as project,",
			fmt.Sprintf("causing port conflicts with dva. Fix: add 'name: %s' to %s", w.DvaName, w.File),
		}
	}
	return []string{
		fmt.Sprintf("%s: name '%s' differs from dva.yml project_name '%s'", w.File, w.ComposeName, w.DvaName),
		fmt.Sprintf("Fix: change 'name: %s' to 'name: %s' in %s", w.ComposeName, w.DvaName, w.File),
	}
}

// printComposeNameWarnings prints compose file name mismatch warnings to w.
func printComposeNameWarnings(w io.Writer, warnings []config.ComposeNameWarning) {
	for _, warning := range warnings {
		lines := composeNameWarningLines(warning)
		_, _ = fmt.Fprintf(w, "[warn] semantic: %s\n", lines[0])
		for _, detail := range lines[1:] {
			_, _ = fmt.Fprintf(w, "       %s\n", detail)
		}
	}
}

// fixComposeNameWarnings auto-fixes compose file name mismatches.
func fixComposeNameWarnings(c *config.Config, warnings []config.ComposeNameWarning) {
	notice := validateNoticeWriter()
	for _, w := range warnings {
		if err := c.FixComposeProjectName(w); err != nil {
			fmt.Fprintf(os.Stderr, "[error] failed to fix %s: %v\n", w.File, err)
		} else {
			_, _ = fmt.Fprintf(notice, "[fixed] %s: set 'name: %s'\n", w.File, w.DvaName)
		}
	}
}

func printConfigDriftWarnings(w io.Writer, warnings []string) {
	for _, warning := range warnings {
		_, _ = fmt.Fprintf(w, "[warn] config drift: %s\n", warning)
	}
}

func detectConfigDriftWarnings(c *config.Config) []string {
	warnings, _ := detectConfigDriftWarningsWithSuppressions(c)
	return warnings
}

// detectConfigDriftWarningsWithSuppressions is detectConfigDriftWarnings plus the
// accounting docs/56 §2 principle 1 requires: what `drift_ignore` hid, and which of its
// patterns hid nothing. The plain wrapper above stays because most callers — every drift
// test among them — only ask what the operator would see, and threading an out-parameter
// through them would obscure that.
func detectConfigDriftWarningsWithSuppressions(c *config.Config) ([]string, *suppressionSummary) {
	var warnings []string
	suppressed := &suppressionSummary{}

	// configuredRootComposeFiles' file list is unused here (Finding 2/3 replaced the symmetric
	// comparison it fed); its deferredRootCompose flag is still authoritative for "a root
	// entry's compose path needs plan/site/entry vars that validation cannot resolve yet" and
	// gates the unregistered-file scan the same way it gated the old comparison — comparing
	// against an incomplete configured corpus produces false positives, not signal.
	//
	// The stale-pattern check is gated with it for the same reason in reverse: when the scan
	// does not run, no file was examined, so calling every drift_ignore pattern stale would
	// be a verdict drawn from an empty sample.
	if _, deferredRootCompose := configuredRootComposeFiles(c); !deferredRootCompose {
		warnings = append(warnings, detectUnregisteredComposeFileWarnings(c, suppressed)...)
	}
	for _, file := range missingConfiguredComposeFiles(c) {
		warnings = append(warnings, fmt.Sprintf("compose file %q is configured by dva.yml but does not exist", file))
	}

	availableServices, complete := configuredComposeServices(c)
	if !complete || len(availableServices) == 0 {
		// Not merely an optimization: this is what stops a compose file built entirely out
		// of `include:` or an incomplete configured corpus from producing a false positive
		// on every interaction. See configuredComposeServices' doc comment.
		return warnings, suppressed
	}

	tree := runner.NewInteractionTree(c.Interaction)
	for name, cmd := range tree.List() {
		if cmd.Service == "" {
			continue
		}
		if !availableServices[cmd.Service] {
			warnings = append(warnings,
				fmt.Sprintf("interaction %q references compose service %q, but configured compose files expose %s",
					name, cmd.Service, formatList(sortedSetKeys(availableServices))))
		}
	}

	return warnings, suppressed
}

// configuredRootComposeFiles returns configured compose files that live directly
// beside dva.yml. Root autodiscovery deliberately does not walk subdirectories,
// so comparing it to every configured file treats an explicit, isolated compose
// project (for example compose/e2e.yaml) as drift on every strict validation.
//
// Existing root paths remain in the comparison even when absent, making a stale
// root compose declaration visible instead of silently skipping it. The boolean
// result is true when a root file still needs plan/site resolution; callers must
// defer the comparison rather than compare autodiscovery to an incomplete set.
func configuredRootComposeFiles(c *config.Config) ([]string, bool) {
	root := c.FileDir()
	var files []string
	configured, deferredRootCompose, _ := configuredComposeFiles(c)
	for _, file := range configured {
		if !file.root {
			continue
		}
		path := file.path
		rel, err := filepath.Rel(root, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		if filepath.Dir(rel) != "." {
			continue
		}
		files = append(files, filepath.ToSlash(rel))
	}
	return deduplicateComposeFiles(root, files), deferredRootCompose
}

// missingConfiguredComposeFiles checks every configured Compose path, rather
// than only the root-level paths compared with autodiscovery. That keeps
// isolated subdirectory projects out of the root drift comparison without
// letting a stale subdirectory or absolute path pass strict validation.
func missingConfiguredComposeFiles(c *config.Config) []string {
	seen := make(map[string]bool)
	var missing []string
	configured, _, _ := configuredComposeFiles(c)
	for _, file := range configured {
		identity := file.sourceIdentity + "\x00" + file.path
		if seen[identity] {
			continue
		}
		seen[identity] = true
		if !fileExists(file.path) {
			if file.root {
				missing = append(missing, file.file)
			} else {
				missing = append(missing, fmt.Sprintf("%s (stack entry %q)", file.file, file.entryName))
			}
		}
	}
	sort.Strings(missing)
	return missing
}

type configuredComposeFile struct {
	entryName      string
	file           string
	path           string
	sourceIdentity string
	root           bool
}

// configuredComposeFiles resolves each compose file the same way the lifecycle
// runner does: ordinary entries are relative to dva.yml, while source entries
// are relative to their SourceDir. The base environment is cloned for every
// entry before entry vars are merged, matching the orchestrator's scope. A source
// that has not been made available yet is deferred to source readiness; validation
// must not mistake an unfetched git cache (or absent path source) for a missing
// compose file. Paths still containing interpolation after those static layers
// are deferred too: plan/site/plan-entry variables belong to lifecycle resolution,
// and this validator intentionally does not duplicate that resolver.
func configuredComposeFiles(c *config.Config) ([]configuredComposeFile, bool, bool) {
	// TASK-248: no env-file I/O during validation. A compose path that still
	// interpolates after the literal layers below is deferred by the existing
	// `strings.Contains(resolved, "$")` guard, exactly as plan and site variables
	// already are — one more deferral, not a new failure mode, and never a claim
	// that a path the validator cannot resolve is present.
	env := config.NewEnvironment(c.Vars, c.FileDir(), c.FileDir())
	env.MergeVars(c.Environment)

	names := make([]string, 0, len(c.Stack))
	for name := range c.Stack {
		names = append(names, name)
	}
	sort.Strings(names)

	var files []configuredComposeFile
	deferredRootCompose := false
	complete := true
	for _, name := range names {
		entry := c.Stack[name]
		if entry == nil || entry.ComposeConfig() == nil {
			continue
		}

		baseDir := c.FileDir()
		root := true
		sourceIdentity := "root"
		if entry.Source != nil {
			resolved, err := config.SourceDir(entry.Source, name, c.FileDir())
			if err != nil {
				complete = false
				continue
			}
			info, err := os.Stat(resolved)
			if err != nil || !info.IsDir() {
				complete = false
				continue
			}
			baseDir = resolved
			root = false
			sourceIdentity = "source:" + name + ":" + filepath.Clean(resolved)
		}

		entryEnv := env.Clone()
		entryEnv.MergeVars(entry.Vars)
		for _, declared := range entry.ComposeConfig().Files {
			if strings.TrimSpace(declared) == "" {
				continue
			}
			resolved := entryEnv.Interpolate(declared)
			if strings.Contains(resolved, "$") {
				complete = false
				if root {
					deferredRootCompose = true
				}
				continue
			}
			if !filepath.IsAbs(resolved) {
				resolved = filepath.Join(baseDir, resolved)
			}
			resolved = filepath.Clean(resolved)
			if !fileExists(resolved) {
				complete = false
			}
			files = append(files, configuredComposeFile{
				entryName:      name,
				file:           declared,
				path:           resolved,
				sourceIdentity: sourceIdentity,
				root:           root,
			})
		}
	}
	return files, deferredRootCompose, complete
}

// detectUnregisteredComposeFileWarnings reports compose files that autodiscovery finds in a
// scanned directory but no stack entry lists under runners.compose.files (TASK-316 Finding 2)
// — the mirror image of missingConfiguredComposeFiles, which already reports the opposite
// direction (registered but absent). A registered file that exists is never drift here: the
// two warnings are deliberately asymmetric because they answer different questions.
//
// The scan set is root + the directory of every root-entry configured compose file + every
// directory reached by following those files' `include:` chains (Finding 3), so overlays and
// subdirectory compose corpora that dva.yml only partially declares are visible instead of
// invisible outside the root directory. `source:` entries are excluded — they own an
// externally-managed corpus dva.yml is not expected to fully enumerate. The registered set
// tracked per scanned directory is configured files ∪ include-reached files, compared by
// canonicalComposePath so a symlink alias (e.g. docker-compose.yml -> compose.yaml) still
// counts as the one file it is (TestDetectConfigDriftWarnings_SymlinkAliasRepresentsOneComposeFile).
// composeScanDirLocation renders a scanned directory for a drift warning, relative to the
// dva.yml directory when it sits underneath it. Directories reached through `include:` are
// symlink-resolved (canonicalComposePath), so on platforms where the config directory itself
// is behind a symlink — macOS /tmp → /private/tmp — a plain Rel against baseDir escapes with
// a long `../..` chain; comparing against the canonical base too keeps those readable.
func composeScanDirLocation(baseDir, dir string) string {
	for _, base := range []string{baseDir, canonicalComposePath(baseDir)} {
		rel, err := filepath.Rel(base, dir)
		if err != nil || strings.HasPrefix(rel, "..") {
			continue
		}
		if rel == "." || rel == "" {
			return "beside dva.yml"
		}
		return fmt.Sprintf("in %s/", filepath.ToSlash(rel))
	}
	return fmt.Sprintf("in %s/", filepath.ToSlash(dir))
}

func detectUnregisteredComposeFileWarnings(c *config.Config, suppressed *suppressionSummary) []string {
	configured, _, _ := configuredComposeFiles(c)

	registered := map[string]bool{}
	scanDirs := map[string]bool{c.FileDir(): true}
	for _, file := range configured {
		if !file.root {
			continue
		}
		scanDirs[filepath.Dir(file.path)] = true
		for _, reachable := range composeReachablePaths(file.path) {
			registered[reachable] = true
			scanDirs[filepath.Dir(reachable)] = true
		}
	}

	dirs := make([]string, 0, len(scanDirs))
	for dir := range scanDirs {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	var warnings []string
	var discovered []string
	for _, dir := range dirs {
		var unregistered []string
		for _, name := range detectComposeFilesInDir(dir) {
			ignoreName := driftIgnoreName(c.FileDir(), dir, name)
			discovered = append(discovered, ignoreName)
			if registered[canonicalComposePath(filepath.Join(dir, name))] {
				continue
			}
			// docs/56 §6-3: drift_ignore applies to this rule and to nothing else. The
			// other two drift findings — a configured file that is missing, an interaction
			// naming a service no compose file exposes — describe a config that fails the
			// moment it runs, and suppressing those would only move the failure from
			// validate to `dva up`.
			if pattern, ok := matchIgnorePattern(ignoreName, c.DriftIgnore); ok {
				suppressed.add(suppressedDrift, ignoreName, fmt.Sprintf("drift_ignore: %s", pattern))
				continue
			}
			unregistered = append(unregistered, name)
		}
		if len(unregistered) == 0 {
			continue
		}
		sort.Strings(unregistered)

		location := composeScanDirLocation(c.FileDir(), dir)
		warnings = append(warnings, fmt.Sprintf(
			"compose files %s exist %s but no stack entry lists them under runners.compose.files; add them to an entry or leave them out on purpose (suppression: TASK-309)",
			formatList(unregistered), location))
	}

	sort.Strings(discovered)
	suppressed.stale = append(suppressed.stale,
		staleIgnoreWarnings("drift_ignore", c.DriftIgnore, discovered, "compose file that autodiscovery found")...)

	return warnings
}

// driftIgnoreName is what a drift_ignore pattern is matched against: the discovered file's
// path relative to dva.yml, slash-separated. It is a relative path rather than a bare
// basename because autodiscovery already scans the directories of registered root compose
// files, and two directories may hold a `compose.yaml` each — matching on the basename
// would let one project's ignore silently cover the other's file. A path that escapes the
// config directory falls back to the bare name, since a pattern cannot usefully spell it.
//
// The two bases are the same defence composeScanDirLocation makes: directories reached
// through `include:` are symlink-resolved, so where the config directory itself sits behind
// a symlink — macOS /tmp → /private/tmp — a plain Rel against root escapes and the name
// would silently collapse to the basename. A collapsed name is worse here than in a warning
// string: it decides whether a drift_ignore pattern matches, so the same file would be
// suppressed on one checkout path and reported on another.
func driftIgnoreName(root, dir, name string) string {
	path := filepath.Join(dir, name)
	for _, base := range []string{root, canonicalComposePath(root)} {
		rel, err := filepath.Rel(base, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		return filepath.ToSlash(rel)
	}
	return name
}

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
