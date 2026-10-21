package cirun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// The supervisor's subproject name reaches the step process. Run rejects a
// step environment that sets it, and ciEnv keeps the supervisor's value too.
func TestRunStepReceivesSupervisorSubproject(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "scoped", StateDir: state, lockDirectory: state,
		Env:     append(os.Environ(), config.CISubprojectEnv+"=site-a"),
		Profile: profile(config.CIStep{Name: "record", Run: `printf %s "$DVA_CI_SUBPROJECT" > seen`}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "succeeded" {
		t.Fatalf("status %q", r.Status)
	}
	seen, err := os.ReadFile(filepath.Join(root, "seen"))
	if err != nil || string(seen) != "site-a" {
		t.Fatalf("step saw %q (err %v), want site-a", seen, err)
	}
	env := "\n" + strings.Join(ciEnv([]string{config.CISubprojectEnv + "=site-a"}, map[string]string{config.CISubprojectEnv: "spoofed"}, 1), "\n") + "\n"
	if !strings.Contains(env, "\n"+config.CISubprojectEnv+"=site-a\n") {
		t.Fatalf("step environment replaced the supervisor value: %s", env)
	}
}
