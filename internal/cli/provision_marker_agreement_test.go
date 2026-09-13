package cli

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// markerTransientClass picks the marker class out of the doctor table by what it asks about
// rather than by position, so reordering the table does not silently retarget this test at the
// pids class — where every assertion below would pass vacuously.
func markerTransientClass(t *testing.T) transientClass {
	t.Helper()

	prefix := path.Join(config.DotDirName, provisionMarkerPrefix)
	for _, c := range dvaTransientClasses() {
		if strings.HasPrefix(c.name, prefix) {
			return c
		}
	}
	t.Fatalf("no transient class asks about provision markers")
	return transientClass{}
}

// TestDoctorAndPurgeAgreeOnWhatAMarkerIs holds the two answers to "is this file DVA's own
// marker?" against each other over one fixture.
//
// They are computed by different machinery that cannot be shared: doctor asks git with a
// pathspec, because the question is about the index and only git can read it, while `--purge`
// walks the directory with isProvisionMarker, because it deletes and an index has nothing to do
// with that. So the agreement is a coupling kept by hand, and every disagreement is a defect in
// one direction or the other. A file doctor reports and purge will not delete is doctor naming
// something as DVA's to manage that DVA does not; a file purge deletes without doctor reporting
// is a deletion nobody was warned about — that one was the original bug, when the bare prefix
// matched a user's `provisioned-base.yml`.
//
// Both sides are also checked against the table, not only against each other, because two
// predicates that drift together are still wrong and agreeing about it proves nothing.
func TestDoctorAndPurgeAgreeOnWhatAMarkerIs(t *testing.T) {
	cases := []struct {
		name   string // relative to the dot directory
		marker bool
	}{
		{name: provisionMarkerName("default"), marker: true},
		{name: legacyProvisionMarkerName("legacy"), marker: true},
		// A profile whose name carries a dot: the legacy spelling gives it extension ".0",
		// which is exactly the case a positive "looks like a marker" rule would strand.
		{name: legacyProvisionMarkerName("setup-v1.0"), marker: true},
		{name: legacyProvisionMarkerName("child/profile"), marker: true},
		// Authored content wearing the prefix, in all three spellings the exclusions name.
		{name: provisionMarkerPrefix + "base.yml"},
		{name: provisionMarkerPrefix + "base.yaml"},
		{name: provisionMarkerPrefix + "notes.md"},
		// Content without the prefix, so neither side has any business with it.
		{name: "modules.yml"},
	}

	dir := t.TempDir()
	initGitRepo(t, dir)
	for _, tt := range cases {
		full := filepath.Join(dir, config.DotDirName, filepath.FromSlash(tt.name))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("MkdirAll %q: %v", tt.name, err)
		}
		if err := os.WriteFile(full, []byte("x\n"), 0o644); err != nil {
			t.Fatalf("WriteFile %q: %v", tt.name, err)
		}
		gitInRepo(t, dir, "add", "-f", path.Join(config.DotDirName, tt.name))
	}

	class := markerTransientClass(t)
	out := gitInRepo(t, dir, append([]string{"ls-files", "--cached", "--"}, class.pathspecs()...)...)
	fromDoctor := map[string]bool{}
	for _, line := range strings.Fields(out) {
		fromDoctor[line] = true
	}

	entries, err := os.ReadDir(filepath.Join(dir, config.DotDirName))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	fromPurge := map[string]bool{}
	for _, e := range entries {
		if isProvisionMarker(e) {
			fromPurge[path.Join(config.DotDirName, e.Name())] = true
		}
	}

	for _, tt := range cases {
		full := path.Join(config.DotDirName, tt.name)
		if fromDoctor[full] != tt.marker {
			t.Errorf("doctor: %q reported as a marker = %v, want %v", full, fromDoctor[full], tt.marker)
		}
		if fromPurge[full] != tt.marker {
			t.Errorf("--purge: %q treated as a marker = %v, want %v", full, fromPurge[full], tt.marker)
		}
		delete(fromDoctor, full)
		delete(fromPurge, full)
	}
	// Whatever is left is a name the table never wrote — the directory entry itself, say, which
	// only appears once a side starts claiming directories.
	for leftover := range fromDoctor {
		t.Errorf("doctor claimed %q, which the fixture never wrote as a file", leftover)
	}
	for leftover := range fromPurge {
		t.Errorf("--purge claimed %q, which the fixture never wrote as a file", leftover)
	}
}
