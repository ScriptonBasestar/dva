//go:build darwin || linux || freebsd || netbsd || openbsd

package resourceadmission

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStatusEmptyStatesMarshalArray(t *testing.T) {
	for _, scenario := range []string{"absent", "no-lock", "existing-lock", "completed-lease"} {
		t.Run(scenario, func(t *testing.T) {
			base := t.TempDir()
			dir := filepath.Join(base, "state")
			if scenario != "absent" {
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "existing-lock" {
				if err := os.WriteFile(filepath.Join(dir, "state.lock"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "completed-lease" {
				ctx, cancel := context.WithTimeout(context.Background(), time.Second)
				defer cancel()
				p := Policy{Capacity: 2, MaxRunning: 2, MaxHeavy: 1, Weights: map[string]int{"heavy": 2, "light": 1}, AgingInterval: "1m"}
				lease, err := Acquire(ctx, dir, p, "same", "heavy", "normal")
				if err != nil {
					t.Fatal(err)
				}
				if err = lease.Complete(ctx); err != nil {
					t.Fatal(err)
				}
			}
			entries, err := Status(context.Background(), dir, "same")
			if err != nil {
				t.Fatal(err)
			}
			b, err := json.Marshal(entries)
			if err != nil {
				t.Fatal(err)
			}
			if string(b) != "[]" {
				t.Fatalf("JSON = %s, want []", b)
			}
		})
	}
}
