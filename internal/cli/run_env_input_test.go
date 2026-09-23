package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// withRunEnvInputState isolates the package globals read by runCmd. In particular,
// dryRun and the confirmation streams make the ordering claim below observable.
func withRunEnvInputState(t *testing.T, c *config.Config) {
	t.Helper()
	oldCfg, oldEnv := cfg, env
	oldDryRun, oldProject, oldConfirm := dryRun, projectName, confirmYes
	oldTerminal, oldStdout := isStdinTerminal, stdoutWriter
	cfg, env, dryRun, projectName, confirmYes = c, nil, false, "", false
	t.Cleanup(func() {
		cfg, env = oldCfg, oldEnv
		dryRun, projectName, confirmYes = oldDryRun, oldProject, oldConfirm
		isStdinTerminal, stdoutWriter = oldTerminal, oldStdout
	})
}

func loadRunEnvInputConfig(t *testing.T, body string) *config.Config {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.FileName), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestRunRefusesIncompleteOwnerEnvironmentBeforePreviewExecutionAndConfirmation(t *testing.T) {
	for _, tt := range []struct {
		name    string
		envFile string
	}{
		{name: "required missing", envFile: "missing.env"},
		{name: "malformed", envFile: ".env"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "ran")
			c := loadRunEnvInputConfig(t, fmt.Sprintf(`env_file:
  files:
    - path: %s
      required: true
interaction:
  probe:
    destructive: true
    script: %q
`, tt.envFile, "touch "+marker))
			if tt.name == "malformed" {
				if err := os.WriteFile(filepath.Join(c.FileDir(), ".env"), []byte("not dotenv\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}

			for _, mode := range []struct {
				name  string
				setup func()
			}{
				{name: "execution"},
				{name: "preview", setup: func() { dryRun = true }},
				{name: "confirmation", setup: func() { isStdinTerminal = func() bool { return false } }},
			} {
				t.Run(mode.name, func(t *testing.T) {
					withRunEnvInputState(t, c)
					if mode.setup != nil {
						mode.setup()
					}
					err := runCmd.RunE(runCmd, []string{"probe"})
					if err == nil || !strings.Contains(err.Error(), "environment inputs are incomplete") {
						t.Fatalf("run error = %v, want incomplete environment error", err)
					}
					if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
						t.Fatalf("run reached execution after environment failure: %v", statErr)
					}
				})
			}
		})
	}
}

func TestRunEnvironmentFailuresUseTheInteractionOwner(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(child, "ran")
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(parent, config.FileName), `env_file:
  files: [root-missing.env]
  required: true
subprojects:
  child:
    path: child
    import:
      interactions: [probe]
`)
	write(filepath.Join(child, config.FileName), fmt.Sprintf(`env_file:
  files: [child-missing.env]
  required: true
interaction:
  probe:
    script: %q
`, "touch "+marker))
	c, err := config.Load(parent)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name  string
		setup func()
		args  []string
	}{
		{name: "imported interaction", args: []string{"child/probe"}},
		{name: "direct child interaction", setup: func() { projectName = "child" }, args: []string{"probe"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			withRunEnvInputState(t, c)
			if tt.setup != nil {
				tt.setup()
			}
			err := runCmd.RunE(runCmd, tt.args)
			if err == nil || !strings.Contains(err.Error(), "child-missing.env: missing required file") {
				t.Fatalf("run error = %v, want child environment failure", err)
			}
			if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
				t.Fatalf("run reached child execution after environment failure: %v", statErr)
			}
		})
	}
}

func TestRunAllowsOptionalMissingOwnerEnvironmentAndIgnoresBrokenParent(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(child, "ran")
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(parent, config.FileName), `env_file:
  files: [root-missing.env]
  required: true
subprojects:
  child:
    path: child
    import:
      interactions: [probe]
`)
	write(filepath.Join(child, config.FileName), fmt.Sprintf(`env_file:
  files: [optional.env]
  required: false
interaction:
  probe:
    script: %q
`, "touch "+marker))
	c, err := config.Load(parent)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range []struct {
		name  string
		setup func()
		args  []string
	}{
		{name: "imported interaction", args: []string{"child/probe"}},
		{name: "direct child interaction", setup: func() { projectName = "child" }, args: []string{"probe"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.Remove(marker); err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			withRunEnvInputState(t, c)
			if tt.setup != nil {
				tt.setup()
			}
			if err := runCmd.RunE(runCmd, tt.args); err != nil {
				t.Fatalf("run %v: %v", tt.args, err)
			}
			if _, err := os.Stat(marker); err != nil {
				t.Fatalf("run did not execute with optional missing env_file: %v", err)
			}
		})
	}
}
