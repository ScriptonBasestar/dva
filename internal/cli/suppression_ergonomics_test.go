package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// TASK-309 / docs/56 §6. Every test here is about the same invariant from a different
// side: an ignore surface may remove a warning from the list, and may never remove the
// fact that something was removed.

// suppressionFixture writes a dva.yml plus the sibling files the suggestion and drift
// detectors read, and returns the config path for runValidateCommandForTest.
func suppressionFixture(t *testing.T, dvaYML string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, config.FileName)
	if err := os.WriteFile(path, []byte(dvaYML), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}
	for name, body := range files {
		full := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir for %s: %v", name, err)
		}
		if err := os.WriteFile(full, []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return path
}

func loadSuppressionConfig(t *testing.T, configPath string) *config.Config {
	t.Helper()
	t.Setenv(config.EnvFileKey, configPath)
	c, err := config.Load(filepath.Dir(configPath))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return c
}

const composeStub = "services:\n  web:\n    image: nginx\n"

// --- drift_ignore ---------------------------------------------------------------------

func TestDriftIgnoreSuppressesTheUnregisteredComposeFile(t *testing.T) {
	configPath := suppressionFixture(t, `version: "0.1.0"
drift_ignore:
  - "compose.ci.yaml"
`, map[string]string{
		"compose.yaml":    composeStub,
		"compose.ci.yaml": composeStub,
	})
	c := loadSuppressionConfig(t, configPath)

	warnings, suppressed := detectConfigDriftWarningsWithSuppressions(c)
	for _, w := range warnings {
		if strings.Contains(w, "compose.ci.yaml") {
			t.Errorf("drift_ignore did not suppress the warning: %s", w)
		}
	}
	if got := suppressed.count(suppressedDrift); got != 1 {
		t.Fatalf("suppressed drift count = %d, want 1 (an ignored finding must stay counted)", got)
	}
	if lines := suppressed.showIgnoredLines(); !strings.Contains(strings.Join(lines, "\n"), "drift_ignore: compose.ci.yaml") {
		t.Errorf("--show-ignored must name the responsible line, got: %v", lines)
	}
}

// A drift_ignore pattern is a path relative to dva.yml, and it has to keep meaning that
// when the config directory is reached through a symlink. Directories reached through
// `include:` come back symlink-resolved, so without canonicalising the base too, the
// relative name collapses to the bare basename: the pattern below stops matching, the
// warning comes back, and the pattern is additionally condemned as stale — the same file
// suppressed on one checkout path and reported on another.
func TestDriftIgnoreMatchesThroughASymlinkedConfigDirectory(t *testing.T) {
	base := t.TempDir()
	real := filepath.Join(base, "real")
	if err := os.MkdirAll(filepath.Join(real, "sub"), 0755); err != nil {
		t.Fatalf("mkdir real/sub: %v", err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(real, name), []byte(body), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	write(config.FileName, `version: "0.1.0"
drift_ignore:
  - "sub/compose.ci.yaml"
stack:
  web:
    runners:
      compose:
        files: [compose.yaml]
`)
	write("compose.yaml", "include:\n  - ./sub/compose.base.yaml\nservices:\n  web:\n    image: nginx\n")
	write("sub/compose.base.yaml", "services:\n  worker:\n    image: alpine\n")
	write("sub/compose.ci.yaml", composeStub)

	c := loadSuppressionConfig(t, filepath.Join(link, config.FileName))

	warnings, suppressed := detectConfigDriftWarningsWithSuppressions(c)
	for _, w := range warnings {
		if strings.Contains(w, "compose.ci.yaml") {
			t.Errorf("drift_ignore must match through the symlinked config dir, got warning: %s", w)
		}
	}
	if got := suppressed.count(suppressedDrift); got != 1 {
		t.Errorf("suppressed drift count = %d, want 1", got)
	}
	if joined := strings.Join(suppressed.stale, "\n"); strings.Contains(joined, "compose.ci.yaml") {
		t.Errorf("a pattern that did match must not be reported stale: %q", joined)
	}
}

// The unregistered-file rule is the only drift finding drift_ignore touches. Both of the
// findings below describe a config that fails when it runs, and docs/56 §2 principle 2
// puts those out of reach of every ignore surface.
func TestDriftIgnoreNeverSuppressesAnExecutionBreakingFact(t *testing.T) {
	missingFile := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.1.0"
drift_ignore:
  - "*"
stack:
  web:
    runners:
      compose:
        files: [compose.gone.yaml]
`, map[string]string{
		"compose.yaml": composeStub,
	}))
	if joined := strings.Join(detectConfigDriftWarnings(missingFile), "\n"); !strings.Contains(joined, "compose.gone.yaml") {
		t.Errorf("a configured-but-missing compose file must survive drift_ignore: %q", joined)
	}

	// A separate fixture, because the service check stands down whenever the configured
	// compose corpus is incomplete — a missing file already makes its answer a guess.
	missingService := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.1.0"
drift_ignore:
  - "*"
stack:
  web:
    runners:
      compose:
        files: [compose.yaml]
interaction:
  shell:
    service: nope
    command: sh
`, map[string]string{
		"compose.yaml": composeStub,
	}))
	if joined := strings.Join(detectConfigDriftWarnings(missingService), "\n"); !strings.Contains(joined, `references compose service "nope"`) {
		t.Errorf("an interaction naming a missing service must survive drift_ignore: %q", joined)
	}
}

