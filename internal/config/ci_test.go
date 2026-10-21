package config

import (
	"os"
	"path/filepath"
	"slices"
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
	if profile.Timeout != "10m0s" || profile.WarnAfter != "5m0s" || profile.MaxParallel != 1 || profile.ToolJobs != nil || profile.Locks == nil {
		t.Fatalf("commit defaults = timeout %q, warn_after %q, max_parallel %d, locks %#v", profile.Timeout, profile.WarnAfter, profile.MaxParallel, profile.Locks)
	}
}

func TestLoadMergesCIProfileToolJobs(t *testing.T) {
	root := `ci:
  profiles:
    full:
      timeout: 5m
      tool_jobs: 2
      steps: [{name: unit, run: "true"}]`
	module := `ci:
  profiles:
    full:
      tool_jobs: 4`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, DotDirName), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, DotDirName, "x.yml"), []byte(module), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("modules: [x]\n"+root), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.CI.Profiles["full"].ToolJobs; got == nil || *got != 4 {
		t.Fatalf("tool_jobs=%v, want 4", got)
	}
	for _, override := range []string{"description: local", "tool_jobs: 7"} {
		if err := os.WriteFile(filepath.Join(dir, "dva.override.yml"), []byte("ci:\n  profiles:\n    full:\n      "+override+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg, err := Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		want := 4
		if override == "tool_jobs: 7" {
			want = 7
		}
		if got := cfg.CI.Profiles["full"].ToolJobs; got == nil || *got != want {
			t.Fatalf("override %s got %v want %d", override, got, want)
		}
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
      locks: [root-db, root.cache]
      steps:
        - name: root
          run: go test ./root
`
	module := `ci:
  profiles:
    commit:
      description: module description
      locks: []
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
	if profile.Locks == nil || len(profile.Locks) != 0 {
		t.Fatalf("locks were not explicitly cleared: %#v", profile.Locks)
	}
}

func TestLoadCIProfileLocksRetainsWhenOmitted(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, DotDirName), 0o755); err != nil {
		t.Fatalf("make module dir: %v", err)
	}
	root := `modules: [ci]
ci:
  profiles:
    full:
      timeout: 20m
      locks: [database]
      steps: [{name: root, run: "true"}]
`
	module := `ci:
  profiles:
    full:
      description: module description
`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(root), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, DotDirName, "ci.yml"), []byte(module), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := cfg.CI.Profiles["full"].Locks; !slices.Equal(got, []string{"database"}) {
		t.Fatalf("locks = %#v, want root value retained", got)
	}
}

func TestValidateRejectsInvalidDuplicateLocksAndReservedStepEnvironment(t *testing.T) {
	for _, tc := range []struct {
		name, content, want string
	}{
		{"invalid lock", "locks: [bad/key]", "lock \"bad/key\" must match"},
		{"duplicate case sensitive lock", "locks: [database, database]", "duplicate lock \"database\""},
		{"reserved environment", "", "environment must not set reserved DVA_CI_PARENT_RUN"},
		{"reserved subproject environment", "", "environment must not set reserved DVA_CI_SUBPROJECT"},
		{"zero tool jobs", "tool_jobs: 0", "tool_jobs"},
		{"negative tool jobs", "tool_jobs: -1", "tool_jobs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stepEnvironment := ""
			switch tc.name {
			case "reserved environment":
				stepEnvironment = "\n          environment: {DVA_CI_PARENT_RUN: spoofed}"
			case "reserved subproject environment":
				stepEnvironment = "\n          environment: {DVA_CI_SUBPROJECT: spoofed}"
			}
			content := "ci:\n  profiles:\n    full:\n      timeout: 20m\n      " + tc.content + "\n      steps:\n        - name: unit\n          run: true" + stepEnvironment + "\n"
			err := loadCIConfig(t, content)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Load() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCIProfileLockNamesAreCaseSensitive(t *testing.T) {
	err := loadCIConfig(t, `ci:
  profiles:
    full:
      timeout: 20m
      locks: [database, Database]
      steps: [{name: unit, run: "true"}]
`)
	if err != nil {
		t.Fatalf("case-distinct locks rejected: %v", err)
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
