package cirun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// hostSlotFilename names the single host-wide advisory slot. Unlike the
// per-root and per-resource CI locks it has no hash component: every dva
// invocation on the host — any repository, worktree, or session — competes
// for the same file, so declared heavy steps serialize against each other.
const hostSlotFilename = "host-slot.lock"

// hostSlotPollInterval bounds acquisition latency and keeps the wait
// cancellable: flock has no interruptible blocking form, so the waiter polls
// non-blocking attempts instead of entering a blocking syscall.
const hostSlotPollInterval = 100 * time.Millisecond

// hostSlotMetadataLimit bounds the occupancy record a waiter reads. Records
// beyond it (unreachable in practice: it would take multi-kilobyte root paths
// plus step names) fail to parse and report as an unrecorded holder.
const hostSlotMetadataLimit = 8 << 10

// hostSlotOriginKey carries the pre-budget step context past the step timeout
// arming, so a slot wait is never charged to the declared step timeout.
type hostSlotOriginKey struct{}

// hostSlotMetadata is the occupancy record a holder writes into the slot file.
// A waiter reads it to report exactly whom it is queued behind.
type hostSlotMetadata struct {
	RunID      string    `json:"run_id"`
	PID        int       `json:"pid"`
	Root       string    `json:"root"`
	Profile    string    `json:"profile"`
	Step       string    `json:"step"`
	AcquiredAt time.Time `json:"acquired_at"`
}

func (m hostSlotMetadata) describe() string {
	return fmt.Sprintf("run=%s pid=%d profile=%s step=%s root=%s since=%s",
		m.RunID, m.PID, m.Profile, m.Step, m.Root, m.AcquiredAt.Format(time.RFC3339))
}

func hostSlotPath(state string) string { return filepath.Join(state, "locks", hostSlotFilename) }

func hostSlotState(ctx context.Context) string {
	if runtime, ok := ctx.Value(compositionRuntimeKey{}).(compositionRuntime); ok && runtime.state != "" {
		return runtime.state
	}
	state, err := stateDirectory("")
	if err != nil {
		return ""
	}
	return state
}

// readHostSlotMetadata reads the current occupancy record. The record length
// is not fixed: root paths and step names vary, so the reader sizes the buffer
// from the file (capped at 8 KiB) instead of truncating at a constant. The
// flock is the authority; a stale record from a crashed holder is reported
// as-is.
func readHostSlotMetadata(f *os.File) hostSlotMetadata {
	info, err := f.Stat()
	if err != nil || info.Size() == 0 {
		return hostSlotMetadata{}
	}
	size := min(info.Size(), hostSlotMetadataLimit)
	b := make([]byte, size)
	n, err := f.ReadAt(b, 0)
	if err != nil && !errors.Is(err, io.EOF) {
		return hostSlotMetadata{}
	}
	var m hostSlotMetadata
	if json.Unmarshal(b[:n], &m) != nil {
		return hostSlotMetadata{}
	}
	return m
}

// acquireHostSlot blocks until the host-wide slot is free, then holds it for
// the caller's step. The wait is cancellable through ctx, logs each distinct
// holder it queues behind, and refuses immediately when the holder is an
// composition ancestor — a child re-declaring host_slot under a parent that
// already holds it would otherwise wait for its own parent forever.
func acquireHostSlot(ctx context.Context, state string, holder hostSlotMetadata, step string, out io.Writer, ancestry []compositionFrame) (release func(), waited time.Duration, retErr error) {
	path := hostSlotPath(state)
	if state == "" {
		return nil, 0, errors.New("host slot state directory unavailable")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, 0, fmt.Errorf("prepare host slot: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, 0, fmt.Errorf("open host slot: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = f.Close()
		}
	}()
	start := time.Now()
	reported := hostSlotMetadata{RunID: "\x00none"}
	announce := func(current hostSlotMetadata) {
		if current.RunID == reported.RunID {
			return
		}
		reported = current
		if current.RunID == "" {
			_, _ = fmt.Fprintf(out, "ci step %s host_slot=waiting holder=<unrecorded>\n", step)
			return
		}
		_, _ = fmt.Fprintf(out, "ci step %s host_slot=waiting holder %s\n", step, current.describe())
	}
	for {
		if err := ctx.Err(); err != nil {
			return nil, time.Since(start), err
		}
		err := tryLock(f)
		if err == nil {
			holder.AcquiredAt = time.Now()
			record, marshalErr := json.Marshal(holder)
			if marshalErr == nil {
				if truncateErr := f.Truncate(0); truncateErr == nil {
					_, marshalErr = f.WriteAt(append(record, '\n'), 0)
				} else {
					marshalErr = truncateErr
				}
			}
			if marshalErr != nil {
				_ = unlock(f)
				return nil, time.Since(start), fmt.Errorf("record host slot holder: %w", marshalErr)
			}
			waited = time.Since(start)
			acquiredAt := time.Now()
			_, _ = fmt.Fprintf(out, "ci step %s host_slot=acquired waited=%s\n", step, waited.Round(time.Millisecond))
			heldFrom := acquiredAt
			return func() {
				_ = unlock(f)
				_ = f.Close()
				_, _ = fmt.Fprintf(out, "ci step %s host_slot=released held=%s\n", step, time.Since(heldFrom).Round(time.Millisecond))
			}, waited, nil
		}
		if !lockContended(err) {
			return nil, time.Since(start), fmt.Errorf("acquire host slot: %w", err)
		}
		current := readHostSlotMetadata(f)
		for _, frame := range ancestry {
			if current.RunID != "" && current.RunID == frame.RunID {
				return nil, time.Since(start), fmt.Errorf(
					"ci step %s host_slot: held by ancestor run %s (step %q); a composed child must not re-declare host_slot under a parent that holds it",
					step, current.RunID, frame.Step)
			}
		}
		announce(current)
		select {
		case <-ctx.Done():
			return nil, time.Since(start), ctx.Err()
		case <-time.After(hostSlotPollInterval):
		}
	}
}
