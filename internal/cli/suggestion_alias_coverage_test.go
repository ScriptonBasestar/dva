package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestSuggestionAliasCoverageSuppressesOnlyNamedDocumentedMakefileTarget(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.4.2"
interaction:
  build-app:
    runner: local
    command: go build ./cmd/app
    suggestion_aliases:
      makefile: [legacy-build]
`, map[string]string{
		"Makefile": "legacy-build: ## Legacy build\norphan: ## Needs mapping\n",
	}))

	warnings, _ := detectConfigSuggestionWarningsWithSuppressions(c)
	joined := strings.Join(warnings, "\n")
	if strings.Contains(joined, `"legacy-build"`) {
		t.Fatalf("declared Makefile alias must cover its target: %s", joined)
	}
	if !strings.Contains(joined, `"orphan"`) {
		t.Fatalf("alias must not suppress another target: %s", joined)
	}
}

func TestSuggestionAliasCoverageWarnsForUndocumentedOrUnrunnableAlias(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.4.2"
interaction:
  build-app:
    runner: local
    command: go build ./cmd/app
    suggestion_aliases:
      makefile: [gone]
  group:
    suggestion_aliases:
      makefile: [real]
`, map[string]string{
		"Makefile": "real: ## Real target\n",
	}))

	warnings, _ := detectConfigSuggestionWarningsWithSuppressions(c)
	joined := strings.Join(warnings, "\n")
	for _, want := range []string{"gone", "not a documented root Makefile target", "interaction.group.suggestion_aliases.makefile", "no runnable route"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("alias warning missing %q: %s", want, joined)
		}
	}
	if !strings.Contains(joined, `Makefile defines "real"`) {
		t.Fatalf("unrunnable interaction alias must not cover target: %s", joined)
	}
}

func TestSuggestionAliasCoverageFromImportedChildDoesNotSuppressRootMakefile(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatalf("mkdir child: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("root-target: ## Root workflow\n"), 0o644); err != nil {
		t.Fatalf("write root Makefile: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, config.FileName), []byte(`version: "0.4.2"
subprojects:
  child:
    path: child
    import:
      interactions: [child-run]
`), 0o644); err != nil {
		t.Fatalf("write root dva.yml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(child, config.FileName), []byte(`version: "0.4.2"
interaction:
  child-run:
    runner: local
    command: go build ./cmd/app
    suggestion_aliases:
      makefile: [root-target]
`), 0o644); err != nil {
		t.Fatalf("write child dva.yml: %v", err)
	}

	c, err := config.Load(root)
	if err != nil {
		t.Fatalf("load root: %v", err)
	}
	warnings, _ := detectConfigSuggestionWarningsWithSuppressions(c)
	if joined := strings.Join(warnings, "\n"); !strings.Contains(joined, `Makefile defines "root-target"`) {
		t.Fatalf("child-owned metadata must not cover root Makefile target: %s", joined)
	}
}
