package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestScaffoldDvaYml_ReturnsErrorWithoutCreatingConfig_whenComposeFileIsMissing(t *testing.T) {
	// Given
	dir := t.TempDir()

	// When
	created, err := scaffoldDvaYml(dir, "")

	// Then
	if created {
		t.Fatal("scaffoldDvaYml() created a config without a Compose file")
	}
	if !errors.Is(err, errComposeFileNotFound) {
		t.Fatalf("scaffoldDvaYml() error = %v, want %v", err, errComposeFileNotFound)
	}
	if !strings.Contains(err.Error(), "am run dva-discover") {
		t.Fatalf("missing Compose error should recommend discovery, got: %v", err)
	}
	if !strings.Contains(err.Error(), "am run dva-improve -p mode=rewrite") {
		t.Fatalf("missing Compose error should document explicit rewrite syntax, got: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, config.FileName)); !os.IsNotExist(statErr) {
		t.Fatalf("dva.yml was created or could not be checked: %v", statErr)
	}
}

// scriptonDashboardRootFiles is the root file set of the scripton-dashboard
// devbox as observed in docs/dogfood/scripton-dashboard.md: a pnpm workspace
// whose only package.json lives in dashboard-webui/, so the root carries no
// classic language manifest at all. Before TASK-322 this classified as
// outcomeNoDiscovery and `dva init` exited 1 without writing anything.
//
// Only mise.toml is read today — it is what makes this root classify at all.
// PORT_MAPPINGS.yaml, .gz-git.yaml and the Makefile are carried here because
// they are part of the observed shape, but nothing reads them yet and the test
// below asserts len(cfg.Stack) == 0 precisely because of that. Deriving native
// entries from those three is TASK-332, not a gap in this test.
var scriptonDashboardRootFiles = map[string]string{
	"mise.toml": `[tools]
node = "24"
pnpm = "10"
"npm:typescript-language-server" = "5.3.0"
"npm:svelte-language-server" = "0.18.3"

[settings]
env_shell_expand = false
`,
	"PORT_MAPPINGS.yaml": `project: scripton-dashboard
services:
  application:
    dashboard:
      port: 11600
      env: DASHBOARD_PORT
`,
	".gz-git.yaml": `kind: workspace
workspaces:
  scripton-ui-components:
    targetPath: scripton-ui-components
`,
	"Makefile": `.PHONY: dev-dashboard
dev-dashboard: ## Run the dashboard host
	cd dashboard-webui && pnpm dev
`,
	".env.example":                 "DASHBOARD_PORT=11600\n",
	"README.md":                    "# scripton-dashboard\n",
	"dashboard-webui/package.json": `{"name":"dashboard-webui"}`,
}

func writeFixture(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		target := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", target, err)
		}
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			t.Fatalf("write %s: %v", target, err)
		}
	}
}

// TestClassifyDiscovery_ScriptonDashboardRootIsNativeOnly is TASK-322
// completion criterion 1: the observed scripton-dashboard root must classify as
// native-only and produce a valid, stack-less dva.yml instead of exiting 1.
func TestClassifyDiscovery_ScriptonDashboardRootIsNativeOnly(t *testing.T) {
	// Given
	dir := t.TempDir()
	writeFixture(t, dir, scriptonDashboardRootFiles)

	// When
	outcome, composeFiles, nativeLang := classifyDiscovery(dir)

	// Then
	if outcome != outcomeNativeOnly {
		t.Fatalf("classifyDiscovery() outcome = %v, want outcomeNativeOnly", outcome)
	}
	if len(composeFiles) != 0 {
		t.Fatalf("scripton-dashboard root has no Compose file, got %v", composeFiles)
	}
	if nativeLang != "node" {
		t.Fatalf("classifyDiscovery() lang = %q, want %q (mise.toml [tools] node)", nativeLang, "node")
	}

	created, err := scaffoldDvaYml(dir, "")
	if err != nil || !created {
		t.Fatalf("scaffoldDvaYml() = (%v, %v), want (true, nil)", created, err)
	}
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if len(cfg.Stack) != 0 {
		t.Fatalf("native-only output must not guess a stack entry, got: %+v", cfg.Stack)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("generated config does not validate: %v", err)
	}
}

