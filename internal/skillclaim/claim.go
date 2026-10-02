// Package skillclaim defines a strict, producer-neutral Agent Skills claim protocol.
package skillclaim

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
	"strconv"
	"strings"
	"unicode"
)

const Schema = 1
const (
	KindDirectory  = "directory"
	KindFile       = "file"
	StateReserved  = "reserved"
	StateActive    = "active"
	StateUpdating  = "updating"
	StateReleasing = "releasing"
	StateRestoring = "restoring"
)

type FileHash struct {
	Path string `json:"path"`
	SHA  string `json:"sha256"`
}
type Claim struct {
	Schema       int        `json:"schema"`
	Name         string     `json:"name"`
	Kind         string     `json:"kind"`
	State        string     `json:"state"`
	OperationID  string     `json:"operation_id"`
	Generation   uint64     `json:"generation"`
	Destination  string     `json:"destination"`
	Producer     string     `json:"producer"`
	Format       string     `json:"format"`
	Scope        string     `json:"scope"`
	Consumers    []string   `json:"consumers"`
	SourceDigest string     `json:"source_digest"`
	Files        []FileHash `json:"files"`
}

// Path takes the neutral XDG state root, never a producer state directory.
func Path(root, destination string) string {
	sum := sha256.Sum256([]byte(destination))
	return filepath.Join(root, "agent-skills", "claims", "v1", hex.EncodeToString(sum[:])+".json")
}

// CanonicalDestination resolves the nearest existing ancestor, including its symlinks, then appends a missing tail.
func CanonicalDestination(destination string) (string, error) {
	current, err := filepath.Abs(destination)
	if err != nil {
		return "", err
	}
	var tail []string
	for {
		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				if len(tail) == 0 {
					return "", fmt.Errorf("skill destination %s is a symlink", current)
				}
				current, err = filepath.EvalSymlinks(current)
				if err != nil {
					return "", err
				}
			}
			if resolved, err := filepath.EvalSymlinks(current); err == nil {
				current = resolved
			}
			for _, part := range slices.Backward(tail) {
				current = filepath.Join(current, part)
			}
			return filepath.Clean(current), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		tail = append(tail, filepath.Base(current))
		current = parent
	}
}

