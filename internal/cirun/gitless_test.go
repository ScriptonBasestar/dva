package cirun

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestCIWithoutGitRefusesBeforeExecutingSteps(t *testing.T) {
	root, state, tools := t.TempDir(), t.TempDir(), t.TempDir()
	sh, err := exec.LookPath("sh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sh, filepath.Join(tools, "sh")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tools)
	report, err := Run(t.Context(), Options{
		Root: root, StateDir: state, lockDirectory: state,
		ProfileName: "gitless", Env: os.Environ(),
		Profile: profile(config.CIStep{Name: "never", Run: ": > executed"}),
	})
	if err == nil || !errors.Is(err, exec.ErrNotFound) || !strings.Contains(err.Error(), "probe git inputs") {
		t.Fatalf("missing Git error = %v", err)
	}
	if len(report.Steps) != 0 {
		t.Fatalf("profile steps ran before the Git precondition: %+v", report)
	}
	if _, err := os.Stat(filepath.Join(root, "executed")); !os.IsNotExist(err) {
		t.Fatalf("profile step executed: marker stat = %v", err)
	}
}
