package cirun

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParentMetadataRejectsMalformedAndLookupErrors(t *testing.T) {
	state := t.TempDir()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 32)
	for _, value := range []string{"null", `{}`, `{"id":"bad","root":"/tmp"}`, `{"id":"` + id + `","root":"relative"}`, `{"id":"` + id + `","root":"` + root + `"} {}`} {
		if err := checkParent(state, value); err == nil || errors.Is(err, ErrBusy) {
			t.Fatalf("metadata %q: %v", value, err)
		}
	}
	token, err := json.Marshal(parentRun{ID: id, Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(lockPath(state, "root", root), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := checkParent(state, string(token)); err == nil || errors.Is(err, ErrBusy) {
		t.Fatalf("lookup failure must not be stale or busy: %v", err)
	}
	if err := writeReport(state, Report{ID: id, Root: root, Status: "running", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if _, err := Status(state); err == nil {
		t.Fatal("status hid lock lookup failure")
	}
}

func TestScopedLockAccessErrorRollsBack(t *testing.T) {
	state, root := t.TempDir(), t.TempDir()
	// A directory at the lock-file location makes open fail, even for privileged users.
	if err := os.MkdirAll(lockPath(state, "resource", "z"), 0o700); err != nil {
		t.Fatal(err)
	}
	_, err := acquire(state, root, strings.Repeat("a", 32), "a", "z")
	if err == nil || errors.Is(err, ErrBusy) {
		t.Fatalf("expected file access failure: %v", err)
	}
	h, err := acquire(state, root, strings.Repeat("b", 32), "a")
	if err != nil {
		t.Fatalf("partial acquisition survived error: %v", err)
	}
	h.release()
}

func TestUnknownLockOwnerIsNotInvented(t *testing.T) {
	state, root := t.TempDir(), t.TempDir()
	h, err := acquire(state, root, "invalid-metadata")
	if err != nil {
		t.Fatal(err)
	}
	defer h.release()
	_, err = acquire(state, root, strings.Repeat("b", 32))
	var busy *BusyError
	if !errors.As(err, &busy) || busy.ActiveRunID != "" || busy.Conflict.RunID != "" {
		t.Fatalf("unknown occupant: %#v %v", busy, err)
	}
}

func TestReceiptWithoutIdentityCannotBeLive(t *testing.T) {
	active, err := receiptActive(t.TempDir(), t.TempDir(), "")
	if err != nil || active {
		t.Fatalf("missing identity appears live: %v %v", active, err)
	}
}
