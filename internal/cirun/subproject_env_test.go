package cirun

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Options.Subproject is the only source of the step's subproject name: an
// inherited value is replaced, and the stored receipt names the same scope.
func TestRunStepReceivesAndRecordsSubproject(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "scoped", StateDir: state, lockDirectory: state,
		Subproject: "site-a",
		Env:        append(os.Environ(), config.CISubprojectEnv+"=ambient"),
		Profile:    profile(config.CIStep{Name: "record", Run: `printf %s "$DVA_CI_SUBPROJECT" > seen`}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if r.Status != "succeeded" || r.Subproject != "site-a" {
		t.Fatalf("status %q subproject %q", r.Status, r.Subproject)
	}
	seen, err := os.ReadFile(filepath.Join(root, "seen"))
	if err != nil || string(seen) != "site-a" {
		t.Fatalf("step saw %q (err %v), want site-a", seen, err)
	}
	data, err := os.ReadFile(filepath.Join(state, r.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var stored Report
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatal(err)
	}
	if stored.Subproject != "site-a" {
		t.Fatalf("stored receipt subproject %q in %s", stored.Subproject, data)
	}
	env := "\n" + strings.Join(ciEnv([]string{config.CISubprojectEnv + "=site-a"}, map[string]string{config.CISubprojectEnv: "spoofed"}, 1), "\n") + "\n"
	if !strings.Contains(env, "\n"+config.CISubprojectEnv+"=site-a\n") {
		t.Fatalf("step environment replaced the supervisor value: %s", env)
	}
}

// Without --subproject an inherited value never reaches steps, so a step
// guarding with ${DVA_CI_SUBPROJECT:?} fails and the receipt omits the field.
func TestRunWithoutSubprojectDropsInheritedValue(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "unscoped", StateDir: state, lockDirectory: state,
		Env:     append(os.Environ(), config.CISubprojectEnv+"=ambient"),
		Profile: profile(config.CIStep{Name: "guard", Run: `: "${DVA_CI_SUBPROJECT:?}"`}),
	})
	if err == nil || r.Status != "failed" {
		t.Fatalf("status %q err %v, want the guard to fail", r.Status, err)
	}
	data, err := os.ReadFile(filepath.Join(state, r.ID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), `"subproject"`) {
		t.Fatalf("unscoped receipt names a subproject: %s", data)
	}
}
