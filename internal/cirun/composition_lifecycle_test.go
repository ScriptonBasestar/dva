//go:build darwin || linux || freebsd || netbsd || openbsd

package cirun

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func compositionScopeFixture(t *testing.T) (compositionScope, compositionGrant) {
	t.Helper()
	grant, _, _ := compositionTestGrant(t)
	grant.Ancestry = grant.Ancestry[:1]
	grant.Deadline = time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano)
	parent := grant.Ancestry[0]
	locks, err := acquire(grant.LockState, parent.Root, parent.RunID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(locks.release)
	scope := scopeFor(grant.LockState, parent)
	if err := scope.createGrant(grant); err != nil {
		t.Fatal(err)
	}
	return scope, grant
}

func TestCompositionScopeAdmissionClosure(t *testing.T) {
	scope, grant := compositionScopeFixture(t)
	state := grant.LockState
	firstRoot, secondRoot := compositionTestDir(t), compositionTestDir(t)
	firstID, secondID := strings.Repeat("b", 32), strings.Repeat("c", 32)
	parent, value := compositionTestParent(t, grant), compositionTestJSON(t, grant)
	first, _, err := acquireComposition(context.Background(), state, firstRoot, firstID, state, parent, value, nil)
	if err != nil {
		t.Fatal(err)
	}
	firstHeld := true
	t.Cleanup(func() {
		if firstHeld {
			first.release()
		}
	})
	_, _, err = acquireComposition(context.Background(), state, secondRoot, secondID, state, parent, value, nil)
	var busy *BusyError
	if !errors.As(err, &busy) || busy.Conflict.Kind != "composition_child" || busy.ActiveRunID != firstID {
		t.Fatalf("duplicate concurrent grant: %v", err)
	}
	first.release()
	firstHeld = false
	if err := finishCompositionChild(scope, state, Report{ID: firstID, Root: firstRoot}, nil); err != nil {
		t.Fatal(err)
	}
	second, _, err := acquireComposition(context.Background(), state, secondRoot, secondID, state, parent, value, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer second.release()
	if err := scope.close(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if err := scope.canceled(); err != nil {
		t.Fatalf("normal closure canceled admitted child: %v", err)
	}
	_, _, err = acquireComposition(context.Background(), state, compositionTestDir(t), strings.Repeat("d", 32), state, parent, value, nil)
	if err == nil {
		t.Fatal("closed scope admitted delayed child")
	}
	if err := scope.close(context.Background(), true); err != nil {
		t.Fatal(err)
	}
	if scope.canceled() == nil {
		t.Fatal("cancel marker not observed")
	}
}

func TestCompositionScopeCloseRegisterRace(t *testing.T) {
	scope, grant := compositionScopeFixture(t)
	state, root, id := grant.LockState, compositionTestDir(t), strings.Repeat("b", 32)
	parent, value := compositionTestParent(t, grant), compositionTestJSON(t, grant)
	var wg sync.WaitGroup
	var registered, closed error
	var locks heldLocks
	wg.Add(2)
	go func() {
		defer wg.Done()
		locks, _, registered = acquireComposition(context.Background(), state, root, id, state, parent, value, nil)
	}()
	go func() { defer wg.Done(); closed = scope.close(context.Background(), false) }()
	wg.Wait()
	defer locks.release()
	if closed != nil {
		t.Fatal(closed)
	}
	children, err := scope.children()
	if err != nil {
		t.Fatal(err)
	}
	if (registered == nil) != (len(children) == 1) {
		t.Fatalf("admission lost: registered=%v children=%+v", registered, children)
	}
	owner, err := lockOwner(state, "root", root)
	if err != nil || (registered == nil && owner != id) || (registered != nil && owner != "") {
		t.Fatalf("registration/lock not atomic: admitted=%v owner=%s err=%v", registered, owner, err)
	}
	if scope.open() == nil {
		t.Fatal("closure lost")
	}
}

func TestCompositionScopeCorruptRecordFailsClosed(t *testing.T) {
	scope, grant := compositionScopeFixture(t)
	if err := os.WriteFile(filepath.Join(scope.dir, strings.Repeat("b", 32)+".json"), []byte(`{"finished":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := acquireComposition(context.Background(), grant.LockState, compositionTestDir(t), strings.Repeat("c", 32), grant.LockState, compositionTestParent(t, grant), compositionTestJSON(t, grant), nil)
	if err == nil {
		t.Fatal("corrupt child record ignored")
	}
}
