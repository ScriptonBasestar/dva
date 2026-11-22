package cirun

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestObservabilityUsesPrivateDirectoryAndCanonicalEnvironment(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	p := profile(config.CIStep{Name: "one", Run: `test "$DVA_CI_RUN_ID" != forged && test -d "$DVA_CI_EVENTS_DIR"`, Environment: map[string]string{"DVA_CI_RUN_ID": "forged", "DVA_CI_EVENTS_DIR": "/forged"}})
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "fast", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: p})
	if err != nil {
		t.Fatal(err)
	}
	if r.Observability == nil || r.Observability.EventsStatus != "available" || r.Observability.CacheCondition != "unknown" || r.Steps[0].Workdir != r.Root {
		t.Fatalf("report: %+v", r)
	}
	if r.Observability.ExecutorVersion != config.Version || r.Observability.ExecutorRevision != config.ResolvedCommit().Value {
		t.Fatalf("executor identity: %+v", r.Observability)
	}
	info, err := os.Stat(r.Observability.EventsDir)
	if err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("directory: %v %v", info, err)
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err = json.Unmarshal(b, &decoded); err != nil || decoded.Observability.EventsDir != r.Observability.EventsDir {
		t.Fatalf("roundtrip: %v %+v", err, decoded)
	}
	if strings.Contains(string(b), "forged") {
		t.Fatal("command/environment leaked")
	}
}

func TestSnapshotFailureIsDiagnosticOnly(t *testing.T) {
	ctx := context.WithValue(context.Background(), progressIntervalKey{}, 5*time.Millisecond)
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	observed := 0
	reports, err := execute(ctx, root, profile(config.CIStep{Name: "slow", Run: "sleep 0.1"}), os.Environ(), &lockedWriter{w: &output},
		func([]StepReport) error { return nil },
		func([]StepReport) error { observed++; return errors.New("snapshot unavailable") })
	if err != nil || len(reports) != 1 || reports[0].Status != "succeeded" || observed == 0 || !strings.Contains(output.String(), "ci native snapshot warning: snapshot unavailable") {
		t.Fatalf("reports=%+v calls=%d err=%v output=%s", reports, observed, err, output.String())
	}
}

func TestProgressStopsWhenExecutionCanceled(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), progressIntervalKey{}, 5*time.Millisecond), 80*time.Millisecond)
	defer cancel()
	var b bytes.Buffer
	w := &lockedWriter{w: &b}
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = execute(ctx, root, profile(config.CIStep{Name: "slow", Run: "sleep 1"}), os.Environ(), w, func([]StepReport) error { return nil })
	if err == nil || !strings.Contains(b.String(), "ci progress completed=0/1 running=1 pending=0") {
		t.Fatalf("err=%v output=%s", err, b.String())
	}
}

func TestObservabilityFailureDoesNotChangeCheckVerdict(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(state, "events"), []byte("occupied"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := Run(context.Background(), Options{Root: root, ProfileName: "fast", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(config.CIStep{Name: "one", Run: "true"})})
	if err != nil || r.Status != "succeeded" || r.Observability.EventsStatus != "degraded" || r.Observability.EventsDir != "" {
		t.Fatalf("report=%+v err=%v", r, err)
	}
}

func TestOCIWorkdirIsNotReportedAsHostDirectory(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r, _ := runStep(context.Background(), root, config.CIStep{Name: "oci", OCI: &config.CIOCI{}}, nil, 1, "test", &bytes.Buffer{}, time.Now())
	if r.Workdir != "" {
		t.Fatalf("OCI host directory mislabeled: %q", r.Workdir)
	}
}
