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
	cmd = newResourceRecoverCommand()
	cmd.SetArgs(append(args, "--apply"))
	if err := cmd.Execute(); err == nil {
		t.Fatal("nonterminal apply accepted")
	}
	if _, err := os.Stat(filepath.Join(dir, id+".json")); err != nil {
		t.Fatal(err)
	}
}
