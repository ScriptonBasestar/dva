package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPinStagedManifestRejectsBeforeQueue(t *testing.T) {
	bin := t.TempDir()
	called := filepath.Join(t.TempDir(), "called")
	stub := filepath.Join(bin, "taskchain-task-manager")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\ntouch \"$QUEUE_CALLED\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QUEUE_CALLED", called)
	var output bytes.Buffer
	err := execute(context.Background(), "tasks", "feat", &output)
	if err == nil || !strings.Contains(err.Error(), "no authorized TaskChain binary pin") {
		t.Fatalf("start error = %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("unexpected output: %q", output.String())
	}
	if _, err := os.Stat(called); !os.IsNotExist(err) {
		t.Fatalf("queue binary ran: %v", err)
	}
}

func approvedTestPin(digest string) artifactPin {
	return artifactPin{
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		SourceCommit: strings.Repeat("a", 40), SourceTree: strings.Repeat("b", 40),
		GoVersion: "go-test", CGOEnabled: "0", BuildCommand: "go build ./cmd/taskchain-task-manager",
		Status: "published-approved", Distribution: "published",
		SHA256: digest, MutationAuthorized: true,
	}
}

func TestPinSnapshotBindsSelectedBytes(t *testing.T) {
	bin := t.TempDir()
	first := filepath.Join(bin, "taskchain-task-manager")
	verified := []byte("#!/bin/sh\nprintf verified\n")
	if err := os.WriteFile(first, verified, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	sha := sha256.Sum256(verified)
	pin := artifactPin{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SHA256: hex.EncodeToString(sha[:]), MutationAuthorized: true}
	snapshot, cleanup, err := pinnedQueueBinary(pin)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first, []byte("#!/bin/sh\nprintf substituted\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(snapshot).Output()
	if err != nil || string(output) != "verified" {
		t.Fatalf("snapshot output = %q, err=%v", output, err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(snapshot); !os.IsNotExist(err) {
		t.Fatalf("snapshot remains: %v", err)
	}
	if _, _, err := pinnedQueueBinary(pin); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("substitution error = %v", err)
	}
}

func TestPinFirstPathResultCannotFallThrough(t *testing.T) {
	firstDir, secondDir := t.TempDir(), t.TempDir()
	bad, good := []byte("#!/bin/sh\nprintf bad\n"), []byte("#!/bin/sh\nprintf good\n")
	for _, file := range []struct {
		path string
		data []byte
	}{
		{filepath.Join(firstDir, "taskchain-task-manager"), bad},
		{filepath.Join(secondDir, "taskchain-task-manager"), good},
	} {
		if err := os.WriteFile(file.path, file.data, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", firstDir+string(os.PathListSeparator)+secondDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	sha := sha256.Sum256(good)
	pin := artifactPin{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SHA256: hex.EncodeToString(sha[:]), MutationAuthorized: true}
	if _, _, err := pinnedQueueBinary(pin); err == nil || !strings.Contains(err.Error(), filepath.Join(firstDir, "taskchain-task-manager")) {
		t.Fatalf("first PATH mismatch = %v", err)
	}
}

func TestPinCompatibleWrongBinaryDoesNotReachQueueOrCE(t *testing.T) {
	bin := t.TempDir()
	queueCalled := filepath.Join(t.TempDir(), "queue-called")
	ceCalled := filepath.Join(t.TempDir(), "ce-called")
	wrong := []byte("#!/bin/sh\ntouch \"$QUEUE_CALLED\"\nprintf '{\"outputVersion\":1,\"runnableCount\":0,\"agentRunnableCount\":0,\"runnable\":[],\"agentRunnable\":[]}'\n")
	if err := os.WriteFile(filepath.Join(bin, "taskchain-task-manager"), wrong, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "ce"), []byte("#!/bin/sh\ntouch \"$CE_CALLED\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("QUEUE_CALLED", queueCalled)
	t.Setenv("CE_CALLED", ceCalled)
	approved := sha256.Sum256([]byte("reviewed different bytes"))
	pins := artifactPins{SchemaVersion: 1, Artifacts: []artifactPin{approvedTestPin(hex.EncodeToString(approved[:]))}}
	var output bytes.Buffer
	err := executeWithPins(context.Background(), "tasks", "feat", &output, pins)
	if err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") || output.Len() != 0 {
		t.Fatalf("mismatch result: err=%v output=%q", err, output.String())
	}
	for _, path := range []string{queueCalled, ceCalled} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unexpected call marker %s: %v", path, err)
		}
	}
}

func TestPinSymlinkRetargetKeepsSnapshot(t *testing.T) {
	bin := t.TempDir()
	verified := filepath.Join(bin, "verified")
	substituted := filepath.Join(bin, "substituted")
	link := filepath.Join(bin, "taskchain-task-manager")
	bytesVerified := []byte("#!/bin/sh\nprintf verified\n")
	if err := os.WriteFile(verified, bytesVerified, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(substituted, []byte("#!/bin/sh\nprintf substituted\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(verified, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	sha := sha256.Sum256(bytesVerified)
	snapshot, cleanup, err := pinnedQueueBinary(artifactPin{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, SHA256: hex.EncodeToString(sha[:])})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(substituted, link); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(snapshot).Output()
	if err != nil || string(output) != "verified" {
		t.Fatalf("retargeted snapshot output=%q err=%v", output, err)
	}
	if err := cleanup(); err != nil {
		t.Fatal(err)
	}
}

func TestPinNoPlatformOrMalformedManifest(t *testing.T) {
	for _, pins := range []artifactPins{
		{SchemaVersion: 1, Artifacts: []artifactPin{{GOOS: "unsupported", GOARCH: runtime.GOARCH, MutationAuthorized: true}}},
		{SchemaVersion: 0},
		{SchemaVersion: 1, Artifacts: []artifactPin{{GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, MutationAuthorized: true, SHA256: "wrong"}}},
	} {
		if _, err := pins.activeForPlatform(); err == nil {
			t.Fatalf("invalid pin accepted: %+v", pins)
		}
	}
}

func TestPinCandidateCannotBeActivatedByBooleanAlone(t *testing.T) {
	pins := embeddedPins
	if pins.loadErr != nil {
		t.Fatal(pins.loadErr)
	}
	pins.Artifacts = append([]artifactPin(nil), pins.Artifacts...)
	for i := range pins.Artifacts {
		if pins.Artifacts[i].GOOS == runtime.GOOS && pins.Artifacts[i].GOARCH == runtime.GOARCH {
			pins.Artifacts[i].MutationAuthorized = true
			if _, err := pins.activeForPlatform(); err == nil || !strings.Contains(err.Error(), "lacks approved release/build provenance") {
				t.Fatalf("staged candidate activated by boolean alone: %v", err)
			}
			return
		}
	}
	// The checked-in candidate is darwin/arm64, so other platforms have no
	// staged entry to flip and must still fail to select an active pin.
	if _, err := pins.activeForPlatform(); err == nil {
		t.Fatal("unlisted platform accepted")
	}
}
