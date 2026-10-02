package skillclaim

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
)

type LockSet struct {
	paths    []string
	released bool
}

func AcquireLocks(root string, destinations []string) (*LockSet, error) {
	unique := map[string]bool{}
	paths := make([]string, 0, len(destinations))
	for _, destination := range destinations {
		canonical, err := CanonicalDestination(destination)
		if err != nil {
			return nil, err
		}
		path := Path(root, canonical) + ".lock"
		if !unique[path] {
			unique[path] = true
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	if len(paths) > 0 {
		if err := os.MkdirAll(filepath.Dir(paths[0]), 0o700); err != nil {
			return nil, err
		}
	}
	for index, path := range paths {
		if err := os.Mkdir(path, 0o700); err != nil {
			for i := index - 1; i >= 0; i-- {
				_ = os.Remove(paths[i])
			}
			if errors.Is(err, os.ErrExist) {
				return nil, fmt.Errorf("claim mutation lock exists at %s", path)
			}
			return nil, err
		}
	}
	return &LockSet{paths: paths}, nil
}
func (locks *LockSet) Release() error {
	if locks == nil || locks.released {
		return errors.New("claim locks already released")
	}
	locks.released = true
	var first error
	for _, path := range slices.Backward(locks.paths) {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) && first == nil {
			first = err
		}
	}
	return first
}

// LockedStore is a multi-claim transaction boundary. Its locks must be acquired once, in canonical order.
type LockedStore struct {
	root         string
	locks        *LockSet
	destinations map[string]bool
}

func Begin(root string, destinations []string) (*LockedStore, error) {
	if len(destinations) == 0 {
		return nil, errors.New("claim transaction has no destinations")
	}
	authorized := map[string]bool{}
	for _, destination := range destinations {
		value, err := CanonicalDestination(destination)
		if err != nil {
			return nil, err
		}
		authorized[value] = true
	}
	locks, err := AcquireLocks(root, destinations)
	if err != nil {
		return nil, err
	}
	return &LockedStore{root: root, locks: locks, destinations: authorized}, nil
}
func (store *LockedStore) Close() error { return store.locks.Release() }
func (store *LockedStore) Read(destination string) (Claim, bool, error) {
	canonical, err := store.authorize(destination)
	if err != nil {
		return Claim{}, false, err
	}
	return Read(store.root, canonical)
}
func (store *LockedStore) authorize(destination string) (string, error) {
	if store == nil || store.locks == nil || store.locks.released {
		return "", errors.New("claim transaction is closed")
	}
	canonical, err := CanonicalDestination(destination)
	if err != nil {
		return "", err
	}
	if !store.destinations[canonical] {
		return "", fmt.Errorf("destination %s is outside locked transaction", canonical)
	}
	return canonical, nil
}
func (store *LockedStore) Reserve(claim Claim) error {
	claim.State = StateReserved
	claim.Generation = 1
	return store.CompareAndSwap(claim, 0, "")
}
func (store *LockedStore) CompareAndSwap(next Claim, generation uint64, previous string) error {
	canonical, err := CanonicalDestination(next.Destination)
	if err != nil {
		return err
	}
	next.Destination = canonical
	if _, err := store.authorize(canonical); err != nil {
		return err
	}
	if err := Validate(next, canonical); err != nil {
		return err
	}
	current, found, err := store.Read(canonical)
	if err != nil {
		return err
	}
	if !found {
		if generation != 0 || previous != "" || next.State != StateReserved || next.Generation != 1 {
			return errors.New("new claim must be reserved with empty predecessor")
		}
		return write(store.root, next, false)
	}
	digest, err := Digest(current)
	if err != nil {
		return err
	}
	if current.Generation != generation || digest != previous || current.Producer != next.Producer {
		return errors.New("claim reservation changed")
	}
	if current.Generation == ^uint64(0) || next.Generation != current.Generation+1 {
		return errors.New("claim generation must advance exactly one")
	}
	if !allowedTransition(current, next) {
		return errors.New("claim state or operation transition is invalid")
	}
	return write(store.root, next, true)
}
func (store *LockedStore) Remove(destination, producer, operationID string, generation uint64, previous string) error {
	canonical, err := store.authorize(destination)
	if err != nil {
		return err
	}
	current, found, err := store.Read(canonical)
	if err != nil {
		return err
	}
	if !found {
		return errors.New("claim reservation is absent")
	}
	digest, err := Digest(current)
	if err != nil {
		return err
	}
	if current.Producer != producer || current.OperationID != operationID || current.Generation != generation || digest != previous || (current.State != StateReleasing && current.State != StateRestoring) {
		return errors.New("claim removal precondition failed")
	}
	if err := os.Remove(Path(store.root, current.Destination)); err != nil {
		return err
	}
	return syncDir(filepath.Dir(Path(store.root, current.Destination)))
}
func allowedTransition(from, to Claim) bool {
	if from.State == StateReserved {
		return to.State == StateActive && to.OperationID == from.OperationID && samePayload(from, to)
	}
	if from.State == StateActive {
		if to.OperationID == from.OperationID || !sameIdentity(from, to) {
			return false
		}
		return (to.State == StateUpdating) || ((to.State == StateReleasing || to.State == StateRestoring) && samePayload(from, to))
	}
	if from.OperationID != to.OperationID {
		return false
	}
	if to.State != StateActive || !sameIdentity(from, to) {
		return false
	}
	return from.State == StateUpdating || ((from.State == StateReleasing || from.State == StateRestoring) && samePayload(from, to))
}
func sameIdentity(left, right Claim) bool {
	return left.Name == right.Name && left.Kind == right.Kind && left.Destination == right.Destination && left.Producer == right.Producer && left.Format == right.Format && left.Scope == right.Scope
}
func samePayload(left, right Claim) bool {
	return sameIdentity(left, right) && sameStrings(left.Consumers, right.Consumers) && sameFiles(left.Files, right.Files) && left.SourceDigest == right.SourceDigest
}
func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
func sameFiles(left, right []FileHash) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
