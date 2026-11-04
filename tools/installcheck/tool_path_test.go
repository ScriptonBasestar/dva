package installcheck

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// Expose only the install recipe's tools. In particular, PATH diagnostics must
// inspect fixture DVA binaries, never the developer's installed CLI or the
// native-recovery guard. Tests can prepend their own failing tool fixtures.
func isolateInstallerPath(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{
		"make", "sh", "cat", "chmod", "cmp", "cp", "head", "mkdir",
		"mktemp", "mv", "rm", "sleep", "git", "shasum", "grep", "sed", "column",
	} {
		path, err := exec.LookPath(name)
		if err != nil {
			t.Fatalf("locate installer tool %s: %v", name, err)
		}
		if err := os.Symlink(path, filepath.Join(dir, name)); err != nil {
			t.Fatalf("expose installer tool %s: %v", name, err)
		}
	}
	t.Setenv("PATH", dir)
}