func TestDetectNativeMarkerIn(t *testing.T) {
	tests := []struct {
		name     string
		files    map[string]string
		wantLang string
		wantOK   bool
	}{
		{
			name:     "go.work workspace root without go.mod",
			files:    map[string]string{"go.work": "go 1.25\n\nuse ./cmd\n"},
			wantLang: "go",
			wantOK:   true,
		},
		{
			name:     "mise.toml declaring node",
			files:    map[string]string{"mise.toml": "[tools]\nnode = \"24\"\n"},
			wantLang: "node",
			wantOK:   true,
		},
		{
			name:     "asdf .tool-versions",
			files:    map[string]string{".tool-versions": "# pinned\npython 3.13.1\n"},
			wantLang: "python",
			wantOK:   true,
		},
		{
			name:     "mise.toml with only backend-prefixed tools is not language evidence",
			files:    map[string]string{"mise.toml": "[tools]\n\"npm:typescript\" = \"5.9.3\"\n"},
			wantLang: "",
			wantOK:   false,
		},
		{
			name:     "a language name outside the [tools] table is not evidence",
			files:    map[string]string{"mise.toml": "[env]\nnode = \"whatever\"\n"},
			wantLang: "",
			wantOK:   false,
		},
		{
			name:     "classic manifest still wins",
			files:    map[string]string{"go.mod": "module x\n", "mise.toml": "[tools]\nnode = \"24\"\n"},
			wantLang: "go",
			wantOK:   true,
		},
		{
			name:     "[tools] header with a trailing comment",
			files:    map[string]string{"mise.toml": "[tools] # runtimes\nnode = \"24\"\n"},
			wantLang: "node",
			wantOK:   true,
		},
		{
			name:     "[tools] header with inner spacing",
			files:    map[string]string{"mise.toml": "[ tools ]\nruby = \"3.4\"\n"},
			wantLang: "rails",
			wantOK:   true,
		},
		{
			name:     "TOML dotted key before any table header",
			files:    map[string]string{"mise.toml": "tools.node = \"24\"\n"},
			wantLang: "node",
			wantOK:   true,
		},
		{
			name:     "a dotted key inside another table is not a tools pin",
			files:    map[string]string{"mise.toml": "[settings]\ntools.node = \"24\"\n"},
			wantLang: "",
			wantOK:   false,
		},
		{
			name:     "no evidence at all",
			files:    map[string]string{"README.md": "# x\n"},
			wantLang: "",
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixture(t, dir, tc.files)

			lang, ok := detectNativeMarkerIn(dir)

			if lang != tc.wantLang || ok != tc.wantOK {
				t.Fatalf("detectNativeMarkerIn() = (%q, %v), want (%q, %v)", lang, ok, tc.wantLang, tc.wantOK)
			}
		})
	}
}

