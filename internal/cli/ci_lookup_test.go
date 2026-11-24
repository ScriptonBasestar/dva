package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestCILookupDoesNotExecuteOrCreateState(t *testing.T) {
	t.Setenv(ciSupervisorEnv, "invalid-for-execution")
	dir := t.TempDir()
	state := filepath.Join(t.TempDir(), "absent-state")
	t.Setenv("XDG_STATE_HOME", state)
	text := "ci:\n  profiles:\n    verify:\n      timeout: 1m\n      steps:\n        - name: forbidden\n          run: touch executed\n"
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	oldContext := ciCmd.Context()
	ciCmd.SetContext(t.Context())
	t.Cleanup(func() { ciCmd.SetContext(oldContext) })
	oldLookup, oldFresh, oldProject, oldSubproject := ciLookup, ciFresh, ciProject, ciSubproject
	t.Cleanup(func() { ciLookup, ciFresh, ciProject, ciSubproject = oldLookup, oldFresh, oldProject, oldSubproject })
	ciLookup, ciFresh, ciProject, ciSubproject = true, false, "", ""
	dryRun = false
	var runErr error
	data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"verify"}) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	var report struct {
		Covered bool `json:"covered"`
		Steps   []struct {
			State string `json:"state"`
		} `json:"steps"`
	}
	if err := json.Unmarshal([]byte(data), &report); err != nil || report.Covered || len(report.Steps) != 1 || report.Steps[0].State != "not-reusable" {
		t.Fatalf("unexpected lookup: %s: %v", data, err)
	}
	for _, path := range []string{state, filepath.Join(dir, "executed")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("lookup created %s: %v", path, err)
		}
	}
	if _, ok := buildManifest(c).StaticCommands["ci"].Options["lookup"]; !ok {
		t.Fatal("lookup flag missing from manifest")
	}
}

func TestCILookupAndDryRunAreDistinct(t *testing.T) {
	oldLookup, oldDry := ciLookup, dryRun
	t.Cleanup(func() { ciLookup, dryRun = oldLookup, oldDry })
	ciLookup, dryRun = true, true
	if err := ciCmd.RunE(ciCmd, nil); err == nil || !strings.Contains(err.Error(), "separate") {
		t.Fatalf("combined lookup/dry-run: %v", err)
	}
}
