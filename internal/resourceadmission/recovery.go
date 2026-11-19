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

// RecoveryEvidence is an operator attestation, not machine proof of process state.
type RecoveryEvidence struct {
	ClaimID           string `json:"claim_id"`
	ClaimSHA256       string `json:"claim_sha256"`
	PolicyFingerprint string `json:"policy_fingerprint"`
	WorkloadEnded     bool   `json:"workload_ended"`
	Explanation       string `json:"explanation"`
}

type RecoveryResult struct {
	ClaimID           string `json:"claim_id"`
	ClaimSHA256       string `json:"claim_sha256"`
	PolicyFingerprint string `json:"policy_fingerprint"`
	Applied           bool   `json:"applied"`
	Receipt           string `json:"receipt,omitempty"`
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

// Recover plans or releases exactly one stale running claim. apply must be
// authorized by the interactive CLI; a free lock does not prove child exit.
func Recover(ctx context.Context, dir, id, expected, policy, policyPath, evidencePath string, apply bool) (result RecoveryResult, retErr error) {
	result = RecoveryResult{ClaimID: id, ClaimSHA256: expected, PolicyFingerprint: policy}
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
		var att RecoveryEvidence
		if err := decodeRecoveryJSON(evidence, &att); err != nil {
			return fmt.Errorf("invalid evidence: %w", err)
		}
		if att.ClaimID != id || att.ClaimSHA256 != expected || att.PolicyFingerprint != policy || !att.WorkloadEnded || strings.TrimSpace(att.Explanation) == "" {
			return errors.New("evidence does not attest to the exact ended workload")
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
		if err := d.Sync(); err != nil {
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
		if err := d.Sync(); err != nil {
			return fmt.Errorf("claim archived but directory sync failed; operator review required: %w", err)
		}
		result.Applied = true
		result.Receipt = receiptPath
		return nil
	})
	return result, err
}
