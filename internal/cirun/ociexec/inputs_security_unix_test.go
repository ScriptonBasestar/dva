//go:build darwin || linux

package ociexec

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestSnapshotInputRejectsFIFO(t *testing.T) {
	_, source, destination := snapshotDirs(t)
	if err := syscall.Mkfifo(filepath.Join(source, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotInput(filepath.Dir(source), "source", destination); err == nil {
		t.Fatal("FIFO accepted")
	}
}

func TestOpenInputFileRefusesFIFOWithoutBlocking(t *testing.T) {
	root := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(root, "pipe"), 0o600); err != nil {
		t.Fatal(err)
	}
	descriptor, err := os.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer descriptor.Close()
	file, err := openInputFile(descriptor, "pipe")
	if err == nil {
		_ = file.Close()
		t.Fatal("FIFO open accepted")
	}
}

func TestRemoveSnapshotAtRefusesInputSymlink(t *testing.T) {
	parentPath := filepath.Join(t.TempDir(), "parent")
	if err := os.Mkdir(parentPath, 0o700); err != nil {
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
	if err := os.Symlink(outside, filepath.Join(parentPath, "input")); err != nil {
		t.Fatal(err)
	}
	parent, err := os.OpenRoot(parentPath)
	if err != nil {
		t.Fatal(err)
	}
	defer parent.Close()
	if err := removeSnapshotAt(parent); err == nil {
		t.Fatal("input symlink accepted")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("outside touched: %v", err)
	}
}
