package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SopsConfigFileName is the only name sops 3.x searches for. It ignores
// `.sops.yml` with a warning, so DVA does not look for that spelling either:
// finding a file sops would refuse to read would be a resolution sops disagrees
// with.
const SopsConfigFileName = ".sops.yaml"

// Lookup modes for SopsConfig.Lookup.
const (
	// SopsLookupNearest uses the .sops.yaml closest to the encrypted file,
	// searching from that file's directory up to the config root.
	SopsLookupNearest = "nearest"
	// SopsLookupRoot uses the .sops.yaml in the config root and ignores any
	// nested one.
	SopsLookupRoot = "root"
)

// SopsConfig is the root-only `sops:` section. It decides which .sops.yaml DVA
// hands to sops through --config.
//
// The section exists because sops on its own searches upward from the working
// directory, not from the file being encrypted. DVA discovers dva.yml upward
// from the working directory too, so without an explicit --config the same
// command on the same file picks different creation rules depending on which
// subdirectory it was typed in.
type SopsConfig struct {
	// Lookup is nearest (the default when empty) or root.
	Lookup string `yaml:"lookup,omitempty" json:"lookup,omitempty"`
	// Config names one rules file, relative to the config root. It is
	// mutually exclusive with Lookup.
	Config string `yaml:"config,omitempty" json:"config,omitempty"`
}

// ErrNoSopsConfig reports that no .sops.yaml applies to a file. Callers that
// need creation rules (encrypt) fail on it; callers that only decrypt may
// proceed, because sops reads a decrypting key from the file's own metadata.
var ErrNoSopsConfig = errors.New("no .sops.yaml applies")

func validateSopsConfig(s *SopsConfig) error {
	if s == nil {
		return nil
	}
	if s.Config != "" && s.Lookup != "" {
		return fmt.Errorf("sops: config and lookup are mutually exclusive")
	}
	switch s.Lookup {
	case "", SopsLookupNearest, SopsLookupRoot:
	default:
		return fmt.Errorf("sops: lookup must be %q or %q, got %q", SopsLookupNearest, SopsLookupRoot, s.Lookup)
	}
	if s.Config != "" {
		clean := filepath.Clean(s.Config)
		if filepath.IsAbs(s.Config) || !filepath.IsLocal(s.Config) || clean == "." || strings.ContainsAny(s.Config, "\\\x00\r\n") {
			return fmt.Errorf("sops: config must be a relative file inside the config directory, got %q", s.Config)
		}
	}
	return nil
}

// ResolveSopsConfig returns the absolute path of the .sops.yaml that governs
// target, or ErrNoSopsConfig (wrapped) when none does.
//
// root is the config root and bounds every search: a .sops.yaml above root, or
// one named by SOPS_CONFIG, never applies, because neither is something this
// project declares. target must be absolute and inside root.
//
// A symlinked rules file is refused rather than followed, for the same reason
// the env bridge refuses symlinked path components: the file sops reads would
// not be the file the project shows.
func ResolveSopsConfig(root string, s *SopsConfig, target string) (string, error) {
	root = filepath.Clean(root)
	if s != nil && s.Config != "" {
		path := filepath.Join(root, s.Config)
		// Not ErrNoSopsConfig even when missing: a named file that is absent is
		// a broken declaration, and a decrypting caller must not shrug it off
		// the way it may shrug off "no rules file was found by searching".
		if err := checkSopsConfigFile(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", fmt.Errorf("sops.config %s does not exist", s.Config)
			}
			return "", fmt.Errorf("sops.config %s: %w", s.Config, err)
		}
		return path, nil
	}
	if s != nil && s.Lookup == SopsLookupRoot {
		path := filepath.Join(root, SopsConfigFileName)
		if err := checkSopsConfigFile(path); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return "", fmt.Errorf("%w: %s does not exist in the config root", ErrNoSopsConfig, SopsConfigFileName)
			}
			return "", fmt.Errorf("%s: %w", SopsConfigFileName, err)
		}
		return path, nil
	}

	rel, err := filepath.Rel(root, filepath.Clean(target))
	if err != nil || !filepath.IsLocal(rel) {
		return "", fmt.Errorf("%s is outside the config directory", target)
	}
	for dir := filepath.Dir(filepath.Join(root, rel)); ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, SopsConfigFileName)
		err := checkSopsConfigFile(path)
		if err == nil {
			return path, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("%s: %w", displayRel(root, path), err)
		}
		if dir == root {
			start := displayRel(root, filepath.Dir(filepath.Join(root, rel)))
			if start == "." {
				return "", fmt.Errorf("%w: none in the config root", ErrNoSopsConfig)
			}
			return "", fmt.Errorf("%w: none between %s and the config root", ErrNoSopsConfig, start)
		}
	}
}

func checkSopsConfigFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("is a symlink; DVA does not follow a symlinked sops config")
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("is not a regular file")
	}
	return nil
}

func displayRel(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return path
}
