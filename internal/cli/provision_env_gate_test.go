package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// A provision profile must not run, or print a plan, from env input that failed: the marker
// file proves the step never ran, and the dry-run case proves the gate precedes that branch.
// The child case pins the owner's report rather than only the root's.
func TestProvisionStopsOnEnvFileFailure(t *testing.T) {
	const profileBody = "version: \"0.1.0\"\nproject_name: gate\nenv_file: ${A:?x}/.env\nprovision:\n  setup:\n    - step: record\n      run: 'touch \"MARKER\"'\n"
	for _, tc := range []struct {
		name    string
		dryRun  bool
		child   bool
		profile string
	}{
		{"root live", false, false, "setup"},
		{"root dry-run", true, false, "setup"},
		{"child live", false, true, "child-setup"},
		{"child dry-run", true, true, "child-setup"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "ran.out")
			body := strings.ReplaceAll(profileBody, "MARKER", marker)
			if tc.child {
				writeFile(t, filepath.Join(dir, "child", config.FileName), body)
				body = `version: "0.1.0"
project_name: parent
subprojects:
  child:
    path: child
    import:
      provision:
        - name: setup
          as: child-setup
`
			}
			writeFile(t, filepath.Join(dir, config.FileName), body)
			c, err := config.Load(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}

			f := &ownerFixture{root: c}
			f.withCLIState(t, func() { dryRun = tc.dryRun })
			err = provisionCmd.RunE(provisionCmd, []string{tc.profile})
			if err == nil || !strings.Contains(err.Error(), "${A:?x}") {
				t.Fatalf("provision error = %v, want the env_file failure", err)
			}
			if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
				t.Fatalf("step ran despite env_file failure: stat = %v", statErr)
			}
		})
	}
}
