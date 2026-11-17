//go:build darwin || linux || freebsd || netbsd || openbsd

package resourceadmission

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestPolicyRefusesFIFOWithoutBlocking(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fifo")
	if err := syscall.Mkfifo(path, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { _, _, err := ReadPolicy(path); done <- err }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("fifo accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("fifo open blocked")
	}
}

func TestStateReadsRefuseFIFOAndSymlink(t *testing.T) {
	dir := t.TempDir()
	fifo := filepath.Join(dir, "state.fifo")
	if err := syscall.Mkfifo(fifo, 0600); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		f, err := openPrivateRegular(fifo, os.O_RDONLY)
		if f != nil {
			_ = f.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FIFO accepted")
		}
	case <-time.After(time.Second):
		t.Fatal("FIFO state read blocked")
	}
	link := filepath.Join(dir, "state.lock")
	if err := os.Symlink(fifo, link); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := locked(ctx, link, func() error { return nil }); err == nil {
		t.Fatal("symlink lock accepted")
	}
}

func TestStateEntryRejectsOversize(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "entry.json"), make([]byte, 8193), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readEntries(dir); err == nil {
		t.Fatal("oversized entry accepted")
	}
}
