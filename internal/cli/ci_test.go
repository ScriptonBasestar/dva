package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/cirun"
	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/output"
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

func TestCIManifestAndDryRunExposeLocks(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte("ci:\n  profiles:\n    commit:\n      locks: [integration-db]\n      steps:\n        - name: test\n          run: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	m := buildManifest(c)
	if got := m.CIProfiles["commit"].Locks; len(got) != 1 || got[0] != "integration-db" {
		t.Fatalf("manifest locks = %#v", got)
	}
	oldProject := ciProject
	ciProject = ""
	t.Cleanup(func() { ciProject = oldProject })
	dryRun = true
	var runErr error
	data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, nil) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	var profile config.CIProfile
	if err := json.Unmarshal([]byte(data), &profile); err != nil {
		t.Fatal(err)
	}
	if got := profile.Locks; len(got) != 1 || got[0] != "integration-db" {
		t.Fatalf("dry-run locks = %#v", got)
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

// Both routes must pass the child's owning configuration directory to cirun.
// Holding that directory's OS root lock makes this an admission test: neither
// invocation can reach its step command.
func TestCIProjectAndDirectChildShareRootLock(t *testing.T) {
	// A surrounding dva ci run deliberately sets this for its steps. This test
	// is a separate top-level invocation, so it must exercise root admission.
	t.Setenv(config.CIParentRunEnv, "")
	oldContext := ciCmd.Context()
	if oldContext == nil {
		oldContext = context.Background()
	}
	ciCmd.SetContext(context.Background())
	t.Cleanup(func() { ciCmd.SetContext(oldContext) })
	dir := t.TempDir()
	childDir := filepath.Join(dir, "child")
	if err := os.Mkdir(childDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		filepath.Join(dir, "dva.yml"):      "subprojects:\n  child:\n    path: child\n",
		filepath.Join(childDir, "dva.yml"): "ci:\n  profiles:\n    commit:\n      steps:\n        - name: must-not-run\n          run: touch executed\n",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	parent, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	child, err := config.Load(childDir)
	if err != nil {
		t.Fatal(err)
	}
	canonicalChildDir, err := filepath.EvalSymlinks(child.FileDir())
	if err != nil {
		t.Fatal(err)
	}
	canonicalChildDir, err = filepath.Abs(canonicalChildDir)
	if err != nil {
		t.Fatal(err)
	}
	holder := holdCIRootLock(t, child.FileDir())
	defer holder.release()

	for _, route := range []struct {
		name    string
		config  *config.Config
		project string
	}{
		{name: "parent project", config: parent, project: "child"},
		{name: "child direct", config: child},
	} {
		t.Run(route.name, func(t *testing.T) {
			withEnvPolicyGlobals(t, route.config, true)
			oldProject := ciProject
			ciProject = route.project
			t.Cleanup(func() { ciProject = oldProject })
			output.ResetStdoutDocument()
			var runErr error
			data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, nil) })
			if !errors.Is(runErr, cirun.ErrBusy) {
				t.Fatalf("RunE() error = %v, want busy", runErr)
			}
			var report cirun.Report
			if err := json.Unmarshal([]byte(data), &report); err != nil {
				t.Fatalf("busy output is not a report: %v; output=%s", err, data)
			}
			if report.Status != "busy" || report.Conflict == nil || report.Conflict.Kind != "root" || report.Conflict.Key != canonicalChildDir || report.Conflict.RunID != holder.id {
				t.Fatalf("report = %#v, want child root conflict", report)
			}
			if _, err := os.Stat(filepath.Join(childDir, "executed")); !os.IsNotExist(err) {
				t.Fatalf("busy route executed CI step: stat error = %v", err)
			}
		})
	}
}
