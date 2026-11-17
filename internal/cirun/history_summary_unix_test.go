//go:build darwin || linux

package cirun

import (
	"encoding/json"
	"golang.org/x/sys/unix"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestReadSummaryReceiptsBoundsAndRejectsUnsafeInputs(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 32)
	data, err := json.Marshal(Report{ID: id, Root: "/a", Profile: "full", Status: "succeeded", Duration: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, strings.Repeat("b", 32)+".json"), []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, strings.Repeat("c", 32)+".json"), make([]byte, (4<<20)+1), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(dir, id+".json"), filepath.Join(dir, strings.Repeat("d", 32)+".json")); err != nil {
		t.Fatal(err)
	}
	reports, excluded, err := ReadSummaryReceipts(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 || excluded["invalid_receipt"] != 1 || excluded["oversized"] != 1 || excluded["non_regular"] != 1 {
		t.Fatalf("reports %v excluded %v", reports, excluded)
	}
}

func TestSummaryRejectsFIFOAndUntrustedModes(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(dir, strings.Repeat("a", 32)+".json"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, strings.Repeat("b", 32)+".json"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	_, excluded, err := ReadSummaryReceipts(dir)
	if err != nil || excluded["non_regular"] != 1 || excluded["untrusted_owner_or_mode"] != 1 || time.Since(started) > time.Second {
		t.Fatalf("exclusions=%v err=%v elapsed=%v", excluded, err, time.Since(started))
	}
}

func TestSummaryRejectsNonprivateState(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	_, excluded, err := ReadSummaryReceipts(dir)
	if err != nil || excluded["state_directory_unavailable"] != 1 {
		t.Fatalf("state: %v %v", excluded, err)
	}
}

func TestSummaryRejectsSymlinkStateAndPinsDirectoryAcrossSwap(t *testing.T) {
	base := t.TempDir()
	dir := filepath.Join(base, "state")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 32)
	data, err := json.Marshal(Report{ID: id, Root: "/a", Profile: "full", Status: "succeeded", Duration: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	opened, err := openSummaryState(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := opened.Close(); err != nil {
			t.Errorf("close directory: %v", err)
		}
	})
	if err := os.Rename(dir, filepath.Join(base, "original")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "original"), dir); err != nil {
		t.Fatal(err)
	}
	_, excluded, err := ReadSummaryReceipts(dir)
	if err != nil || excluded["state_directory_unavailable"] != 1 {
		t.Fatalf("symlink: %v %v", excluded, err)
	}
	reports, excluded, err := readSummaryFromOpenedDir(opened)
	if err != nil || len(reports) != 1 || len(excluded) != 0 {
		t.Fatalf("pinned: %v %v %v", reports, excluded, err)
	}
}
