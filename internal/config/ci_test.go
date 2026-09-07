package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCIProfileCommitDefaults(t *testing.T) {
	cfg := &Config{CI: &CIConfig{Profiles: map[string]CIProfile{
		"commit": {
			Steps: []CIStep{{Name: "unit", Run: "go test ./..."}},
		},
	}}}

	profile, err := cfg.ResolveCIProfile("")
	if err != nil {
		t.Fatalf("ResolveCIProfile(\"\") error: %v", err)
	}
	if profile.Timeout != "10m0s" || profile.WarnAfter != "5m0s" || profile.MaxParallel != 1 {
		t.Fatalf("commit defaults = timeout %q, warn_after %q, max_parallel %d", profile.Timeout, profile.WarnAfter, profile.MaxParallel)
	}
}

func TestLoadMergesCIProfileFieldsAndReplacesSteps(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, DotDirName), 0o755); err != nil {
		t.Fatalf("make module dir: %v", err)
	}
	root := `modules: [ci]
ci:
  profiles:
    commit:
      description: root description
      timeout: 8m
      warn_after: 4m
      max_parallel: 2
      steps:
        - name: root
          run: go test ./root
`
	module := `ci:
  profiles:
    commit:
      description: module description
      steps:
        - name: module
          run: go test ./module
`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(root), 0o644); err != nil {
		t.Fatalf("write root config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, DotDirName, "ci.yml"), []byte(module), 0o644); err != nil {
		t.Fatalf("write module config: %v", err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	profile := cfg.CI.Profiles["commit"]
	if profile.Description != "module description" || profile.Timeout != "8m" || profile.MaxParallel != 2 {
		t.Fatalf("merged profile = %#v", profile)
	}
	if len(profile.Steps) != 1 || profile.Steps[0].Name != "module" {
		t.Fatalf("steps were not replaced: %#v", profile.Steps)
	}
}

func TestLoadRejectsCIStepDependencyCycle(t *testing.T) {
	err := loadCIConfig(t, `ci:
  profiles:
    full:
      timeout: 20m
      steps:
        - name: lint
          run: make lint
          depends_on: [test]
        - name: test
          run: make test
          depends_on: [lint]
`)
	if err == nil || !strings.Contains(err.Error(), "dependencies contain a cycle") {
		t.Fatalf("Load() error = %v, want dependency-cycle error", err)
	}
}

func TestValidateRejectsCIProfileBudgetAndUnknownSchemaField(t *testing.T) {
	dir := t.TempDir()
	content := `ci:
  profiles:
    commit:
      timeout: 11m
      steps:
        - name: unit
          run: go test ./...
      unknown: true
`
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := loadFile(path)
	if err != nil {
		t.Fatalf("loadFile() error: %v", err)
	}
	cfg.filePath = path
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "Additional property unknown is not allowed") || !strings.Contains(err.Error(), "timeout must not exceed") {
		t.Fatalf("Validate() error = %v, want schema and budget errors", err)
	}
}

func loadCIConfig(t *testing.T, content string) error {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	_, err := Load(dir)
	return err
}
