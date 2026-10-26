package cli

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestCIChildDryRunUsesOwningWorkerEnvironment(t *testing.T) {
	// A surrounding `dva ci` exports these worker values for its child steps. This
	// fixture verifies the child config's own environment, so remove ambient
	// worker values after registering cleanup instead of letting the parent run
	// change the preview it is meant to inspect.
	for _, key := range []string{"DVA_CI_JOBS", "CARGO_BUILD_JOBS", "RUST_TEST_THREADS"} {
		t.Setenv(key, "ambient-"+key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o700); err != nil {
		t.Fatal(err)
	}
	for path, body := range map[string]string{
		filepath.Join(dir, "dva.yml"):   "vars:\n  CARGO_BUILD_JOBS: parent\nenv_file: [missing.env]\nsubprojects:\n  child:\n    path: child\n",
		filepath.Join(child, "dva.yml"): "vars:\n  CARGO_BUILD_JOBS: '7'\nci:\n  profiles:\n    commit:\n      tool_jobs: 3\n      steps:\n        - name: test\n          run: touch executed\n",
	} {
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	oldProject := ciProject
	t.Cleanup(func() { ciProject = oldProject })
	ciProject, dryRun = "child", true
	var runErr error
	data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, nil) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	var preview struct {
		Workers []cirun.StepWorkers `json:"workers"`
	}
	if err := json.Unmarshal([]byte(data), &preview); err != nil {
		t.Fatal(err)
	}
	owner, err := ciOwner(c, "child")
	if err != nil {
		t.Fatal(err)
	}
	p, err := owner.ResolveCIProfile("commit")
	if err != nil {
		t.Fatal(err)
	}
	want := cirun.WorkerPlan(p, newOwnedConfigEnvironment(owner).env.EnvSlice())
	if !reflect.DeepEqual(preview.Workers, want) || len(want) != 1 || want[0].WorkerBudget == nil || want[0].WorkerBudget.CargoBuildJobs != 7 || want[0].WorkerBudget.ToolJobs != 3 {
		t.Fatalf("preview %s want %+v", data, want)
	}
	if _, err := os.Stat(filepath.Join(child, "executed")); !os.IsNotExist(err) {
		t.Fatal("preview executed step")
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

// --subproject keeps the root's profile and directory and only names a declared
// child, so a root step can scope a shared check (for example a board gate) to
// that child. Undeclared names and the --project combination fail before any
// profile is resolved.
func TestCISubprojectRunsRootProfileWithDeclaredName(t *testing.T) {
	dir := t.TempDir()
	childDir := filepath.Join(dir, "site-a")
	if err := os.Mkdir(childDir, 0o700); err != nil {
		t.Fatal(err)
	}
	for path, contents := range map[string]string{
		filepath.Join(dir, "dva.yml"):      "subprojects:\n  site-a:\n    path: site-a\nci:\n  profiles:\n    predeploy:\n      timeout: 1m\n      steps:\n        - name: root-step\n          run: touch executed\n",
		filepath.Join(childDir, "dva.yml"): "ci:\n  profiles:\n    predeploy:\n      timeout: 1m\n      steps:\n        - name: child-step\n          run: true\n",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	withEnvPolicyGlobals(t, c, true)
	oldProject, oldSubproject := ciProject, ciSubproject
	oldDryRun := dryRun
	t.Cleanup(func() { ciProject, ciSubproject, dryRun = oldProject, oldSubproject, oldDryRun })
	dryRun = true

	ciProject, ciSubproject = "", "site-a"
	var runErr error
	data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"predeploy"}) })
	if runErr != nil {
		t.Fatal(runErr)
	}
	var preview struct {
		config.CIProfile
		Subproject string `json:"subproject"`
	}
	if err := json.Unmarshal([]byte(data), &preview); err != nil {
		t.Fatal(err)
	}
	if preview.Subproject != "site-a" || len(preview.Steps) != 1 || preview.Steps[0].Name != "root-step" {
		t.Fatalf("preview %s, want the root's profile scoped to site-a", data)
	}
	if _, err := os.Stat(filepath.Join(dir, "executed")); !os.IsNotExist(err) {
		t.Fatal("preview executed a check")
	}

	for _, bad := range []struct{ project, subproject, want string }{
		{"", "site-b", `unknown CI subproject "site-b"`},
		{"site-a", "site-a", "cannot be combined with --project"},
	} {
		ciProject, ciSubproject = bad.project, bad.subproject
		output.ResetStdoutDocument()
		captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"predeploy"}) })
		if runErr == nil || !strings.Contains(runErr.Error(), bad.want) {
			t.Fatalf("--project=%q --subproject=%q: err = %v, want %q", bad.project, bad.subproject, runErr, bad.want)
		}
	}
}

