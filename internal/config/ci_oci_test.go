package config

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const validOCIImage = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestLoadCIOCI(t *testing.T) {
	err := loadCIConfig(t, "ci:\n  profiles:\n    full:\n      timeout: 1m\n      steps:\n        - name: isolated\n          timeout: 10s\n          oci:\n            image: "+validOCIImage+"\n            argv: [echo, ok]\n            source: app\n            pids_limit: 8\n            memory_bytes: 16777216\n            cpus: 1\n            output_limit_bytes: 1024\n")
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
}

func TestCIOCIValidation(t *testing.T) {
	base := "ci:\n  profiles:\n    full:\n      timeout: 1m\n      steps:\n        - name: isolated\n          timeout: 10s\n          oci:\n            image: %s\n            argv: [echo]\n            source: %s\n            pids_limit: %d\n            memory_bytes: %d\n            cpus: %d\n            output_limit_bytes: %d\n"
	for _, tc := range []struct {
		name, image, source, want  string
		pids, memory, cpus, output int64
	}{
		{"digest", "busybox:latest", "app", "immutable", 8, 16777216, 1, 1024},
		{"escape", validOCIImage, "../app", "source", 8, 16777216, 1, 1024},
		{"limits", validOCIImage, "app", "pids_limit", 7, 16777216, 1, 1024},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := loadCIConfig(t, fmt.Sprintf(base, tc.image, tc.source, tc.pids, tc.memory, tc.cpus, tc.output))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Load() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestCIOCIRejectsBoundaryInputs(t *testing.T) {
	valid := CIOCI{Image: validOCIImage, Argv: []string{"echo", "literal value"}, Source: "app", PidsLimit: 8, MemoryBytes: 16 << 20, CPUs: 1, OutputLimitBytes: 1024}
	for _, tc := range []struct {
		name string
		edit func(*CIOCI)
	}{
		{"empty argv", func(o *CIOCI) { o.Argv = nil }},
		{"empty argv command", func(o *CIOCI) { o.Argv = []string{""} }},
		{"nul argv", func(o *CIOCI) { o.Argv = []string{"echo", "bad\x00arg"} }},
		{"transport completion", func(o *CIOCI) { o.Argv = []string{"echo", "__complete"} }},
		{"transport completion no desc", func(o *CIOCI) { o.Argv = []string{"echo", "__completeNoDesc"} }},
		{"dot source", func(o *CIOCI) { o.Source = "." }},
		{"parent source", func(o *CIOCI) { o.Source = "../app" }},
		{"absolute source", func(o *CIOCI) { o.Source = "/app" }},
		{"backslash source", func(o *CIOCI) { o.Source = "app\\child" }},
		{"comma source", func(o *CIOCI) { o.Source = "app,child" }},
		{"workspace evil", func(o *CIOCI) { o.Workdir = "/workspaceevil" }},
		{"workspace escape", func(o *CIOCI) { o.Workdir = "/workspace/../escape" }},
		{"nul workdir", func(o *CIOCI) { o.Workdir = "/workspace/\x00bad" }},
		{"home env", func(o *CIOCI) { o.Environment = map[string]string{"HOME": "x"} }},
		{"tmp env", func(o *CIOCI) { o.Environment = map[string]string{"TMPDIR": "x"} }},
		{"xdg env", func(o *CIOCI) { o.Environment = map[string]string{"XDG_CACHE_HOME": "x"} }},
		{"parent env", func(o *CIOCI) { o.Environment = map[string]string{CIParentRunEnv: "x"} }},
		{"subproject env", func(o *CIOCI) { o.Environment = map[string]string{CISubprojectEnv: "x"} }},
		{"pids low", func(o *CIOCI) { o.PidsLimit = 7 }},
		{"pids high", func(o *CIOCI) { o.PidsLimit = 4097 }},
		{"memory low", func(o *CIOCI) { o.MemoryBytes = 16<<20 - 1 }},
		{"cpu low", func(o *CIOCI) { o.CPUs = 0 }},
		{"cpu high", func(o *CIOCI) { o.CPUs = 129 }},
		{"output low", func(o *CIOCI) { o.OutputLimitBytes = 1023 }},
		{"output high", func(o *CIOCI) { o.OutputLimitBytes = 64<<20 + 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oci := valid
			tc.edit(&oci)
			if err := validateCIOCI("full", "isolated", oci); err == nil {
				t.Fatal("validateCIOCI() unexpectedly succeeded")
			}
		})
	}
	if err := validateCIOCI("full", "isolated", valid); err != nil {
		t.Fatalf("valid OCI: %v", err)
	}
	valid.Workdir = "/workspace"
	if err := validateCIOCI("full", "isolated", valid); err != nil {
		t.Fatalf("workspace root: %v", err)
	}
	valid.Workdir = "/workspace/foo..bar"
	if err := validateCIOCI("full", "isolated", valid); err != nil {
		t.Fatalf("workspace foo..bar: %v", err)
	}
	valid.Image = "registry.example:5000/team/image@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	if err := validateCIOCI("full", "isolated", valid); err != nil {
		t.Fatalf("registry port image: %v", err)
	}
}

func TestCIOCIRejectsStepFieldConflicts(t *testing.T) {
	for _, field := range []string{"run: ' '", "workdir: app", "environment: {SAFE: value}", "timeout: ''"} {
		yaml := "ci:\n  profiles:\n    full:\n      timeout: 1m\n      steps:\n        - name: isolated\n          " + field + "\n          oci:\n            image: " + validOCIImage + "\n            argv: [echo]\n            source: app\n            pids_limit: 8\n            memory_bytes: 16777216\n            cpus: 1\n            output_limit_bytes: 1024\n"
		if err := loadCIConfig(t, yaml); err == nil {
			t.Fatalf("field %q unexpectedly accepted", field)
		}
	}
}

func TestCIOCIYAMLRoundTripOmitsEmptyShellFields(t *testing.T) {
	cfg := Config{
		CI: &CIConfig{Profiles: map[string]CIProfile{
			"full": {
				Timeout: "1m",
				Steps: []CIStep{{Name: "isolated", Timeout: "10s", OCI: &CIOCI{
					Image: validOCIImage, Argv: []string{"echo", ""}, Source: "app", PidsLimit: 8,
					MemoryBytes: 16 << 20, CPUs: 1, OutputLimitBytes: 1024,
				}}},
			},
		}},
	}
	raw, err := yaml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "\n                          run:") || strings.Contains(string(raw), "\n                          workdir:") || strings.Contains(string(raw), "\n                          environment:") {
		t.Fatalf("OCI YAML retained empty shell fields: %s", raw)
	}
	if err := loadCIConfig(t, string(raw)); err != nil {
		t.Fatalf("OCI YAML reload: %v\n%s", err, raw)
	}
}

func TestNativeCIStepYAMLRetainsEmptyPublicFields(t *testing.T) {
	raw, err := yaml.Marshal(CIStep{Name: "native", Run: "true", Environment: map[string]string{}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "run: \"true\"") || !strings.Contains(string(raw), "workdir: \"\"") || !strings.Contains(string(raw), "environment: {}") {
		t.Fatalf("native wire changed: %s", raw)
	}
}
