package resourceadmission

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type entry struct {
	ID, Policy, Class, Priority, State string
	QueuedAt                           time.Time
}
type Lease struct {
	dir, id string
	lock    *os.File
}

func (l *Lease) ID() string {
	if l == nil {
		return ""
	}
	return l.id
}

func privateDirectory(path string) error {
	if !filepath.IsAbs(path) {
		return errors.New("resource state path must be absolute")
	}
	if err := os.MkdirAll(path, 0700); err != nil {
		return err
	}
	i, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !i.IsDir() || i.Mode().Perm() != 0700 || !owned(i) {
		return errors.New("resource state must be a current-user-owned 0700 directory")
	}
	return nil
}

func locked(ctx context.Context, path string, fn func() error) (retErr error) {
	return lockFile(ctx, path, true, fn)
}

func lockFile(ctx context.Context, path string, create bool, fn func() error) (retErr error) {
	flags := os.O_RDWR
	if create {
		flags |= os.O_CREATE
	}
	f, err := openPrivateRegular(path, flags)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	current, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || !owned(info) || !current.Mode().IsRegular() || !os.SameFile(info, current) {
		return errors.New("resource lock is not a private regular file")
	}
	for {
		err = tryLock(f)
		if err == nil {
			break
		}
		if !busy(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
	defer func() { retErr = errors.Join(retErr, unlock(f)) }()
	return fn()
}

func writeEntry(dir string, e entry) error {
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, ".entry-")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, e.ID+".json"))
}

