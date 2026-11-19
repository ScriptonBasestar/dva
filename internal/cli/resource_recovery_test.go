//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
	"github.com/spf13/cobra"
)

func TestResourceRecoverPlanAndNonTerminalApply(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	policyPath := filepath.Join(dir, "policy")
	if err := os.WriteFile(policyPath, []byte(`{"capacity":2,"max_running":2,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"1m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	_, fingerprint, err := resourceadmission.ReadPolicy(policyPath)
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("a", 32)
	claim, _ := json.Marshal(map[string]any{"ID": id, "Policy": fingerprint, "Class": "heavy", "Priority": "normal", "State": "running", "QueuedAt": "2026-11-19T00:00:00Z"})
	for name, data := range map[string][]byte{id + ".json": claim, id + ".lock": nil, "state.lock": nil} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	sum := sha256.Sum256(claim)
	digest := hex.EncodeToString(sum[:])
	evidence := filepath.Join(dir, "evidence")
	b, _ := json.Marshal(resourceadmission.RecoveryEvidence{ClaimID: id, ClaimSHA256: digest, PolicyFingerprint: fingerprint, WorkloadEnded: true, Explanation: "verified"})
	if err := os.WriteFile(evidence, b, 0600); err != nil {
		t.Fatal(err)
	}
	old := resourceStateDirectory
	resourceStateDirectory = func() (string, error) { return dir, nil }
	defer func() { resourceStateDirectory = old }()
	oldTTY := bridgeOpenTTY
	bridgeOpenTTY = func() (*os.File, error) { return nil, os.ErrNotExist }
	defer func() { bridgeOpenTTY = oldTTY }()
	args := []string{"--policy", policyPath, "--claim-id", id, "--claim-sha256", digest, "--policy-fingerprint", fingerprint, "--evidence-file", evidence}
	cmd := newResourceRecoverCommand()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `"applied":false`) {
		t.Fatalf("plan: %s", output.String())
	}
	before, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	oldDryRun := dryRun
	defer func() { dryRun = oldDryRun }()
	ttyCalls := 0
	bridgeOpenTTY = func() (*os.File, error) { ttyCalls++; return nil, os.ErrNotExist }
	parent := &cobra.Command{Use: "dva", SilenceErrors: true, SilenceUsage: true}
	parent.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "plan only")
	parent.AddCommand(newResourceRecoverCommand())
	output.Reset()
	parent.SetOut(&output)
	parent.SetArgs(append([]string{"--dry-run", "recover"}, append(args, "--apply")...))
	if err := parent.Execute(); err != nil {
		t.Fatalf("dry-run apply: %v", err)
	}
	if !strings.Contains(output.String(), `"applied":false`) || ttyCalls != 0 {
		t.Fatalf("dry-run output %q, tty calls %d", output.String(), ttyCalls)
	}
	after, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("dry-run changed directory entries: %d -> %d", len(before), len(after))
	}
	for i := range before {
		if before[i].Name() != after[i].Name() {
			t.Fatalf("dry-run changed directory entry %q -> %q", before[i].Name(), after[i].Name())
		}
	}
	actualClaim, err := os.ReadFile(filepath.Join(dir, id+".json"))
	if err != nil || !bytes.Equal(actualClaim, claim) {
		t.Fatalf("dry-run changed original claim: %v", err)
	}
	dryRun = false
	cmd = newResourceRecoverCommand()
	cmd.SetArgs(append(args, "--apply"))
	if err := cmd.Execute(); err == nil {
		t.Fatal("nonterminal apply accepted")
	}
	if _, err := os.Stat(filepath.Join(dir, id+".json")); err != nil {
		t.Fatal(err)
	}
}
