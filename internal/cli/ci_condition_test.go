package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/cirun"
	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestCIConditionPreviewAndLookupNeverExecuteSelection(t *testing.T) {
	dir := t.TempDir()
	state := filepath.Join(t.TempDir(), "absent-state")
	t.Setenv("XDG_STATE_HOME", state)
	text := `ci:
  profiles:
    verify:
      timeout: 1m
      steps:
        - name: selected-unit
          run: touch body-executed
          condition:
            run: touch condition-executed
            timeout: 1s
`
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	manifest := buildManifest(c)
	if manifest.SchemaVersion != "1.11" || manifest.CIProfiles["verify"].Steps[0].Condition == nil {
		t.Fatal("discovery must expose conditional selection under schema 1.11")
	}
	oldContext := ciCmd.Context()
	ciCmd.SetContext(t.Context())
	oldLookup, oldFresh, oldProject, oldSubproject := ciLookup, ciFresh, ciProject, ciSubproject
	t.Cleanup(func() {
		ciCmd.SetContext(oldContext)
		ciLookup, ciFresh, ciProject, ciSubproject = oldLookup, oldFresh, oldProject, oldSubproject
	})
	ciFresh, ciProject, ciSubproject = false, "", ""
	for _, preview := range []bool{true, false} {
		dryRun, ciLookup = preview, !preview
		var runErr error
		data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"verify"}) })
		if runErr != nil {
			t.Fatal(runErr)
		}
		if preview {
			var profile config.CIProfile
			if err := json.Unmarshal([]byte(data), &profile); err != nil || len(profile.Steps) != 1 || profile.Steps[0].Condition == nil {
				t.Fatalf("condition missing from preview: %s: %v", data, err)
			}
		} else {
			var report cirun.LookupReport
			if err := json.Unmarshal([]byte(data), &report); err != nil || report.Covered || len(report.Steps) != 1 || report.Steps[0].Reason != "condition-not-evaluated" {
				t.Fatalf("unexpected conditional lookup: %s: %v", data, err)
			}
		}
	}
	for _, path := range []string{state, filepath.Join(dir, "condition-executed"), filepath.Join(dir, "body-executed")} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("read-only operation created %s: %v", path, err)
		}
	}
}
