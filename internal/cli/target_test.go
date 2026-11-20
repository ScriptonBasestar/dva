package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/worktarget"
)

// A snapshot has no ID and prints "-"; the head is shortened and flags read yes/no.
func TestWriteTargetTable(t *testing.T) {
	var buf bytes.Buffer
	err := writeTargetTable(&buf, []worktarget.Candidate{
		{ID: "main", Kind: worktarget.KindDefault, Path: "/r", Head: "0123456789abcdef"},
		{Kind: worktarget.KindSnapshot, Path: "/s", Head: "fedcba", Dirty: true, Locked: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("table = %q", buf.String())
	}
	if f := strings.Fields(lines[1]); strings.Join(f, " ") != "main default 0123456789ab no no /r" {
		t.Errorf("row 1 = %q", lines[1])
	}
	if f := strings.Fields(lines[2]); strings.Join(f, " ") != "- snapshot fedcba yes yes /s" {
		t.Errorf("row 2 = %q", lines[2])
	}
}
