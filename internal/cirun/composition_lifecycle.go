package cirun

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

// A scope belongs to one parent step. Its tombstone is retained so a delayed
// child cannot register after the parent's final drain. The root lock and scope
// are separate: holding the former alone does not mean this step still admits.
type compositionScope struct{ dir string }

type compositionChild struct {
	RunID    string `json:"run_id"`
	Root     string `json:"root"`
	State    string `json:"state,omitempty"`
	Finished bool   `json:"finished"`
	Error    string `json:"error,omitempty"`
}

func scopeFor(state string, frame compositionFrame) compositionScope {
	hash := sha256.Sum256([]byte(frame.Step))
	return compositionScope{filepath.Join(state, "composition", frame.RunID, hex.EncodeToString(hash[:]))}
}

// Serialize admission and closure, with a short finite wait even during cleanup.
func (s compositionScope) locked(ctx context.Context, fn func() error) (retErr error) {
	f, err := os.OpenFile(filepath.Join(s.dir, "admission.lock"), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return err
	}
	defer func() { retErr = errors.Join(retErr, f.Close()) }()
	tick := time.NewTicker(10 * time.Millisecond)
	defer tick.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		err = tryLock(f)
		if err == nil {
			break
		}
		if !lockContended(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
	defer func() { retErr = errors.Join(retErr, unlock(f)) }()
	return fn()
}

func (s compositionScope) open() error {
	info, err := os.Stat(s.dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return compositionFailure("scope", "scope is not a directory")
	}
	_, err = os.Stat(filepath.Join(s.dir, "closed"))
	if err == nil {
		return compositionFailure("scope", "parent step no longer admits children")
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s compositionScope) close(ctx context.Context, canceled bool) error {
	return s.locked(ctx, func() error {
		if err := os.WriteFile(filepath.Join(s.dir, "closed"), []byte("closed\n"), 0o600); err != nil {
			return err
		}
		if canceled {
			return os.WriteFile(filepath.Join(s.dir, "cancel"), []byte("cancel\n"), 0o600)
		}
		return nil
	})
}

func (s compositionScope) canceled() error {
	info, err := os.Stat(s.dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return compositionFailure("scope", "scope is not a directory")
	}
	_, err = os.Stat(filepath.Join(s.dir, "cancel"))
	if err == nil {
		return compositionFailure("scope", "ancestor step canceled")
	}
	if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s compositionScope) children() ([]compositionChild, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	var children []compositionChild
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		if !safeID(entry.Name()[:len(entry.Name())-5]) || entry.IsDir() {
			return nil, compositionFailure("scope", "invalid child record name")
		}
		data, err := os.ReadFile(filepath.Join(s.dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var child compositionChild
		if err := decodeCompositionJSON(string(data), &child); err != nil {
			return nil, err
		}
		if !safeID(child.RunID) || entry.Name() != child.RunID+".json" {
			return nil, compositionFailure("scope", "invalid child identity")
		}
		if err := canonicalCompositionDir(child.Root); err != nil {
			return nil, err
		}
		children = append(children, child)
	}
	return children, nil
}

func (s compositionScope) writeChild(child compositionChild) (retErr error) {
	data, err := json.Marshal(child)
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, ".child-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer func() {
		if err := os.Remove(name); err != nil && !os.IsNotExist(err) {
			retErr = errors.Join(retErr, err)
		}
	}()
	_, writeErr := f.Write(data)
	if err := errors.Join(writeErr, f.Close()); err != nil {
		return err
	}
	return os.Rename(name, filepath.Join(s.dir, child.RunID+".json"))
}

// Every child observes every ancestor scope, so cancellation propagates directly
// to leaves rather than spending a cleanup grace at each level of the chain.
func monitorComposition(ctx context.Context, cancel context.CancelCauseFunc, state string, grant compositionGrant) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			for _, frame := range grant.Ancestry {
				if err := scopeFor(state, frame).canceled(); err != nil {
					cancel(err)
					return
				}
				owner, err := lockOwner(state, "root", frame.Root)
				if err != nil || owner != frame.RunID {
					cancel(errors.Join(compositionFailure("ancestor", "ancestor lock lost during execution"), err))
					return
				}
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}

// Close first, then wait for terminal child registration AND released root lock.
// On a successful shell exit children retain the original finite step deadline.
// OCI leaves retain their existing 10s cleanup budget. All levels observe the
// same ancestor closure concurrently; 15s is a cleanup bound, not extra work time.
func (s compositionScope) drain(stepCtx context.Context, state string, canceled bool, signalWork func()) (retErr error) {
	if _, bounded := stepCtx.Deadline(); !bounded && !canceled {
		return compositionFailure("scope", "child drain requires a finite step deadline")
	}
	// Admission closure must work even when the work context already expired.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	err := s.close(ctx, canceled)
	if canceled && signalWork != nil {
		signalWork()
	}
	if err != nil {
		return err
	}
	if !canceled {
		cancel()
	}
	reported := map[string]bool{}
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		children, err := s.children()
		if err != nil {
			return err
		}
		pending := false
		for _, child := range children {
			owner, err := lockOwner(state, "root", child.Root)
			if err != nil {
				return errors.Join(retErr, err)
			}
			if !child.Finished || owner == child.RunID {
				pending = true
			} else if !reported[child.RunID] {
				// Do not turn a swallowed child exit failure into a successful step.
				retErr = errors.Join(retErr, verifyCompositionChild(child))
				reported[child.RunID] = true
			}
		}
		if !pending {
			return retErr
		}
		if !canceled {
			select {
			case <-stepCtx.Done():
				retErr = errors.Join(retErr, context.Cause(stepCtx))
				ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				err := s.close(ctx, true)
				if signalWork != nil {
					signalWork()
				}
				if err != nil {
					return errors.Join(retErr, err)
				}
				canceled = true
			case <-ticker.C:
			}
			continue
		}
		select {
		case <-ctx.Done():
			return errors.Join(retErr, compositionFailure("cleanup", "child terminal receipt or lock release unconfirmed within cleanup bound"))
		case <-ticker.C:
		}
	}
}
