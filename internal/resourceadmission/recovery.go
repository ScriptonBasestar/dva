package resourceadmission

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var recoveryID = regexp.MustCompile(`^[0-9a-f]{32}$`)
var recoveryDigest = regexp.MustCompile(`^[0-9a-f]{64}$`)
var recoveryDirectorySync = func(f *os.File) error { return f.Sync() }

// RecoveryEvidence is an operator attestation, not machine proof of process state.
type RecoveryEvidence struct {
	ClaimID           string `json:"claim_id"`
	ClaimSHA256       string `json:"claim_sha256"`
	PolicyFingerprint string `json:"policy_fingerprint"`
	WorkloadEnded     bool   `json:"workload_ended"`
	Explanation       string `json:"explanation"`
}

// UnknownRecoveryEvidence records an operator's acceptance of an unverified workload.
type UnknownRecoveryEvidence struct {
	ClaimID                   string `json:"claim_id"`
	ClaimSHA256               string `json:"claim_sha256"`
	PolicyFingerprint         string `json:"policy_fingerprint"`
	WorkloadState             string `json:"workload_state"`
	Decision                  string `json:"decision"`
	AcceptOverlappingWorkload bool   `json:"accept_overlapping_workload"`
	AuthorizationRef          string `json:"authorization_ref"`
	Reason                    string `json:"reason"`
}

type RecoveryResult struct {
	ClaimID            string `json:"claim_id"`
	ClaimSHA256        string `json:"claim_sha256"`
	PolicyFingerprint  string `json:"policy_fingerprint"`
	Applied            bool   `json:"applied"`
	Receipt            string `json:"receipt,omitempty"`
	WorkloadState      string `json:"workload_state,omitempty"`
	AccountingReleased *bool  `json:"accounting_released,omitempty"`
}

func readRecoveryFile(path string, limit int64) (_ []byte, retErr error) {
	f, err := openPrivateRegular(path, os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	data, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("recovery file too large")
	}
	return data, nil
}

func decodeRecoveryJSON(data []byte, dst any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		return errors.New("expected one JSON value")
	}
	return nil
}

func rejectDuplicateEvidenceFields(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	opening, err := d.Token()
	if err != nil || opening != json.Delim('{') {
		return errors.New("evidence must be an object")
	}
	seen := make(map[string]bool)
	allowed := map[string]bool{"claim_id": true, "claim_sha256": true, "policy_fingerprint": true, "workload_state": true, "decision": true, "accept_overlapping_workload": true, "authorization_ref": true, "reason": true}
	for d.More() {
		key, err := d.Token()
		if err != nil {
			return err
		}
		name, ok := key.(string)
		if !ok || !allowed[name] || seen[name] {
			return errors.New("duplicate or invalid evidence field")
		}
		seen[name] = true
		var value json.RawMessage
		if err := d.Decode(&value); err != nil {
			return err
		}
	}
	_, err = d.Token()
	return err
}

// Recover plans or releases exactly one stale running claim. apply must be
// authorized by the interactive CLI; a free lock does not prove child exit.
func Recover(ctx context.Context, dir, id, expected, policy, policyPath, evidencePath string, apply bool) (result RecoveryResult, retErr error) {
	return recoverClaim(ctx, dir, id, expected, policy, policyPath, evidencePath, apply, false)
}

// RecoverUnknown accepts an explicit operator risk decision, never proof of termination.
func RecoverUnknown(ctx context.Context, dir, id, expected, policy, policyPath, evidencePath string, apply bool) (RecoveryResult, error) {
	return recoverClaim(ctx, dir, id, expected, policy, policyPath, evidencePath, apply, true)
}

