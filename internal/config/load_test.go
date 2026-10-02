package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindConfigWalksUp(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	subDir := filepath.Join(projectDir, "src", "deep")
	os.MkdirAll(subDir, 0755)

	// Write dva.yml in project root
	dvaYml := filepath.Join(projectDir, FileName)
	os.WriteFile(dvaYml, []byte("version: '0.1.0'\n"), 0644)

	// Find from deep subdir
	found, err := findConfig(subDir)
	if err != nil {
		t.Fatalf("findConfig(%s) error: %v", subDir, err)
	}
	if found != dvaYml {
		t.Errorf("findConfig(%s) = %s, want %s", subDir, found, dvaYml)
	}
}

func TestFindConfigNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	_, err := findConfig(tmpDir)
	if err == nil {
		t.Error("findConfig should fail when no dva.yml exists")
	}
}

func TestFindConfigDVAFILE(t *testing.T) {
	tmpDir := t.TempDir()
	customFile := filepath.Join(tmpDir, "custom.yml")
	os.WriteFile(customFile, []byte("version: '0.1.0'\n"), 0644)

	t.Setenv(EnvFileKey, customFile)

	found, err := findConfig(tmpDir)
	if err != nil {
		t.Fatalf("findConfig with DVA_FILE error: %v", err)
	}
	if found != customFile {
		t.Errorf("got %s, want %s", found, customFile)
	}
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `version: "0.1.0"
stack:
  compose:
    default_runner: compose
    order: 10
    runners:
      compose:
        files:
          - docker-compose.yml
        project_name: myapp

environment:
  RAILS_ENV: development
  NODE_ENV: development

interaction:
  shell:
    description: "Open shell"
    service: app
    command: /bin/bash
  test:
    description: "Run tests"
    service: app
    command: bundle exec rspec
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Version != "0.1.0" {
		t.Errorf("version = %s, want 0.1.0", cfg.Version)
	}
	if cfg.ComposeProjectName() != "myapp" {
		t.Errorf("project_name = %s, want myapp", cfg.ComposeProjectName())
	}
	files := cfg.AllComposeFiles()
	if len(files) != 1 || files[0] != "docker-compose.yml" {
		t.Errorf("compose.files = %v, want [docker-compose.yml]", files)
	}
	if len(cfg.Interaction) != 2 {
		t.Errorf("interaction count = %d, want 2", len(cfg.Interaction))
	}
	if cfg.Interaction["shell"].Command != "/bin/bash" {
		t.Errorf("shell command = %s, want /bin/bash", cfg.Interaction["shell"].Command)
	}
	if len(cfg.Environment) != 2 {
		t.Errorf("environment count = %d, want 2", len(cfg.Environment))
	}
}

func TestLoadConfigWithModules(t *testing.T) {
	tmpDir := t.TempDir()
	dvaDir := filepath.Join(tmpDir, DotDirName)
	os.MkdirAll(dvaDir, 0755)

	// Main config with module reference
	os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`
modules:
  - extra

interaction:
  shell:
    description: "Open shell"
    service: app
    command: /bin/bash
`), 0644)

	// Module file
	os.WriteFile(filepath.Join(dvaDir, "extra.yml"), []byte(`
interaction:
  test:
    description: "Run tests"
    service: app
    command: bundle exec rspec
`), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// Should have both shell and test commands
	if len(cfg.Interaction) != 2 {
		t.Errorf("interaction count = %d, want 2 (merged module)", len(cfg.Interaction))
	}
	if _, ok := cfg.Interaction["test"]; !ok {
		t.Error("module command 'test' not found after merge")
	}
}

func TestLoadConfigWithOverride(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`
stack:
  compose:
    default_runner: compose
    order: 10
    runners:
      compose:
        project_name: original
interaction:
  shell:
    service: app
    command: /bin/bash
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "dva.override.yml"), []byte(`
stack:
  compose-override:
    default_runner: compose
    order: 20
    runners:
      compose:
        project_name: overridden
`), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	// "compose" (order 10) is primary over "compose-override" (order 20)
	if cfg.ComposeProjectName() != "original" {
		t.Errorf("project_name = %s, want original (lower order is primary)", cfg.ComposeProjectName())
	}
	if len(cfg.Stack) != 2 {
		t.Errorf("stack entries = %d, want 2 (merged)", len(cfg.Stack))
	}
}

