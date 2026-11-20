//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

// The operator never has to look up a digest, fingerprint or evidence file:
// the listing names the claim and the exact command, and a plan derives the rest.
func TestResourceRecoverDerivesEveryValue(t *testing.T) {
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
	id := strings.Repeat("b", 32)
	claim, _ := json.Marshal(map[string]any{"ID": id, "Policy": fingerprint, "Class": "heavy", "Priority": "normal", "State": "running", "QueuedAt": "2026-11-18T00:00:00Z"})
	for name, data := range map[string][]byte{id + ".json": claim, id + ".lock": nil, "state.lock": nil} {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	old := resourceStateDirectory
	resourceStateDirectory = func() (string, error) { return dir, nil }
	defer func() { resourceStateDirectory = old }()
	oldTTY := bridgeOpenTTY
	bridgeOpenTTY = func() (*os.File, error) { return nil, os.ErrNotExist }
	defer func() { bridgeOpenTTY = oldTTY }()

	run := func(args ...string) (string, error) {
		cmd := newResourceRecoverCommand()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs(append([]string{"--policy", policyPath}, args...))
		err := cmd.Execute()
		return out.String(), err
	}
	listing, err := run()
	if err != nil {
		t.Fatal(err)
	}
	var candidates []recoveryCandidate
	if err := json.Unmarshal([]byte(listing), &candidates); err != nil || len(candidates) != 1 {
		t.Fatalf("listing: %s %v", listing, err)
	}
	c := candidates[0]
	if c.ClaimID != id || !c.Liveness.NeedsOperator || strings.Contains(c.Next, "<") || !strings.Contains(c.Next, "--claim-id "+id+" --apply") {
		t.Fatalf("candidate: %+v", c)
	}
	plan, err := run("--claim-id", id)
	if err != nil || !strings.Contains(plan, `"applied":false`) {
		t.Fatalf("derived plan: %s %v", plan, err)
	}
	if _, err := run("--claim-id", id, "--apply"); err == nil {
		t.Fatal("legacy claim released without a human terminal confirmation")
	}
	if _, err := os.Stat(filepath.Join(dir, id+".json")); err != nil {
		t.Fatalf("claim removed: %v", err)
	}
	leftovers, _ := filepath.Glob(filepath.Join(os.TempDir(), "dva-recovery-*", "evidence.json"))
	for _, l := range leftovers {
		if b, _ := os.ReadFile(l); strings.Contains(string(b), id) {
			t.Fatalf("generated evidence left behind: %s", l)
		}
	}
}
