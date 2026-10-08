package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestDetectConfigSuggestionWarnings_FromMakefileAndPackageJSON(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	makefile := "build: ## Build project\nlint: ## Run lint\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	packageJSON := `{"scripts":{"dev":"vite","test":"vitest","pretest":"echo pre"}}`
	if err := os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644); err != nil {
		t.Fatalf("write package.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte("version: \"0.1.0\"\ninteraction:\n  build:\n    runner: local\n    command: make build\n"), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	warnings := detectConfigSuggestionWarnings(c)
	if len(warnings) == 0 {
		t.Fatal("expected suggestion warnings")
	}
	joined := strings.Join(warnings, "\n")
	if !strings.Contains(joined, `Makefile defines "lint"`) {
		t.Fatalf("expected Makefile suggestion, got: %s", joined)
	}
	if !strings.Contains(joined, `package.json defines "dev"`) || !strings.Contains(joined, `package.json defines "test"`) {
		t.Fatalf("expected package.json suggestions, got: %s", joined)
	}
	if strings.Contains(joined, "pretest") {
		t.Fatalf("pre/post scripts should be ignored, got: %s", joined)
	}
}

func TestShouldIgnoreMakefileTarget(t *testing.T) {
	// Exact matches: DVA reserved commands and meta targets — plus `ps` and `clean`, which
	// are ignored on their own merit and not because a built-in shares the name. `clean`
	// stayed on this list when the built-in was removed; see the grouping in validate.go.
	exactIgnored := []string{
		"help", "all", "default",
		"stop", "up", "down", "restart",
		"ps", "run", config.LogsDirName, "build", "clean",
		"infra-up", "infra-down", "infra-start", "infra-stop",
		"deps", "install", "prepare", "setup", "install-hooks",
		"docs", "docs-build", "docs-serve",
	}
	for _, name := range exactIgnored {
		if !shouldIgnoreMakefileTarget(name) {
			t.Errorf("expected %q to be ignored (exact match)", name)
		}
	}

	// Suffix patterns: compose lifecycle
	suffixIgnored := []string{
		"dev-full-up", "dev-full-down", "dev-full-logs", "dev-full-ps",
		"dev-full-build",
		"e2e-up", "e2e-down", "e2e-stop", "e2e-restart",
		"app-logs", "backend-ps", "frontend-build",
	}
	for _, name := range suffixIgnored {
		if !shouldIgnoreMakefileTarget(name) {
			t.Errorf("expected %q to be ignored (suffix pattern)", name)
		}
	}

	// Should NOT be ignored: legitimate development workflow targets
	kept := []string{
		"build-ce", "build-ee", "build-mirror", "build-all",
		"test-ce", "test-all", "test-cloud",
		"e2e-smoke", "e2e-full", "e2e-rust",
		"clippy", "clippy-all", "fmt-check", "check-all",
		"lint", "dev", "test",
	}
	for _, name := range kept {
		if shouldIgnoreMakefileTarget(name) {
			t.Errorf("expected %q to be kept, but was ignored", name)
		}
	}
}

func TestIsDVAWrapperRecipe(t *testing.T) {
	wrappers := [][]string{
		{"dva up"},
		{"dva down"},
		{"dva up", "dva status"},
	}
	for _, recipe := range wrappers {
		if !isDVAWrapperRecipe(recipe) {
			t.Errorf("expected %v to be DVA wrapper", recipe)
		}
	}

	notWrappers := [][]string{
		{},
		{"go build ./..."},
		{"dva up", "echo done"},
		{"cargo test"},
		{"docker compose up -d"},
	}
	for _, recipe := range notWrappers {
		if isDVAWrapperRecipe(recipe) {
			t.Errorf("expected %v NOT to be DVA wrapper", recipe)
		}
	}
}

func TestMatchIgnorePattern(t *testing.T) {
	patterns := []string{"*-release", "clippy*", "test-e2e-*"}

	matches := []string{"build-ce-release", "clippy", "clippy-all", "test-e2e-ci", "test-e2e-dev"}
	for _, name := range matches {
		if _, matched := matchIgnorePattern(name, patterns); !matched {
			t.Errorf("expected %q to match suggestion_ignore patterns", name)
		}
	}

	noMatches := []string{"build-ce", "test-ce", "e2e-smoke", "clipboard", "lint"}
	for _, name := range noMatches {
		if _, matched := matchIgnorePattern(name, patterns); matched {
			t.Errorf("expected %q NOT to match suggestion_ignore patterns", name)
		}
	}

	// Empty patterns never match
	if _, matched := matchIgnorePattern("anything", nil); matched {
		t.Error("empty patterns should never match")
	}
}

