//go:build darwin || linux || freebsd || netbsd || openbsd

package compare

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// A second Lock of the same comparison fails at once; another comparison is unaffected; the
// lock can be taken again after release.
func TestLockExcludesSecondHolderOfSameComparison(t *testing.T) {
	dir := t.TempDir()
	release, err := Lock(dir, "layout")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Lock(dir, "layout"); !errors.Is(err, ErrLocked) {
		t.Fatalf("second Lock err = %v, want ErrLocked", err)
	}
	other, err := Lock(dir, "other")
	if err != nil {
		t.Fatalf("a different comparison must not be locked: %v", err)
	}
	other()
	release()
	again, err := Lock(dir, "layout")
	if err != nil {
		t.Fatalf("Lock after release: %v", err)
	}
	again()
}

// The lock belongs to the process: when its holder dies without releasing, the file stays but
// locks nothing.
func TestLockDoesNotSurviveItsProcess(t *testing.T) {
	if os.Getenv("DVA_LOCK_HOLDER") == "1" {
		if _, err := Lock(os.Getenv("DVA_LOCK_DIR"), "layout"); err != nil {
			os.Exit(3)
		}
		os.Exit(0) // exits holding the lock, never releasing it
	}
	dir := t.TempDir()
	cmd := exec.Command(os.Args[0], "-test.run=^TestLockDoesNotSurviveItsProcess$")
	cmd.Env = append(os.Environ(), "DVA_LOCK_HOLDER=1", "DVA_LOCK_DIR="+dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("holder process: %v\n%s", err, out)
	}
	path, _ := LockPath(dir, "layout")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the holder should have left its lock file: %v", err)
	}
	release, err := Lock(dir, "layout")
	if err != nil {
		t.Fatalf("a dead process's lock file still locks: %v", err)
	}
	release()
	if strings.HasSuffix(path, ".json") {
		t.Errorf("lock path %s collides with the record", path)
	}
}
