package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// Exercise Execute in a fresh process, including Cobra help and os.Exit paths.
func TestRootOverviewProcess(t *testing.T) {
	if os.Getenv("DVA_ROOT_OVERVIEW_CHILD") != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		t.Fatal("missing argument separator")
	}
	os.Args = append([]string{"dva"}, os.Args[separator+1:]...)
	Execute()
	os.Exit(0)
}

func TestRootOverview(t *testing.T) {
	t.Setenv("DVA_FILE", "")
	t.Setenv("DVA_ROOT_OVERVIEW_CHILD", "1")
	for _, tc := range []struct {
		name      string
		args      []string
		config    string
		filename  string
		parent    bool
		explicit  bool
		wantError bool
		want      string
		absent    string
		json      bool
	}{
		{name: "missing config", want: "dva config init", absent: "Core Commands"},
		{name: "missing config JSON", args: []string{"--json"}, want: "not_configured", json: true},
		{name: "explicit help", args: []string{"--help"}, want: "Core Commands", absent: "No DVA configuration found"},
		{name: "help command", args: []string{"help"}, want: "Core Commands", absent: "No DVA configuration found"},
		{name: "version", args: []string{"version"}, want: "dva version", absent: "No DVA configuration found"},
		{name: "init help", args: []string{"config", "init", "--help"}, want: "--template", absent: "No DVA configuration found"},
		{name: "missing config execution", args: []string{"up", "local-dev"}, wantError: true, want: "dva config init", absent: "Core Commands"},
		{name: "unknown command", args: []string{"not-a-command"}, wantError: true, want: "unknown command", absent: "No DVA configuration found"},
		{name: "typo suggestions", args: []string{"sta"}, wantError: true, want: "Did you mean this?", absent: "No DVA configuration found"},
		{name: "valid config", config: "version: \"0.1.22\"\n", want: "Core Commands", absent: "No DVA configuration found"},
		{name: "parent config", config: "version: \"0.1.22\"\n", parent: true, want: "Core Commands", absent: "No DVA configuration found"},
		{name: "legacy filename", config: "version: \"0.1.22\"\n", filename: "dva.yaml", want: "Core Commands", absent: "No DVA configuration found"},
		{name: "explicit config", config: "version: \"0.1.22\"\n", filename: "custom.yml", explicit: true, want: "Core Commands", absent: "No DVA configuration found"},
		{name: "invalid config", config: "stack: [\n", wantError: true, want: "ERROR:", absent: "dva config init"},
		{name: "help with invalid config", config: "stack: [\n", args: []string{"--help"}, want: "Core Commands", absent: "ERROR:"},
		{name: "missing explicit config", explicit: true, wantError: true, want: "DVA_FILE=", absent: "dva config init"},
		{name: "missing explicit config JSON", explicit: true, args: []string{"--json"}, wantError: true, want: "DVA_FILE=", json: true, absent: "not_configured"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			filename := tc.filename
			if filename == "" {
				filename = "dva.yml"
			}
			configPath := filepath.Join(dir, filename)
			if tc.config != "" {
				if err := os.WriteFile(configPath, []byte(tc.config), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if tc.parent {
				dir = filepath.Join(dir, "child")
				if err := os.Mkdir(dir, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			args := append([]string{"-test.run=^TestRootOverviewProcess$", "--"}, tc.args...)
			cmd := exec.Command(os.Args[0], args...)
			cmd.Dir = dir
			cmd.Env = os.Environ()
			if tc.explicit {
				cmd.Env = append(cmd.Env, "DVA_FILE="+configPath)
			}
			var stdout, stderr bytes.Buffer
			cmd.Stdout, cmd.Stderr = &stdout, &stderr
			err := cmd.Run()
			combined := stdout.String() + stderr.String()
			if (err != nil) != tc.wantError {
				t.Fatalf("exit = %v, wantError = %v:\n%s", err, tc.wantError, combined)
			}
			if !strings.Contains(combined, tc.want) || (tc.absent != "" && strings.Contains(combined, tc.absent)) {
				t.Fatalf("want %q, absent %q:\n%s", tc.want, tc.absent, combined)
			}
			if tc.json && !json.Valid(stdout.Bytes()) {
				t.Fatalf("stdout is not one JSON document: %s", stdout.String())
			}
			if tc.config == "" {
				if _, err := os.Stat(configPath); !os.IsNotExist(err) {
					t.Fatalf("guidance created a configuration: %v", err)
				}
			}
		})
	}
}