func recoverClaim(ctx context.Context, dir, id, expected, policy, policyPath, evidencePath string, apply, unknown bool) (result RecoveryResult, retErr error) {
	result = RecoveryResult{ClaimID: id, ClaimSHA256: expected, PolicyFingerprint: policy}
	if unknown {
		result.WorkloadState = "unknown"
		released := false
		result.AccountingReleased = &released
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if !recoveryID.MatchString(id) || !recoveryDigest.MatchString(expected) || !recoveryDigest.MatchString(policy) {
		return result, errors.New("invalid claim ID or digest")
	}
	if !filepath.IsAbs(dir) || !filepath.IsAbs(evidencePath) {
		return result, errors.New("state and evidence paths must be absolute")
	}
	info, err := os.Lstat(dir)
	if err != nil {
		return result, err
	}
	if !info.IsDir() || info.Mode().Perm() != 0700 || !owned(info) {
		return result, errors.New("invalid private state directory")
	}
	err = lockFile(ctx, filepath.Join(dir, "state.lock"), false, func() (retErr error) {
		lf, err := openPrivateRegular(filepath.Join(dir, id+".lock"), os.O_RDWR)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, lf.Close()) }()
		if err = tryLock(lf); err != nil {
			return fmt.Errorf("claim lease unavailable: %w", err)
		}
		defer func() { retErr = errors.Join(retErr, unlock(lf)) }()
		opened, err := lf.Stat()
		if err != nil {
			return err
		}
		current, err := os.Lstat(filepath.Join(dir, id+".lock"))
		if err != nil || !os.SameFile(opened, current) {
			return errors.New("claim lease lock changed")
		}
		_, currentPolicy, err := ReadPolicy(policyPath)
		if err != nil {
			return err
		}
		if currentPolicy != policy {
			return errors.New("policy fingerprint changed")
		}
		evidence, err := readRecoveryFile(evidencePath, 16384)
		if err != nil {
			return fmt.Errorf("read evidence: %w", err)
		}
		if unknown {
			if err := rejectDuplicateEvidenceFields(evidence); err != nil {
				return fmt.Errorf("invalid unknown-workload evidence: %w", err)
			}
			var att UnknownRecoveryEvidence
			if err := decodeRecoveryJSON(evidence, &att); err != nil {
				return fmt.Errorf("invalid unknown-workload evidence: %w", err)
			}
			if att.ClaimID != id || att.ClaimSHA256 != expected || att.PolicyFingerprint != policy || att.WorkloadState != "unknown" || att.Decision != "release-accounting" || !att.AcceptOverlappingWorkload || strings.TrimSpace(att.AuthorizationRef) == "" || strings.TrimSpace(att.Reason) == "" {
				return errors.New("evidence does not authorize the exact unknown-workload risk")
			}
		} else {
			var att RecoveryEvidence
			if err := decodeRecoveryJSON(evidence, &att); err != nil {
				return fmt.Errorf("invalid evidence: %w", err)
			}
			if att.ClaimID != id || att.ClaimSHA256 != expected || att.PolicyFingerprint != policy || !att.WorkloadEnded || strings.TrimSpace(att.Explanation) == "" {
				return errors.New("evidence does not attest to the exact ended workload")
			}
		}
		path := filepath.Join(dir, id+".json")
		claim, err := readRecoveryFile(path, 8192)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(claim)
		if hex.EncodeToString(sum[:]) != expected {
			return errors.New("claim digest changed")
		}
		var e entry
		if err := decodeRecoveryJSON(claim, &e); err != nil {
			return fmt.Errorf("invalid claim: %w", err)
		}
		if e.ID != id || e.Policy != policy || e.State != "running" || (e.Class != "heavy" && e.Class != "light") || (e.Priority != "background" && e.Priority != "normal" && e.Priority != "interactive") || e.QueuedAt.IsZero() {
			return errors.New("claim is not the expected running claim and policy")
		}
		if !apply {
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// All evidence is durable before the atomic capacity exclusion.
		receiptPath := filepath.Join(dir, id+".recovery-receipt")
		releasedPath := filepath.Join(dir, id+".released")
		if _, err := os.Lstat(releasedPath); !errors.Is(err, os.ErrNotExist) {
			return errors.New("released claim path already exists or is inaccessible")
		}
		receipt, err := os.OpenFile(receiptPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		if err != nil {
			return err
		}
		payload := struct {
			Schema   string    `json:"schema"`
			Time     time.Time `json:"time"`
			Claim    []byte    `json:"claim_base64"`
			Evidence []byte    `json:"evidence_base64"`
		}{"dva-resource-recovery/1", time.Now().UTC(), claim, evidence}
		err = json.NewEncoder(receipt).Encode(payload)
		if err == nil {
			err = receipt.Chmod(0400)
		}
		if err == nil {
			err = receipt.Sync()
		}
		err = errors.Join(err, receipt.Close())
		if err != nil {
			return fmt.Errorf("persist recovery receipt: %w", err)
		}
		d, err := os.Open(dir)
		if err != nil {
			return err
		}
		defer func() { retErr = errors.Join(retErr, d.Close()) }()
		if err := recoveryDirectorySync(d); err != nil {
			return fmt.Errorf("sync recovery receipt: %w", err)
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		currentClaim, err := readRecoveryFile(path, 8192)
		if err != nil || !bytes.Equal(currentClaim, claim) {
			return errors.New("claim changed during receipt preparation")
		}
		if err := os.Rename(path, releasedPath); err != nil {
			return fmt.Errorf("archive claim: %w", err)
		}
		if unknown {
			result.AccountingReleased = nil
		}
		if err := recoveryDirectorySync(d); err != nil {
			return fmt.Errorf("claim archived but directory sync failed; operator review required: %w", err)
		}
		result.Applied = true
		if unknown {
			released := true
			result.AccountingReleased = &released
		}
		result.Receipt = receiptPath
		return nil
	})
	return result, err
}
