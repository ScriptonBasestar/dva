// Package skillinstall installs the DVA-owned Agent Skills without requiring an AI runtime.
package skillinstall

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/skillclaim"
)

type Scope string

const (
	ScopeUser    Scope = "user"
	ScopeProject Scope = "project"
)

type Runtime string

const (
	RuntimeClaudeCode  Runtime = "claude-code"
	RuntimeCodex       Runtime = "codex"
	RuntimeOpenCode    Runtime = "opencode"
	RuntimeGrok        Runtime = "grok"
	RuntimeAntigravity Runtime = "antigravity"
	RuntimeAgentMesh   Runtime = "agent-mesh"
)

// Options supplies filesystem roots. Empty roots are resolved from the process environment.
type Options struct {
	Scope                 Scope
	Runtimes              []Runtime
	HomeDir               string
	ProjectRoot           string
	StateRoot             string
	ClaimRoot             string
	DryRun                bool
	Takeover              bool
	RestoreTakeoverBackup bool
	Version               string
}

type Result struct {
	Scope        Scope               `json:"scope"`
	Destinations []DestinationResult `json:"destinations"`
}

type DestinationResult struct {
	Destination        string          `json:"destination"`
	Runtimes           []Runtime       `json:"runtimes"`
	Skills             []string        `json:"skills"`
	Status             string          `json:"status"`
	Detail             string          `json:"detail,omitempty"`
	SourceVersion      string          `json:"source_version,omitempty"`
	SourceBundleSHA    string          `json:"source_bundle_sha256,omitempty"`
	InstalledVersion   string          `json:"installed_version,omitempty"`
	InstalledBundleSHA string          `json:"installed_bundle_sha256,omitempty"`
	RuntimeStatuses    []RuntimeStatus `json:"runtime_statuses"`
	TakeoverBackup     string          `json:"takeover_backup,omitempty"`
	BackupStatus       string          `json:"backup_status,omitempty"`
}

type RuntimeStatus struct {
	Runtime Runtime `json:"runtime"`
	Status  string  `json:"status"`
}

type receipt struct {
	Schema       int              `json:"schema"`
	Installation string           `json:"installation,omitempty"`
	Format       string           `json:"format,omitempty"`
	Scope        Scope            `json:"scope"`
	Destination  string           `json:"destination"`
	Runtimes     []Runtime        `json:"runtimes"`
	Version      string           `json:"version"`
	BundleSHA    string           `json:"bundle_sha256"`
	Files        []fileHash       `json:"files"`
	Takeovers    []takeoverBackup `json:"takeovers,omitempty"`
}

const (
	receiptSchemaCurrent = 3
	receiptFormatNative  = "agent-skills-directory"
	receiptFormatFlat    = "agent-mesh-flat-markdown"
)

type takeoverBackup struct {
	Skill          string        `json:"skill"`
	BackupID       string        `json:"backup_id"`
	Kind           string        `json:"kind"`
	ManifestDigest string        `json:"manifest_digest"`
	Entries        []backupEntry `json:"entries"`
}

type backupEntry struct {
	Path string `json:"path"`
	Kind string `json:"kind"`
	Mode uint32 `json:"mode"`
	SHA  string `json:"sha256,omitempty"`
}

type fileHash struct {
	Path string `json:"path"`
	SHA  string `json:"sha256"`
}

type destination struct {
	path     string
	runtimes []Runtime
}

type skillBundle struct {
	files    []fileHash
	contents map[string][]byte
}

// DefaultRuntimes returns every supported runtime.
func DefaultRuntimes() []Runtime {
	return []Runtime{RuntimeClaudeCode, RuntimeCodex, RuntimeOpenCode, RuntimeGrok, RuntimeAntigravity, RuntimeAgentMesh}
}

