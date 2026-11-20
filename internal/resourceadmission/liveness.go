package resourceadmission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// QuarantineBlockedError reports that only quarantined claims prevent
// admission, so waiting cannot succeed until they are recovered.
type QuarantineBlockedError struct{ ClaimIDs []string }

func (e *QuarantineBlockedError) Error() string {
	return fmt.Sprintf("resource admission blocked by quarantined claim(s) %s; run: dva resource recover", strings.Join(e.ClaimIDs, ", "))
}

// ClaimLiveness is the machine verdict for a running claim whose supervisor
// lease is no longer held.
type ClaimLiveness struct {
	Ended         bool   `json:"ended"`
	NeedsOperator bool   `json:"needs_operator"`
	Reason        string `json:"reason"`
}

// claimLiveness proves a claim ended only from recorded process evidence:
// a different boot, or a dead supervisor with every recorded group gone.
// Claims written before the evidence fields existed are never proven ended.
func claimLiveness(e entry) ClaimLiveness {
	if e.Boot == "" || e.SupervisorPID <= 0 {
		return ClaimLiveness{NeedsOperator: true, Reason: "claim predates process evidence; operator confirmation required"}
	}
	// Only a boot ID that cannot change within one boot proves a reboot;
	// darwin and Linux provide one, kern.boottime does not (clock steps move it).
	if current := bootID(); stableBoot(current) && stableBoot(e.Boot) && current != e.Boot {
		return ClaimLiveness{Ended: true, Reason: "host rebooted since the claim was admitted"}
	}
	if processAlive(e.SupervisorPID) {
		return ClaimLiveness{Reason: fmt.Sprintf("supervisor pid %d is still present", e.SupervisorPID)}
	}
	for _, pgid := range e.PGIDs {
		if !processGroupGone(pgid) {
			return ClaimLiveness{Reason: fmt.Sprintf("recorded process group %d still has members", pgid)}
		}
	}
	// The supervisor died between announcing a launch and recording its
	// group: a child may or may not exist, so only a person can decide.
	if e.Launching > 0 {
		return ClaimLiveness{NeedsOperator: true, Reason: fmt.Sprintf("supervisor died with %d process launch(es) unrecorded; operator confirmation required", e.Launching)}
	}
	return ClaimLiveness{Ended: true, Reason: fmt.Sprintf("supervisor pid %d and recorded process groups %v have exited", e.SupervisorPID, e.PGIDs)}
}

func stableBoot(id string) bool { return id != "" && !strings.HasPrefix(id, "boottime:") }

func (l *Lease) updateClaim(ctx context.Context, change func(*entry)) error {
	if l == nil || l.lock == nil {
		return errors.New("resource lease already closed")
	}
	return locked(ctx, filepath.Join(l.dir, "state.lock"), func() error {
		e, _, err := readClaim(l.dir, l.id)
		if err != nil {
			return err
		}
		change(&e)
		return writeEntry(l.dir, e)
	})
}

// BeginLaunch announces, before Start, that a child is about to run. Until
// TrackProcessGroup or EndLaunch settles it, the claim cannot be proven
// ended, so a supervisor crash in between never releases a live workload.
func (l *Lease) BeginLaunch(ctx context.Context) error {
	return l.updateClaim(ctx, func(e *entry) { e.Launching++ })
}

// EndLaunch settles a BeginLaunch whose Start failed.
func (l *Lease) EndLaunch(ctx context.Context) error {
	return l.updateClaim(ctx, func(e *entry) {
		if e.Launching > 0 {
			e.Launching--
		}
	})
}

// TrackProcessGroup records a started child process group on the claim and
// settles its BeginLaunch, so a later recovery can prove the workload ended.
// Call it right after the child starts, before waiting on it.
func (l *Lease) TrackProcessGroup(ctx context.Context, pgid int) error {
	if pgid <= 0 {
		return errors.New("invalid process group")
	}
	return l.updateClaim(ctx, func(e *entry) {
		if !slices.Contains(e.PGIDs, pgid) {
			e.PGIDs = append(e.PGIDs, pgid)
		}
		if e.Launching > 0 {
			e.Launching--
		}
	})
}

func readClaim(dir, id string) (entry, []byte, error) {
	data, err := readRecoveryFile(filepath.Join(dir, id+".json"), 8192)
	if err != nil {
		return entry{}, nil, err
	}
	var e entry
	if err := json.Unmarshal(data, &e); err != nil || e.ID != id {
		return entry{}, nil, fmt.Errorf("invalid resource entry %s", id)
	}
	return e, data, nil
}

