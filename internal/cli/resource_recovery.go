package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
	"github.com/spf13/cobra"
)

// recoveryCandidate is one quarantined claim with the machine verdict and
// the single command that recovers it; nothing in it needs operator lookup.
type recoveryCandidate struct {
	ClaimID  string                          `json:"claim_id"`
	Class    string                          `json:"class"`
	Priority string                          `json:"priority"`
	Liveness resourceadmission.ClaimLiveness `json:"liveness"`
	Next     string                          `json:"next"`
}

func recoveryNext(id string, verdict resourceadmission.ClaimLiveness) string {
	switch {
	case verdict.Ended:
		return "dva resource recover --claim-id " + id + " --apply"
	case verdict.NeedsOperator:
		return "dva resource recover --claim-id " + id + " --apply  (human terminal: type the claim ID to confirm)"
	default:
		return "wait: the workload is still running"
	}
}

// writeGeneratedEvidence stores dva-written evidence in a fresh private
// directory and returns its path and a cleanup function.
func writeGeneratedEvidence(data []byte) (string, func() error, error) {
	tmp, err := os.MkdirTemp("", "dva-recovery-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() error { return os.RemoveAll(tmp) }
	if err := os.Chmod(tmp, 0700); err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	path := filepath.Join(tmp, "evidence.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	return path, cleanup, nil
}

func newResourceRecoverCommand() *cobra.Command {
	var policyPath, claim, digest, fingerprint, evidence string
	var apply, acceptUnknown bool
	cmd := &cobra.Command{Use: "recover", Short: "Recover quarantined resource claims", Long: "Without --claim-id, list quarantined claims with a machine liveness verdict and the exact next command. With --claim-id, dva derives the claim digest and policy fingerprint itself (explicit values must match). --apply releases a claim whose recorded supervisor and process groups are proven gone, writing its own evidence; a claim from an older dva without process evidence needs a human to type the claim ID on the controlling terminal. A claim whose workload is still present is refused. --evidence-file supplies operator evidence instead, but any claim not machine-proven ended still needs that terminal confirmation. --accept-unknown-workload with unknown-workload --evidence-file releases accounting for a workload whose state is unknown after a human types RELEASE UNKNOWN and the claim ID. Recovery does not terminate processes or alter CI success records.", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) (retErr error) {
		resolvedPolicy, err := resolveResourcePolicyPath(policyPath)
		if err != nil {
			return err
		}
		policyPath = resolvedPolicy
		_, actual, err := resourceadmission.ReadPolicy(policyPath)
		if err != nil {
			return err
		}
		if fingerprint == "" {
			fingerprint = actual
		} else if actual != fingerprint {
			return errors.New("policy fingerprint mismatch")
		}
		dir, err := resourceStateDirectory()
		if err != nil {
			return err
		}
		if claim == "" {
			if apply {
				return errors.New("--apply needs --claim-id; run without flags to list quarantined claims")
			}
			statusCtx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
			entries, err := resourceadmission.Status(statusCtx, dir, actual)
			cancel()
			if err != nil {
				return err
			}
			candidates := []recoveryCandidate{}
			for _, e := range entries {
				if e.State != "quarantined" {
					continue
				}
				verdict, err := resourceadmission.Liveness(dir, e.ID)
				if err != nil {
					return err
				}
				candidates = append(candidates, recoveryCandidate{e.ID, e.Class, e.Priority, verdict, recoveryNext(e.ID, verdict)})
			}
			return json.NewEncoder(cmd.OutOrStdout()).Encode(candidates)
		}
		if digest == "" {
			if digest, err = resourceadmission.ClaimDigest(dir, claim); err != nil {
				return err
			}
		}
		verdict, err := resourceadmission.Liveness(dir, claim)
		if err != nil {
			return err
		}
		if acceptUnknown && evidence == "" {
			return errors.New("--accept-unknown-workload requires --evidence-file with unknown-workload evidence")
		}
		human := !verdict.Ended || acceptUnknown
		if evidence == "" {
			if !verdict.Ended && !verdict.NeedsOperator {
				return fmt.Errorf("claim %s workload is still present: %s", claim, verdict.Reason)
			}
			explanation := "dva recover: " + verdict.Reason
			if human {
				explanation = "operator typed the claim ID on the controlling terminal; machine check: " + verdict.Reason
			}
			data, err := resourceadmission.MachineEvidence(claim, digest, fingerprint, explanation)
			if err != nil {
				return err
			}
			path, cleanup, err := writeGeneratedEvidence(data)
			if err != nil {
				return err
			}
			defer func() { retErr = errors.Join(retErr, cleanup()) }()
			evidence = path
		}
		recoverFn := resourceadmission.Recover
		if acceptUnknown {
			recoverFn = resourceadmission.RecoverUnknown
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		// The preview uses the same locks and validations but never writes state.
		plan, err := recoverFn(ctx, dir, claim, digest, fingerprint, policyPath, evidence, false)
		cancel()
		if err != nil {
			return err
		}
		if !apply || dryRun {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
				resourceadmission.RecoveryResult
				Liveness resourceadmission.ClaimLiveness `json:"liveness"`
				Next     string                          `json:"next"`
			}{plan, verdict, recoveryNext(claim, verdict)})
		}
		if human {
			if err := confirmOnTerminal(claim, acceptUnknown); err != nil {
				return err
			}
		}
		// The operation revalidates all state after confirmation.
		applyCtx, applyCancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer applyCancel()
		result, err := recoverFn(applyCtx, dir, claim, digest, fingerprint, policyPath, evidence, true)
		if err != nil {
			if result.Applied {
				return fmt.Errorf("claim was released but cleanup failed; operator review required: %w", err)
			}
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	cmd.Flags().StringVar(&policyPath, "policy", "", "Absolute host policy path (or DVA_RESOURCE_POLICY)")
	cmd.Flags().StringVar(&claim, "claim-id", "", "Claim ID; omit to list quarantined claims")
	cmd.Flags().StringVar(&digest, "claim-sha256", "", "Expected claim SHA-256 (derived when omitted)")
	cmd.Flags().StringVar(&fingerprint, "policy-fingerprint", "", "Expected policy fingerprint (derived when omitted)")
	cmd.Flags().StringVar(&evidence, "evidence-file", "", "Operator evidence JSON (dva writes its own when omitted)")
	cmd.Flags().BoolVar(&apply, "apply", false, "Release the claim")
	cmd.Flags().BoolVar(&acceptUnknown, "accept-unknown-workload", false, "Explicitly accept overlapping workload risk using unknown-state evidence")
	return cmd
}

// confirmOnTerminal is the one human step left for claims that are not
// machine-proven ended: a person types the claim ID on the controlling terminal.
func confirmOnTerminal(claim string, unknown bool) (retErr error) {
	if detectAgentEnvironment() {
		return errors.New("this claim is not machine-proven ended; a human must run the apply on a terminal and type the claim ID")
	}
	tty, err := bridgeOpenTTY()
	if err != nil {
		return errors.New("recovery apply requires a controlling terminal")
	}
	defer func() { retErr = errors.Join(retErr, tty.Close()) }()
	confirmation := claim
	prompt := fmt.Sprintf("Confirm workload and descendants have ended. Type claim ID %s to release: ", claim)
	if unknown {
		confirmation = "RELEASE UNKNOWN " + claim
		prompt = fmt.Sprintf("Workload state is unknown; overlapping workloads may run. Type %s to release accounting: ", confirmation)
	}
	if _, err := fmt.Fprint(tty, prompt); err != nil {
		return err
	}
	line, err := bufio.NewReader(tty).ReadString('\n')
	response := strings.TrimSpace(line)
	if unknown {
		// The risk phrase must match exactly; only the line ending is dropped.
		response = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
	}
	if err != nil || response != confirmation {
		return errors.New("operator confirmation did not match claim")
	}
	return nil
}