func readEntries(dir string) ([]entry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var entries []entry
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		if len(entries) >= 1000 {
			return nil, errors.New("too many resource claims")
		}
		info, err := f.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || !owned(info) || info.Size() > 8192 {
			return nil, fmt.Errorf("invalid resource entry %s", f.Name())
		}
		handle, err := openPrivateRegular(filepath.Join(dir, f.Name()), os.O_RDONLY)
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(handle, 8193))
		closeErr := handle.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(b) > 8192 {
			return nil, errors.New("resource entry exceeds size limit")
		}
		var e entry
		if err = json.Unmarshal(b, &e); err != nil || e.ID+".json" != f.Name() {
			return nil, fmt.Errorf("invalid resource entry %s", f.Name())
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Acquire registers a cooperative request. Interrupted active requests remain
// recorded and consume capacity until independently resolved by an operator.
func Acquire(ctx context.Context, dir string, p Policy, digest, class, priority string) (lease *Lease, retErr error) {
	weight := p.Weights[class]
	if weight < 1 {
		return nil, errors.New("unknown resource class")
	}
	rank := map[string]int{"background": 0, "normal": 1, "interactive": 2}
	if _, ok := rank[priority]; !ok {
		return nil, errors.New("unknown resource priority")
	}
	age, err := time.ParseDuration(p.AgingInterval)
	if err != nil || age <= 0 {
		return nil, errors.New("invalid aging interval")
	}
	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("resource admission requires a finite wait deadline")
	}
	if err = privateDirectory(dir); err != nil {
		return nil, err
	}
	idbytes := make([]byte, 16)
	if _, err = rand.Read(idbytes); err != nil {
		return nil, err
	}
	id := hex.EncodeToString(idbytes)
	lockPath := filepath.Join(dir, id+".lock")
	lf, err := openPrivateRegular(lockPath, os.O_CREATE|os.O_EXCL|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	if err = tryLock(lf); err != nil {
		return nil, errors.Join(err, lf.Close(), os.Remove(lockPath))
	}
	entryPath := filepath.Join(dir, id+".json")
	created := false
	success := false
	defer func() {
		if !success {
			removed := !created
			if created {
				cleanupCtx, cancel := context.WithTimeout(context.Background(), time.Second)
				cleanupErr := locked(cleanupCtx, filepath.Join(dir, "state.lock"), func() error { return os.Remove(entryPath) })
				cancel()
				retErr = errors.Join(retErr, cleanupErr)
				removed = cleanupErr == nil
			}
			retErr = errors.Join(retErr, unlock(lf), lf.Close())
			if removed {
				retErr = errors.Join(retErr, os.Remove(lockPath))
			}
		}
	}()
	request := entry{ID: id, Policy: digest, Class: class, Priority: priority, State: "waiting", QueuedAt: time.Now().UTC()}
	for {
		admitted := false
		err = locked(ctx, filepath.Join(dir, "state.lock"), func() error {
			entries, err := readEntries(dir)
			if err != nil {
				return err
			}
			live := entries[:0]
			for _, e := range entries {
				if e.State == "waiting" && e.ID != id {
					path := filepath.Join(dir, e.ID+".lock")
					f, err := openPrivateRegular(path, os.O_RDWR)
					if err != nil {
						return fmt.Errorf("inspect waiting lease: %w", err)
					}
					err = tryLock(f)
					if err == nil {
						if removeErr := os.Remove(filepath.Join(dir, e.ID+".json")); removeErr != nil {
							_ = unlock(f)
							_ = f.Close()
							return removeErr
						}
						if unlockErr := unlock(f); unlockErr != nil {
							_ = f.Close()
							return unlockErr
						}
						if closeErr := f.Close(); closeErr != nil {
							return closeErr
						}
						if removeErr := os.Remove(path); removeErr != nil {
							return removeErr
						}
						continue
					}
					closeErr := f.Close()
					if !busy(err) {
						return err
					}
					if closeErr != nil {
						return closeErr
					}
				}
				live = append(live, e)
			}
			entries = live
			for _, e := range entries {
				if e.Policy != digest {
					return errors.New("resource policy differs from existing state")
				}
			}
			if !created {
				if err = writeEntry(dir, request); err != nil {
					return err
				}
				created = true
				entries = append(entries, request)
			}
			var waiting []entry
			used, running, heavy := 0, 0, 0
			for _, e := range entries {
				switch e.State {
				case "running":
					w := p.Weights[e.Class]
					if w < 1 {
						return errors.New("invalid active resource class")
					}
					used += w
					running++
					if e.Class == "heavy" {
						heavy++
					}
				case "waiting":
					if _, ok := rank[e.Priority]; !ok || e.QueuedAt.IsZero() {
						return errors.New("invalid waiting resource priority or timestamp")
					}
					waiting = append(waiting, e)
				default:
					return errors.New("invalid resource state")
				}
			}
			now := time.Now()
			score := func(e entry) int {
				v, ok := rank[e.Priority]
				if !ok {
					return -1
				}
				boost := now.Sub(e.QueuedAt) / age
				if boost < 0 {
					boost = 0
				}
				if boost >= 2 {
					return 2
				}
				return min(2, v+int(boost))
			}
			sort.SliceStable(waiting, func(i, j int) bool {
				a, b := score(waiting[i]), score(waiting[j])
				if a != b {
					return a > b
				}
				if waiting[i].QueuedAt.Equal(waiting[j].QueuedAt) {
					return waiting[i].ID < waiting[j].ID
				}
				return waiting[i].QueuedAt.Before(waiting[j].QueuedAt)
			})
			selected := ""
			for _, candidate := range waiting {
				w := p.Weights[candidate.Class]
				if w < 1 {
					return errors.New("invalid waiting resource class")
				}
				// Reserve free capacity for an aged heavy request when only
				// other work's capacity prevents it from starting. A heavy
				// already occupying the heavy lane permits light backfill.
				if candidate.Class == "heavy" && now.Sub(candidate.QueuedAt) >= age && heavy < p.MaxHeavy && used+w > p.Capacity {
					break
				}
				if used+w <= p.Capacity && running < p.MaxRunning && (candidate.Class != "heavy" || heavy < p.MaxHeavy) {
					selected = candidate.ID
					break
				}
			}
			if selected == id {
				request.State = "running"
				if err = writeEntry(dir, request); err != nil {
					return err
				}
				admitted = true
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if admitted {
			success = true
			return &Lease{dir: dir, id: id, lock: lf}, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

// Complete releases a lease only after the owning supervisor has observed child
// termination. It does not attempt to recover interrupted running leases.
func (l *Lease) Complete(ctx context.Context) error {
	if l == nil || l.lock == nil {
		return errors.New("resource lease already closed")
	}
	err := locked(ctx, filepath.Join(l.dir, "state.lock"), func() error { return os.Remove(filepath.Join(l.dir, l.id+".json")) })
	if err != nil {
		return err
	}
	err = unlock(l.lock)
	err = errors.Join(err, l.lock.Close())
	l.lock = nil
	return errors.Join(err, os.Remove(filepath.Join(l.dir, l.id+".lock")))
}

// Quarantine retains the active claim while relinquishing the supervisor lock.
func (l *Lease) Quarantine() error {
	if l == nil || l.lock == nil {
		return errors.New("resource lease already closed")
	}
	err := errors.Join(unlock(l.lock), l.lock.Close())
	l.lock = nil
	return err
}