// TestLoad_ModuleIncompatibleVersion_Fails locks TASK-042: version: on a module
// must refuse Load the same way root version does (not silently dropped).
func TestLoad_ModuleIncompatibleVersion_Fails(t *testing.T) {
	// Given: root is compatible; module requires a future DVA
	tmpDir := t.TempDir()
	dvaDir := filepath.Join(tmpDir, DotDirName)
	if err := os.MkdirAll(dvaDir, 0o755); err != nil {
		t.Fatalf("mkdir modules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`
version: "0.1.0"
modules:
  - extra
interaction:
  shell:
    command: echo root
`), 0o644); err != nil {
		t.Fatalf("write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dvaDir, "extra.yml"), []byte(`
version: "99.0.0"
interaction:
  from_module:
    command: echo HELLO_FROM_MODULE
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	// When
	_, err := Load(tmpDir)

	// Then
	if err == nil {
		t.Fatal("Load() error = nil, want version gate failure for module")
	}
	msg := err.Error()
	if !strings.Contains(msg, "99.0.0") {
		t.Errorf("error = %q, want required version 99.0.0", msg)
	}
	if !strings.Contains(msg, "extra") {
		t.Errorf("error = %q, want module name extra", msg)
	}
}

func TestLoad_ModuleIncompatibleVersion_SkipVersionCheck(t *testing.T) {
	// Given: same incompatible module as the fail case
	tmpDir := t.TempDir()
	dvaDir := filepath.Join(tmpDir, DotDirName)
	if err := os.MkdirAll(dvaDir, 0o755); err != nil {
		t.Fatalf("mkdir modules dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`
version: "0.1.0"
modules:
  - extra
`), 0o644); err != nil {
		t.Fatalf("write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dvaDir, "extra.yml"), []byte(`
version: "99.0.0"
interaction:
  from_module:
    command: echo ok
`), 0o644); err != nil {
		t.Fatalf("write module: %v", err)
	}

	// When: escape hatch used (e.g. config improve)
	cfg, err := Load(tmpDir, SkipVersionCheck())

	// Then
	if err != nil {
		t.Fatalf("Load(SkipVersionCheck) error = %v, want nil", err)
	}
	if _, ok := cfg.Interaction["from_module"]; !ok {
		t.Error("module interaction not merged under SkipVersionCheck")
	}
}

func TestLoad_OverrideIncompatibleVersion_Fails(t *testing.T) {
	// Given: root ok; override requires future DVA
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`
version: "0.1.0"
interaction:
  shell:
    command: echo root
`), 0o644); err != nil {
		t.Fatalf("write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "dva.override.yml"), []byte(`
version: "99.0.0"
interaction:
  from_override:
    command: echo override
`), 0o644); err != nil {
		t.Fatalf("write override: %v", err)
	}

	// When
	_, err := Load(tmpDir)

	// Then
	if err == nil {
		t.Fatal("Load() error = nil, want version gate failure for override")
	}
	if !strings.Contains(err.Error(), "99.0.0") {
		t.Errorf("error = %q, want required version 99.0.0", err)
	}
}

func TestLoad_SubprojectIncompatibleVersion_Fails(t *testing.T) {
	// Given: root imports a subproject whose dva.yml requires future DVA
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "backend")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`
version: "0.1.0"
subprojects:
  backend:
    path: backend
    import:
      interactions:
        - name: shell
`), 0o644); err != nil {
		t.Fatalf("write root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subDir, FileName), []byte(`
version: "99.0.0"
interaction:
  shell:
    command: echo sub
`), 0o644); err != nil {
		t.Fatalf("write sub: %v", err)
	}

	// When
	_, err := Load(tmpDir)

	// Then
	if err == nil {
		t.Fatal("Load() error = nil, want version gate failure for subproject")
	}
	msg := err.Error()
	if !strings.Contains(msg, "99.0.0") {
		t.Errorf("error = %q, want required version 99.0.0", msg)
	}
	if !strings.Contains(msg, "backend") {
		t.Errorf("error = %q, want subproject name backend", msg)
	}
}
