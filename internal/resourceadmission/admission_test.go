package resourceadmission

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

func TestCapacityAndCanceledWait(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	p := Policy{Capacity: 4, MaxRunning: 3, MaxHeavy: 1, Weights: map[string]int{"heavy": 2, "light": 1}, AgingInterval: "1m"}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	heavy, err := Acquire(ctx, dir, p, "fingerprint", "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	one, err := Acquire(ctx, dir, p, "fingerprint", "light", "normal")
	if err != nil {
		t.Fatal(err)
	}
	two, err := Acquire(ctx, dir, p, "fingerprint", "light", "normal")
	if err != nil {
		t.Fatal(err)
	}
	wait, cancelWait := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancelWait()
	if _, err := Acquire(wait, dir, p, "fingerprint", "light", "normal"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected timeout, got %v", err)
	}
	if err = one.Complete(ctx); err != nil {
		t.Fatal(err)
	}
	if err = two.Complete(ctx); err != nil {
		t.Fatal(err)
	}
	if err = heavy.Complete(ctx); err != nil {
		t.Fatal(err)
	}
	entries, err := readEntries(dir)
	if err != nil || len(entries) != 0 {
		t.Fatalf("left entries: %+v %v", entries, err)
	}
}

func TestBlockedHeavyDoesNotOccupyLightSlots(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	p := Policy{Capacity: 4, MaxRunning: 3, MaxHeavy: 1, Weights: map[string]int{"heavy": 2, "light": 1}, AgingInterval: "1m"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	heavy, err := Acquire(ctx, dir, p, "same", "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	waitCtx, stop := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { _, err := Acquire(waitCtx, dir, p, "same", "heavy", "normal"); done <- err }()
	for {
		status, err := Status(ctx, dir, "same")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range status {
			if e.State == "waiting" {
				found = true
			}
		}
		if found {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("waiting request not registered")
		case <-time.After(10 * time.Millisecond):
		}
	}
	light, err := Acquire(ctx, dir, p, "same", "light", "normal")
	if err != nil {
		t.Fatal(err)
	}
	stop()
	if err = <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("wait cancellation: %v", err)
	}
	if err = light.Complete(ctx); err != nil {
		t.Fatal(err)
	}
	if err = heavy.Complete(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestAgedHeavyReservesCapacityAgainstRollingLights(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	p := Policy{Capacity: 2, MaxRunning: 2, MaxHeavy: 1, Weights: map[string]int{"heavy": 2, "light": 1}, AgingInterval: "10ms"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	light, err := Acquire(ctx, dir, p, "same", "light", "normal")
	if err != nil {
		t.Fatal(err)
	}
	acquired := make(chan *Lease, 1)
	fail := make(chan error, 1)
	go func() {
		lease, err := Acquire(ctx, dir, p, "same", "heavy", "normal")
		if err != nil {
			fail <- err
			return
		}
		acquired <- lease
	}()
	for {
		entries, err := Status(ctx, dir, "same")
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range entries {
			if e.State == "waiting" && e.Class == "heavy" {
				found = true
			}
		}
		if found {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("heavy not waiting")
		case <-time.After(10 * time.Millisecond):
		}
	}
	time.Sleep(30 * time.Millisecond)
	wait, stop := context.WithTimeout(ctx, 150*time.Millisecond)
	defer stop()
	if _, err := Acquire(wait, dir, p, "same", "light", "normal"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("light should wait for aged heavy: %v", err)
	}
	if err = light.Complete(ctx); err != nil {
		t.Fatal(err)
	}
	select {
	case h := <-acquired:
		if err = h.Complete(ctx); err != nil {
			t.Fatal(err)
		}
	case err = <-fail:
		t.Fatal(err)
	case <-ctx.Done():
		t.Fatal("heavy starved")
	}
}
