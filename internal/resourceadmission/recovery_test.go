//go:build darwin || linux || freebsd || netbsd || openbsd

package resourceadmission

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
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

func TestUnknownRecoveryRiskEvidenceAndScopedArchive(t *testing.T) {
	f := newRecoveryFixture(t)
	e := UnknownRecoveryEvidence{ClaimID: f.id, ClaimSHA256: f.digest, PolicyFingerprint: f.policy, WorkloadState: "unknown", Decision: "release-accounting", AcceptOverlappingWorkload: true, AuthorizationRef: "user instruction", Reason: "operator accepts potential overlap"}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.evidence, b, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := f.recover(false); err == nil {
		t.Fatal("unknown evidence accepted without opt-in")
	}
	other := strings.Repeat("b", 32) + ".json"
	if err := os.WriteFile(filepath.Join(f.dir, other), []byte("untouched"), 0600); err != nil {
		t.Fatal(err)
	}
	plan, err := RecoverUnknown(context.Background(), f.dir, f.id, f.digest, f.policy, f.policyPath, f.evidence, false)
	if err != nil || plan.Applied || plan.WorkloadState != "unknown" || plan.AccountingReleased == nil || *plan.AccountingReleased {
		t.Fatalf("plan: %+v %v", plan, err)
	}
	result, err := RecoverUnknown(context.Background(), f.dir, f.id, f.digest, f.policy, f.policyPath, f.evidence, true)
	if err != nil || !result.Applied || result.WorkloadState != "unknown" || result.AccountingReleased == nil || !*result.AccountingReleased {
		t.Fatalf("apply: %+v %v", result, err)
	}
	var receipt struct {
		Claim    []byte `json:"claim_base64"`
		Evidence []byte `json:"evidence_base64"`
	}
	stored, err := os.ReadFile(result.Receipt)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(stored, &receipt); err != nil {
		t.Fatal(err)
	}
	archived, err := os.ReadFile(filepath.Join(f.dir, f.id+".released"))
	if err != nil {
		t.Fatal(err)
	}
	if string(receipt.Claim) != string(archived) || string(receipt.Evidence) != string(b) {
		t.Fatal("receipt did not preserve exact input bytes")
	}
	if value, err := os.ReadFile(filepath.Join(f.dir, other)); err != nil || string(value) != "untouched" {
		t.Fatalf("other claim changed: %v", err)
	}
}

func TestUnknownRecoveryRejectsInvalidEvidence(t *testing.T) {
	for _, scenario := range []string{"ended-evidence", "mixed", "missing-risk", "missing-ref", "missing-reason", "wrong-decision", "unknown-field", "mismatched-id", "conflicting-case"} {
		t.Run(scenario, func(t *testing.T) {
			f := newRecoveryFixture(t)
			m := map[string]any{"claim_id": f.id, "claim_sha256": f.digest, "policy_fingerprint": f.policy, "workload_state": "unknown", "decision": "release-accounting", "accept_overlapping_workload": true, "authorization_ref": "approved", "reason": "risk accepted"}
			switch scenario {
			case "ended-evidence":
				if _, err := RecoverUnknown(context.Background(), f.dir, f.id, f.digest, f.policy, f.policyPath, f.evidence, true); err == nil {
					t.Fatal("ended evidence accepted as unknown")
				}
				return
			case "mixed":
				m["workload_ended"] = true
			case "missing-risk":
				delete(m, "accept_overlapping_workload")
			case "missing-ref":
				delete(m, "authorization_ref")
			case "missing-reason":
				m["reason"] = " "
			case "wrong-decision":
				m["decision"] = "ended"
			case "unknown-field":
				m["extra"] = true
			case "mismatched-id":
				m["claim_id"] = strings.Repeat("b", 32)
			case "conflicting-case":
				m["accept_overlapping_workload"] = false
			}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "conflicting-case" {
				b = append(b[:len(b)-1], []byte(`,"ACCEPT_OVERLAPPING_WORKLOAD":true}`)...)
			}
			if err = os.WriteFile(f.evidence, b, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err = RecoverUnknown(context.Background(), f.dir, f.id, f.digest, f.policy, f.policyPath, f.evidence, true); err == nil {
				t.Fatal("invalid evidence accepted")
			}
			if _, err = os.Stat(filepath.Join(f.dir, f.id+".json")); err != nil {
				t.Fatalf("claim not retained: %v", err)
			}
		})
	}
}

func TestUnknownRecoveryPostArchiveSyncFailureIsIndeterminate(t *testing.T) {
	f := newRecoveryFixture(t)
	e := UnknownRecoveryEvidence{ClaimID: f.id, ClaimSHA256: f.digest, PolicyFingerprint: f.policy, WorkloadState: "unknown", Decision: "release-accounting", AcceptOverlappingWorkload: true, AuthorizationRef: "user request", Reason: "risk accepted"}
	b, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(f.evidence, b, 0600); err != nil {
		t.Fatal(err)
	}
	originalSync := recoveryDirectorySync
	defer func() { recoveryDirectorySync = originalSync }()
	calls := 0
	recoveryDirectorySync = func(file *os.File) error {
		calls++
		if calls == 2 {
			return errors.New("simulated sync failure")
		}
		return originalSync(file)
	}
	result, err := RecoverUnknown(context.Background(), f.dir, f.id, f.digest, f.policy, f.policyPath, f.evidence, true)
	if err == nil || !strings.Contains(err.Error(), "claim archived") || result.AccountingReleased != nil {
		t.Fatalf("result %+v, error %v", result, err)
	}
	if _, err := os.Stat(filepath.Join(f.dir, f.id+".released")); err != nil {
		t.Fatal(err)
	}
}
