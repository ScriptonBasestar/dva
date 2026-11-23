package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// StartedByCwd is the starter of an entry `dva up` ran in its own checkout. `dva target` runs
// record "target:<id>" and `dva compare` runs "compare:<name>" (TASK-538).
const StartedByCwd = "cwd"

// DefaultStartedBy is what a new orchestrator records as the starter of the entries it starts.
// The CLI sets it once per process; compare overrides it per orchestrator with SetStartedBy.
var DefaultStartedBy = StartedByCwd

// Identity says who started a background entry and in which checkout. It is written next to
// the entry's pid file when the process starts, removed with the pid file, and only trusted
// after ReadIdentity checks it still describes that pid in that directory.
type Identity struct {
	StartedBy string `json:"started_by"`
	// Path is the config directory the entry was started from.
	Path string `json:"path"`
	// Repository is the checkout's common git directory; empty outside a git checkout.
	Repository string    `json:"repository,omitempty"`
	Branch     string    `json:"branch,omitempty"`
	Commit     string    `json:"commit,omitempty"`
	PID        int       `json:"pid"`
	StartedAt  time.Time `json:"started_at"`
}

// ErrIdentityStale marks an identity file that no longer describes the entry's pid file:
// another pid, another directory, or no pid file at all.
var ErrIdentityStale = errors.New("identity does not match the running entry")

// IdentityFile is the identity file of the named entry for this run.
func (p *PluginContext) IdentityFile(name string) string {
	return filepath.Join(p.StateDir(config.PidsDirName), name+".identity.json")
}

// removeEntryState removes the entry's pid and identity files together.
func removeEntryState(pctx *PluginContext, name string) {
	_ = os.Remove(pctx.PidFile(name))
	_ = os.Remove(pctx.IdentityFile(name))
	clearHalted(pctx, name)
}

func writeIdentity(pctx *PluginContext, name string, pid int) error {
	startedBy := pctx.StartedBy
	if startedBy == "" {
		startedBy = StartedByCwd
	}
	id := Identity{StartedBy: startedBy, Path: filepath.Clean(pctx.ConfigDir), PID: pid, StartedAt: time.Now().UTC()}
	id.Repository, id.Branch, id.Commit = checkoutOf(pctx.ConfigDir)
	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pctx.IdentityFile(name), append(data, '\n'), 0o644)
}

// checkoutOf describes the git checkout dir belongs to. A directory outside git has no
// repository, and the identity keeps only its path.
func checkoutOf(dir string) (repository, branch, commit string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "-C", dir, "rev-parse", "--path-format=absolute", "--git-common-dir", "HEAD").Output()
	if err != nil {
		return "", "", ""
	}
	lines := strings.Fields(string(out))
	if len(lines) != 2 {
		return "", "", ""
	}
	repository, commit = lines[0], lines[1]
	if b, err := exec.CommandContext(ctx, "git", "-C", dir, "symbolic-ref", "-q", "--short", "HEAD").Output(); err == nil {
		branch = strings.TrimSpace(string(b))
	}
	return repository, branch, commit
}

// CheckoutDrift records the branch and commit a running entry started on and what its
// checkout holds now. A field pair is left empty when that part did not change.
type CheckoutDrift struct {
	StartedBranch string `json:"started_branch,omitempty"`
	CurrentBranch string `json:"current_branch,omitempty"`
	StartedCommit string `json:"started_commit,omitempty"`
	CurrentCommit string `json:"current_commit,omitempty"`
}

// String describes the drift for status text, for example
// "branch pilot/a → pilot/b, commit 1a2b3c4 → 5d6e7f8".
func (d *CheckoutDrift) String() string {
	var parts []string
	if d.StartedBranch != d.CurrentBranch {
		parts = append(parts, fmt.Sprintf("branch %s → %s", orDetached(d.StartedBranch), orDetached(d.CurrentBranch)))
	}
	if d.StartedCommit != d.CurrentCommit {
		parts = append(parts, fmt.Sprintf("commit %s → %s", shortCommit(d.StartedCommit), shortCommit(d.CurrentCommit)))
	}
	return strings.Join(parts, ", ")
}

func orDetached(branch string) string {
	if branch == "" {
		return "(detached)"
	}
	return branch
}

func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

// checkoutDrift compares a running entry's identity with its checkout now. It returns nil when
// the entry has no trustworthy identity, was started outside git, or nothing changed.
func checkoutDrift(pctx *PluginContext, name string) *CheckoutDrift {
	id, err := ReadIdentity(pctx.ConfigDir, pctx.Instance, name)
	if err != nil || id.Repository == "" {
		return nil
	}
	_, branch, commit := checkoutOf(pctx.ConfigDir)
	if commit == "" || (branch == id.Branch && commit == id.Commit) {
		return nil
	}
	d := &CheckoutDrift{}
	if branch != id.Branch {
		d.StartedBranch, d.CurrentBranch = id.Branch, branch
	}
	if commit != id.Commit {
		d.StartedCommit, d.CurrentCommit = id.Commit, commit
	}
	return d
}

// ReadIdentity reads the identity of the named entry in configDir and checks it against the
// pid file beside it. A missing identity file returns an error wrapping fs.ErrNotExist; one
// whose pid or directory no longer matches returns ErrIdentityStale.
func ReadIdentity(configDir, instance, name string) (*Identity, error) {
	pctx := &PluginContext{ConfigDir: configDir, Instance: instance}
	data, err := os.ReadFile(pctx.IdentityFile(name))
	if err != nil {
		return nil, err
	}
	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		return nil, fmt.Errorf("read identity of %s: %w", name, err)
	}
	pidData, err := os.ReadFile(pctx.PidFile(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("%s: no pid file: %w", name, ErrIdentityStale)
	}
	if err != nil {
		return nil, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil || pid != id.PID {
		return nil, fmt.Errorf("%s: pid file holds %q, identity pid %d: %w", name, strings.TrimSpace(string(pidData)), id.PID, ErrIdentityStale)
	}
	if id.Path != filepath.Clean(configDir) {
		return nil, fmt.Errorf("%s: identity names %s, not %s: %w", name, id.Path, configDir, ErrIdentityStale)
	}
	return &id, nil
}
