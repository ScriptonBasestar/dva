package cirun

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const parentRunEnv = config.CIParentRunEnv

type parentRun struct {
	ID   string `json:"id"`
	Root string `json:"root"`
}

type heldLocks struct{ files []*os.File }

func lockPath(state, kind, key string) string {
	hash := sha256.Sum256([]byte(key))
	return filepath.Join(state, "locks", kind+"-"+hex.EncodeToString(hash[:])+".lock")
}

func acquire(state, root, id string, resources ...string) (heldLocks, error) {
	h := heldLocks{}
	keys := []Conflict{{Kind: "root", Key: root}}
	for _, key := range slices.Sorted(slices.Values(resources)) {
		keys = append(keys, Conflict{Kind: "resource", Key: key})
	}
	for _, key := range keys {
		p := lockPath(state, key.Kind, key.Key)
		if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
			h.release()
			return heldLocks{}, err
		}
		f, err := os.OpenFile(p, os.O_CREATE|os.O_RDWR, 0o600)
		if err != nil {
			h.release()
			return heldLocks{}, fmt.Errorf("open CI %s lock: %w", key.Kind, err)
		}
		if err = tryLock(f); err != nil {
			var owner string
			var readErr error
			if lockContended(err) {
				owner, readErr = readLockID(f)
			}
			_ = f.Close()
			h.release()
			if !lockContended(err) {
				return heldLocks{}, fmt.Errorf("acquire CI %s lock: %w", key.Kind, err)
			}
			if readErr != nil {
				return heldLocks{}, fmt.Errorf("read CI %s lock owner: %w", key.Kind, readErr)
			}
			key.RunID = owner
			return heldLocks{}, &BusyError{ActiveRunID: owner, Conflict: key}
		}
		h.files = append(h.files, f)
		if err := f.Truncate(0); err != nil {
			h.release()
			return heldLocks{}, err
		}
		if _, err := f.WriteAt([]byte(id+"\n"), 0); err != nil {
			h.release()
			return heldLocks{}, err
		}
	}
	return h, nil
}

func readLockID(f *os.File) (string, error) {
	b := make([]byte, 128)
	n, err := f.ReadAt(b, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	id := strings.TrimSpace(string(b[:n]))
	if !safeID(id) {
		return "", nil
	}
	return id, nil
}

func (h heldLocks) release() {
	for _, f := range slices.Backward(h.files) {
		_ = unlock(f)
		_ = f.Close()
	}
}

// lockOwner probes the actual OS lock; metadata alone never establishes liveness.
func lockOwner(state, kind, key string) (owner string, retErr error) {
	f, err := os.OpenFile(lockPath(state, kind, key), os.O_RDWR, 0o600)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("inspect CI %s lock: %w", kind, err)
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	err = tryLock(f)
	if err == nil {
		return "", unlock(f)
	}
	if !lockContended(err) {
		return "", fmt.Errorf("inspect CI %s lock: %w", kind, err)
	}
	return readLockID(f)
}

func checkParent(state, value string) error {
	if value == "" {
		return nil
	}
	var parent parentRun
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&parent); err != nil {
		return fmt.Errorf("invalid %s: %w", parentRunEnv, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid %s: trailing data", parentRunEnv)
	}
	if !safeID(parent.ID) || !filepath.IsAbs(parent.Root) || filepath.Clean(parent.Root) != parent.Root {
		return fmt.Errorf("invalid %s: expected run ID and canonical absolute root", parentRunEnv)
	}
	canonical, err := filepath.EvalSymlinks(parent.Root)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("inspect %s root: %w", parentRunEnv, err)
	}
	if err == nil && canonical != parent.Root {
		return fmt.Errorf("invalid %s: root is not canonical", parentRunEnv)
	}
	owner, err := lockOwner(state, "root", parent.Root)
	if err != nil {
		return fmt.Errorf("inspect %s: %w", parentRunEnv, err)
	}
	if owner == parent.ID {
		return &BusyError{ActiveRunID: owner, Conflict: Conflict{Kind: "nested", Key: parent.Root, RunID: owner}}
	}
	return nil
}

func environmentValue(env []string, key string) string {
	value := ""
	for _, entry := range env {
		if v, ok := strings.CutPrefix(entry, key+"="); ok {
			value = v
		}
	}
	return value
}
