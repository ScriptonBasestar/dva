package ociexec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func snapshotDirs(t *testing.T) (string, string, string) {
	t.Helper()
	root := t.TempDir()
	source, destination := filepath.Join(root, "source"), filepath.Join(root, "destination")
	if err := os.MkdirAll(source, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(destination, 0o755); err != nil {
		t.Fatal(err)
	}
	return root, source, destination
}

func TestSnapshotInputRejectsSparseAndEntryDepthBounds(t *testing.T) {
	_, source, destination := snapshotDirs(t)
	if err := os.WriteFile(filepath.Join(source, "large"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(filepath.Join(source, "large"), maxInputBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotInput(filepath.Dir(source), "source", destination); err == nil || !strings.Contains(err.Error(), "64 MiB") {
		t.Fatalf("sparse bound: %v", err)
	}

	_, source, destination = snapshotDirs(t)
	for i := 0; i <= maxInputFiles; i++ {
		if err := os.WriteFile(filepath.Join(source, fmt.Sprintf("f%05d", i)), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := snapshotInput(filepath.Dir(source), "source", destination); err == nil || !strings.Contains(err.Error(), "10000") {
		t.Fatalf("entry bound: %v", err)
	}

	_, source, destination = snapshotDirs(t)
	current := source
	for range 33 {
		current = filepath.Join(current, "d")
		if err := os.Mkdir(current, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := snapshotInput(filepath.Dir(source), "source", destination); err == nil || !strings.Contains(err.Error(), "32") {
		t.Fatalf("depth bound: %v", err)
	}
}

func TestSnapshotInputDigestIsStableAcrossCreationOrder(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"first", "second"} {
		source, destination := filepath.Join(root, name, "source"), filepath.Join(root, name, "destination")
		if err := os.MkdirAll(source, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(destination, 0o755); err != nil {
			t.Fatal(err)
		}
		files := []string{"a", "b", "nested/c"}
		if name == "second" {
			files = []string{"nested/c", "b", "a"}
		}
		for _, file := range files {
			if err := os.MkdirAll(filepath.Dir(filepath.Join(source, file)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(source, file), []byte(file), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	a, err := snapshotInput(filepath.Join(root, "first"), "source", filepath.Join(root, "first", "destination"))
	if err != nil {
		t.Fatal(err)
	}
	b, err := snapshotInput(filepath.Join(root, "second"), "source", filepath.Join(root, "second", "destination"))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("digest differs by creation order: %s != %s", a, b)
	}
}

func TestSnapshotInputRejectsNestedSymlinkAndSpecialObject(t *testing.T) {
	root, source, destination := snapshotDirs(t)
	if err := os.Symlink(filepath.Join(root, "outside"), filepath.Join(source, "link")); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotInput(filepath.Dir(source), "source", destination); err == nil {
		t.Fatal("nested symlink accepted")
	}
}

func TestRemoveSnapshotAtKeepsReplacementOutside(t *testing.T) {
	parentPath := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(parentPath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(parentPath, "input"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(parentPath, "input", "owned"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	moved := parentPath + "-moved"
	if err := os.Rename(parentPath, moved); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.Mkdir(outside, 0o700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(outside, "marker")
	if err := os.WriteFile(marker, []byte("safe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, parentPath); err != nil {
		t.Fatal(err)
	}
	if err := removeSnapshotAt(parent); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("outside touched: %v", err)
	}
}

func TestSecureStateDirectoryRefusesExistingPermissiveDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state")
	if err := os.Mkdir(path, 0o770); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o770); err != nil {
		t.Fatal(err)
	}
	if _, err := secureStateDirectory(path); err == nil {
		t.Fatal("0770 state accepted")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o770 {
		t.Fatalf("state mode changed to %o", info.Mode().Perm())
	}
}
