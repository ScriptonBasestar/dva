package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunRejectsStartTypeBeforeAnySubprocess(t *testing.T) {
	bin, marker := t.TempDir(), filepath.Join(t.TempDir(), "ce-called")
	if err := os.WriteFile(filepath.Join(bin, "ce"), []byte("#!/bin/sh\ntouch \"$CE_CALLED\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CE_CALLED", marker)
	err := run(context.Background(), []string{"--start-type", "feat"}, t.TempDir(), &bytes.Buffer{})
	if err == nil {
		t.Fatal("run accepted --start-type")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("CE ran: %v", err)
	}
}