func TestDriftIgnoreReportsAPatternThatMatchesNothing(t *testing.T) {
	configPath := suppressionFixture(t, `version: "0.1.0"
drift_ignore:
  - "compose.ci.yaml"
  - "compose.deleted.yaml"
`, map[string]string{
		"compose.yaml":    composeStub,
		"compose.ci.yaml": composeStub,
	})
	c := loadSuppressionConfig(t, configPath)

	_, suppressed := detectConfigDriftWarningsWithSuppressions(c)
	joined := strings.Join(suppressed.stale, "\n")
	if !strings.Contains(joined, `drift_ignore[1] "compose.deleted.yaml" matches no compose file`) {
		t.Errorf("stale pattern not reported: %q", joined)
	}
	if strings.Contains(joined, "compose.ci.yaml") {
		t.Errorf("a pattern that hid a real file must not be called stale: %q", joined)
	}
}

// --- suggestion_ignore and suggestions: -------------------------------------------------

func TestSuggestionIgnoreIsCountedAndAttributed(t *testing.T) {
	configPath := suppressionFixture(t, `version: "0.1.0"
suggestion_ignore:
  - "release*"
`, map[string]string{
		"Makefile": "release: ## Cut a release\nrelease-notes: ## Notes\nbench: ## Benchmarks\n",
	})
	c := loadSuppressionConfig(t, configPath)

	warnings, suppressed := detectConfigSuggestionWarningsWithSuppressions(c)
	if joined := strings.Join(warnings, "\n"); !strings.Contains(joined, `"bench"`) || strings.Contains(joined, `"release"`) {
		t.Fatalf("unexpected surviving warnings: %q", joined)
	}
	if got := suppressed.count(suppressedSuggestion); got != 2 {
		t.Fatalf("suppressed suggestion count = %d, want 2", got)
	}
	if lines := strings.Join(suppressed.showIgnoredLines(), "\n"); !strings.Contains(lines, `suggestion "release" — suggestion_ignore: release*`) {
		t.Errorf("--show-ignored must attribute the suppression to its pattern, got:\n%s", lines)
	}
}

