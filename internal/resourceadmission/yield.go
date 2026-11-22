package resourceadmission

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// YieldWanted reports whether releasing the running claim id would let the
// existing ordering admit a waiting request that it does not admit now, and
// whose current score is at least the rank of priority. It never changes
// resource state: abandoned waiting requests (free lease lock) are ignored
// here and left for Acquire to remove.
func YieldWanted(ctx context.Context, dir string, p Policy, digest, id, priority string) (bool, error) {
	own, ok := priorityRank[priority]
	if !ok {
		return false, errors.New("unknown resource priority")
	}
	age, err := time.ParseDuration(p.AgingInterval)
	if err != nil || age <= 0 {
		return false, errors.New("invalid aging interval")
	}
	wanted := false
	err = locked(ctx, filepath.Join(dir, "state.lock"), func() error {
		entries, err := readEntries(dir)
		if err != nil {
			return err
		}
		live := entries[:0]
		found := false
		for _, e := range entries {
			if e.Policy != digest {
				return errors.New("resource policy differs from existing state")
			}
			if e.ID == id {
				if e.State != "running" {
					return errors.New("yield check requires the caller's running claim")
				}
				found = true
			}
			if e.State == "waiting" {
				held, err := leaseHeld(dir, e.ID)
				if err != nil {
					return err
				}
				if !held {
					continue
				}
			}
			live = append(live, e)
		}
		if !found {
			return errors.New("yield check: claim is not recorded")
		}
		now := time.Now()
		with, err := selectNext(live, p, nil, now, age)
		if err != nil {
			return err
		}
		without, err := selectNext(live, p, map[string]bool{id: true}, now, age)
		if err != nil || without == "" || without == with {
			return err
		}
		for _, e := range live {
			if e.ID == without {
				// Base rank only, strictly higher: aging orders waiters but never
				// displaces a running claim, so a yield goes to a higher class.
				wanted = priorityRank[e.Priority] > own
			}
		}
		return nil
	})
	return wanted, err
}

// leaseHeld reports whether a request's supervisor still holds its lease lock.
func leaseHeld(dir, id string) (bool, error) {
	f, err := openPrivateRegular(filepath.Join(dir, id+".lock"), os.O_RDWR)
	if err != nil {
		return false, err
	}
	err = tryLock(f)
	if err == nil {
		return false, errors.Join(unlock(f), f.Close())
	}
	closeErr := f.Close()
	if !busy(err) {
		return false, errors.Join(err, closeErr)
	}
	return true, closeErr
}