// releaseClaim archives a claim after durably writing its recovery receipt.
// The caller holds the state lock and the claim's lease lock.
// onArchived runs once the claim file is renamed, before the final directory
// sync, so a caller can mark a released-but-unsynced state.
func releaseClaim(ctx context.Context, dir, id string, claim, evidence []byte, onArchived func()) (receiptPath string, retErr error) {
	path := filepath.Join(dir, id+".json")
	receiptPath = filepath.Join(dir, id+".recovery-receipt")
	releasedPath := filepath.Join(dir, id+".released")
	if _, err := os.Lstat(releasedPath); !errors.Is(err, os.ErrNotExist) {
		return "", errors.New("released claim path already exists or is inaccessible")
	}
	receipt, err := os.OpenFile(receiptPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return "", err
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
		return "", fmt.Errorf("persist recovery receipt: %w", err)
	}
	d, err := os.Open(dir)
	if err != nil {
		return "", err
	}
	defer func() { retErr = errors.Join(retErr, d.Close()) }()
	if err := recoveryDirectorySync(d); err != nil {
		return "", fmt.Errorf("sync recovery receipt: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	current, err := readRecoveryFile(path, 8192)
	if err != nil || string(current) != string(claim) {
		return "", errors.New("claim changed during receipt preparation")
	}
	if err := os.Rename(path, releasedPath); err != nil {
		return "", fmt.Errorf("archive claim: %w", err)
	}
	if onArchived != nil {
		onArchived()
	}
	if err := recoveryDirectorySync(d); err != nil {
		return "", fmt.Errorf("claim archived but directory sync failed; operator review required: %w", err)
	}
	return receiptPath, nil
}

// MachineEvidence is the attestation dva writes itself when process evidence
// proves the workload ended, or when an operator confirmed on a terminal.
func MachineEvidence(id, digest, policy, explanation string) ([]byte, error) {
	return json.Marshal(RecoveryEvidence{ClaimID: id, ClaimSHA256: digest, PolicyFingerprint: policy, WorkloadEnded: true, Explanation: explanation})
}

// sweepUnheldClaims runs under the state lock. It releases running claims
// whose lease is free and whose workload is proven ended, and returns the
// IDs of the remaining quarantined claims.
func sweepUnheldClaims(dir string, entries []entry) (kept []entry, quarantined []string, err error) {
	for _, e := range entries {
		if e.State != "running" {
			kept = append(kept, e)
			continue
		}
		f, openErr := openPrivateRegular(filepath.Join(dir, e.ID+".lock"), os.O_RDWR)
		if openErr != nil {
			kept = append(kept, e)
			continue
		}
		lockErr := tryLock(f)
		if lockErr != nil {
			closeErr := f.Close()
			if !busy(lockErr) {
				return nil, nil, errors.Join(lockErr, closeErr)
			}
			kept = append(kept, e)
			continue
		}
		verdict := claimLiveness(e)
		released := false
		if verdict.Ended {
			_, claim, readErr := readClaim(dir, e.ID)
			if readErr == nil {
				digest := sha256Hex(claim)
				evidence, evErr := MachineEvidence(e.ID, digest, e.Policy, "dva admission: "+verdict.Reason)
				if evErr == nil {
					_, readErr = releaseClaim(context.Background(), dir, e.ID, claim, evidence, nil)
				} else {
					readErr = evErr
				}
			}
			if readErr != nil {
				_ = unlock(f)
				_ = f.Close()
				return nil, nil, fmt.Errorf("auto-release claim %s: %w", e.ID, readErr)
			}
			released = true
		}
		if err := errors.Join(unlock(f), f.Close()); err != nil {
			return nil, nil, err
		}
		if released {
			continue
		}
		kept = append(kept, e)
		quarantined = append(quarantined, e.ID)
	}
	return kept, quarantined, nil
}

// ClaimDigest returns the SHA-256 of a claim's current bytes, the value
// Recover expects, so operators never have to compute it.
func ClaimDigest(dir, id string) (string, error) {
	if !recoveryID.MatchString(id) {
		return "", errors.New("invalid claim ID")
	}
	_, data, err := readClaim(dir, id)
	if err != nil {
		return "", err
	}
	return sha256Hex(data), nil
}

// Liveness reports the machine verdict for one claim without changing state.
func Liveness(dir, id string) (ClaimLiveness, error) {
	e, _, err := readClaim(dir, id)
	if err != nil {
		return ClaimLiveness{}, err
	}
	return claimLiveness(e), nil
}