func TestCIRootProfileSkipsUnrelatedImportedChildren(t *testing.T) {
	dir := t.TempDir()
	rootConfig := `subprojects:
  site-a:
    path: site-a
    import:
      interactions: [shell]
  unrelated:
    path: missing-child
    import:
      interactions: [shell]
ci:
  profiles:
    predeploy:
      timeout: 1m
      steps:
        - name: root-step
          run: true
`
	if err := os.WriteFile(filepath.Join(dir, "dva.yml"), []byte(rootConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	withCIWorkingDirectory(t, dir)
	withEnvPolicyGlobals(t, nil, true)
	oldProject, oldSubproject, oldDryRun := ciProject, ciSubproject, dryRun
	ciProject, ciSubproject, dryRun = "", "site-a", true
	t.Cleanup(func() { ciProject, ciSubproject, dryRun = oldProject, oldSubproject, oldDryRun })

	var runErr error
	data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"predeploy"}) })
	if runErr != nil {
		t.Fatalf("root CI dry-run rejected an unrelated missing child: %v", runErr)
	}
	var preview struct {
		config.CIProfile
		Subproject string `json:"subproject"`
	}
	if err := json.Unmarshal([]byte(data), &preview); err != nil {
		t.Fatalf("decode preview %q: %v", data, err)
	}
	if preview.Subproject != "site-a" || len(preview.Steps) != 1 || preview.Steps[0].Name != "root-step" {
		t.Fatalf("preview = %s, want root profile with site-a scope", data)
	}

	ciSubproject = ""
	output.ResetStdoutDocument()
	data = captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"predeploy"}) })
	if runErr != nil {
		t.Fatalf("root-only CI dry-run rejected an unrelated missing child: %v", runErr)
	}
	preview = struct {
		config.CIProfile
		Subproject string `json:"subproject"`
	}{}
	if err := json.Unmarshal([]byte(data), &preview); err != nil {
		t.Fatalf("decode root-only preview %q: %v", data, err)
	}
	if preview.Subproject != "" || len(preview.Steps) != 1 || preview.Steps[0].Name != "root-step" {
		t.Fatalf("preview = %s, want unscoped root profile", data)
	}
}

func TestCIProjectLoadsOnlySelectedChild(t *testing.T) {
	dir := t.TempDir()
	selectedDir := filepath.Join(dir, "selected")
	if err := os.Mkdir(selectedDir, 0o700); err != nil {
		t.Fatal(err)
	}
	rootConfig := `subprojects:
  selected:
    path: selected
    import:
      interactions: [shell]
  declared-missing:
    path: missing-selected
  unrelated:
    path: missing-child
    import:
      interactions: [shell]
`
	childConfig := `ci:
  profiles:
    predeploy:
      timeout: 1m
      steps:
        - name: selected-step
          run: true
`
	for path, contents := range map[string]string{
		filepath.Join(dir, "dva.yml"):         rootConfig,
		filepath.Join(selectedDir, "dva.yml"): childConfig,
	} {
		if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	withCIWorkingDirectory(t, dir)
	withEnvPolicyGlobals(t, nil, true)
	oldProject, oldSubproject, oldDryRun := ciProject, ciSubproject, dryRun
	ciProject, ciSubproject, dryRun = "selected", "", true
	t.Cleanup(func() { ciProject, ciSubproject, dryRun = oldProject, oldSubproject, oldDryRun })

	var runErr error
	data := captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"predeploy"}) })
	if runErr != nil {
		t.Fatalf("selected child CI dry-run rejected unrelated child: %v", runErr)
	}
	var preview config.CIProfile
	if err := json.Unmarshal([]byte(data), &preview); err != nil {
		t.Fatalf("decode preview %q: %v", data, err)
	}
	if len(preview.Steps) != 1 || preview.Steps[0].Name != "selected-step" {
		t.Fatalf("preview = %s, want selected child's profile", data)
	}

	ciProject = "declared-missing"
	output.ResetStdoutDocument()
	captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"predeploy"}) })
	if runErr == nil || !strings.Contains(runErr.Error(), "loading subproject \"declared-missing\"") {
		t.Fatalf("missing selected child error = %v, want selected config load failure", runErr)
	}
	ciProject = "unknown"
	output.ResetStdoutDocument()
	captureStdout(t, func() { runErr = ciCmd.RunE(ciCmd, []string{"predeploy"}) })
	if runErr == nil || !strings.Contains(runErr.Error(), "unknown CI project") {
		t.Fatalf("unregistered selected child error = %v, want unknown CI project", runErr)
	}
}

func withCIWorkingDirectory(t *testing.T, dir string) {
	t.Helper()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})
}