// TestInitRecursive_ScansSubprojectsWithoutRootEvidence is TASK-322 completion
// criterion 2: --recursive advertises a sub-project scan, so a root with no
// Compose file and no language manifest must not abort before the scan.
func TestInitRecursive_ScansSubprojectsWithoutRootEvidence(t *testing.T) {
	savedTemplate, savedRecursive, savedDevcontainer, savedAll := initTemplate, initRecursive, initDevcontainer, initAll
	t.Cleanup(func() {
		initTemplate, initRecursive, initDevcontainer, initAll = savedTemplate, savedRecursive, savedDevcontainer, savedAll
	})

	tests := []struct {
		name       string
		files      map[string]string
		recursive  bool
		wantErr    bool
		wantCreate []string
		wantAbsent []string
	}{
		{
			name: "sub-projects are scaffolded even though the root has no evidence",
			files: map[string]string{
				"README.md":                "# workspace\n",
				"webui/package.json":       `{"name":"webui"}`,
				"api/go.mod":               "module example.com/api\n",
				"infra/docker-compose.yml": "services: {}\n",
			},
			recursive:  true,
			wantErr:    false,
			wantCreate: []string{"webui/dva.yml", "api/dva.yml"},
			// The root itself still has no verified evidence, so nothing is written there.
			// infra/ is a config directory the scan deliberately skips.
			wantAbsent: []string{"dva.yml", "infra/dva.yml"},
		},
		{
			name: "a go.work member workspace is a recognized sub-project",
			files: map[string]string{
				"README.md":     "# workspace\n",
				"tools/go.work": "go 1.25\n",
			},
			recursive:  true,
			wantErr:    false,
			wantCreate: []string{"tools/dva.yml"},
			wantAbsent: []string{"dva.yml"},
		},
		{
			// Regression: counting "no error" instead of "created" let a
			// leftover sub-project dva.yml from an earlier run stand in for
			// progress, so a still-unusable root exited 0.
			name: "a pre-existing sub-project dva.yml is not progress",
			files: map[string]string{
				"README.md":          "# workspace\n",
				"webui/package.json": `{"name":"webui"}`,
				"webui/dva.yml":      "version: \"0.1.0\"\n",
			},
			recursive:  true,
			wantErr:    true,
			wantAbsent: []string{"dva.yml"},
		},
		{
			name:       "an evidence-less tree still fails, with nothing written",
			files:      map[string]string{"README.md": "# empty\n"},
			recursive:  true,
			wantErr:    true,
			wantAbsent: []string{"dva.yml"},
		},
		{
			name: "without --recursive an evidence-less root still fails",
			files: map[string]string{
				"README.md":          "# workspace\n",
				"webui/package.json": `{"name":"webui"}`,
			},
			recursive:  false,
			wantErr:    true,
			wantAbsent: []string{"dva.yml", "webui/dva.yml"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixture(t, dir, tc.files)

			origWd, err := os.Getwd()
			if err != nil {
				t.Fatalf("getwd: %v", err)
			}
			if err := os.Chdir(dir); err != nil {
				t.Fatalf("chdir: %v", err)
			}
			t.Cleanup(func() { _ = os.Chdir(origWd) })

			initTemplate, initRecursive, initDevcontainer, initAll = "", tc.recursive, false, false

			stderr, runErr := captureCommandStderr(t, func() error {
				return initCmd.RunE(initCmd, nil)
			})

			// The refusal body is emitted exactly once. When RunE returns the
			// error, root.go prints it, so RunE must not have printed it too —
			// otherwise the user sees the same multi-line block twice.
			const refusalBody = "dva.yml was not created"
			if runErr != nil && strings.Contains(stderr, refusalBody) {
				t.Errorf("RunE returned an error and also printed its body; root.go prints it again.\nstderr:\n%s", stderr)
			}
			if runErr == nil && tc.recursive && !tc.wantErr && len(tc.wantCreate) > 0 &&
				!strings.Contains(stderr, refusalBody) {
				t.Errorf("a refused root that the scan rescued should still explain itself once.\nstderr:\n%s", stderr)
			}

			if tc.wantErr && runErr == nil {
				t.Fatal("initCmd.RunE() = nil, want an error")
			}
			if !tc.wantErr && runErr != nil {
				t.Fatalf("initCmd.RunE() = %v, want nil", runErr)
			}
			for _, rel := range tc.wantCreate {
				if _, statErr := os.Stat(filepath.Join(dir, rel)); statErr != nil {
					t.Errorf("expected %s to be scaffolded: %v", rel, statErr)
				}
			}
			for _, rel := range tc.wantAbsent {
				if _, statErr := os.Stat(filepath.Join(dir, rel)); !os.IsNotExist(statErr) {
					t.Errorf("%s must not be written, stat err = %v", rel, statErr)
				}
			}
		})
	}
}

