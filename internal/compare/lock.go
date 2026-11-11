package compare

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// ErrLocked reports that another dva process holds the lock of a comparison.
var ErrLocked = errors.New("compare is locked by another dva process")

// LockPath is the lock file of comparison name, beside its record.
func LockPath(configDir, name string) (string, error) {
	rec, err := RecordPath(configDir, name)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(rec), name+".lock"), nil
}

// Lock takes the exclusive lock of comparison name without waiting and returns its release
// function. A second holder gets an error satisfying errors.Is(err, ErrLocked). The lock is an
// advisory file lock owned by the open file, so the operating system drops it when the process
// dies; a lock file left behind by a crash does not lock anything.
func Lock(configDir, name string) (release func(), err error) {
	path, err := LockPath(configDir, name)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create %s directory: %w", config.CompareDirName, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open compare lock: %w", err)
	}
	if err := lockFile(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() { unlockFile(f); _ = f.Close() }, nil
}
