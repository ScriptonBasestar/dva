package cirun

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func compositionTestDir(t *testing.T) string {
	t.Helper()
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func compositionTestJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func compositionTestGrant(t *testing.T) (compositionGrant, string, time.Time) {
	t.Helper()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return compositionGrant{
		Version: 1, JobsLimit: 2, LockState: compositionTestDir(t),
		Deadline: now.Add(time.Minute).Format(time.RFC3339Nano),
		Ancestry: []compositionFrame{
			{RunID: strings.Repeat("a", 32), Root: compositionTestDir(t), Step: "outer"},
			{RunID: strings.Repeat("b", 32), Root: compositionTestDir(t), Step: "inner"},
		},
	}, compositionTestDir(t), now
}

func compositionTestParent(t *testing.T, grant compositionGrant) string {
	t.Helper()
	last := grant.Ancestry[len(grant.Ancestry)-1]
	return compositionTestJSON(t, parentRun{ID: last.RunID, Root: last.Root})
}

func TestCompositionAdmissionChecksEveryAncestor(t *testing.T) {
	grant, child, now := compositionTestGrant(t)
	var visited []string
	probe := func(state, kind, root string) (string, error) {
		if state != grant.LockState || kind != "root" {
			t.Fatalf("incorrect lock identity %s %s", state, kind)
		}
		visited = append(visited, root)
		for _, frame := range grant.Ancestry {
			if frame.Root == root {
				return frame.RunID, nil
			}
		}
		return "", errors.New("unexpected ancestor")
	}
	admitted, err := admitComposition(grant.LockState, child, compositionTestParent(t, grant), compositionTestJSON(t, grant), now, probe)
	if err != nil || admitted.JobsLimit != 2 || len(visited) != 2 {
		t.Fatalf("admission=%#v err=%v visited=%v", admitted, err, visited)
	}
	for _, staleIndex := range []int{0, 1} {
		probe := func(_, _, root string) (string, error) {
			for index, frame := range grant.Ancestry {
				if frame.Root == root {
					if index == staleIndex {
						return "", nil
					}
					return frame.RunID, nil
				}
			}
			return "", nil
		}
		_, err := admitComposition(grant.LockState, child, compositionTestParent(t, grant), compositionTestJSON(t, grant), now, probe)
		if err == nil || !strings.Contains(err.Error(), "ancestor") {
			t.Fatalf("stale frame %d accepted: %v", staleIndex, err)
		}
	}
	_, err = admitComposition(grant.LockState, child, compositionTestParent(t, grant), compositionTestJSON(t, grant), now,
		func(_, _, _ string) (string, error) { return "", errors.New("lookup failure") })
	if err == nil || !strings.Contains(err.Error(), "lookup failure") {
		t.Fatalf("lookup failure was hidden: %v", err)
	}
}

func TestCompositionAdmissionRejectsCyclesAndInvalidMetadata(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*compositionGrant, *string)
	}{
		{"version", func(g *compositionGrant, _ *string) { g.Version = 2 }},
		{"empty ancestry", func(g *compositionGrant, _ *string) { g.Ancestry = nil }},
		{"depth", func(g *compositionGrant, _ *string) { g.Ancestry = make([]compositionFrame, 9) }},
		{"jobs", func(g *compositionGrant, _ *string) { g.JobsLimit = 0 }},
		{"worker default key", func(g *compositionGrant, _ *string) { g.WorkerDefaults = map[string]string{"OTHER": "1"} }},
		{"worker default format", func(g *compositionGrant, _ *string) { g.WorkerDefaults = map[string]string{"CARGO_BUILD_JOBS": "+1"} }},
		{"worker default excess", func(g *compositionGrant, _ *string) { g.WorkerDefaults = map[string]string{"RUST_TEST_THREADS": "9"} }},
		{"expired", func(g *compositionGrant, _ *string) { g.Deadline = "2025-01-01T00:00:00Z" }},
		{"nonUTC", func(g *compositionGrant, _ *string) { g.Deadline = "2026-01-01T00:01:00+00:00" }},
		{"state mismatch", func(g *compositionGrant, _ *string) { g.LockState += "/other" }},
		{"run ID", func(g *compositionGrant, _ *string) { g.Ancestry[0].RunID = "bad" }},
		{"step", func(g *compositionGrant, _ *string) { g.Ancestry[0].Step = "../outside" }},
		{"root", func(g *compositionGrant, _ *string) { g.Ancestry[0].Root = "." }},
		{"duplicate root", func(g *compositionGrant, _ *string) { g.Ancestry[0].Root = g.Ancestry[1].Root }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			grant, child, now := compositionTestGrant(t)
			tc.edit(&grant, &child)
			parent := `{"id":"missing","root":"/"}`
			if len(grant.Ancestry) > 0 {
				parent = compositionTestParent(t, grant)
			}
			_, err := admitComposition(grant.LockState, child, parent, compositionTestJSON(t, grant), now,
				func(_, _, _ string) (string, error) { t.Fatal("invalid metadata reached lock probe"); return "", nil })
			if err == nil {
				t.Fatal("invalid grant accepted")
			}
			if strings.Contains(tc.name, "cycle") {
				var busy *BusyError
				if !errors.As(err, &busy) || busy.Conflict.Kind != "composition_cycle" {
					t.Fatalf("cycle lacks structured conflict: %v", err)
				}
			}
		})
	}
}

