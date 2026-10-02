package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	bundled "github.com/ScriptonBasestar/dva/skills"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
)

func verifyOwnedArtifacts(project, stateRoot string) error {
	for suffix, runtimes := range runtimeDestinations {
		destination := filepath.Join(project, filepath.FromSlash(suffix))
		files, err := installedFilesFor(destination, suffix == ".agent-mesh/skills/dva")
		if err != nil {
			return err
		}
		record, err := readReceipt(stateRoot, destination)
		if err != nil {
			return err
		}
		if err := requireReceiptContract(destination, record, runtimes, files, receiptFormatForSuffix(suffix)); err != nil {
			return err
		}
		if err := requireClaimContract(filepath.Dir(stateRoot), destination, runtimes, files, receiptFormatForSuffix(suffix)); err != nil {
			return err
		}
	}
	return nil
}

func sharedReceiptSnapshot(project, stateRoot string) (receiptRecord, []fileHash, error) {
	destination := filepath.Join(project, ".agents", "skills")
	files, err := installedSkillFiles(destination)
	if err != nil {
		return receiptRecord{}, nil, err
	}
	record, err := readReceipt(stateRoot, destination)
	if err != nil {
		return receiptRecord{}, nil, err
	}
	if err := requireReceiptContract(destination, record, []string{"antigravity", "codex"}, files, "agent-skills-directory"); err != nil {
		return receiptRecord{}, nil, err
	}
	return record, files, nil
}

func verifySharedUnlink(project, stateRoot string, before receiptRecord, beforeFiles []fileHash) error {
	destination := filepath.Join(project, ".agents", "skills")
	files, err := installedSkillFiles(destination)
	if err != nil {
		return fmt.Errorf("shared skill files not retained: %w", err)
	}
	record, err := readReceipt(stateRoot, destination)
	if err != nil {
		return err
	}
	if err := requireReceiptContract(destination, record, []string{"antigravity"}, files, "agent-skills-directory"); err != nil {
		return err
	}
	if err := requireSharedUnlinkPreservation(destination, before, record, beforeFiles, files); err != nil {
		return err
	}
	return requireClaimContract(filepath.Dir(stateRoot), destination, []string{"antigravity"}, files, "agent-skills-directory")
}

// requireSharedUnlinkPreservation pins the uninstall contract: removing one
// runtime from a shared destination changes membership only. Installed bytes and
// every other receipt field must remain exactly as they were before it.
func requireSharedUnlinkPreservation(destination string, before, after receiptRecord, beforeFiles, afterFiles []fileHash) error {
	if !sameFiles(afterFiles, beforeFiles) {
		return fmt.Errorf("shared installed files changed after unlink for %s", destination)
	}
	if before.Schema != after.Schema {
		return fmt.Errorf("shared receipt schema changed after unlink for %s", destination)
	}
	if before.Installation != after.Installation {
		return fmt.Errorf("shared receipt installation state changed after unlink for %s", destination)
	}
	if before.Format != after.Format {
		return fmt.Errorf("shared receipt format changed after unlink for %s", destination)
	}
	if before.Scope != after.Scope {
		return fmt.Errorf("shared receipt scope changed after unlink for %s", destination)
	}
	if before.Destination != after.Destination {
		return fmt.Errorf("shared receipt destination changed after unlink for %s", destination)
	}
	if before.Version != after.Version {
		return fmt.Errorf("shared receipt version changed after unlink for %s", destination)
	}
	if !sameFiles(before.Files, after.Files) {
		return fmt.Errorf("shared receipt files changed after unlink for %s", destination)
	}
	if before.BundleSHA != after.BundleSHA {
		return fmt.Errorf("shared receipt bundle_sha256 changed after unlink for %s", destination)
	}
	return nil
}

// requireReceiptContract independently checks the installer's external receipt
// receipt. The black-box gate must not call internal/skillinstall's reader because
// that would make the producer and verifier share the same decoding assumptions.
func requireReceiptContract(destination string, record receiptRecord, runtimes []string, files []fileHash, format string) error {
	if record.Schema != 3 {
		return fmt.Errorf("receipt for %s has schema=%d, want 3", destination, record.Schema)
	}
	if record.Installation != "active" {
		return fmt.Errorf("receipt for %s has installation=%q, want active", destination, record.Installation)
	}
	if record.Format != format {
		return fmt.Errorf("receipt for %s has format=%q, want %q", destination, record.Format, format)
	}
	if record.Scope != "project" {
		return fmt.Errorf("receipt for %s has scope=%q, want project", destination, record.Scope)
	}
	if record.Destination != destination {
		return fmt.Errorf("receipt destination=%q, want %q", record.Destination, destination)
	}
	if record.Version == "" {
		return fmt.Errorf("receipt for %s has empty version", destination)
	}
	if !sameStrings(record.Runtimes, runtimes) {
		return fmt.Errorf("receipt for %s has runtimes=%v, want %v", destination, record.Runtimes, runtimes)
	}
	if len(files) == 0 || len(record.Files) == 0 {
		return fmt.Errorf("receipt for %s has no installed file records", destination)
	}
	if !sameFiles(record.Files, files) {
		return fmt.Errorf("receipt for %s does not match installed files", destination)
	}
	if record.BundleSHA != bundleSHA(files) {
		return fmt.Errorf("receipt for %s has bundle_sha256=%q that does not match installed files", destination, record.BundleSHA)
	}
	return nil
}

