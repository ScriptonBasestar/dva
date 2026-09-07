package cirun

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

type scopedReadyWriter struct {
	once  sync.Once
	ready chan struct{}
}

func (w *scopedReadyWriter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte("READY")) {
		w.once.Do(func() { close(w.ready) })
	}
	return len(p), nil
}

// These tests deliberately use a fresh test-binary process for the holder.
// A separate process exercises the OS admission boundary of independently
// invoked dva runs.
func TestScopedLockHelperProcess(t *testing.T) {
	if os.Getenv("DVA_CIRUN_SCOPED_HELPER") != "hold" {
		return
	}
	state, root, id := os.Getenv("DVA_CIRUN_STATE"), os.Getenv("DVA_CIRUN_ROOT"), os.Getenv("DVA_CIRUN_ID")
	h, err := acquire(state, root, id, strings.Fields(os.Getenv("DVA_CIRUN_LOCKS"))...)
	if err != nil {
		fmt.Fprintln(os.Stdout, "ERROR", err)
		os.Exit(2)
	}
	defer h.release()
	if os.Getenv("DVA_CIRUN_WORKLOAD") == "1" {
		child := exec.Command(os.Args[0], "-test.run=^TestScopedLockHelperWorkload$")
		child.Env = append(os.Environ(), "DVA_CIRUN_SCOPED_HELPER=workload")
		readyR, readyW, err := os.Pipe()
		if err != nil {
			os.Exit(3)
		}
		child.Stdout, child.Stderr = readyW, io.Discard
		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stdout, "ERROR", err)
			os.Exit(3)
		}
		_ = readyW.Close()
		_ = readyR.SetReadDeadline(time.Now().Add(5 * time.Second))
		line, err := bufio.NewReader(readyR).ReadString('\n')
		_ = readyR.Close()
		if err != nil || line != "READY\n" {
			_ = child.Process.Kill()
			_ = child.Wait()
			os.Exit(3)
		}
		fmt.Fprintf(os.Stdout, "READY %d\n", child.Process.Pid)
	} else {
		fmt.Fprintln(os.Stdout, "READY")
	}
	// FD 3 is a one-way release pipe from the parent test.  It gives the tests
	// a readiness/release handshake without depending on scheduler timing.
	_, _ = io.ReadAll(os.NewFile(3, "release"))
	h.release()
	os.Exit(0) // avoid the testing package writing PASS into a closed ready pipe.
}

func TestScopedLockHelperWorkload(t *testing.T) {
	if os.Getenv("DVA_CIRUN_SCOPED_HELPER") != "workload" {
		return
	}
	fmt.Fprintln(os.Stdout, "READY")
	// The parent test kills this deliberately orphaned workload.  The timeout is
	// only a cleanup backstop if the parent itself is interrupted.
	<-time.After(2 * time.Minute)
	os.Exit(0)
}

type scopedHolder struct {
	cmd     *exec.Cmd
	release *os.File
	child   int
}

func canonicalScopedRoot(t *testing.T, root string) string {
	t.Helper()
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("canonicalize root: %v", err)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		t.Fatalf("absolute root: %v", err)
	}
	return canonical
}

func startScopedHolder(t *testing.T, state, root, id string, locks ...string) *scopedHolder {
	return startScopedHolderWithWorkload(t, state, root, id, false, locks...)
}

