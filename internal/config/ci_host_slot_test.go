package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveCIProfileAcceptsNativeHostSlot(t *testing.T) {
	cfg := &Config{CI: &CIConfig{Profiles: map[string]CIProfile{
		"full": {Timeout: "30m", Steps: []CIStep{{Name: "suite", Run: "make test", HostSlot: true}}},
	}}}
	profile, err := cfg.ResolveCIProfile("full")
	if err != nil {
		t.Fatalf("ResolveCIProfile() error: %v", err)
	}
	if !profile.Steps[0].HostSlot {
		t.Fatalf("host_slot not resolved: %#v", profile.Steps[0])
	}
}

func TestResolveCIProfileRejectsHostSlotOnOCIStep(t *testing.T) {
	cfg := &Config{CI: &CIConfig{Profiles: map[string]CIProfile{
		"full": {Timeout: "30m", Steps: []CIStep{{
			Name: "image", Timeout: "5m", HostSlot: true,
			OCI: &CIOCI{Image: "sha256:" + strings.Repeat("0", 64), Argv: []string{"/bin/true"}, Source: "fixtures",
				PidsLimit: 64, MemoryBytes: 1 << 24, CPUs: 2, OutputLimitBytes: 1 << 20},
		}}},
	}}}
	_, err := cfg.ResolveCIProfile("full")
	if err == nil || !strings.Contains(err.Error(), "host_slot is only supported by native run steps") {
		t.Fatalf("error = %v, want host_slot native-only rejection", err)
	}
}

func TestLoadAcceptsBooleanHostSlot(t *testing.T) {
	dir := t.TempDir()
	content := `ci:
  profiles:
    full:
      timeout: 5m
      steps:
        - name: suite
          run: "true"
          host_slot: true
`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if !cfg.CI.Profiles["full"].Steps[0].HostSlot {
		t.Fatalf("host_slot not loaded: %#v", cfg.CI.Profiles["full"].Steps[0])
	}
}

func TestLoadRejectsNonBooleanHostSlot(t *testing.T) {
	dir := t.TempDir()
	content := `ci:
  profiles:
    full:
      timeout: 5m
      steps:
        - name: suite
          run: "true"
          host_slot: "true"
`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(dir)
	if err == nil {
		t.Fatal("Load() accepted a quoted host_slot")
	}
	// The struct decoder reports the line and target type, not the field name.
	if !strings.Contains(err.Error(), "cannot unmarshal") || !strings.Contains(err.Error(), "bool") {
		t.Fatalf("Load() error = %v, want a bool unmarshal rejection", err)
	}
}

// The struct decoder tolerates YAML 1.1 booleans ("yes"), so the raw-bytes
// schema gate is the layer that keeps the on-disk spelling to real booleans.
func TestValidateRejectsYAML11BooleanHostSlot(t *testing.T) {
	dir := t.TempDir()
	content := `ci:
  profiles:
    full:
      timeout: 5m
      steps:
        - name: suite
          run: "true"
          host_slot: "yes"
`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "ci.profiles.full.steps.0.host_slot") {
		t.Fatalf("Validate() error = %v, want it to name the host_slot field", err)
	}
}
