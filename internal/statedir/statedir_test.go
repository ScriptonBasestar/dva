package statedir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootUnderFreshBase(t *testing.T) {
	base := t.TempDir()
	root, err := RootUnder(base)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(base, "sb", "dva"); root != want {
		t.Fatalf("root = %s, want %s", root, want)
	}
	if _, err := os.Lstat(filepath.Join(base, "dva")); !os.IsNotExist(err) {
		t.Fatalf("fresh base must not gain a legacy link: %v", err)
	}
}

func TestRootUnderMigratesLegacyAndLinksIt(t *testing.T) {
	base := t.TempDir()
	legacyFile := filepath.Join(base, "dva", "ci", "receipt.json")
	if err := os.MkdirAll(filepath.Dir(legacyFile), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyFile, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := RootUnder(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "ci", "receipt.json")); err != nil {
		t.Fatalf("migrated data missing: %v", err)
	}
	link, err := os.Readlink(filepath.Join(base, "dva"))
	if err != nil || link != filepath.Join("sb", "dva") {
		t.Fatalf("legacy link = %q, %v", link, err)
	}
	// Legacy path keeps resolving to the same data for older binaries.
	if _, err := os.Stat(legacyFile); err != nil {
		t.Fatalf("legacy path no longer resolves: %v", err)
	}
	// Second call is a no-op.
	if again, err := RootUnder(base); err != nil || again != root {
		t.Fatalf("second call = %s, %v", again, err)
	}
}

func TestRootUnderRefusesToMergeBothLayouts(t *testing.T) {
	base := t.TempDir()
	for _, dir := range []string{filepath.Join(base, "dva"), filepath.Join(base, "sb", "dva")} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := RootUnder(base); err == nil {
		t.Fatal("expected an error when both legacy and new state exist")
	}
}
