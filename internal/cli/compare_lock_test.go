//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"context"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/compare"
)

// While one `compare up` of a name holds the lock, a second one of the same name fails before it
// resolves anything or starts a process, and leaves the first one's record alone.
func TestCompareUpRefusesWhileAnotherUpHoldsTheLock(t *testing.T) {
	c, host, _ := compareFixture(t)
	release, err := compare.Lock(c.FileDir(), "layout")
	if err != nil {
		t.Fatal(err)
	}
	_, err = upConfig(t, c, "a=config")
	if err == nil || errorCode(err) != codeComparePreflight || !strings.Contains(err.Error(), "another `dva compare up layout` is running") {
		t.Fatalf("err = %v", err)
	}
	if len(host.calls) != 0 {
		t.Errorf("a locked up must start nothing: %v", host.calls)
	}
	if _, err := compare.ReadRecord(c.FileDir(), "layout"); err == nil {
		t.Error("a locked up wrote a record")
	}
	release()
	if _, err := upConfig(t, c, "a=config"); err != nil {
		t.Fatalf("up after the lock was released: %v", err)
	}
}

// While a `compare up` of the same name holds the lock, `compare down` fails before it stops
// anything or touches the record, and works once the lock is released.
func TestCompareDownRefusesWhileAnotherRunHoldsTheLock(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := upConfig(t, c, "a=config"); err != nil {
		t.Fatal(err)
	}
	host.calls = nil
	release, err := compare.Lock(c.FileDir(), "layout")
	if err != nil {
		t.Fatal(err)
	}
	_, err = captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) })
	if err == nil || errorCode(err) != codeComparePreflight || !strings.Contains(err.Error(), "nothing was stopped: another `dva compare` of layout is running") {
		t.Fatalf("err = %v", err)
	}
	if len(host.calls) != 0 {
		t.Errorf("a locked down must stop nothing: %v", host.calls)
	}
	if _, err := compare.ReadRecord(c.FileDir(), "layout"); err != nil {
		t.Errorf("a locked down must leave the record alone: %v", err)
	}
	release()
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) }); err != nil {
		t.Fatalf("down after the lock was released: %v", err)
	}
}

// A down releases the lock on every way out: success, a nothing-to-take-down preflight failure
// and a dry run (which never takes it).
func TestCompareDownReleasesTheLock(t *testing.T) {
	c, _, _ := compareFixture(t)
	// No record: the preflight failure after the lock was taken.
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) }); err == nil {
		t.Fatal("down without a record must fail")
	}
	release, err := compare.Lock(c.FileDir(), "layout")
	if err != nil {
		t.Fatalf("lock still held after a failed down: %v", err)
	}
	release()
	if _, err := upConfig(t, c, "a=config"); err != nil {
		t.Fatal(err)
	}
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) }); err != nil {
		t.Fatal(err)
	}
	release, err = compare.Lock(c.FileDir(), "layout")
	if err != nil {
		t.Fatalf("lock still held after a successful down: %v", err)
	}
	release()
}

// A dry-run down takes no lock, so it still reports the plan while another run holds it.
func TestCompareDownDryRunIgnoresTheLock(t *testing.T) {
	c, host, _ := compareFixture(t)
	if _, err := upConfig(t, c, "a=config"); err != nil {
		t.Fatal(err)
	}
	host.calls = nil
	release, err := compare.Lock(c.FileDir(), "layout")
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	old := dryRun
	dryRun = true
	defer func() { dryRun = old }()
	if _, err := captureJSON(t, func() error { return runCompareDown(context.Background(), c, "layout", nil) }); err != nil {
		t.Fatalf("dry-run down under a held lock: %v", err)
	}
	if len(host.calls) != 0 {
		t.Errorf("dry run stopped something: %v", host.calls)
	}
}