func startScopedHolderWithWorkload(t *testing.T, state, root, id string, workload bool, locks ...string) *scopedHolder {
	t.Helper()
	root = canonicalScopedRoot(t, root)
	releaseR, releaseW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	readyR, readyW, err := os.Pipe()
	if err != nil {
		_ = releaseR.Close()
		_ = releaseW.Close()
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestScopedLockHelperProcess$")
	cmd.ExtraFiles = []*os.File{releaseR}
	cmd.Stdout, cmd.Stderr = readyW, os.Stderr
	cmd.Env = append(os.Environ(),
		"DVA_CIRUN_SCOPED_HELPER=hold",
		"DVA_CIRUN_STATE="+state,
		"DVA_CIRUN_ROOT="+root,
		"DVA_CIRUN_ID="+id,
		"DVA_CIRUN_LOCKS="+strings.Join(locks, " "),
	)
	if workload {
		cmd.Env = append(cmd.Env, "DVA_CIRUN_WORKLOAD=1")
	}
	if err := cmd.Start(); err != nil {
		_ = releaseR.Close()
		_ = releaseW.Close()
		_ = readyR.Close()
		_ = readyW.Close()
		t.Fatal(err)
	}
	_ = releaseR.Close()
	_ = readyW.Close()
	h := &scopedHolder{cmd: cmd, release: releaseW}
	t.Cleanup(func() {
		if h.release != nil {
			_ = h.release.Close()
		}
		_ = h.cmd.Wait()
		if h.child != 0 {
			if p, err := os.FindProcess(h.child); err == nil {
				_ = p.Kill()
			}
		}
	})
	if err := readyR.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(readyR).ReadString('\n')
	_ = readyR.Close()
	if err != nil || strings.HasPrefix(line, "ERROR") {
		_ = releaseW.Close()
		_ = cmd.Wait()
		t.Fatalf("holder was not ready: line=%q err=%v", line, err)
	}
	if fields := strings.Fields(line); len(fields) == 2 {
		if _, err := fmt.Sscanf(fields[1], "%d", &h.child); err != nil {
			t.Fatalf("parse workload pid %q: %v", fields[1], err)
		}
	}
	return h
}

func requireBusy(t *testing.T, err error, kind, key string) {
	t.Helper()
	if !errors.Is(err, ErrBusy) {
		t.Fatalf("error=%v, want busy", err)
	}
	runID := "aabbccddeeff00112233445566778899"
	var busy *BusyError
	if !errors.As(err, &busy) {
		t.Fatalf("busy error type missing: %T", err)
	}
	if busy.Conflict.Kind != kind || busy.Conflict.Key != key || busy.Conflict.RunID != runID {
		t.Fatalf("conflict=%#v, want kind=%q key=%q run=%q", busy.Conflict, kind, key, runID)
	}
}

func TestScopedLocksPermitIndependentRootsAndResources(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state := t.TempDir()
	rootA, rootB := t.TempDir(), t.TempDir()
	holder := startScopedHolder(t, state, rootA, "aabbccddeeff00112233445566778899", "database-a")
	defer holder.release.Close()
	marker := filepath.Join(rootB, "executed")
	r, err := Run(context.Background(), Options{Root: rootB, ProfileName: "independent", StateDir: state, lockDirectory: state, Profile: config.CIProfile{
		Timeout: "5s", MaxParallel: 1, Locks: []string{"database-b"},
		Steps: []config.CIStep{{Name: "runs", Run: "touch " + marker}},
	}})
	if err != nil || r.Status != "succeeded" {
		t.Fatalf("independent roots/resources serialized: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("independent run did not execute: %v", err)
	}
}

func TestScopedLocksCanonicalRootAndSharedResourceConflict(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	canonicalRoot := canonicalScopedRoot(t, root)
	alias := filepath.Join(t.TempDir(), "root-alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	id := "aabbccddeeff00112233445566778899"
	holder := startScopedHolder(t, state, root, id, "shared-db")
	defer holder.release.Close()
	_, err := Run(context.Background(), Options{
		Root: alias, ProfileName: "alias", StateDir: state, lockDirectory: state,
		Profile: config.CIProfile{Timeout: "5s", MaxParallel: 1, Steps: []config.CIStep{{Name: "must-not-run", Run: "exit 1"}}},
	})
	requireBusy(t, err, "root", canonicalRoot)
	_, err = acquire(state, t.TempDir(), "11223344556677889900aabbccddeeff", "shared-db")
	requireBusy(t, err, "resource", "shared-db")
}

func TestScopedNestedParentAdmission(t *testing.T) {
	state, root := t.TempDir(), t.TempDir()
	root = canonicalScopedRoot(t, root)
	id := "aabbccddeeff00112233445566778899"
	holder := startScopedHolder(t, state, root, id)
	defer holder.release.Close()
	parent := fmt.Sprintf(`{"id":%q,"root":%q}`, id, root)
	marker := filepath.Join(root, "nested-ran")
	p := config.CIProfile{Timeout: "5s", MaxParallel: 1, Steps: []config.CIStep{{Name: "must-not-run", Run: "touch " + marker}}}
	t.Setenv(config.CIParentRunEnv, parent)
	r, err := Run(context.Background(), Options{Root: t.TempDir(), ProfileName: "child", StateDir: state, lockDirectory: state, Profile: p})
	requireBusy(t, err, "nested", root)
	if r.Conflict == nil || r.Conflict.Kind != "nested" {
		t.Fatalf("nested report=%#v", r)
	}
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("nested run executed a step: %v", statErr)
	}

	// A well-formed receipt whose root lock is no longer owned is stale parent
	// information, not a reason to reject an independent run.
	_ = holder.release.Close()
	if err := holder.cmd.Wait(); err != nil {
		t.Fatalf("release parent holder: %v", err)
	}
	holder.release = nil
	r, err = Run(context.Background(), Options{Root: t.TempDir(), ProfileName: "stale-parent", StateDir: state, lockDirectory: state, Profile: config.CIProfile{Timeout: "5s", MaxParallel: 1, Steps: []config.CIStep{{Name: "ok", Run: "exit 0"}}}})
	if err != nil || r.Status != "succeeded" {
		t.Fatalf("stale parent report=%#v err=%v", r, err)
	}
	t.Setenv(config.CIParentRunEnv, "{broken")
	_, err = Run(context.Background(), Options{Root: t.TempDir(), ProfileName: "bad-parent", StateDir: state, lockDirectory: state, Profile: config.CIProfile{Timeout: "5s", MaxParallel: 1, Steps: []config.CIStep{{Name: "never", Run: "exit 0"}}}})
	if err == nil || !strings.Contains(err.Error(), config.CIParentRunEnv) {
		t.Fatalf("malformed parent error=%v", err)
	}
}

func TestScopedRunPassesProtectedParentToken(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	marker := filepath.Join(root, "parent.json")
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "parent-token", StateDir: state, lockDirectory: state, Profile: config.CIProfile{
		Timeout: "5s", MaxParallel: 1,
		Steps: []config.CIStep{{Name: "capture", Run: "printf '%s' \"$" + config.CIParentRunEnv + "\" > " + marker}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	value, err := os.ReadFile(marker)
	if err != nil {
		t.Fatal(err)
	}
	var got parentRun
	if err := json.Unmarshal(value, &got); err != nil {
		t.Fatalf("parent token=%q: %v", value, err)
	}
	if got.ID != r.ID || got.Root != r.Root {
		t.Fatalf("parent token=%#v report=%#v", got, r)
	}
	_, err = Run(context.Background(), Options{Root: t.TempDir(), ProfileName: "reserved-step-env", StateDir: state, lockDirectory: state, Profile: config.CIProfile{
		Timeout: "5s", MaxParallel: 1,
		Steps: []config.CIStep{{Name: "invalid", Run: "exit 0", Environment: map[string]string{config.CIParentRunEnv: "forged"}}},
	}})
	if err == nil || !strings.Contains(err.Error(), config.CIParentRunEnv) {
		t.Fatalf("reserved environment error=%v", err)
	}
}

func TestScopedConflictJSONRoundTrip(t *testing.T) {
	want := Report{ID: "aabbccddeeff00112233445566778899", Status: "busy", Duration: time.Second, Conflict: &Conflict{Kind: "resource", Key: "acme.db", RunID: "00112233445566778899aabbccddeeff"}}
	b, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got Report
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.Conflict == nil || *got.Conflict != *want.Conflict {
		t.Fatalf("conflict after JSON roundtrip=%#v", got.Conflict)
	}
}

func TestScopedLocksRollbackPartialResourceAcquisition(t *testing.T) {
	state := t.TempDir()
	holder := startScopedHolder(t, state, t.TempDir(), "aabbccddeeff00112233445566778899", "second")
	defer holder.release.Close()
	_, err := acquire(state, t.TempDir(), "00112233445566778899aabbccddeeff", "first", "second")
	requireBusy(t, err, "resource", "second")
	// The failed caller must not retain "first" while rolling back.
	h, err := acquire(state, t.TempDir(), "11223344556677889900aabbccddeeff", "first")
	if err != nil {
		t.Fatalf("partial acquisition leaked a resource lock: %v", err)
	}
	h.release()
}

func TestScopedRunResourceConflictDoesNotExecuteStep(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), t.TempDir()
	holder := startScopedHolder(t, state, t.TempDir(), "aabbccddeeff00112233445566778899", "integration-db")
	defer holder.release.Close()
	marker := filepath.Join(root, "executed")
	r, err := Run(context.Background(), Options{
		Root: root, ProfileName: "other", StateDir: state, lockDirectory: state,
		Profile: config.CIProfile{Timeout: "5s", MaxParallel: 1, Locks: []string{"integration-db"}, Steps: []config.CIStep{{Name: "must-not-run", Run: "touch " + marker}}},
	})
	requireBusy(t, err, "resource", "integration-db")
	if _, statErr := os.Stat(marker); !os.IsNotExist(statErr) {
		t.Fatalf("busy run executed a step: %v", statErr)
	}
	if r.Status != "busy" || r.Conflict == nil || r.Conflict.Kind != "resource" {
		t.Fatalf("busy report=%#v", r)
	}
}

func TestScopedLocksReleaseAfterRunOutcomes(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	for _, test := range []struct {
		name    string
		ctx     func() (context.Context, context.CancelFunc)
		profile config.CIProfile
	}{
		{
			name:    "ordinary error",
			ctx:     func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			profile: config.CIProfile{Timeout: "5s", MaxParallel: 1, Locks: []string{"reusable"}, Steps: []config.CIStep{{Name: "fail", Run: "exit 1"}}},
		},
		{
			name:    "profile timeout",
			ctx:     func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) },
			profile: config.CIProfile{Timeout: "25ms", MaxParallel: 1, Locks: []string{"reusable"}, Steps: []config.CIStep{{Name: "block", Run: scopedWorkloadCommand(), Environment: map[string]string{"DVA_CIRUN_SCOPED_HELPER": "workload"}}}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := test.ctx()
			defer cancel()
			state, root := t.TempDir(), canonicalScopedRoot(t, t.TempDir())
			if _, err := Run(ctx, Options{Root: root, ProfileName: "outcome", StateDir: state, lockDirectory: state, Profile: test.profile}); err == nil {
				t.Fatal("run unexpectedly succeeded")
			}
			h, err := acquire(state, root, "00112233445566778899aabbccddeeff", "reusable")
			if err != nil {
				t.Fatalf("outcome retained lock: %v", err)
			}
			h.release()
		})
	}
}

func TestScopedLocksReleaseAfterRunningCancellation(t *testing.T) {
	t.Setenv(config.CIParentRunEnv, "")
	state, root := t.TempDir(), canonicalScopedRoot(t, t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	writer := &scopedReadyWriter{ready: make(chan struct{})}
	done := make(chan error, 1)
	go func() {
		_, err := Run(ctx, Options{Root: root, ProfileName: "cancel", StateDir: state, lockDirectory: state, Output: writer, Profile: config.CIProfile{
			Timeout: "5s", MaxParallel: 1, Locks: []string{"reusable"},
			Steps: []config.CIStep{{Name: "block", Run: scopedWorkloadCommand(), Environment: map[string]string{"DVA_CIRUN_SCOPED_HELPER": "workload"}}},
		}})
		done <- err
	}()
	select {
	case <-writer.ready:
	case <-time.After(5 * time.Second):
		t.Fatal("canceled run did not reach its ready point")
	}
	cancel()
	if err := <-done; err == nil {
		t.Fatal("canceled run unexpectedly succeeded")
	}
	h, err := acquire(state, root, "00112233445566778899aabbccddeeff", "reusable")
	if err != nil {
		t.Fatalf("canceled running process retained lock: %v", err)
	}
	h.release()
}

func TestScopedLocksRecoverAfterSupervisorKillAndIgnoreMachineLock(t *testing.T) {
	state, root := t.TempDir(), canonicalScopedRoot(t, t.TempDir())
	old := filepath.Join(state, "locks", "machine.lock")
	if err := os.MkdirAll(filepath.Dir(old), 0o700); err != nil {
		t.Fatal(err)
	}
	oldFile, err := os.OpenFile(old, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer oldFile.Close()
	if err := tryLock(oldFile); err != nil {
		t.Fatal(err)
	}
	holder := startScopedHolderWithWorkload(t, state, root, "aabbccddeeff00112233445566778899", true, "killable")
	if err := holder.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := holder.cmd.Wait(); err == nil {
		t.Fatal("killed supervisor unexpectedly exited cleanly")
	}
	_ = holder.release.Close()
	holder.release = nil // Wait already consumed it; cleanup only releases workload when present.
	h, err := acquire(state, root, "00112233445566778899aabbccddeeff", "killable")
	if err != nil {
		t.Fatalf("OS did not release supervisor locks: %v", err)
	}
	h.release()
}

func TestScopedStatusSeparatesLiveAndStaleReceipts(t *testing.T) {
	state, root := t.TempDir(), t.TempDir()
	root = canonicalScopedRoot(t, root)
	id := "aabbccddeeff00112233445566778899"
	holder := startScopedHolder(t, state, root, id)
	defer holder.release.Close()
	secondRoot := canonicalScopedRoot(t, t.TempDir())
	secondID := "00112233445566778899aabbccddeeff"
	second := startScopedHolder(t, state, secondRoot, secondID)
	defer second.release.Close()
	if err := writeReport(state, Report{ID: id, Root: root, Profile: "live", Status: "running", StartedAt: time.Now()}); err != nil {
		t.Fatal(err)
	}
	if err := writeReport(state, Report{ID: secondID, Root: secondRoot, Profile: "second-live", Status: "running", StartedAt: time.Now().Add(time.Millisecond)}); err != nil {
		t.Fatal(err)
	}
	staleID := "11223344556677889900aabbccddeeff"
	if err := writeReport(state, Report{ID: staleID, Root: canonicalScopedRoot(t, t.TempDir()), Profile: "old", Status: "running", StartedAt: time.Now().Add(-time.Second)}); err != nil {
		t.Fatal(err)
	}
	reports, err := Status(state)
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	for _, report := range reports {
		statuses[report.ID] = report.Status
	}
	if statuses[id] != "running" || statuses[secondID] != "running" || statuses[staleID] != "stale" {
		t.Fatalf("statuses=%#v", statuses)
	}
}

func scopedWorkloadCommand() string {
	return "exec '" + strings.ReplaceAll(os.Args[0], "'", "'\"'\"'") + "' -test.run=^TestScopedLockHelperWorkload$"
}
