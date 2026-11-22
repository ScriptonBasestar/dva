package resourceadmission

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func yieldFixture(t *testing.T) (string, Policy, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "resource")
	path := filepath.Join(t.TempDir(), "policy.json")
	if err := os.WriteFile(path, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	p, digest, err := ReadPolicy(path)
	if err != nil {
		t.Fatal(err)
	}
	return dir, p, digest
}

// queueWaiter starts a request that stays waiting until the test ends.
func queueWaiter(t *testing.T, dir string, p Policy, digest, class, priority string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	done := make(chan struct{})
	go func() {
		defer close(done)
		if lease, err := Acquire(ctx, dir, p, digest, class, priority); err == nil {
			_ = lease.Complete(context.Background())
		}
	}()
	t.Cleanup(func() { cancel(); <-done })
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		entries, _ := readEntries(dir)
		for _, e := range entries {
			if e.State == "waiting" && e.Priority == priority && e.Class == class {
				return
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("waiter not queued")
}

func TestYieldWanted(t *testing.T) {
	for _, tc := range []struct {
		name, own, waiterClass, waiterPriority string
		want, otherHeavy                       bool
	}{
		{"no waiter", "normal", "", "", false, false},
		{"higher heavy waiter blocked by own claim", "background", "heavy", "interactive", true, false},
		{"equal priority heavy waiter", "normal", "heavy", "normal", true, false},
		{"lower priority heavy waiter", "normal", "heavy", "background", false, false},
		{"release would not admit the waiter", "background", "heavy", "interactive", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, p, digest := yieldFixture(t)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			ownClass := "heavy"
			if tc.otherHeavy {
				ownClass = "light"
				other, err := Acquire(ctx, dir, p, digest, "heavy", "normal")
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = other.Complete(context.Background()) }()
			}
			own, err := Acquire(ctx, dir, p, digest, ownClass, tc.own)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = own.Complete(context.Background()) }()
			if tc.waiterClass != "" {
				queueWaiter(t, dir, p, digest, tc.waiterClass, tc.waiterPriority)
			}
			got, err := YieldWanted(ctx, dir, p, digest, own.ID(), tc.own)
			if err != nil || got != tc.want {
				t.Fatalf("YieldWanted=%v err=%v, want %v", got, err, tc.want)
			}
		})
	}
}

func TestYieldWantedIsReadOnlyAndIgnoresAbandonedWaiters(t *testing.T) {
	dir, p, digest := yieldFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	own, err := Acquire(ctx, dir, p, digest, "heavy", "background")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = own.Complete(context.Background()) }()
	// An abandoned waiting request: recorded, lease lock not held.
	abandoned := entry{ID: "0123456789abcdef0123456789abcdef", Policy: digest, Class: "heavy", Priority: "interactive", State: "waiting", QueuedAt: time.Now().UTC()}
	if err := writeEntry(dir, abandoned); err != nil {
		t.Fatal(err)
	}
	lock, err := openPrivateRegular(filepath.Join(dir, abandoned.ID+".lock"), os.O_CREATE|os.O_EXCL|os.O_RDWR)
	if err != nil {
		t.Fatal(err)
	}
	_ = lock.Close()
	got, err := YieldWanted(ctx, dir, p, digest, own.ID(), "background")
	if err != nil || got {
		t.Fatalf("abandoned waiter counted: %v %v", got, err)
	}
	if _, err := os.Stat(filepath.Join(dir, abandoned.ID+".json")); err != nil {
		t.Fatalf("query changed state: %v", err)
	}
}
