//go:build darwin || linux

package cirun

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/unix"
)

func TestNativeStageReaderRejectsSymlinkFIFOModesAndOversizedFiles(t *testing.T) {
	runID := strings.Repeat("a", 32)
	events := nativeTestEventsDir(t, runID)
	valid := nativeTestEvent(runID, "one", "start")
	writeNativeTestEvent(t, events, "native-abcdef0123456789abcdef0123456789-start.json", valid, 0600)

	badMode := nativeTestEvent(runID, "one", "start")
	writeNativeTestEvent(t, events, "native-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa-start.json", badMode, 0644)
	oversized := nativeTestEvent(runID, "one", "start")
	data, err := json.Marshal(oversized)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte(strings.Repeat("x", maxNativeStageFileBytes+1))...)
	if err := os.WriteFile(filepath.Join(events, "native-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb-start.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(events, "native-abcdef0123456789abcdef0123456789-start.json"), filepath.Join(events, "native-cccccccccccccccccccccccccccccccc-start.json")); err != nil {
		t.Fatal(err)
	}
	if err := unix.Mkfifo(filepath.Join(events, "native-dddddddddddddddddddddddddddddddd-start.json"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(events, "native-bad.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	got := collectNativeStages(events, runID, []string{"one"})
	if got.CollectionStatus != "partial" || got.EventsSeen != 6 || got.EventsRead != 1 || got.InvalidEvents != 5 || len(got.Stages) != 1 {
		t.Fatalf("unsafe event inputs not bounded/rejected: %+v", got)
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "native-") || strings.Contains(string(encoded), "unsafe") {
		t.Fatalf("untrusted filename/error data leaked: %s", encoded)
	}
}

func TestNativeStageReaderCapsDirectoryEntries(t *testing.T) {
	runID := strings.Repeat("a", 32)
	events := nativeTestEventsDir(t, runID)
	for i := range maxNativeStageFiles + 1 {
		path := filepath.Join(events, fmt.Sprintf("extra-%04d.txt", i))
		if err := os.WriteFile(path, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	got := collectNativeStages(events, runID, []string{"one"})
	if got.CollectionStatus != "partial" || got.EventsOmitted != 1 || got.EventsSeen != 0 {
		t.Fatalf("directory entry cap not recorded: %+v", got)
	}
}

func TestMalformedNativeFilenameMakesCollectionPartial(t *testing.T) {
	runID := strings.Repeat("a", 32)
	events := nativeTestEventsDir(t, runID)
	if err := os.WriteFile(filepath.Join(events, "native-bad.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	got := collectNativeStages(events, runID, []string{"one"})
	if got.CollectionStatus != "partial" || got.EventsSeen != 1 || got.EventsRead != 0 || got.InvalidEvents != 1 {
		t.Fatalf("malformed filename was not reported partial: %+v", got)
	}
}
