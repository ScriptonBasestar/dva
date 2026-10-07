//go:build integration && (darwin || linux)

package integration

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/secretpush"
)

// TestSecretPushRealSOPS proves real dotenv decryption and exact gh stdin bytes.
// The fake gh writes to a FIFO, never to a regular plaintext file.
func TestSecretPushRealSOPS(t *testing.T) {
	f := newSopsFixture(t, "PUSH_SENTINEL=\""+realSopsSentinel+"\\n\\t\\r\\\"\\\\ # preserved  \"\n")
	git := realTool(t, "git")
	cmd := exec.Command(git, "remote", "add", "origin", "https://github.com/acme/widget.git")
	cmd.Dir = f.dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("add origin: %v: %s", err, out)
	}
	bin, log := t.TempDir(), filepath.Join(t.TempDir(), "gh-args")
	fifo := filepath.Join(t.TempDir(), "gh-stdin")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	keepFIFO, err := os.OpenFile(fifo, os.O_RDWR, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = keepFIFO.Close() }()
	stdin := make(chan []byte, 1)
	stdinErr := make(chan error, 1)
	go func() {
		b, err := os.ReadFile(fifo)
		if err != nil {
			stdinErr <- err
			return
		}
		stdin <- b
	}()
	gh := filepath.Join(bin, "gh")
	if err := os.WriteFile(gh, []byte("#!/bin/sh\nprintf '%s' \"$*\" > \"$GH_ARGS\"\ncat > \"$GH_STDIN_FIFO\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+f.path)
	t.Setenv("SOPS_AGE_KEY_FILE", f.keyFile)
	t.Setenv("GH_ARGS", log)
	t.Setenv("GH_STDIN_FIFO", fifo)
	// secretpush verifies every state-directory ancestor is private. Linux's
	// shared /tmp is mode 1777, so t.TempDir cannot hold this fixture even
	// though the leaf directory is private. Use the repository's designated
	// temporary directory, whose checkout ancestors are not writable by others.
	stateRoot := filepath.Join(repoRoot(), "tmp")
	if err := os.MkdirAll(stateRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	stateRoot, err = filepath.EvalSymlinks(stateRoot)
	if err != nil {
		t.Fatal(err)
	}
	state, err := os.MkdirTemp(stateRoot, "dva-secret-push-state-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		entries, err := os.ReadDir(state)
		if err != nil {
			t.Error(err)
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				t.Errorf("unexpected state directory entry: %s", entry.Name())
				return
			}
			if err := os.Remove(filepath.Join(state, entry.Name())); err != nil {
				t.Error(err)
				return
			}
		}
		if err := os.Remove(state); err != nil {
			t.Error(err)
		}
	})
	report, err := secretpush.Push(context.Background(), secretpush.Options{Root: f.dir, Name: "real-sops", StateDir: state, Target: secretpush.Target{Source: "secrets.env.enc", Repository: "acme/widget", Keys: map[string]string{"PUSH_SENTINEL": "PUSH_DEST"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Keys) != 1 || report.Keys[0].State != secretpush.StateAccepted {
		t.Fatalf("report = %+v", report)
	}
	if err := keepFIFO.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-stdin:
		want := []byte(realSopsSentinel + "\n\t\r\"\\ # preserved  ")
		if string(got) != string(want) {
			t.Fatalf("gh stdin = %q, want %q", got, want)
		}
	case err := <-stdinErr:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out reading gh stdin FIFO")
	}
	args, err := os.ReadFile(log)
	if err != nil || string(args) != "secret set PUSH_DEST --repo github.com/acme/widget" {
		t.Fatalf("gh argv = %q, err = %v", args, err)
	}
	entries, err := os.ReadDir(state)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(state, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), realSopsSentinel) {
			t.Fatalf("receipt leaked plaintext: %s", entry.Name())
		}
	}
}
