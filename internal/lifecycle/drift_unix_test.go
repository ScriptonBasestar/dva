//go:build !windows

package lifecycle

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.invalid", "-c", "commit.gpgsign=false"}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func startInRepo(t *testing.T) (*ProcessPlugin, *PluginContext) {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q", "-b", "main")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "one")
	pctx := identityPctx(dir, "target:main")
	p := &ProcessPlugin{}
	if _, err := p.Up(context.Background(), pctx); err != nil {
		t.Fatalf("up: %v", err)
	}
	killProcessGroupInCleanup(t, readPid(t, pctx))
	return p, pctx
}

func statusDrift(t *testing.T, p *ProcessPlugin, pctx *PluginContext) *CheckoutDrift {
	t.Helper()
	st, err := p.Status(context.Background(), pctx)
	if err != nil || len(st) != 1 || st[0].State != "running" {
		t.Fatalf("status = %+v, %v; want one running row", st, err)
	}
	return st[0].Drift
}

func TestTargetDriftNoneWhenCheckoutUnchanged(t *testing.T) {
	p, pctx := startInRepo(t)
	if d := statusDrift(t, p, pctx); d != nil {
		t.Fatalf("drift = %+v, want nil for an unchanged checkout", d)
	}
}

func TestTargetDriftBranchAndCommit(t *testing.T) {
	p, pctx := startInRepo(t)
	gitIn(t, pctx.ConfigDir, "switch", "-q", "-c", "feat/x")
	gitIn(t, pctx.ConfigDir, "commit", "-q", "--allow-empty", "-m", "two")

	d := statusDrift(t, p, pctx)
	if d == nil {
		t.Fatal("drift = nil, want branch and commit change")
	}
	if d.StartedBranch != "main" || d.CurrentBranch != "feat/x" || d.StartedCommit == "" || d.CurrentCommit == "" || d.StartedCommit == d.CurrentCommit {
		t.Fatalf("drift = %+v", d)
	}
	if s := d.String(); !strings.Contains(s, "branch main → feat/x") || !strings.Contains(s, "commit ") {
		t.Fatalf("drift text = %q", s)
	}
}

func TestTargetDriftTextAndJSON(t *testing.T) {
	drift := &CheckoutDrift{StartedBranch: "main", CurrentBranch: "feat/x"}
	rows := []ServiceStatus{{Name: "api", State: "running", Drift: drift}, {Name: "web", State: "running"}}

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout := os.Stdout
	os.Stdout = w
	printServiceTable(rows)
	os.Stdout = stdout
	_ = w.Close()
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "! api: checkout changed since it started (branch main → feat/x)") || strings.Contains(string(out), "! web") {
		t.Fatalf("status text:\n%s", out)
	}

	data, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"drift":{"started_branch":"main","current_branch":"feat/x"}`) || strings.Count(string(data), `"drift"`) != 1 {
		t.Fatalf("status json = %s", data)
	}
}

func TestTargetDriftCommitOnly(t *testing.T) {
	p, pctx := startInRepo(t)
	gitIn(t, pctx.ConfigDir, "commit", "-q", "--allow-empty", "-m", "two")

	d := statusDrift(t, p, pctx)
	if d == nil || d.StartedBranch != "" || d.CurrentBranch != "" || d.StartedCommit == d.CurrentCommit {
		t.Fatalf("drift = %+v, want a commit-only change", d)
	}
	if strings.Contains(d.String(), "branch") {
		t.Fatalf("drift text = %q, want no branch part", d.String())
	}
}
