package cirun

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

// This bound is independent of the step execution deadline and aging policy.
const hostSlotBridgeWait = 5 * time.Minute

type hostSlotBridgeKey struct{}
type hostSlotBridge struct {
	policy      resourceadmission.Policy
	digest, dir string
}

func lookupCIEnvironment(env []string, key string) (string, bool) {
	value, found := "", false
	for _, entry := range env {
		if v, ok := strings.CutPrefix(entry, key+"="); ok {
			value, found = v, true
		}
	}
	if found {
		return value, true
	}
	return os.LookupEnv(key)
}

func resolveHostSlotBridge(opts Options) (*hostSlotBridge, error) {
	hasSlot := false
	for _, step := range opts.Profile.Steps {
		if step.HostSlot {
			hasSlot = true
		}
	}
	if !hasSlot {
		return nil, nil
	}
	if opts.Profile.Admission != nil {
		return nil, errors.New("profile admission and host_slot cannot be combined")
	}
	if value, _ := lookupCIEnvironment(opts.Env, "DVA_RESOURCE_SUPERVISED"); value != "" || os.Getenv("DVA_RESOURCE_SUPERVISED") != "" {
		return nil, errors.New("nested host_slot resource admission is unsupported")
	}
	path, declared := lookupCIEnvironment(opts.Env, "DVA_RESOURCE_POLICY")
	if !declared {
		return nil, nil
	} // TASK-621 compatibility when no host policy is declared.
	if path == "" {
		return nil, errors.New("DVA_RESOURCE_POLICY is explicitly empty")
	}
	if value, _ := lookupCIEnvironment(opts.Env, parentRunEnv); value != "" || os.Getenv(parentRunEnv) != "" {
		return nil, errors.New("nested host_slot resource admission is unsupported")
	}
	for _, step := range opts.Profile.Steps {
		if step.ChildCI != "" {
			return nil, errors.New("host_slot resource admission does not support child_ci")
		}
	}
	policy, digest, err := resourceadmission.ReadPolicy(path)
	if err != nil {
		return nil, err
	}
	if policy.MaxHeavy != 1 {
		return nil, fmt.Errorf("host_slot bridge requires max_heavy=1, got %d", policy.MaxHeavy)
	}
	dir, err := resourceadmission.DefaultDirectory()
	if err != nil {
		return nil, err
	}
	if opts.resourceDirectory != "" {
		dir = opts.resourceDirectory
	}
	return &hostSlotBridge{policy: policy, digest: digest, dir: dir}, nil
}

func (b *hostSlotBridge) acquire(ctx context.Context, label string, out io.Writer) (*resourceadmission.Lease, *AdmissionObservation, error) {
	waitCtx, cancel := context.WithTimeout(ctx, hostSlotBridgeWait)
	defer cancel()
	return admissionWait(waitCtx, b.dir, b.policy, b.digest, "heavy", "normal", label, out)
}

func finishHostSlotBridge(lease *resourceadmission.Lease, tracker *resourceCleanupTracker) error {
	if tracker.isUncertain() {
		return errors.Join(fmt.Errorf("resource lease %s quarantined: process group cleanup unconfirmed", lease.ID()), lease.Quarantine())
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return lease.Complete(ctx)
}
