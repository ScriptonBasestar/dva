package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
	"github.com/spf13/cobra"
)

func newResourceRecoverCommand() *cobra.Command {
	var policyPath, claim, digest, fingerprint, evidence string
	var apply bool
	cmd := &cobra.Command{Use: "recover", Short: "Plan an operator recovery of a quarantined claim", Long: "Inspect an exact quarantined claim, original digest, policy fingerprint and private operator evidence. By default no state is changed. Apply requires explicit human confirmation on the controlling terminal; a free supervisor lease alone does not prove the workload has ended. Recovery does not terminate processes or alter CI success records.", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) (retErr error) {
		if policyPath == "" {
			policyPath = os.Getenv("DVA_RESOURCE_POLICY")
		}
		_, actual, err := resourceadmission.ReadPolicy(policyPath)
		if err != nil {
			return err
		}
		if actual != fingerprint {
			return errors.New("policy fingerprint mismatch")
		}
		dir, err := resourceStateDirectory()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		// The preview uses the same locks and validations but never writes state.
		plan, err := resourceadmission.Recover(ctx, dir, claim, digest, fingerprint, policyPath, evidence, false)
		cancel()
		if err != nil {
			return err
		}
		if !apply || dryRun {
			return json.NewEncoder(cmd.OutOrStdout()).Encode(plan)
		}
		if detectAgentEnvironment() {
			return errors.New("recovery apply is restricted to an interactive human operator")
		}
		tty, err := bridgeOpenTTY()
		if err != nil {
			return errors.New("recovery apply requires a controlling terminal")
		}
		released := false
		defer func() {
			if closeErr := tty.Close(); closeErr != nil {
				if released {
					closeErr = fmt.Errorf("claim was released but terminal close failed; operator review required: %w", closeErr)
				}
				retErr = errors.Join(retErr, closeErr)
			}
		}()
		_, err = fmt.Fprintf(tty, "Confirm workload and descendants have ended. Type claim ID %s to release: ", claim)
		if err != nil {
			return err
		}
		line, err := bufio.NewReader(tty).ReadString('\n')
		if err != nil || strings.TrimSpace(line) != claim {
			return errors.New("operator confirmation did not match claim")
		}
		// The operation revalidates all state after interactive confirmation.
		applyCtx, applyCancel := context.WithTimeout(cmd.Context(), 5*time.Second)
		defer applyCancel()
		result, err := resourceadmission.Recover(applyCtx, dir, claim, digest, fingerprint, policyPath, evidence, true)
		released = result.Applied
		if err != nil {
			if result.Applied {
				return fmt.Errorf("claim was released but cleanup failed; operator review required: %w", err)
			}
			return err
		}
		return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
	}}
	cmd.Flags().StringVar(&policyPath, "policy", "", "Absolute host policy path (or DVA_RESOURCE_POLICY)")
	cmd.Flags().StringVar(&claim, "claim-id", "", "Exact claim ID")
	cmd.Flags().StringVar(&digest, "claim-sha256", "", "Expected SHA-256 of original claim bytes")
	cmd.Flags().StringVar(&fingerprint, "policy-fingerprint", "", "Expected policy fingerprint")
	cmd.Flags().StringVar(&evidence, "evidence-file", "", "Absolute private operator evidence JSON")
	cmd.Flags().BoolVar(&apply, "apply", false, "Release after interactive human confirmation")
	return cmd
}
