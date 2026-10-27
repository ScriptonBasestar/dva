package cirun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

type compositionRuntimeKey struct{}
type compositionRuntime struct {
	id, root, state string
	ancestry        []compositionFrame
	workers         *compositionWorkers
}

func (s compositionScope) createGrant(grant compositionGrant) error {
	if err := os.MkdirAll(filepath.Dir(s.dir), 0o700); err != nil {
		return err
	}
	if err := os.Mkdir(s.dir, 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(grant)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.dir, "grant"), data, 0o600)
}

func (s compositionScope) verifyGrant(grant compositionGrant) error {
	data, err := os.ReadFile(filepath.Join(s.dir, "grant"))
	if err != nil {
		return err
	}
	var issued compositionGrant
	if err := decodeCompositionJSON(string(data), &issued); err != nil {
		return err
	}
	if !reflect.DeepEqual(issued, grant) {
		return compositionFailure("scope", "grant does not match parent step scope")
	}
	return nil
}

func acquireComposition(ctx context.Context, state, root, id, receiptState, parentValue, grantValue string, resources []string) (heldLocks, *compositionGrant, error) {
	grant, err := admitComposition(state, root, parentValue, grantValue, time.Now(), nil)
	if err != nil {
		return heldLocks{}, nil, err
	}
	scope := scopeFor(state, grant.Ancestry[len(grant.Ancestry)-1])
	var locks heldLocks
	err = scope.locked(ctx, func() error {
		if err := scope.open(); err != nil {
			return err
		}
		if err := scope.verifyGrant(*grant); err != nil {
			return err
		}
		if _, err := admitComposition(state, root, parentValue, grantValue, time.Now(), nil); err != nil {
			return err
		}
		children, err := scope.children()
		if err != nil {
			return err
		}
		if len(children) >= 256 {
			return compositionFailure("scope", "per-step child receipt limit reached")
		}
		for _, child := range children {
			owner, err := lockOwner(state, "root", child.Root)
			if err != nil {
				return err
			}
			if owner == child.RunID {
				return &BusyError{ActiveRunID: owner, Conflict: Conflict{Kind: "composition_child", Key: child.Root, RunID: owner}}
			}
			if !child.Finished {
				return compositionFailure("scope", "previous child completion is unconfirmed")
			}
		}
		locks, err = acquire(state, root, id, resources...)
		if err != nil {
			return err
		}
		if err := scope.writeChild(compositionChild{RunID: id, Root: root, State: receiptState}); err != nil {
			locks.release()
			locks = heldLocks{}
			return err
		}
		return nil
	})
	if err != nil {
		locks.release()
		return heldLocks{}, nil, err
	}
	return locks, grant, nil
}

func verifyCompositionChild(child compositionChild) error {
	if err := canonicalCompositionDir(child.State); err != nil {
		return compositionFailure("receipt", err.Error())
	}
	data, err := os.ReadFile(filepath.Join(child.State, child.RunID+".json"))
	if err != nil {
		return compositionFailure("receipt", err.Error())
	}
	var report Report
	if err := json.Unmarshal(data, &report); err != nil {
		return compositionFailure("receipt", err.Error())
	}
	if report.ID != child.RunID || report.Root != child.Root || report.Status != "succeeded" || report.FinishedAt.IsZero() {
		return compositionFailure("child", fmt.Sprintf("child %s status=%s: %s", child.RunID, report.Status, report.Error))
	}
	if child.Error != "" {
		return compositionFailure("child", child.Error)
	}
	return nil
}

func compositionEnvironment(inherited []string, value string) []string {
	out := make([]string, 0, len(inherited)+1)
	for _, env := range inherited {
		if !strings.HasPrefix(env, "DVA_CI_COMPOSITION=") {
			out = append(out, env)
		}
	}
	if value != "" {
		out = append(out, "DVA_CI_COMPOSITION="+value)
	}
	return out
}

func compositionWorkerEnvironment(inherited []string, grant compositionGrant, explicit map[string]bool) []string {
	out := make([]string, 0, len(inherited))
	for _, env := range inherited {
		key, value, _ := strings.Cut(env, "=")
		if issued, ok := grant.WorkerDefaults[key]; ok && !explicit[key] && issued == value {
			continue
		}
		out = append(out, env)
	}
	return out
}

func finishCompositionChild(scope compositionScope, state string, report Report, retErr error) error {
	child := compositionChild{RunID: report.ID, Root: report.Root, State: state, Finished: true}
	if retErr != nil {
		child.Error = retErr.Error()
	}
	return errors.Join(retErr, scope.writeChild(child))
}
