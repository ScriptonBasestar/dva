//go:build darwin || linux || freebsd || netbsd || openbsd

package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

func TestResourceTimeoutRecordedGrandchildGoneOrQuarantined(t *testing.T) {
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(dir, "state")
	old := resourceStateDirectory
	resourceStateDirectory = func() (string, error) { return state, nil }
	t.Cleanup(func() { resourceStateDirectory = old })
	policy := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(policy, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	null, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer null.Close()
	pidPath := filepath.Join(dir, "pid")
	cmd := newResourceCommand()
	cmd.SetOut(null)
	cmd.SetErr(null)
	cmd.SetArgs([]string{"run", "--policy", policy, "--class", "light", "--execution-timeout", "300ms", "--", "/bin/sh", "-c", `sleep 20 & printf '%s' "$!" > "$1"; wait`, "_", pidPath})
	err = cmd.Execute()
	var child *resourceChildExit
	if !errors.As(err, &child) || child.code != 124 {
		t.Fatalf("wrong timeout outcome: %v", err)
	}
	b, err := os.ReadFile(pidPath)
	if err != nil {
		t.Fatal(err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil || pid < 1 {
		t.Fatalf("invalid recorded PID: %q %v", b, err)
	}
	probe := syscall.Kill(pid, 0)
	if errors.Is(probe, syscall.ESRCH) {
		return
	}
	_, digest, err := resourceadmission.ReadPolicy(policy)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	claims, err := resourceadmission.Status(ctx, state, digest)
	if err != nil {
		t.Fatal(err)
	}
	if len(claims) != 1 || claims[0].State != "quarantined" {
		t.Fatalf("recorded grandchild PID %d may remain (probe %v) without quarantine: %+v", pid, probe, claims)
	}
}
