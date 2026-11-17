//go:build darwin || linux || freebsd || netbsd || openbsd

package resourceadmission

import (
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