// Status reports whether every requested destination is installed and unmodified.
func Status(options Options) (Result, error) {
	resolved, destinations, err := resolve(options)
	if err != nil {
		return Result{}, err
	}
	result := Result{Scope: resolved.Scope, Destinations: make([]DestinationResult, 0, len(destinations))}
	for _, target := range destinations {
		bundle, err := bundleFor(target)
		if err != nil {
			return Result{}, err
		}
		entry := resultEntry(target, resolved.Version, sourceBundleSHA(bundle.files))
		record, found, err := readReceipt(receiptPath(resolved.StateRoot, target.path))
		if err != nil {
			entry.Status, entry.Detail = "invalid-receipt", err.Error()
			setAllRuntimeStatuses(&entry, "invalid-receipt")
			result.Destinations = append(result.Destinations, entry)
			continue
		}
		if found {
			if err := validateReceipt(record, resolved.Scope, target); err != nil {
				entry.Status, entry.Detail = "invalid-receipt", err.Error()
				setAllRuntimeStatuses(&entry, "invalid-receipt")
				result.Destinations = append(result.Destinations, entry)
				continue
			}
		}
		projectionRuntimes := target.runtimes
		projectionFiles := bundle.files
		if found {
			projectionRuntimes = record.Runtimes
			projectionFiles = record.Files
			if record.Schema == receiptSchemaCurrent && record.Installation == "absent" {
				projectionRuntimes = target.runtimes
			}
		}
		projection, projectionErr := projectedClaims(target, resolved.Scope, projectionRuntimes, skillBundle{files: projectionFiles}, skillclaim.StateActive, "status")
		if projectionErr != nil {
			entry.Status, entry.Detail = "recovery-required", projectionErr.Error()
			setAllRuntimeStatuses(&entry, entry.Status)
			result.Destinations = append(result.Destinations, entry)
			continue
		}
		if !found {
			if err := ensureClaimsAbsentUnlocked(resolved.ClaimRoot, projection); err != nil {
				entry.Status, entry.Detail = "recovery-required", err.Error()
			} else if hasForeignCollision(target.path, bundle.files) {
				entry.Status = "foreign-conflict"
			} else {
				entry.Status = "absent"
			}
			setAllRuntimeStatuses(&entry, entry.Status)
		} else if err := validateReceipt(record, resolved.Scope, target); err != nil {
			entry.Status, entry.Detail = "invalid-receipt", err.Error()
			setAllRuntimeStatuses(&entry, "invalid-receipt")
		} else if record.Schema < receiptSchemaCurrent {
			entry.InstalledVersion, entry.InstalledBundleSHA = record.Version, record.BundleSHA
			if err := verifyInstalled(target.path, record.Files); err != nil {
				entry.Status, entry.Detail = "drifted", err.Error()
			} else if err := ensureClaimsAbsentUnlocked(resolved.ClaimRoot, projection); err != nil {
				entry.Status, entry.Detail = "recovery-required", err.Error()
			} else {
				entry.Status = "legacy-unclaimed"
			}
			setAllRuntimeStatuses(&entry, entry.Status)
		} else if record.Installation == "absent" {
			entry.BackupStatus, entry.TakeoverBackup = verifyTakeoverBackups(resolved.StateRoot, record)
			if err := ensureClaimsAbsentUnlocked(resolved.ClaimRoot, projection); err != nil {
				entry.Status, entry.Detail = "recovery-required", err.Error()
			} else if hasForeignCollision(target.path, record.Files) {
				entry.Status, entry.Detail = "recovery-required", "backup-only destination contains a DVA-name collision"
			} else {
				entry.Status = "backup-only"
			}
			if entry.BackupStatus == "corrupt" {
				entry.Detail = "takeover backup is missing or differs from its receipt"
			}
			setAllRuntimeStatuses(&entry, "absent")
		} else if err := verifyInstalled(target.path, record.Files); err != nil {
			entry.InstalledVersion, entry.InstalledBundleSHA = record.Version, record.BundleSHA
			entry.Status, entry.Detail = "drifted", err.Error()
			setAllRuntimeStatuses(&entry, "drifted")
		} else if err := verifyClaimsUnlocked(resolved.ClaimRoot, projection); err != nil {
			entry.InstalledVersion, entry.InstalledBundleSHA = record.Version, record.BundleSHA
			entry.Status, entry.Detail = "recovery-required", err.Error()
			setAllRuntimeStatuses(&entry, entry.Status)
		} else {
			entry.InstalledVersion, entry.InstalledBundleSHA = record.Version, record.BundleSHA
			entry.Status = setMembershipStatuses(&entry, record.Runtimes, "installed", "absent")
			if len(record.Takeovers) > 0 {
				entry.BackupStatus, entry.TakeoverBackup = verifyTakeoverBackups(resolved.StateRoot, record)
				if entry.BackupStatus == "corrupt" {
					entry.Detail = "takeover backup is missing or differs from its receipt"
				}
			}
		}
		result.Destinations = append(result.Destinations, entry)
	}
	return result, nil
}