func Read(root, destination string) (Claim, bool, error) {
	destination, err := CanonicalDestination(destination)
	if err != nil {
		return Claim{}, false, err
	}
	path := Path(root, destination)
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return Claim{}, false, nil
	}
	if err != nil {
		return Claim{}, false, err
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0o077 != 0 {
		return Claim{}, false, fmt.Errorf("claim %s is not a regular file", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Claim{}, false, err
	}
	claim, err := Decode(data)
	if err != nil {
		return Claim{}, false, err
	}
	return claim, true, Validate(claim, destination)
}

func Decode(data []byte) (Claim, error) {
	if err := RejectDuplicateKeys(data); err != nil {
		return Claim{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var claim Claim
	if err := decoder.Decode(&claim); err != nil {
		return Claim{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Claim{}, errors.New("claim has trailing JSON value")
		}
		return Claim{}, err
	}
	return claim, nil
}

// RejectDuplicateKeys rejects ambiguous JSON objects at every nesting depth.
func RejectDuplicateKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	var parse func() error
	parse = func() error {
		token, err := d.Token()
		if err != nil {
			return err
		}
		delimiter, ok := token.(json.Delim)
		if !ok {
			return nil
		}
		switch delimiter {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return errors.New("claim object key is not a string")
				}
				if seen[name] {
					return fmt.Errorf("claim contains duplicate key %q", name)
				}
				seen[name] = true
				if err := parse(); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		case '[':
			for d.More() {
				if err := parse(); err != nil {
					return err
				}
			}
			_, err = d.Token()
			return err
		}
		return nil
	}
	if err := parse(); err != nil {
		return err
	}
	return nil
}

func Validate(c Claim, destination string) error {
	destination, err := CanonicalDestination(destination)
	if err != nil {
		return err
	}
	if c.Schema != Schema || c.Destination != destination {
		return errors.New("claim does not bind canonical destination")
	}
	if strings.ContainsFunc(c.Destination, unicode.IsControl) {
		return errors.New("claim destination contains a control character")
	}
	if !token(c.Name) || !token(c.Producer) || !token(c.Format) || !token(c.Scope) || !token(c.OperationID) {
		return errors.New("claim has invalid identity metadata")
	}
	if c.Kind != KindDirectory && c.Kind != KindFile {
		return errors.New("claim has invalid kind")
	}
	if c.State != StateReserved && c.State != StateActive && c.State != StateUpdating && c.State != StateReleasing && c.State != StateRestoring {
		return errors.New("claim has invalid state")
	}
	if c.Generation == 0 || !digest(c.SourceDigest) || len(c.Consumers) == 0 || len(c.Files) == 0 {
		return errors.New("claim has invalid source metadata")
	}
	if !sortedUnique(c.Consumers) || !sort.SliceIsSorted(c.Files, func(i, j int) bool { return c.Files[i].Path < c.Files[j].Path }) {
		return errors.New("claim lists are not sorted and unique")
	}
	for _, consumer := range c.Consumers {
		if !token(consumer) {
			return errors.New("claim has invalid consumer")
		}
	}
	for _, file := range c.Files {
		if !pathRecord(file.Path, c.Kind) || !digest(file.SHA) {
			return errors.New("claim has invalid file record")
		}
	}
	manifestDigest, err := ManifestDigest(c.Files)
	if err != nil || manifestDigest != c.SourceDigest {
		return errors.New("claim source digest does not bind its files")
	}
	basename := filepath.Base(c.Destination)
	if (c.Kind == KindDirectory && basename != c.Name) || (c.Kind == KindFile && basename != c.Name+".md") {
		return errors.New("claim name does not match its destination")
	}
	return nil
}
func token(value string) bool {
	return value != "" && !strings.ContainsAny(value, "/\\") && !strings.ContainsFunc(value, unicode.IsControl)
}
func pathRecord(value, kind string) bool {
	if kind == KindFile {
		return value == "."
	}
	return value != "" && value != "." && !strings.HasSuffix(value, "/") && !strings.Contains(value, `\`) && !strings.HasPrefix(value, "/") && pathpkg.Clean(value) == value && value != ".." && !strings.HasPrefix(value, "../") && !strings.ContainsFunc(value, unicode.IsControl)
}
func digest(value string) bool {
	raw, err := hex.DecodeString(value)
	return err == nil && len(raw) == sha256.Size && value == strings.ToLower(value)
}
func sortedUnique(values []string) bool {
	if !sort.StringsAreSorted(values) {
		return false
	}
	for i := 1; i < len(values); i++ {
		if values[i-1] == values[i] {
			return false
		}
	}
	return true
}

// ManifestDigest is the portable framing: sorted path UTF-8, NUL, lowercase SHA-256, NUL, repeated.
func ManifestDigest(files []FileHash) (string, error) {
	if len(files) == 0 || !sort.SliceIsSorted(files, func(i, j int) bool { return files[i].Path < files[j].Path }) {
		return "", errors.New("files must be sorted")
	}
	h := sha256.New()
	previous := ""
	for _, file := range files {
		if file.Path == previous || !digest(file.SHA) {
			return "", errors.New("files must be unique with lowercase hashes")
		}
		previous = file.Path
		_, _ = h.Write([]byte(file.Path))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(file.SHA))
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func Digest(claim Claim) (string, error) {
	if err := Validate(claim, claim.Destination); err != nil {
		return "", err
	}
	hash := sha256.New()
	writeDigestFrame(hash, "protocol", "agent-skills-claim-v1")
	writeDigestFrame(hash, "schema", strconv.Itoa(claim.Schema))
	writeDigestFrame(hash, "name", claim.Name)
	writeDigestFrame(hash, "kind", claim.Kind)
	writeDigestFrame(hash, "state", claim.State)
	writeDigestFrame(hash, "operation_id", claim.OperationID)
	writeDigestFrame(hash, "generation", strconv.FormatUint(claim.Generation, 10))
	writeDigestFrame(hash, "destination", claim.Destination)
	writeDigestFrame(hash, "producer", claim.Producer)
	writeDigestFrame(hash, "format", claim.Format)
	writeDigestFrame(hash, "scope", claim.Scope)
	writeDigestFrame(hash, "consumer_count", strconv.Itoa(len(claim.Consumers)))
	for _, consumer := range claim.Consumers {
		writeDigestFrame(hash, "consumer", consumer)
	}
	writeDigestFrame(hash, "source_digest", claim.SourceDigest)
	writeDigestFrame(hash, "file_count", strconv.Itoa(len(claim.Files)))
	for _, file := range claim.Files {
		writeDigestFrame(hash, "file_path", file.Path)
		writeDigestFrame(hash, "file_sha256", file.SHA)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func writeDigestFrame(hash io.Writer, key, value string) {
	_, _ = io.WriteString(hash, key)
	_, _ = hash.Write([]byte{0})
	_, _ = io.WriteString(hash, value)
	_, _ = hash.Write([]byte{0})
}

// Transition is the single-claim convenience around a LockedStore transaction.
func Transition(root string, next Claim, expectedGeneration uint64, previousDigest string) error {
	canonical, err := CanonicalDestination(next.Destination)
	if err != nil {
		return err
	}
	next.Destination = canonical
	if err := Validate(next, canonical); err != nil {
		return err
	}
	locks, err := AcquireLocks(root, []string{canonical})
	if err != nil {
		return err
	}
	defer func() { _ = locks.Release() }()
	store := &LockedStore{root: root, locks: locks, destinations: map[string]bool{canonical: true}}
	return store.CompareAndSwap(next, expectedGeneration, previousDigest)
}

// Reserve persists a new reserved claim with O_EXCL. Call Activate with the same operation ID.
func Reserve(root string, claim Claim) error {
	claim.State = StateReserved
	if claim.Generation == 0 {
		claim.Generation = 1
	}
	return Transition(root, claim, 0, "")
}
func Activate(root string, claim Claim, expectedGeneration uint64, previousDigest string) error {
	claim.State = StateActive
	return Transition(root, claim, expectedGeneration, previousDigest)
}

func write(root string, claim Claim, replace bool) error {
	path := Path(root, claim.Destination)
	data, err := json.MarshalIndent(claim, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if !replace {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return err
		}
		_, writeErr := file.Write(data)
		closeErr := file.Close()
		if writeErr != nil {
			return writeErr
		}
		if closeErr != nil {
			return closeErr
		}
		return syncPath(path)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".claim-")
	if err != nil {
		return err
	}
	name := temp.Name()
	defer func() { _ = os.Remove(name) }()
	if _, err = temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err = temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err = temp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	return syncPath(path)
}
func syncPath(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	err = file.Sync()
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return syncDir(filepath.Dir(path))
}
func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	err = dir.Sync()
	closeErr := dir.Close()
	if err != nil {
		return err
	}
	return closeErr
}
