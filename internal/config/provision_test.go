package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProvisionConfigParsing(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `provision:
  default_profile: setup
  setup:
    - step: Install deps
      run: npm install
  reset:
    - step: Reset DB
      run: db reset
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Provision.DefaultProfile != "setup" {
		t.Errorf("default_profile = %q, want %q", cfg.Provision.DefaultProfile, "setup")
	}
	if len(cfg.Provision.Profiles) != 2 {
		t.Errorf("profiles count = %d, want 2", len(cfg.Provision.Profiles))
	}
	if _, ok := cfg.Provision.Profiles["setup"]; !ok {
		t.Error("profile 'setup' not found")
	}
	if _, ok := cfg.Provision.Profiles["reset"]; !ok {
		t.Error("profile 'reset' not found")
	}
}

func TestProvisionConfigWithoutDefaultProfile(t *testing.T) {
	tmpDir := t.TempDir()
	dvaYml := filepath.Join(tmpDir, FileName)

	content := `provision:
  setup:
    - step: Install deps
      run: npm install
`
	os.WriteFile(dvaYml, []byte(content), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Provision.DefaultProfile != "" {
		t.Errorf("default_profile = %q, want empty", cfg.Provision.DefaultProfile)
	}
	if len(cfg.Provision.Profiles) != 1 {
		t.Errorf("profiles count = %d, want 1", len(cfg.Provision.Profiles))
	}
}

func TestProvisionConfigMarshalYAMLPreservesProfiles(t *testing.T) {
	input := ProvisionConfig{
		DefaultProfile: "setup",
		Profiles: map[string][]ProvisionItem{
			"setup": {{Step: "Install deps", Run: "npm install"}},
		},
	}

	data, err := yaml.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal error: %v", err)
	}

	var output map[string]any
	if err := yaml.Unmarshal(data, &output); err != nil {
		t.Fatalf("Unmarshal marshaled data: %v", err)
	}
	if got := output["default_profile"]; got != "setup" {
		t.Fatalf("default_profile = %v, want setup", got)
	}
	if _, ok := output["setup"]; !ok {
		t.Fatalf("marshaled provision missing setup profile: %s", data)
	}
}

func TestProvisionConfigMergeOverride(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, FileName), []byte(`provision:
  setup:
    - step: Install
      run: npm install
  reset:
    - step: Reset
      run: db reset
`), 0644)

	os.WriteFile(filepath.Join(tmpDir, "dva.override.yml"), []byte(`provision:
  default_profile: reset
`), 0644)

	cfg, err := Load(tmpDir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Provision.DefaultProfile != "reset" {
		t.Errorf("default_profile = %q, want %q after override", cfg.Provision.DefaultProfile, "reset")
	}
	// Original profiles should still be present
	if len(cfg.Provision.Profiles) != 2 {
		t.Errorf("profiles count = %d, want 2 (originals preserved)", len(cfg.Provision.Profiles))
	}
}