func TestSuggestionsCategoryOptOutIsPerSourceKind(t *testing.T) {
	files := map[string]string{
		"Makefile":     "bench: ## Benchmarks\n",
		"package.json": `{"scripts": {"storybook": "storybook dev"}}`,
	}
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.1.0"
suggestions:
  makefile: false
`, files))

	warnings, suppressed := detectConfigSuggestionWarningsWithSuppressions(c)
	joined := strings.Join(warnings, "\n")
	if strings.Contains(joined, `"bench"`) {
		t.Errorf("suggestions.makefile: false must silence the Makefile source: %q", joined)
	}
	if !strings.Contains(joined, `"storybook"`) {
		t.Errorf("the package.json source stays on when only makefile is disabled: %q", joined)
	}
	if lines := strings.Join(suppressed.showIgnoredLines(), "\n"); !strings.Contains(lines, "suggestions.makefile: false") {
		t.Errorf("category suppression must be attributable too, got:\n%s", lines)
	}
}

// An omitted `suggestions:` block, and an omitted key inside one, both mean "on" — a
// pointer-free bool would have silenced every suggestion in every config written before
// TASK-309.
func TestOmittedSuggestionCategoryStaysEnabled(t *testing.T) {
	var declared config.SuggestionCategories
	for _, source := range []string{config.SuggestionSourceMakefile, config.SuggestionSourcePackageJSON, "unknown"} {
		if !(*config.SuggestionCategories)(nil).Enabled(source) {
			t.Errorf("nil suggestions block disabled %q", source)
		}
		if !declared.Enabled(source) {
			t.Errorf("declared block with no keys disabled %q", source)
		}
	}
}

// --- rule C: narrowing the source ------------------------------------------------------

func TestAnAlreadyWrappedTargetIsNotSuggested(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.1.0"
interaction:
  suite:
    runner: local
    command: make bench
  ui:
    runner: local
    command: pnpm run storybook
  elsewhere:
    runner: local
    command: make -C vendor lint
`, map[string]string{
		"Makefile":     "bench: ## Benchmarks\nlint: ## Lint\n",
		"package.json": `{"scripts": {"storybook": "storybook dev"}}`,
	}))

	joined := strings.Join(detectConfigSuggestionWarnings(c), "\n")
	if strings.Contains(joined, `"bench"`) {
		t.Errorf("`make bench` under another interaction name is already the wrapping: %q", joined)
	}
	if strings.Contains(joined, `"storybook"`) {
		t.Errorf("`pnpm run storybook` is already the wrapping: %q", joined)
	}
	// `make -C vendor lint` runs a different project's Makefile, so the root `lint`
	// target is still unwrapped and the suggestion has to survive.
	if !strings.Contains(joined, `"lint"`) {
		t.Errorf("a -C invocation must not count as coverage of the root target: %q", joined)
	}
}

func TestTargetFamiliesDVAReplacesAreNotSuggested(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.1.0"
`, map[string]string{
		"Makefile": strings.Join([]string{
			"docker-shell: ## Shell into the container",
			"compose-pull: ## Pull images",
			"k8s-apply: ## Apply manifests",
			"helm-diff: ## Diff the release",
			"_scaffold: ## Internal helper",
			"bench: ## Benchmarks",
		}, "\n") + "\n",
		"package.json": `{"scripts": {"_prepare-fixtures": "node x.js", "storybook": "storybook dev"}}`,
	}))

	joined := strings.Join(detectConfigSuggestionWarnings(c), "\n")
	for _, gone := range []string{"docker-shell", "compose-pull", "k8s-apply", "helm-diff", "_scaffold", "_prepare-fixtures"} {
		if strings.Contains(joined, gone) {
			t.Errorf("%q should be excluded by the suggestion rules themselves: %q", gone, joined)
		}
	}
	for _, kept := range []string{"bench", "storybook"} {
		if !strings.Contains(joined, kept) {
			t.Errorf("%q is an ordinary workflow target and must still be suggested: %q", kept, joined)
		}
	}
}

// The stale check measures a pattern against every documented target, not against the
// list the built-in rules left behind. Otherwise narrowing those rules — which TASK-309
// just did — would turn correct ignore entries into "remove it" advice overnight.
func TestStaleSuggestionIgnoreIsJudgedBeforeTheBuiltInExclusions(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.1.0"
suggestion_ignore:
  - "docker-*"
  - "vanished-*"
`, map[string]string{
		"Makefile": "docker-shell: ## Shell in\nbench: ## Benchmarks\n",
	}))

	_, suppressed := detectConfigSuggestionWarningsWithSuppressions(c)
	joined := strings.Join(suppressed.stale, "\n")
	if strings.Contains(joined, "docker-*") {
		t.Errorf("docker-shell exists, so `docker-*` is redundant but not stale: %q", joined)
	}
	if !strings.Contains(joined, `suggestion_ignore[1] "vanished-*"`) {
		t.Errorf("a pattern naming no target at all must be reported: %q", joined)
	}
}

