package resourceadmission

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
)

// DefaultDirectory identifies this user's single host-wide resource state.
func DefaultDirectory() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(u.HomeDir, ".local", "state", "dva", "resource-admission"), nil
}

type StatusEntry struct {
	ID       string `json:"id"`
	Class    string `json:"class"`
	Priority string `json:"priority"`
	State    string `json:"state"`
	Liveness string `json:"liveness"`
	Reason   string `json:"reason,omitempty"`
}

// CreateReport exclusively creates a private report in a verified parent.
func CreateReport(path string) (*os.File, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("report path must be absolute")
	}
	i, err := os.Lstat(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if !i.IsDir() || i.Mode().Perm() != 0700 || !owned(i) {
		return nil, fmt.Errorf("report parent must be current-user-owned mode 0700")
	}
	return os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
}

// Status is read-only: stale active claims are reported, never removed.
func Status(ctx context.Context, dir, digest string) ([]StatusEntry, error) {
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return []StatusEntry{}, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode().Perm() != 0700 || !owned(info) {
		return nil, fmt.Errorf("invalid resource state directory")
	}
	var result []StatusEntry
	lockPath := filepath.Join(dir, "state.lock")
	if _, err = os.Lstat(lockPath); os.IsNotExist(err) {
		entries, readErr := readEntries(dir)
		if readErr != nil {
			return nil, readErr
		}
		if len(entries) != 0 {
			return nil, fmt.Errorf("resource state lock missing; operator review required")
		}
		return []StatusEntry{}, nil
	} else if err != nil {
		return nil, err
	}
	err = lockFile(ctx, lockPath, false, func() error {
		entries, err := readEntries(dir)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Policy != digest {
				return fmt.Errorf("resource policy differs from existing state")
			}
			item := StatusEntry{ID: e.ID, Class: e.Class, Priority: e.Priority, State: e.State}
			f, err := os.OpenFile(filepath.Join(dir, e.ID+".lock"), os.O_RDWR, 0)
			if err != nil {
				item.Liveness = "unknown"
				item.Reason = "lease lock unavailable; operator review required"
			} else {
				err = tryLock(f)
				if err == nil {
					item.Liveness = "not-held"
					if e.State == "running" {
						item.State = "quarantined"
						item.Reason = "supervisor lease no longer held; operator review required"
					}
					_ = unlock(f)
				} else if busy(err) {
					item.Liveness = "held"
				} else {
					item.Liveness = "unknown"
					item.Reason = "lease liveness unavailable; operator review required"
				}
				_ = f.Close()
			}
			result = append(result, item)
		}
		return nil
	})
	return result, err
}