func TestCompositionCycleRequiresVerifiedOwners(t *testing.T) {
	for _, index := range []int{0, 1} {
		grant, _, now := compositionTestGrant(t)
		for _, live := range []bool{false, true} {
			visited := 0
			probe := func(_, _, root string) (string, error) {
				visited++
				if !live {
					return "", nil
				}
				for _, f := range grant.Ancestry {
					if f.Root == root {
						return f.RunID, nil
					}
				}
				return "", nil
			}
			_, err := admitComposition(grant.LockState, grant.Ancestry[index].Root, compositionTestParent(t, grant), compositionTestJSON(t, grant), now, probe)
			var busy *BusyError
			if live {
				if !errors.As(err, &busy) || busy.Conflict.Kind != "composition_cycle" || visited != len(grant.Ancestry) {
					t.Fatalf("unverified cycle: %v probes=%d", err, visited)
				}
			} else if err == nil || errors.As(err, &busy) {
				t.Fatalf("stale token published as active owner: %v", err)
			}
		}
	}
}

func TestCompositionAdmissionRejectsAmbiguousJSONAndParentMismatch(t *testing.T) {
	grant, child, now := compositionTestGrant(t)
	valid := compositionTestJSON(t, grant)
	for _, value := range []string{
		valid + "{}", valid[:len(valid)-1] + `,"unknown":1}`,
		valid[:len(valid)-1] + `,"version":1}`, "null", "[]", strings.Repeat(" ", 65537),
	} {
		if _, err := admitComposition(grant.LockState, child, compositionTestParent(t, grant), value, now, nil); err == nil {
			t.Fatalf("ambiguous JSON accepted: %.80s", value)
		}
	}
	if _, err := admitComposition(grant.LockState, child, `{"id":"wrong","root":"/"}`, valid, now, nil); err == nil {
		t.Fatal("mismatched immediate parent accepted")
	}
	if _, err := admitComposition(compositionTestDir(t), child, compositionTestParent(t, grant), valid, now, nil); err == nil || !strings.Contains(err.Error(), "stores differ") {
		t.Fatalf("changing the actual lock store was accepted: %v", err)
	}
}

func TestCompositionAdmissionRejectsRootAlias(t *testing.T) {
	grant, child, now := compositionTestGrant(t)
	alias := filepath.Join(compositionTestDir(t), "alias")
	if err := os.Symlink(grant.Ancestry[0].Root, alias); err != nil {
		t.Skipf("symlink fixture unavailable: %v", err)
	}
	grant.Ancestry[0].Root = alias
	if _, err := admitComposition(grant.LockState, child, compositionTestParent(t, grant), compositionTestJSON(t, grant), now, nil); err == nil || !strings.Contains(err.Error(), "canonical") {
		t.Fatalf("noncanonical root alias accepted: %v", err)
	}
}

func TestCompositionAdmissionUsesActualOSLock(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux", "freebsd", "netbsd", "openbsd":
	default:
		t.Skip("native CI locks are unsupported on this platform")
	}
	grant, child, now := compositionTestGrant(t)
	grant.Ancestry = grant.Ancestry[:1]
	frame := grant.Ancestry[0]
	h, err := acquire(grant.LockState, frame.Root, frame.RunID)
	if err != nil {
		t.Fatal(err)
	}
	released := false
	t.Cleanup(func() {
		if !released {
			h.release()
		}
	})
	if _, err := admitComposition(grant.LockState, child, compositionTestParent(t, grant), compositionTestJSON(t, grant), now, nil); err != nil {
		t.Fatal(err)
	}
	h.release()
	released = true
	if _, err := admitComposition(grant.LockState, child, compositionTestParent(t, grant), compositionTestJSON(t, grant), now, nil); err == nil {
		t.Fatal("stale lock metadata established liveness")
	}
}