func claimDestination(target destination, name string) string {
	if targetReceiptFormat(target) == receiptFormatFlat && !strings.HasSuffix(name, ".md") {
		name += ".md"
	}
	return filepath.Join(target.path, name)
}

func validateTakeover(target destination, bundle skillBundle) error {
	for _, name := range skillNames(bundle) {
		path := claimDestination(target, name)
		info, err := os.Lstat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || (!info.Mode().IsRegular() && !info.IsDir()) {
			return fmt.Errorf("refusing takeover of symlink or special skill %s", path)
		}
		if info.IsDir() {
			err := filepath.WalkDir(path, func(p string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.Type()&os.ModeSymlink != 0 || (!entry.IsDir() && !entry.Type().IsRegular()) {
					return fmt.Errorf("refusing takeover of symlink or special skill %s", p)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func collisionPaths(destination string, files []fileHash) []string {
	paths := make(map[string]bool, len(files))
	for _, file := range files {
		name := file.Path
		if first, _, found := strings.Cut(name, "/"); found {
			name = first
		}
		paths[filepath.Join(destination, filepath.FromSlash(name))] = true
	}
	result := make([]string, 0, len(paths))
	for path := range paths {
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func verifyInstalled(destination string, expected []fileHash) error {
	actual, err := installedFiles(destination, expected)
	if err != nil {
		return err
	}
	if !equalFiles(expected, actual) {
		return errors.New("installed files differ from DVA receipt")
	}
	return nil
}

func installedFiles(destination string, expected []fileHash) ([]fileHash, error) {
	if len(expected) > 0 && !strings.Contains(expected[0].Path, "/") {
		return installedFlatFiles(destination, expected)
	}
	var files []fileHash
	for _, name := range skillNames(skillBundle{files: expected}) {
		root := filepath.Join(destination, name)
		info, err := os.Lstat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%s is not a regular skill directory", root)
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("skill file %s is a symlink", path)
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("skill file %s is not regular", path)
			}
			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(destination, path)
			if err != nil {
				return err
			}
			digest := sha256.Sum256(contents)
			files = append(files, fileHash{Path: filepath.ToSlash(relative), SHA: hex.EncodeToString(digest[:])})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func installedFlatFiles(destination string, expected []fileHash) ([]fileHash, error) {
	files := make([]fileHash, 0, len(expected))
	for _, expectedFile := range expected {
		path := filepath.Join(destination, filepath.FromSlash(expectedFile.Path))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("skill file %s is not regular", path)
		}
		contents, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		digest := sha256.Sum256(contents)
		files = append(files, fileHash{Path: expectedFile.Path, SHA: hex.EncodeToString(digest[:])})
	}
	return files, nil
}

func equalFiles(left, right []fileHash) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sourceBundleSHA(files []fileHash) string {
	hash := sha256.New()
	for _, file := range files {
		_, _ = hash.Write([]byte(file.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(file.SHA))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