func requireClaimContract(neutralRoot, destination string, runtimes []string, installed []fileHash, format string) error {
	flat := format == "agent-mesh-flat-markdown"
	for _, name := range bundled.Names {
		installedName := name
		kind := "directory"
		var files []fileHash
		if flat {
			installedName, kind = name+".md", "file"
			for _, file := range installed {
				if file.Path == installedName {
					files = append(files, fileHash{Path: ".", SHA: file.SHA})
				}
			}
		} else {
			prefix := name + "/"
			for _, file := range installed {
				if relative, found := strings.CutPrefix(file.Path, prefix); found {
					files = append(files, fileHash{Path: relative, SHA: file.SHA})
				}
			}
		}
		claimDestination := filepath.Join(destination, installedName)
		digest := sha256.Sum256([]byte(claimDestination))
		claimPath := filepath.Join(neutralRoot, "agent-skills", "claims", "v1", hex.EncodeToString(digest[:])+".json")
		contents, err := os.ReadFile(claimPath)
		if err != nil {
			return fmt.Errorf("read claim for %s: %w", claimDestination, err)
		}
		var claim claimRecord
		if err := json.Unmarshal(contents, &claim); err != nil {
			return err
		}
		if claim.Schema != 1 || claim.Name != name || claim.Kind != kind || claim.State != "active" || claim.OperationID == "" || claim.Generation == 0 || claim.Destination != claimDestination || claim.Producer != "dva" || claim.Format != format || claim.Scope != "project" {
			return fmt.Errorf("claim identity for %s is invalid: %#v", claimDestination, claim)
		}
		if !sameStrings(claim.Consumers, runtimes) || !sameFiles(claim.Files, files) {
			return fmt.Errorf("claim membership/files for %s differ", claimDestination)
		}
		if claim.SourceDigest != bundleSHA(files) {
			return fmt.Errorf("claim source digest for %s differs from its files", claimDestination)
		}
	}
	return nil
}

func verifyArtifactsAbsent(project, stateRoot string) error {
	for suffix := range runtimeDestinations {
		if suffix == ".agent-mesh/skills/dva" {
			for _, skill := range bundled.Names {
				file := skill + ".md"
				if _, err := os.Lstat(filepath.Join(project, filepath.FromSlash(suffix), file)); !errors.Is(err, os.ErrNotExist) {
					return fmt.Errorf("flat skill artifact remains at %s", filepath.Join(suffix, file))
				}
			}
			continue
		}
		for _, skill := range bundled.Names {
			if _, err := os.Lstat(filepath.Join(project, filepath.FromSlash(suffix), skill)); !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("skill artifact remains at %s", filepath.Join(suffix, skill))
			}
		}
	}
	if err := requireEmptyOrMissingDirectory(filepath.Join(stateRoot, "skill-installs")); err != nil {
		return err
	}
	return requireEmptyOrMissingDirectory(filepath.Join(filepath.Dir(stateRoot), "agent-skills", "claims", "v1"))
}

func readReceipt(stateRoot, destination string) (receiptRecord, error) {
	digest := sha256.Sum256([]byte(destination))
	path := filepath.Join(stateRoot, "skill-installs", hex.EncodeToString(digest[:])+".json")
	contents, err := os.ReadFile(path)
	if err != nil {
		return receiptRecord{}, err
	}
	var record receiptRecord
	if err := json.Unmarshal(contents, &record); err != nil {
		return receiptRecord{}, err
	}
	return record, nil
}

func installedSkillFiles(destination string) ([]fileHash, error) {
	var files []fileHash
	for _, skill := range bundled.Names {
		root := filepath.Join(destination, skill)
		info, err := os.Lstat(root)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("invalid skill root %s", root)
		}
		err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || !entry.Type().IsRegular() {
				return fmt.Errorf("invalid skill file %s", path)
			}
			digest, err := fileSHA256(path)
			if err != nil {
				return err
			}
			relative, err := filepath.Rel(destination, path)
			if err != nil {
				return err
			}
			files = append(files, fileHash{Path: filepath.ToSlash(relative), SHA: digest})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func installedFilesFor(destination string, flat bool) ([]fileHash, error) {
	if !flat {
		return installedSkillFiles(destination)
	}
	files := make([]fileHash, 0, len(bundled.Names))
	for _, skill := range bundled.Names {
		name := skill + ".md"
		path := filepath.Join(destination, name)
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("invalid flat skill file %s", path)
		}
		digest, err := fileSHA256(path)
		if err != nil {
			return nil, err
		}
		files = append(files, fileHash{Path: name, SHA: digest})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}
func sameFiles(left, right []fileHash) bool { return reflect.DeepEqual(left, right) }
func bundleSHA(files []fileHash) string {
	hash := sha256.New()
	for _, file := range files {
		_, _ = hash.Write([]byte(file.Path))
		_, _ = hash.Write([]byte{0})
		_, _ = hash.Write([]byte(file.SHA))
		_, _ = hash.Write([]byte{0})
	}
	return hex.EncodeToString(hash.Sum(nil))
}
