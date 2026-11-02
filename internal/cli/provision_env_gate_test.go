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
func TestProvisionStopsOnEnvFileFailure(t *testing.T) {
	for _, tc := range []struct {
		name   string
		dryRun bool
	}{{"live", false}, {"dry-run", true}} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "ran.out")
			writeFile(t, filepath.Join(dir, config.FileName), `version: "0.1.0"
project_name: gate
env_file: ${A:?x}/.env
provision:
  default:
    - step: record
      run: 'touch "`+marker+`"'
`)
			c, err := config.Load(dir)
			if err != nil {
				t.Fatalf("load: %v", err)
			}

			f := &ownerFixture{root: c}
			f.withCLIState(t, func() { dryRun = tc.dryRun })
			err = provisionCmd.RunE(provisionCmd, []string{"default"})
			if err == nil || !strings.Contains(err.Error(), "${A:?x}") {
				t.Fatalf("provision error = %v, want the env_file failure", err)
			}
			if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
				t.Fatalf("step ran despite env_file failure: stat = %v", statErr)
			}
		})
	}
}
