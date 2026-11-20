//go:build darwin || linux || freebsd || netbsd || openbsd

package resourceadmission

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

var livenessPolicy = Policy{Capacity: 2, MaxRunning: 2, MaxHeavy: 1, Weights: map[string]int{"heavy": 2, "light": 1}, AgingInterval: "1m"}

// crashSupervisor runs a child test process that acquires a heavy lease,
// optionally starts and tracks a sleeping process group, then exits without
// completing the lease. It returns the tracked group, or 0.
func crashSupervisor(t *testing.T, dir string, withGroup bool) int {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestLivenessCrashChild$")
	cmd.Env = append(os.Environ(), "DVA_TEST_LIVENESS_CHILD=1", "DVA_TEST_RESOURCE_DIR="+dir)
	if withGroup {
		cmd.Env = append(cmd.Env, "DVA_TEST_LIVENESS_GROUP=1")
	}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("child: %s %v", out, err)
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		if v, ok := strings.CutPrefix(line, "PGID="); ok {
			pgid, err := strconv.Atoi(v)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = syscall.Kill(-pgid, syscall.SIGKILL) })
			return pgid
		}
	}
	return 0
}

func TestLivenessCrashChild(t *testing.T) {
	if os.Getenv("DVA_TEST_LIVENESS_CHILD") != "1" {
		t.Skip("helper process")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lease, err := Acquire(ctx, os.Getenv("DVA_TEST_RESOURCE_DIR"), livenessPolicy, "same", "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("DVA_TEST_LIVENESS_GROUP") == "1" {
		sleeper := exec.Command("sleep", "30")
		sleeper.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		if err := sleeper.Start(); err != nil {
			t.Fatal(err)
		}
		if err := lease.TrackProcessGroup(ctx, sleeper.Process.Pid); err != nil {
			t.Fatal(err)
		}
		os.Stdout.WriteString("PGID=" + strconv.Itoa(sleeper.Process.Pid) + "\n")
	}
	os.Exit(0)
}

func privateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestCrashedSupervisorWithoutChildrenIsReleasedAutomatically(t *testing.T) {
	dir := privateDir(t)
	crashSupervisor(t, dir, false)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	lease, err := Acquire(ctx, dir, livenessPolicy, "same", "heavy", "normal")
	if err != nil {
		t.Fatalf("ended claim was not released: %v", err)
	}
	defer func() { _ = lease.Complete(ctx) }()
	receipts, _ := filepath.Glob(filepath.Join(dir, "*.recovery-receipt"))
	if len(receipts) != 1 {
		t.Fatalf("expected one machine receipt, got %v", receipts)
	}
	data, err := os.ReadFile(receipts[0])
	if err != nil {
		t.Fatal(err)
	}
	var receipt struct {
		Evidence []byte `json:"evidence_base64"`
	}
	if err := json.Unmarshal(data, &receipt); err != nil {
		t.Fatal(err)
	}
	var ev RecoveryEvidence
	if err := json.Unmarshal(receipt.Evidence, &ev); err != nil || !ev.WorkloadEnded || !strings.HasPrefix(ev.Explanation, "dva admission: ") {
		t.Fatalf("bad machine evidence: %+v %v", ev, err)
	}
}

func TestLiveTrackedGroupKeepsQuarantineAndFailsFast(t *testing.T) {
	dir := privateDir(t)
	pgid := crashSupervisor(t, dir, true)
	if pgid == 0 {
		t.Fatal("child did not report its process group")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	start := time.Now()
	_, err := Acquire(ctx, dir, livenessPolicy, "same", "light", "normal")
	var blocked *QuarantineBlockedError
	if !errors.As(err, &blocked) || len(blocked.ClaimIDs) != 1 {
		t.Fatalf("expected quarantine block, got %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("quarantine block waited %s", time.Since(start))
	}
	verdict, err := Liveness(dir, blocked.ClaimIDs[0])
	if err != nil || verdict.Ended || verdict.Legacy {
		t.Fatalf("live group verdict: %+v %v", verdict, err)
	}
	if err := syscall.Kill(-pgid, syscall.SIGKILL); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for !processGroupGone(pgid) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	lease, err := Acquire(ctx, dir, livenessPolicy, "same", "heavy", "normal")
	if err != nil {
		t.Fatalf("claim not released after its group exited: %v", err)
	}
	_ = lease.Complete(ctx)
}

func TestLegacyClaimIsNeverAutoReleased(t *testing.T) {
	f := newRecoveryFixture(t)
	verdict, err := Liveness(f.dir, f.id)
	if err != nil || verdict.Ended || !verdict.Legacy {
		t.Fatalf("legacy verdict: %+v %v", verdict, err)
	}
	_, policy, err := ReadPolicy(f.policyPath)
	if err != nil {
		t.Fatal(err)
	}
	p, _, _ := ReadPolicy(f.policyPath)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	var blocked *QuarantineBlockedError
	if _, err := Acquire(ctx, f.dir, p, policy, "heavy", "normal"); !errors.As(err, &blocked) {
		t.Fatalf("legacy claim should block with a quarantine error, got %v", err)
	}
	if _, err := os.Lstat(filepath.Join(f.dir, f.id+".json")); err != nil {
		t.Fatalf("legacy claim removed: %v", err)
	}
}
