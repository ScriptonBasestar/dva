package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCIProfileFailFastDefaultsTrueWithoutMaterialization(t *testing.T) {
	cfg := &Config{CI: &CIConfig{Profiles: map[string]CIProfile{
		"full": {Timeout: "1m", Steps: []CIStep{{Name: "unit", Run: "true"}}},
	}}}
	profile, err := cfg.ResolveCIProfile("full")
	if err != nil {
		t.Fatal(err)
	}
	if !profile.FailFastEnabled() {
		t.Fatal("omitted fail_fast must preserve fail-fast behavior")
	}
	if profile.FailFast != nil {
		t.Fatalf("ResolveCIProfile materialized fail_fast default: %#v", profile.FailFast)
	}
}

func TestLoadMergesCIProfileExplicitFailFastFalse(t *testing.T) {
	for _, tc := range []struct {
		name, module, override string
	}{
		{name: "module explicit false", module: "fail_fast: false"},
		{name: "override explicit false", override: "fail_fast: false"},
		{name: "omitted override retains module false", module: "fail_fast: false", override: "description: local"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, DotDirName), 0o755); err != nil {
				t.Fatal(err)
			}
			root := "ci:\n  profiles:\n    full:\n      timeout: 1m\n      fail_fast: true\n      steps: [{name: unit, run: true}]\n"
			if tc.module != "" {
				root = "modules: [ci]\n" + root
			}
			if err := os.WriteFile(filepath.Join(dir, FileName), []byte(root), 0o644); err != nil {
				t.Fatal(err)
			}
			if tc.module != "" {
				module := "ci:\n  profiles:\n    full:\n      " + tc.module + "\n"
				if err := os.WriteFile(filepath.Join(dir, DotDirName, "ci.yml"), []byte(module), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tc.override != "" {
				override := "ci:\n  profiles:\n    full:\n      " + tc.override + "\n"
				if err := os.WriteFile(filepath.Join(dir, "dva.override.yml"), []byte(override), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			cfg, err := Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			profile := cfg.CI.Profiles["full"]
			if profile.FailFast == nil || *profile.FailFast || profile.FailFastEnabled() {
				t.Fatalf("merged fail_fast = %#v, want explicit false", profile.FailFast)
			}
		})
	}
}

func TestValidateCIProfileFailFastSchema(t *testing.T) {
	for _, tc := range []struct {
		name, value string
		wantErr     bool
	}{
		{name: "boolean false", value: "false"},
		{name: "string rejected", value: `"false"`, wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			content := "ci:\n  profiles:\n    full:\n      timeout: 1m\n      fail_fast: " + tc.value + "\n      steps: [{name: unit, run: \"true\"}]\n"
			err := validateYAMLSchema([]byte(content))
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "fail_fast") {
					t.Fatalf("validateYAMLSchema() error = %v, want fail_fast schema rejection", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateYAMLSchema() rejected fail_fast false: %v", err)
			}
			cfg := loadConfigForSchemaTest(t, dir, content)
			if err := cfg.Validate(); err != nil {
				t.Fatalf("Validate() rejected fail_fast false: %v", err)
			}
		})
	}
}
