// Package compare holds the run record and snapshot checkouts behind `dva compare`
// (docs/75-variant-comparison-design.md). It records what a comparison started and creates or
// reclaims the detached worktrees DVA owns; starting plans and the command surface live in
// the CLI (TASK-591).
package compare

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// ModeSnapshot is a variant checked out by DVA from a ref pinned to a commit. ModeWorktree
// (provider.go) and ModeConfig name the other two ways a variant gets its files.
const ModeSnapshot = "snapshot"

// ModeConfig is a variant that shares the calling checkout: no second copy of the files, only a
// separate instance (pid files, logs, compose project) and its own vars.
const ModeConfig = "config"

// SourceConfig is the variant source that selects config mode. A git branch literally named
// "config" is reached as refs/heads/config.
const SourceConfig = "config"

// recordVersion is the schema version written into every record, so a later reader can refuse
// a shape it does not know instead of misreading it.
const recordVersion = 1

// namePattern matches config's compare and variant name rule. It is repeated here because a
// name becomes a path component: a record or snapshot path must never be built from a name
// that config did not validate, such as one containing a separator or "..".
var namePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)

// Record is what one comparison started, kept at `<configDir>/.sb/dva/compare/<name>.json`.
// It holds no pids: process identity stays in `.sb/dva/pids/` of the checkout that owns each
// entry, and Check reads it from there. It is not a target registry — a record says what DVA
// started, and every reader re-verifies it against the filesystem, Git and live processes.
type Record struct {
	Version  int       `json:"version"`
	Name     string    `json:"name"`
	Variants []Variant `json:"variants"`
}

// Variant is one started variant.
type Variant struct {
	Name string `json:"name"`
	Mode string `json:"mode"`
	// Source is what the caller named: a ref for snapshot, a provider target ID for worktree.
	Source string `json:"source"`
	// Commit is the full commit Source resolved to when the variant was created.
	Commit string `json:"commit"`
	// Path is the absolute root of the variant's checkout.
	Path string `json:"path"`
	// ConfigDir is the absolute directory inside Path that holds the variant's dva.yml: the
	// per_variant entries run there and their pid files live under its `.sb/dva/pids/`. It
	// differs from Path when the calling dva.yml sits below its repository root.
	ConfigDir string         `json:"config_dir"`
	Ports     map[string]int `json:"ports,omitempty"`
	// Instance names the run of a config-mode variant: it splits the pid/log directories and
	// the compose project of the entries in the shared ConfigDir. Empty for the modes that have
	// a checkout of their own.
	Instance string `json:"instance,omitempty"`
}

// ConfigVariant is the variant a `config` source starts: the calling checkout itself, run as
// the instance named after the variant.
func ConfigVariant(name, configDir string) Variant {
	return Variant{Name: name, Mode: ModeConfig, Source: SourceConfig, Path: configDir, ConfigDir: configDir, Instance: name}
}

// OwnsCheckout reports whether dva created the variant's checkout, so down may remove it.
func (v Variant) OwnsCheckout() bool { return v.Mode == ModeSnapshot }

// Variant returns the named variant, or nil.
func (r *Record) Variant(name string) *Variant {
	for i := range r.Variants {
		if r.Variants[i].Name == name {
			return &r.Variants[i]
		}
	}
	return nil
}

// RecordPath is where the record for comparison name lives under configDir.
func RecordPath(configDir, name string) (string, error) {
	if !namePattern.MatchString(name) {
		return "", fmt.Errorf("compare name %q must match %s", name, namePattern)
	}
	return filepath.Join(configDir, config.DotDirName, config.CompareDirName, name+".json"), nil
}

// ReadRecord loads the record for name. A comparison that was never started returns an error
// satisfying errors.Is(err, os.ErrNotExist), so a caller can tell "nothing started" from a
// record it cannot read.
func ReadRecord(configDir, name string) (*Record, error) {
	path, err := RecordPath(configDir, name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var rec Record
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, fmt.Errorf("read compare record %s: %w", path, err)
	}
	if rec.Version != recordVersion {
		return nil, fmt.Errorf("compare record %s has version %d, this dva reads version %d",
			path, rec.Version, recordVersion)
	}
	if rec.Name != name {
		return nil, fmt.Errorf("compare record %s names comparison %q, not %q", path, rec.Name, name)
	}
	return &rec, nil
}

// WriteRecord stores rec atomically: it writes a temporary file in the record directory and
// renames it over the old record, so a reader sees the old record or the new one, never a
// truncated file.
func WriteRecord(configDir string, rec *Record) error {
	path, err := RecordPath(configDir, rec.Name)
	if err != nil {
		return err
	}
	for _, v := range rec.Variants {
		if !namePattern.MatchString(v.Name) {
			return fmt.Errorf("compare %s: variant name %q must match %s", rec.Name, v.Name, namePattern)
		}
	}
	out := *rec
	out.Version = recordVersion
	data, err := json.MarshalIndent(&out, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create compare record directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, "."+rec.Name+"-*.json")
	if err != nil {
		return fmt.Errorf("write compare record: %w", err)
	}
	_, writeErr := tmp.Write(append(data, '\n'))
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("write compare record %s: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return fmt.Errorf("write compare record %s: %w", path, err)
	}
	return nil
}
