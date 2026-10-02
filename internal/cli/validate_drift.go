package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/runner"
)

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