func TestDetectConfigSuggestionWarnings_SubcommandCoverage(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	makefile := "build-ce: ## Build CE edition\nbuild-ee: ## Build EE edition\norphan: ## No coverage\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	// cargo:build has subcommands ce and ee — should suppress build-ce and build-ee warnings
	dvaYml := "version: \"0.1.0\"\ninteraction:\n  cargo:build:\n    runner: local\n    command: cargo build\n    subcommands:\n      ce:\n        command: cargo build --features ce\n      ee:\n        command: cargo build --features ee\n"
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(dvaYml), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	warnings := detectConfigSuggestionWarnings(c)
	joined := strings.Join(warnings, "\n")
	if strings.Contains(joined, `"build-ce"`) {
		t.Errorf("build-ce should be suppressed by subcommand coverage, got: %s", joined)
	}
	if strings.Contains(joined, `"build-ee"`) {
		t.Errorf("build-ee should be suppressed by subcommand coverage, got: %s", joined)
	}
	if !strings.Contains(joined, `"orphan"`) {
		t.Errorf("orphan should still warn (not covered), got: %s", joined)
	}
}

func TestDetectConfigSuggestionWarnings_SuggestionIgnore(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	makefile := "build-ce-release: ## Release build\nclippy: ## Lint\norphan: ## No coverage\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	dvaYml := "version: \"0.1.0\"\nsuggestion_ignore:\n  - \"*-release\"\n  - \"clippy*\"\n"
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(dvaYml), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	warnings := detectConfigSuggestionWarnings(c)
	joined := strings.Join(warnings, "\n")
	if strings.Contains(joined, `"build-ce-release"`) {
		t.Errorf("build-ce-release should be suppressed by suggestion_ignore, got: %s", joined)
	}
	if strings.Contains(joined, `"clippy"`) {
		t.Errorf("clippy should be suppressed by suggestion_ignore, got: %s", joined)
	}
	if !strings.Contains(joined, `"orphan"`) {
		t.Errorf("orphan should still warn (not ignored), got: %s", joined)
	}
}

func TestShouldIgnorePackageScript(t *testing.T) {
	for _, name := range []string{"pretest", "postinstall", "prepare"} {
		if !shouldIgnorePackageScript(name) {
			t.Fatalf("expected %q to be ignored", name)
		}
	}
	for _, name := range []string{"test", "dev", "build"} {
		if shouldIgnorePackageScript(name) {
			t.Fatalf("expected %q to be kept", name)
		}
	}
}

// TestDetectConfigSuggestionWarnings_MultiTargetMakefileLine pins the parser against Make's
// own rule that one recipe may serve several targets. Before TASK-320 the left side of
// `a b: ## desc` was taken whole, so the suggestion named a target spelled "a b" — which no
// interaction can be named after — and never named either real target.
func TestDetectConfigSuggestionWarnings_MultiTargetMakefileLine(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	makefile := "log-search-bench perf-log-search: ## Run the log search benchmark\nsolo: ## Single target still works\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte("version: \"0.1.0\"\n"), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	joined := strings.Join(detectConfigSuggestionWarnings(c), "\n")
	if strings.Contains(joined, `"log-search-bench perf-log-search"`) {
		t.Errorf("the two targets must not be reported as one name, got: %s", joined)
	}
	for _, want := range []string{`"log-search-bench"`, `"perf-log-search"`, `"solo"`} {
		if !strings.Contains(joined, want) {
			t.Errorf("expected suggestion for %s, got: %s", want, joined)
		}
	}
}

// TestMakeIncludePaths pins Make's include syntax: a trailing comment is not part of the
// path, and one directive may name several files. cwrapper's
// `include .make/env.mk  # secrets` hid every env-* target before this.
func TestMakeIncludePaths(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"a.mk", "b.mk"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x: ## x\n"), 0644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	got, ok := makeIncludePaths("include a.mk   # secrets — do not edit", dir)
	if !ok || len(got) != 1 || got[0] != filepath.Join(dir, "a.mk") {
		t.Errorf("trailing comment must be dropped, got %v (ok=%v)", got, ok)
	}
	got, _ = makeIncludePaths("-include a.mk b.mk", dir)
	if len(got) != 2 || got[1] != filepath.Join(dir, "b.mk") {
		t.Errorf("each space-separated file must be followed, got %v", got)
	}
	got, _ = makeIncludePaths("include *.mk", dir)
	if len(got) != 2 {
		t.Errorf("a glob must expand to every match, got %v", got)
	}
	if _, ok := makeIncludePaths("includes: ## a target, not a directive", dir); ok {
		t.Error("a target named includes is not an include directive")
	}
}

