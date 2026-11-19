//go:build darwin || linux || freebsd || netbsd || openbsd

package resourceadmission

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type recoveryFixture struct{ dir, id, digest, policy, policyPath, evidence string }

func newRecoveryFixture(t *testing.T) recoveryFixture {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(dir, "policy")
	policyJSON := `{"capacity":2,"max_running":2,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"1m"}`
	if err := os.WriteFile(policyPath, []byte(policyJSON), 0600); err != nil {
		t.Fatal(err)
	}
	_, policy, err := ReadPolicy(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 32)
	b, err := json.Marshal(entry{ID: id, Policy: policy, Class: "heavy", Priority: "normal", State: "running", QueuedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, id+".json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"state.lock", id + ".lock"} {
		if err := os.WriteFile(filepath.Join(dir, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256(b)
	digest := hex.EncodeToString(sum[:])
	evidence := filepath.Join(dir, "evidence")
	data, _ := json.Marshal(RecoveryEvidence{ClaimID: id, ClaimSHA256: digest, PolicyFingerprint: policy, WorkloadEnded: true, Explanation: "operator verified workload ended"})
	if err := os.WriteFile(evidence, data, 0600); err != nil {
		t.Fatal(err)
	}
	return recoveryFixture{dir, id, digest, policy, policyPath, evidence}
}

func (f recoveryFixture) recover(apply bool) (RecoveryResult, error) {
	return Recover(context.Background(), f.dir, f.id, f.digest, f.policy, f.policyPath, f.evidence, apply)
}

func TestRecoveryPlanAndReceiptBeforeExclusion(t *testing.T) {
	f := newRecoveryFixture(t)
	if result, err := f.recover(false); err != nil || result.Applied {
		t.Fatalf("plan: %+v %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(f.dir, f.id+".recovery-receipt")); !os.IsNotExist(err) {
		t.Fatalf("plan wrote receipt: %v", err)
	}
	other := strings.Repeat("b", 32) + ".json"
	if err := os.WriteFile(filepath.Join(f.dir, other), []byte("other"), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := f.recover(true)
	if err != nil || !result.Applied {
		t.Fatalf("apply: %+v %v", result, err)
	}
	for _, name := range []string{f.id + ".recovery-receipt", f.id + ".released", f.id + ".lock", other} {
		if _, err := os.Stat(filepath.Join(f.dir, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(f.dir, f.id+".json")); !os.IsNotExist(err) {
		t.Fatalf("claim still present: %v", err)
	}
	if _, err := f.recover(true); err == nil {
		t.Fatal("repeated recovery accepted")
	}
}

func TestRecoveryRejectsUnsafeInputs(t *testing.T) {
	for _, scenario := range []string{"held", "digest", "policy", "evidence", "symlink", "waiting", "receipt"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRecoveryFixture(t)
			switch scenario {
			case "held":
				lf, err := openPrivateRegular(filepath.Join(f.dir, f.id+".lock"), os.O_RDWR)
				if err != nil {
					t.Fatal(err)
				}
				if err = tryLock(lf); err != nil {
					t.Fatal(err)
				}
				defer lf.Close()
				defer unlock(lf)
			case "digest":
				f.digest = strings.Repeat("0", 64)
			case "policy":
				f.policy = strings.Repeat("0", 64)
			case "evidence":
				if err := os.Chmod(f.evidence, 0644); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				if err := os.Remove(f.evidence); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(f.policyPath, f.evidence); err != nil {
					t.Fatal(err)
				}
			case "waiting":
				b, err := os.ReadFile(filepath.Join(f.dir, f.id+".json"))
				if err != nil {
					t.Fatal(err)
				}
				b = []byte(strings.Replace(string(b), `"running"`, `"waiting"`, 1))
				if err = os.WriteFile(filepath.Join(f.dir, f.id+".json"), b, 0600); err != nil {
					t.Fatal(err)
				}
				sum := sha256.Sum256(b)
				f.digest = hex.EncodeToString(sum[:])
				ev, _ := json.Marshal(RecoveryEvidence{ClaimID: f.id, ClaimSHA256: f.digest, PolicyFingerprint: f.policy, WorkloadEnded: true, Explanation: "verified"})
				if err = os.WriteFile(f.evidence, ev, 0600); err != nil {
					t.Fatal(err)
				}
			case "receipt":
				if err := os.WriteFile(filepath.Join(f.dir, f.id+".recovery-receipt"), []byte("existing"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := f.recover(true); err == nil {
				t.Fatal("unsafe recovery accepted")
			}
			if _, err := os.Stat(filepath.Join(f.dir, f.id+".json")); err != nil {
				t.Fatalf("claim not retained: %v", err)
			}
		})
	}
}

func TestRecoveryConcurrentApplyOnlyOne(t *testing.T) {
	f := newRecoveryFixture(t)
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() { _, err := f.recover(true); errs <- err })
	}
	wg.Wait()
	close(errs)
	success := 0
	for err := range errs {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("successes = %d", success)
	}
}

func TestRecoveryCanceledContextDoesNotRelease(t *testing.T) {
	f := newRecoveryFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Recover(ctx, f.dir, f.id, f.digest, f.policy, f.policyPath, f.evidence, true); err == nil {
		t.Fatal("canceled recovery accepted")
	}
	if _, err := os.Stat(filepath.Join(f.dir, f.id+".json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(f.dir, f.id+".recovery-receipt")); !os.IsNotExist(err) {
		t.Fatalf("unexpected receipt: %v", err)
	}
}
