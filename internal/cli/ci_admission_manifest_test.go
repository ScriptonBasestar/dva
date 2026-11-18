package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestCIAdmissionModeManifestAndDryRun(t *testing.T) {
	dir := t.TempDir()
	data := "ci:\n  profiles:\n    commit:\n      admission: {mode: when-policy-configured, class: heavy, priority: interactive, wait_timeout: 5m}\n      steps:\n        - name: test\n          run: 'true'\n"
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	m := buildManifest(c)
	if m.SchemaVersion != "1.13" || m.CIProfiles["commit"].Admission.Mode != "when-policy-configured" {
		t.Fatalf("manifest: %+v", m.CIProfiles)
	}
	oldProject := ciProject
	ciProject = ""
	t.Cleanup(func() { ciProject = oldProject })
	dryRun = true
	var runErr error
	payload := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, nil) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	var profile config.CIProfile
	if err := json.Unmarshal([]byte(payload), &profile); err != nil {
		t.Fatal(err)
	}
	if profile.Admission == nil || profile.Admission.Mode != "when-policy-configured" {
		t.Fatalf("dry run: %s", payload)
	}
}
