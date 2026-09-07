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

			runErr := initCmd.RunE(initCmd, nil)

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
