package cli

import (
	"context"
	"io"
	"path/filepath"
	"testing"
)

func TestTaskQueueStartUsesOwningConfigRoot(t *testing.T) {
	previous := cfg
	t.Cleanup(func() { cfg = previous })
	cfg = loadTestConfig(t, "version: \"0.1.45\"\n")
	wantRoot, err := filepath.Abs(cfg.FileDir())
	if err != nil {
		t.Fatal(err)
	}
	called := 0
	cmd := newTaskQueueStartCommand(func(_ context.Context, root, branchType string, _ io.Writer) error {
		called++
		if root != wantRoot || branchType != "feat" {
			t.Fatalf("start(root=%q, type=%q), want (%q, feat)", root, branchType, wantRoot)
		}
		return nil
	})
	cmd.SetArgs([]string{"feat"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if called != 1 {
		t.Fatalf("start called %d times, want 1", called)
	}
}
