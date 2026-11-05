package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadForEnvInspectionRetainsDeclarationValidation(t *testing.T) {
	for _, layer := range []string{"root", "module", "override"} {
		t.Run(layer, func(t *testing.T) {
			dir := t.TempDir()
			root := "version: '0.1.45'\n"
			invalid := "env_file: [{path: .env, sops_source: .env}]\n"
			file := filepath.Join(dir, FileName)
			switch layer {
			case "root":
				root += invalid
			case "module":
				root += "modules: [env]\n"
				moduleDir := filepath.Join(dir, DotDirName)
				if err := os.MkdirAll(moduleDir, 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(moduleDir, "env.yml"), []byte(invalid), 0o644); err != nil {
					t.Fatal(err)
				}
			case "override":
				if err := os.WriteFile(filepath.Join(dir, "dva.override.yml"), []byte(invalid), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(file, []byte(root), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(dir, FromFile(file), ForEnvInspection()); err == nil || !strings.Contains(err.Error(), "source_is_target") {
				t.Fatalf("environment declaration validation bypassed: %v", err)
			}
		})
	}
}

func TestLoadRejectsConflictingInspectionModes(t *testing.T) {
	for _, opts := range [][]LoadOption{{ForCI(), ForEnvInspection()}, {ForEnvInspection(), ForCI()}} {
		if _, err := Load(t.TempDir(), opts...); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
			t.Fatalf("conflicting modes accepted: %v", err)
		}
	}
}
