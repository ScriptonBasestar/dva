package skillinstall

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/skillclaim"
	bundled "github.com/ScriptonBasestar/dva/skills"
)

func receiptPath(stateRoot, destination string) string {
	digest := sha256.Sum256([]byte(destination))
	return filepath.Join(stateRoot, "skill-installs", hex.EncodeToString(digest[:])+".json")
}

func readReceipt(path string) (receipt, bool, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return receipt{}, false, nil
	}
	if err != nil {
		return receipt{}, false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return receipt{}, false, fmt.Errorf("receipt %s is not a regular file", path)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return receipt{}, false, err
	}
	if err := skillclaim.RejectDuplicateKeys(contents); err != nil {
		return receipt{}, false, err
	}
	var record receipt
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&record); err != nil {
		return receipt{}, false, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return receipt{}, false, errors.New("receipt has trailing JSON value")
		}
		return receipt{}, false, err
	}
	return record, true, nil
}

func validateReceipt(record receipt, scope Scope, target destination) error {
	if (record.Schema != 1 && record.Schema != 2 && record.Schema != receiptSchemaCurrent) || record.Scope != scope || record.Destination != target.path {
		return fmt.Errorf("receipt does not belong to %s", target.path)
	}
	if len(record.Files) == 0 || record.Version == "" || !validSHA(record.BundleSHA) {
		return errors.New("receipt has invalid source metadata")
	}
	if !sort.SliceIsSorted(record.Files, func(i, j int) bool { return record.Files[i].Path < record.Files[j].Path }) {
		return errors.New("receipt files are not sorted")
	}
	if !slices.IsSorted(record.Runtimes) {
		return errors.New("receipt runtimes are not sorted")
	}
	for index := range record.Runtimes {
		if !validReceiptRuntime(record.Runtimes[index]) || (index > 0 && record.Runtimes[index-1] == record.Runtimes[index]) {
			return errors.New("receipt runtimes are invalid or not unique")
		}
	}
	format := receiptFormatForFiles(record.Files)
	if format == "" || format != targetReceiptFormat(target) {
		return errors.New("receipt file format does not match destination runtime")
	}
	if record.Schema == 1 {
		if format != receiptFormatNative {
			return errors.New("legacy receipt cannot describe a flat skill installation")
		}
	} else if record.Format != format {
		return errors.New("receipt format does not match its files")
	}
	for index, file := range record.Files {
		if !validReceiptPath(file.Path) || !validSHA(file.SHA) || (index > 0 && record.Files[index-1].Path == file.Path) {
			return errors.New("receipt contains an invalid file record")
		}
	}
	if sourceBundleSHA(record.Files) != record.BundleSHA {
		return errors.New("receipt bundle digest differs from its files")
	}
	if record.Schema == 3 {
		if record.Installation != "active" && record.Installation != "absent" {
			return errors.New("receipt has invalid installation state")
		}
		if record.Installation == "active" && len(record.Runtimes) == 0 {
			return errors.New("active receipt has no runtime consumers")
		}
		if record.Installation == "absent" && (len(record.Runtimes) != 0 || len(record.Takeovers) == 0) {
			return errors.New("backup-only receipt has invalid runtime state")
		}
		if !sort.SliceIsSorted(record.Takeovers, func(i, j int) bool { return record.Takeovers[i].Skill < record.Takeovers[j].Skill }) {
			return errors.New("receipt takeover records are not sorted")
		}
		for takeoverIndex, takeover := range record.Takeovers {
			if !validTakeoverSkill(takeover.Skill) || !validBackupID(takeover.BackupID) || (takeover.Kind != backupKindFile && takeover.Kind != backupKindDirectory) || !validSHA(takeover.ManifestDigest) || len(takeover.Entries) == 0 || !sort.SliceIsSorted(takeover.Entries, func(i, j int) bool { return takeover.Entries[i].Path < takeover.Entries[j].Path }) {
				return errors.New("receipt has invalid takeover metadata")
			}
			if takeoverIndex > 0 && record.Takeovers[takeoverIndex-1].Skill == takeover.Skill {
				return errors.New("receipt takeover records are not unique")
			}
			if (format == receiptFormatFlat) != strings.HasSuffix(takeover.Skill, ".md") || (takeover.Kind == backupKindFile && len(takeover.Entries) != 1) {
				return errors.New("receipt takeover kind does not match its destination")
			}
			for entryIndex, entry := range takeover.Entries {
				if !validBackupEntry(entry, takeover.Kind) || (entryIndex > 0 && takeover.Entries[entryIndex-1].Path == entry.Path) {
					return errors.New("receipt takeover has invalid file record")
				}
			}
			if backupManifestDigest(takeover.Entries) != takeover.ManifestDigest {
				return errors.New("receipt takeover manifest digest differs from its entries")
			}
		}
	} else if len(record.Takeovers) > 0 {
		return errors.New("legacy receipt has takeover metadata")
	}
	return nil
}

func validReceiptRuntime(runtime Runtime) bool {
	switch runtime {
	case RuntimeClaudeCode, RuntimeCodex, RuntimeOpenCode, RuntimeGrok, RuntimeAntigravity, RuntimeAgentMesh:
		return true
	default:
		return false
	}
}

func targetReceiptFormat(target destination) string {
	if hasRuntime(target.runtimes, RuntimeAgentMesh) {
		return receiptFormatFlat
	}
	return receiptFormatNative
}

func receiptFormatForFiles(files []fileHash) string {
	if len(files) == 0 {
		return ""
	}
	flat := true
	native := true
	for _, file := range files {
		flat = flat && validFlatReceiptPath(file.Path)
		native = native && validNativeReceiptPath(file.Path)
	}
	switch {
	case flat && !native:
		return receiptFormatFlat
	case native && !flat:
		return receiptFormatNative
	default:
		return ""
	}
}

func validReceiptPath(value string) bool {
	return validFlatReceiptPath(value) || validNativeReceiptPath(value)
}

func validFlatReceiptPath(value string) bool {
	if !strings.HasSuffix(value, ".md") {
		return false
	}
	return isBundledSkillName(strings.TrimSuffix(value, ".md"))
}

func validNativeReceiptPath(value string) bool {
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, `\`) || pathpkg.Clean(value) != value {
		return false
	}
	parts := strings.Split(value, "/")
	if len(parts) < 2 || !isBundledSkillName(parts[0]) {
		return false
	}
	for _, part := range parts {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func isBundledSkillName(value string) bool {
	return slices.Contains(bundled.Names, value)
}

func validSHA(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func validBackupEntry(entry backupEntry, rootKind string) bool {
	if entry.Kind != backupKindFile && entry.Kind != backupKindDirectory || entry.Mode > 0o777 {
		return false
	}
	if rootKind == backupKindFile {
		return entry.Path == "." && entry.Kind == backupKindFile && validSHA(entry.SHA)
	}
	if entry.Path == "." {
		return entry.Kind == backupKindDirectory && entry.SHA == ""
	}
	if strings.HasPrefix(entry.Path, "/") || strings.Contains(entry.Path, `\`) || strings.HasSuffix(entry.Path, "/") || pathpkg.Clean(entry.Path) != entry.Path || strings.HasPrefix(entry.Path, "../") || entry.Path == ".." {
		return false
	}
	return (entry.Kind == backupKindFile && validSHA(entry.SHA)) || (entry.Kind == backupKindDirectory && entry.SHA == "")
}

func writeReceipt(path string, record receipt) error {
	contents, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	contents = append(contents, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".receipt-")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(contents); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempName, path); err != nil {
		return err
	}
	return syncDirectory(filepath.Dir(path))
}
