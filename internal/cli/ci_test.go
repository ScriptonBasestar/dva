package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestCIDiscoveryAndDryRunDoNotExecute(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte("ci:\n  profiles:\n    commit:\n      steps:\n        - name: forbidden\n          run: touch executed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	oldProject := ciProject
	ciProject = ""
	t.Cleanup(func() { ciProject = oldProject })
	m := buildManifest(c)
	if len(m.CIProfiles) != 1 || m.CIProfiles["commit"].Timeout != "10m0s" {
		t.Fatalf("missing resolved CI discovery: %#v", m.CIProfiles)
	}
	dryRun = true
	var runErr error
	data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, nil) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	var p config.CIProfile
	if err := json.Unmarshal([]byte(data), &p); err != nil {
		t.Fatal(err)
	}
	if len(p.Steps) != 1 {
		t.Fatalf("unexpected preview: %s", data)
	}
	if _, err := os.Stat(filepath.Join(dir, "executed")); !os.IsNotExist(err) {
		t.Fatal("preview executed a check")
	}
}

func TestCIChildOwnsProfileAndEnvironment(t *testing.T) {
	dir := t.TempDir()
	childDir := filepath.Join(dir, "child")
	if err := os.Mkdir(childDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		filepath.Join(dir, "dva.yml"):      "env_file: [missing.env]\nsubprojects:\n  child:\n    path: child\n",
		filepath.Join(childDir, "dva.yml"): "vars:\n  CI_OWNER_TEST: child\nci:\n  profiles:\n    commit:\n      steps:\n        - name: child\n          run: true\n",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := ciOwner(c, "child")
	if err != nil {
		t.Fatal(err)
	}
	if owner.FileDir() != childDir {
		t.Fatalf("owner root = %s", owner.FileDir())
	}
	if _, err := owner.ResolveCIProfile("commit"); err != nil {
		t.Fatal(err)
	}
	owned := newOwnedConfigEnvironment(owner)
	if err := envIncompleteError(owned.report); err != nil {
		t.Fatalf("parent env contaminated child: %v", err)
	}
	if _, err := ciOwner(c, "missing"); err == nil {
		t.Fatal("unknown child accepted")
	}
}