// A repository with neither a Makefile nor a package.json gives the stale check nothing to
// judge against. Condemning every pattern there would read as "these entries are dead" when
// the truth is "nothing was examined" — the drift side already refuses that verdict when its
// scan does not run.
func TestStaleSuggestionIgnoreStandsDownWhenThereIsNothingToJudgeAgainst(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.1.0"
suggestion_ignore:
  - "release*"
  - "bench"
`, nil))

	_, suppressed := detectConfigSuggestionWarningsWithSuppressions(c)
	if len(suppressed.stale) != 0 {
		t.Errorf("no Makefile and no package.json means an empty sample, got: %v", suppressed.stale)
	}
}

// --- the summary line and the two flags --------------------------------------------------

// The count is the whole reason the ignore surfaces above are allowed to exist (docs/56
// §6-4), so it is pinned end to end rather than at the formatter.
func TestValidateSummaryAlwaysCountsWhatWasIgnored(t *testing.T) {
	configPath := suppressionFixture(t, `version: "0.1.0"
suggestion_ignore:
  - "release"
  - "bench"
drift_ignore:
  - "compose.ci.yaml"
`, map[string]string{
		"Makefile":        "release: ## Cut a release\nbench: ## Benchmarks\n",
		"compose.yaml":    composeStub,
		"compose.ci.yaml": composeStub,
	})

	run := runValidateCommandForTest(t, configPath, "validate")
	if run.err != "" {
		t.Fatalf("validate failed: %s", run.err)
	}
	if !strings.Contains(run.stdout, "✅ dva.yml is valid (2 suggestions, 1 drift file ignored by dva.yml)") {
		t.Errorf("summary line does not account for the suppressions:\n%s", run.stdout)
	}
	// Without --show-ignored the names stay out of the way; the count does not.
	if strings.Contains(run.stdout, "[info] ignored:") {
		t.Errorf("--show-ignored was not asked for:\n%s", run.stdout)
	}

	shown := runValidateCommandForTest(t, configPath, "validate", "--show-ignored")
	for _, want := range []string{
		`[info] ignored: drift file "compose.ci.yaml" — drift_ignore: compose.ci.yaml`,
		`[info] ignored: suggestion "bench" — suggestion_ignore: bench`,
		`[info] ignored: suggestion "release" — suggestion_ignore: release`,
	} {
		if !strings.Contains(shown.stdout, want) {
			t.Errorf("--show-ignored missing %q:\n%s", want, shown.stdout)
		}
	}
}

func TestValidateSummaryIsUnchangedWhenNothingIsIgnored(t *testing.T) {
	configPath := suppressionFixture(t, "version: \"0.1.0\"\n", nil)
	run := runValidateCommandForTest(t, configPath, "validate")
	if !strings.Contains(run.stdout, "✅ dva.yml is valid\n") {
		t.Errorf("a config with no ignore lists must read exactly as before:\n%s", run.stdout)
	}
}

// docs/56 §6-1: --suggest-ignore prints a pasteable block and changes nothing on disk.
func TestSuggestIgnorePrintsAPasteableBlockAndDoesNotEditTheConfig(t *testing.T) {
	body := "version: \"0.1.0\"\n"
	configPath := suppressionFixture(t, body, map[string]string{
		"Makefile": "bench: ## Benchmarks\nrelease: ## Cut a release\n",
	})

	run := runValidateCommandForTest(t, configPath, "validate", "--suggest-ignore")
	if run.err != "" {
		t.Fatalf("validate failed: %s", run.err)
	}
	want := "suggestion_ignore:\n  - \"bench\"\n  - \"release\"\n"
	if !strings.Contains(run.stdout, want) {
		t.Errorf("--suggest-ignore output is not the pasteable block:\n%s", run.stdout)
	}

	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config back: %v", err)
	}
	if string(after) != body {
		t.Errorf("--suggest-ignore rewrote dva.yml; it must only print:\n%s", after)
	}
}

func TestSuggestIgnorePrintsNothingWhenThereAreNoSuggestions(t *testing.T) {
	configPath := suppressionFixture(t, "version: \"0.1.0\"\n", nil)
	run := runValidateCommandForTest(t, configPath, "validate", "--suggest-ignore")
	if strings.Contains(run.stdout, "suggestion_ignore:") {
		t.Errorf("an empty block is a paste that silences nothing and reads as advice:\n%s", run.stdout)
	}
}

// A stale pattern is a defect in dva.yml, so it is reported on an ordinary run — asking
// for --show-ignored must not be the price of learning that an ignore entry is dead.
func TestStaleIgnoreWarningsAppearWithoutAnyFlag(t *testing.T) {
	configPath := suppressionFixture(t, `version: "0.1.0"
suggestion_ignore:
  - "vanished"
`, map[string]string{
		"Makefile": "bench: ## Benchmarks\n",
	})

	run := runValidateCommandForTest(t, configPath, "validate")
	if !strings.Contains(run.stdout, `[warn] stale ignore: suggestion_ignore[0] "vanished" matches no Makefile target or package.json script`) {
		t.Errorf("stale ignore warning missing from a plain run:\n%s", run.stdout)
	}
}