func TestDocumentedTargetsFollowCommentedInclude(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".make"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".make", "env.mk"), []byte("env-edit: ## Edit secrets\n"), 0644); err != nil {
		t.Fatalf("write env.mk: %v", err)
	}
	makefile := "include .make/env.mk          # secrets (SOPS) — SSoT copy\nbench: ## Benchmarks\n"
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	got := strings.Join(allDocumentedMakefileTargetNamesInDir(dir), ",")
	if got != "bench,env-edit" {
		t.Errorf("targets from a commented include must be collected, got %q", got)
	}
}

func TestImportedLeafNameDoesNotSuppressUnroutableTarget(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	makefile := "test: ## Run every suite\ndeploy: ## Deploy the whole stack\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	for name, childYml := range map[string]string{
		"frontend": "version: \"0.1.0\"\ninteraction:\n  test:\n    runner: local\n    command: pnpm test\n  deploy:\n    runner: local\n    command: pnpm deploy\n",
		"backend":  "version: \"0.1.0\"\ninteraction:\n  test:\n    runner: local\n    command: go test ./...\n",
	} {
		childDir := filepath.Join(tmpDir, name)
		if err := os.MkdirAll(childDir, 0755); err != nil {
			t.Fatalf("mkdir %s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(childDir, config.FileName), []byte(childYml), 0644); err != nil {
			t.Fatalf("write %s dva.yml: %v", name, err)
		}
	}

	parentYml := "version: \"0.1.0\"\nsubprojects:\n  frontend:\n    path: frontend\n    import:\n      interactions:\n        - name: test\n        - name: deploy\n  backend:\n    path: backend\n    import:\n      interactions:\n        - name: test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(parentYml), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	joined := strings.Join(detectConfigSuggestionWarnings(c), "\n")
	for _, target := range []string{"test", "deploy"} {
		if !strings.Contains(joined, strconv.Quote(target)) {
			t.Errorf("imported leaf %q must remain a root suggestion, got: %s", target, joined)
		}
	}
	for _, want := range []string{"frontend", "backend", "as: test", "as: deploy"} {
		if !strings.Contains(joined, want) {
			t.Errorf("warning must give an actionable alias hint containing %q, got: %s", want, joined)
		}
	}
}

func TestAliasedImportSuppressesTheRootTarget(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte("test: ## Run tests\n"), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	childDir := filepath.Join(tmpDir, "frontend")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if err := os.WriteFile(filepath.Join(childDir, config.FileName), []byte("version: \"0.1.0\"\ninteraction:\n  test:\n    runner: local\n    command: pnpm test\n"), 0644); err != nil {
		t.Fatalf("write child dva.yml: %v", err)
	}
	parentYml := "version: \"0.1.0\"\nsubprojects:\n  frontend:\n    path: frontend\n    import:\n      interactions:\n        - name: test\n          as: test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(parentYml), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := c.Interaction["test"]; !ok {
		t.Fatal("as: test must create a root interaction key")
	}
	if joined := strings.Join(detectConfigSuggestionWarnings(c), "\n"); strings.Contains(joined, `"test"`) {
		t.Errorf("aliased root interaction should suppress make test, got: %s", joined)
	}
}

func TestPathStyleMakefileTargetStaysCovered(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte("frontend/test-e2e: ## End to end\n"), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}
	childDir := filepath.Join(tmpDir, "frontend")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	childYml := "version: \"0.1.0\"\ninteraction:\n  test:\n    runner: local\n    command: pnpm test\n    subcommands:\n      e2e:\n        command: pnpm test:e2e\n"
	if err := os.WriteFile(filepath.Join(childDir, config.FileName), []byte(childYml), 0644); err != nil {
		t.Fatalf("write child dva.yml: %v", err)
	}
	parentYml := "version: \"0.1.0\"\nsubprojects:\n  frontend:\n    path: frontend\n    import:\n      interactions:\n        - name: test\n"
	if err := os.WriteFile(filepath.Join(tmpDir, config.FileName), []byte(parentYml), 0644); err != nil {
		t.Fatalf("write dva.yml: %v", err)
	}

	c, err := config.Load(".")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if joined := strings.Join(detectConfigSuggestionWarnings(c), "\n"); strings.Contains(joined, `"frontend/test-e2e"`) {
		t.Errorf("path-style target should be covered by frontend/test e2e, got: %s", joined)
	}
}

func TestDotPrefixedTargetIgnoredAfterMultiTargetSplit(t *testing.T) {
	tmpDir := t.TempDir()
	makefile := "foo .bar $(BIN) %pattern: ## Mixed targets\nexport DOCKER_BUILDKIT := 1 ## Build setting\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "Makefile"), []byte(makefile), 0644); err != nil {
		t.Fatalf("write Makefile: %v", err)
	}

	targets := extractDocumentedMakefileTargetNamesInDir(tmpDir)
	if len(targets) != 1 || targets[0] != "foo" {
		t.Fatalf("target filtering after split = %v, want [foo]", targets)
	}
}
