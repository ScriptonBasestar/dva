package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvFilePathUnsupportedOrEmptyIsUnresolved(t *testing.T) {
	for _, tt := range []struct {
		name string
		path string
		vars map[string]string
	}{
		{name: "error syntax", path: ".env.${DVA_TEST_A:?need}"},
		{name: "alternate syntax", path: ".env.${DVA_TEST_A:+x}"},
		{name: "empty default", path: "${DVA_TEST_A:-}"},
		{name: "empty value", path: "${DVA_TEST_EMPTY}", vars: map[string]string{"DVA_TEST_EMPTY": ""}},
		{name: "empty root", path: "${DVA_TEST_EMPTY}/.env", vars: map[string]string{"DVA_TEST_EMPTY": ""}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			// A file named literally must not satisfy the unsupported reference.
			if err := os.WriteFile(filepath.Join(dir, ".env.${DVA_TEST_A:?need}"), []byte("A=1\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			env := NewEnvironment(tt.vars, dir, dir)
			report := ApplyEnvFiles(tt.path, dir, env)
			failures := report.Failures()
			if len(failures) != 1 || failures[0].Kind != EnvInputUnresolvedPath {
				t.Fatalf("failures = %+v", failures)
			}
			if !strings.Contains(report.Message(), tt.path) {
				t.Errorf("message lacks the declared path: %s", report.Message())
			}
		})
	}
}

func TestEnvFilePathEmptyVariableInsideNameStillOpens(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("LOADED=yes\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := NewEnvironment(map[string]string{"DVA_TEST_SUFFIX": ""}, dir, dir)
	report := ApplyEnvFiles(".env${DVA_TEST_SUFFIX}", dir, env)
	if report.State != EnvInputComplete || env.Vars["LOADED"] != "yes" {
		t.Fatalf("state %q, LOADED %q", report.State, env.Vars["LOADED"])
	}
}
