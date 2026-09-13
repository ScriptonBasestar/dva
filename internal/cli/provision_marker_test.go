package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// writeMarkerFile puts a file in the dot directory under an exact name, which is what these
// tests need and writeProvisionMarker cannot give: the legacy spelling is read but never
// written, so there is no production writer to borrow for it.
func writeMarkerFile(t *testing.T, configDir, name string) string {
	t.Helper()
	dir := filepath.Join(configDir, config.DotDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, nil, 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
	return p
}

// The rename must not make a provisioned repository look unprovisioned. Anyone who ran
// provision before the extension existed has a bare marker on disk, and if that stops
// counting, `dva up` starts suggesting a profile that has already run — and `--purge` stops
// clearing it, so the file lingers forever.
func TestProvisionMarkerExistsAcceptsBothSpellings(t *testing.T) {
	for _, tc := range []struct {
		name    string
		spelled func(string) string
	}{
		{"current", provisionMarkerName},
		{"legacy", legacyProvisionMarkerName},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if provisionMarkerExists(dir, "setup") {
				t.Fatal("premise failed: marker reported before anything was written")
			}
			writeMarkerFile(t, dir, tc.spelled("setup"))
			if !provisionMarkerExists(dir, "setup") {
				t.Errorf("marker spelled %q was not recognised", tc.spelled("setup"))
			}
		})
	}
}

// writeProvisionMarker writes the current spelling and only that one. Asserted separately
// from the reader above, because a writer that silently kept emitting the legacy name would
// pass every read test while leaving the collision this change exists to remove.
func TestWriteProvisionMarkerUsesTheExtension(t *testing.T) {
	dir := t.TempDir()
	writeProvisionMarker(dir, "setup")

	if filepath.Ext(provisionMarkerName("setup")) != provisionMarkerExt {
		t.Fatalf("premise failed: %q carries no extension", provisionMarkerName("setup"))
	}
	if _, err := os.Stat(filepath.Join(dir, config.DotDirName, legacyProvisionMarkerName("setup"))); err == nil {
		t.Error("writeProvisionMarker wrote the legacy spelling; it is read-only")
	}
}

// The bug the extension was introduced for: a module is a file the user wrote and must
// commit, and `dva down --purge` deleted it because `provisioned-base.yml` carries the
// marker prefix. The dry-run listing and the delete run off one predicate, so this covers
// both — but the delete is asserted for real, since a preview that lies is a smaller
// failure than data loss.
func TestProvisionMarkersNeverClaimAModule(t *testing.T) {
	dir := t.TempDir()

	module := writeMarkerFile(t, dir, "provisioned-base"+moduleExt)
	writeProvisionMarker(dir, "setup")
	legacy := writeMarkerFile(t, dir, legacyProvisionMarkerName("old"))

	listed := provisionMarkers(dir)
	if slices.Contains(listed, module) {
		t.Errorf("module %q listed as a provision marker: %v", module, listed)
	}
	if len(listed) != 2 {
		t.Errorf("expected both markers listed and the module left out, got %v", listed)
	}

	clearProvisionMarkers(dir)

	if _, err := os.Stat(module); err != nil {
		t.Errorf("--purge deleted the user's module: %v", err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Error("legacy marker survived --purge; it would suppress provisioning forever")
	}
	if provisionMarkerExists(dir, "setup") {
		t.Error("current marker survived --purge")
	}
}

// A profile name containing a dot is why isProvisionMarker excludes modules rather than
// requiring a marker shape: under a positive "ends in .marker or has no extension" rule this
// legacy file has extension ".0" and would never be cleared.
func TestLegacyMarkerWithADottedProfileIsStillAMarker(t *testing.T) {
	dir := t.TempDir()
	marker := writeMarkerFile(t, dir, legacyProvisionMarkerName("setup-v1.0"))

	if filepath.Ext(marker) == "" {
		t.Fatalf("premise failed: %q has no extension, so it proves nothing", marker)
	}

	clearProvisionMarkers(dir)

	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Errorf("dotted legacy marker %q was not cleared", filepath.Base(marker))
	}
}
