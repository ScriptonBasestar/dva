package installcheck

import (
	"path/filepath"
	"strings"
	"testing"
)

// An explicitly empty GO_BIN_DIR restores the second copy, resolved from GOBIN first.
func TestInstallerOnlyTargetFallsBackToGOBINWhenGoBinDirEmpty(t *testing.T) {
	isolateInstallerPath(t)
	repo := repositoryRoot(t)
	fixture := t.TempDir()
	source := fixtureDVA(t, fixture)
	localDir := filepath.Join(fixture, "local", "bin")
	goDir := filepath.Join(fixture, "gobin")

	output := runInstaller(t, repo, source, "LOCAL_BIN_DIR="+localDir, "GO_BIN_DIR=", "GOBIN="+goDir)
	if !strings.Contains(output, "make install: verified Go destination:") {
		t.Fatalf("install did not verify the GOBIN destination:\n%s", output)
	}
	assertSameBinary(t, source, filepath.Join(localDir, "dva"))
	assertSameBinary(t, source, filepath.Join(goDir, "dva"))
}