// TestHybridTemplateSelection covers what the hybrid path generates, and pins
// the evidence grade that is allowed to decide it.
//
// Two defects live here. classifyDiscovery and detectTemplateIn once carried
// separate copies of the manifest table, so a go.work root was announced as
// "a go project manifest" and then generated the "minimal" template. Closing
// that by sharing detectNativeMarkerIn then over-corrected: it let a mise /
// .tool-versions pin pick a template, so a repo pinning python only for its
// pre-commit hooks would get `python manage.py` and `pip` generated against its
// compose service. Direct manifests and go.work select a template; a tool pin
// classifies and words the output but must never select one.
func TestHybridTemplateSelection(t *testing.T) {
	tests := []struct {
		name  string
		files map[string]string
		// wantLang is what classifyDiscovery announces to the user.
		wantLang string
		// wantTmpl is what the generator is allowed to author from it.
		wantTmpl  string
		wantHas   string
		wantLacks string
	}{
		{
			name:     "go.work workspace root with a Compose file selects the go template",
			files:    map[string]string{"go.work": "go 1.25\n", "docker-compose.yml": "services: {}\n"},
			wantLang: "go",
			wantTmpl: "go",
			wantHas:  "go test ./...",
		},
		{
			name:     "a direct manifest is unaffected",
			files:    map[string]string{"Gemfile": "source 'x'\n", "docker-compose.yml": "services: {}\n"},
			wantLang: "rails",
			wantTmpl: "rails",
			wantHas:  "bundle exec rspec",
		},
		{
			// The reviewer's case: python pinned for pre-commit hooks only.
			name:      "a .tool-versions pin must not select a template",
			files:     map[string]string{".tool-versions": "python 3.12.0\n", "docker-compose.yml": "services: {}\n"},
			wantLang:  "python",
			wantTmpl:  "minimal",
			wantHas:   "/bin/bash",
			wantLacks: "python manage.py",
		},
		{
			name:      "a mise.toml pin must not select a template",
			files:     map[string]string{"mise.toml": "[tools]\nnode = \"24\"\n", "docker-compose.yml": "services: {}\n"},
			wantLang:  "node",
			wantTmpl:  "minimal",
			wantHas:   "/bin/bash",
			wantLacks: "npm run dev",
		},
		{
			name:     "a Compose file with no manifest at all is still minimal",
			files:    map[string]string{"docker-compose.yml": "services: {}\n"},
			wantLang: "",
			wantTmpl: "minimal",
			wantHas:  "/bin/bash",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFixture(t, dir, tc.files)

			outcome, _, nativeLang := classifyDiscovery(dir)
			wantOutcome := outcomeHybrid
			if tc.wantLang == "" {
				wantOutcome = outcomeComposeOnly
			}
			if outcome != wantOutcome {
				t.Fatalf("classifyDiscovery() outcome = %v, want %v", outcome, wantOutcome)
			}
			if nativeLang != tc.wantLang {
				t.Fatalf("classifyDiscovery() lang = %q, want %q", nativeLang, tc.wantLang)
			}

			if tmpl := detectTemplateIn(dir); tmpl != tc.wantTmpl {
				t.Fatalf("detectTemplateIn() = %q, want %q (announced %q)", tmpl, tc.wantTmpl, nativeLang)
			}

			created, err := scaffoldDvaYml(dir, "")
			if err != nil || !created {
				t.Fatalf("scaffoldDvaYml() = (%v, %v), want (true, nil)", created, err)
			}
			data, err := os.ReadFile(filepath.Join(dir, config.FileName))
			if err != nil {
				t.Fatalf("read generated config: %v", err)
			}
			if !strings.Contains(string(data), tc.wantHas) {
				t.Errorf("generated config should carry the %q template (%q), got:\n%s", tc.wantTmpl, tc.wantHas, data)
			}
			if tc.wantLacks != "" && strings.Contains(string(data), tc.wantLacks) {
				t.Errorf("a tool pin must not author %q — it says a runtime is available, not that the repo is written in it. Got:\n%s", tc.wantLacks, data)
			}
			if !strings.Contains(string(data), "stack:") {
				t.Errorf("hybrid output must still generate the verified Compose stack, got:\n%s", data)
			}
		})
	}
}
