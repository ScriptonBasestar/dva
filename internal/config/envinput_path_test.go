package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvFilePathExpandsAgainstEnvironment(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env.stg"), []byte("LOADED=yes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		path string
		vars map[string]string
	}{
		{name: "defined", path: ".env.${DEPLOY_ENV_T}", vars: map[string]string{"DEPLOY_ENV_T": "stg"}},
		{name: "default", path: ".env.${DEPLOY_ENV_T:-stg}"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment(tt.vars, dir, dir)
			report := ApplyEnvFiles(tt.path, dir, env)
			if report.State != EnvInputComplete || env.Vars["LOADED"] != "yes" {
				t.Fatalf("state %q, LOADED %q", report.State, env.Vars["LOADED"])
			}
			if report.Entries[0].File != tt.path {
				t.Errorf("File = %q, want the declared path", report.Entries[0].File)
			}
		})
	}
}

func TestEnvFileUnresolvedPathFailsRegardlessOfRequired(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("LOADED=yes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	const path = ".env.${DVA_TEST_UNSET_X}"
	for _, tt := range []struct {
		name string
		decl any
	}{
		{name: "optional", decl: []any{".env", path}},
		{name: "required", decl: map[string]any{"files": []any{".env", path}, "required": true}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			env := NewEnvironment(nil, dir, dir)
			report := ApplyEnvFiles(tt.decl, dir, env)
			if !report.Incomplete() {
				t.Fatalf("state %q, want incomplete", report.State)
			}
			failures := report.Failures()
			if len(failures) != 1 || failures[0].Kind != EnvInputUnresolvedPath {
				t.Fatalf("failures = %+v", failures)
			}
			if !strings.Contains(report.Message(), "unresolved variable reference ${DVA_TEST_UNSET_X} in path") {
				t.Errorf("message lacks the cause: %s", report.Message())
			}
			if _, ok := env.Vars["LOADED"]; ok {
				t.Error("a failed report applied an earlier file's values")
			}
		})
	}
}

func TestEnvFileValueReferenceKeptVerbatimAndCounted(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("A=${DVA_TEST_UNSET_X}\nB=plain\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := NewEnvironment(nil, dir, dir)
	report := ApplyEnvFiles(".env", dir, env)
	if report.Incomplete() || env.Vars["A"] != "${DVA_TEST_UNSET_X}" {
		t.Fatalf("state %q, A %q: runtime behaviour changed", report.State, env.Vars["A"])
	}
	if got := report.Entries[0].UnresolvedValues; got != 1 {
		t.Errorf("UnresolvedValues = %d, want 1", got)
	}
}

func TestValidateWarnsOnUnresolvedEnvFilePath(t *testing.T) {
	c := &Config{
		Vars:    map[string]string{"KNOWN_T": "x"},
		EnvFile: []any{".env.${KNOWN_T}", ".env.${DVA_TEST_UNSET_X}"},
	}
	warnings := c.warnUnresolvedEnvFilePaths()
	if len(warnings) != 1 || !strings.Contains(warnings[0], "${DVA_TEST_UNSET_X}") {
		t.Fatalf("warnings = %v", warnings)
	}
}
