//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
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
