package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestCIFreshPreviewDoesNotExecuteProbes(t *testing.T) {
	dir := t.TempDir()
	text := "ci:\n  profiles:\n    full:\n      timeout: 1m\n      steps:\n        - name: unit\n          run: touch executed\n          reuse:\n            inputs: [source]\n            tools: [sh]\n            max_age: 1h\n            probes: [touch probed]\n"
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	oldProject, oldSubproject, oldFresh, oldDry := ciProject, ciSubproject, ciFresh, dryRun
	t.Cleanup(func() { ciProject, ciSubproject, ciFresh, dryRun = oldProject, oldSubproject, oldFresh, oldDry })
	ciProject, ciSubproject, ciFresh, dryRun = "", "", true, true
	var runErr error
	data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"full"}) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	var p config.CIProfile
	if err := json.Unmarshal([]byte(data), &p); err != nil || !p.Fresh {
		t.Fatalf("preview %s, %v", data, err)
	}
	for _, name := range []string{"executed", "probed"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("preview wrote %s", name)
		}
	}
}
