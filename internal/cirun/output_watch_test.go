//go:build darwin || linux || freebsd || netbsd || openbsd

package cirun

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestOutputWatchQuietEpisodes(t *testing.T) {
	var output bytes.Buffer
	w := newOutputWatch(&output, "sample", 3*time.Minute, time.Hour)
	w.start()
	base := w.started
	w.check(base.Add(3 * time.Minute))
	w.check(base.Add(4 * time.Minute))
	if n := strings.Count(output.String(), "ci output idle"); n != 1 {
		t.Fatalf("warnings = %d", n)
	}
	// Supervisor output bypasses the child-output writer and must not rearm it.
	_, _ = output.WriteString("ci progress heartbeat\n")
	w.check(base.Add(5 * time.Minute))
	if n := strings.Count(output.String(), "ci output idle"); n != 1 {
		t.Fatalf("heartbeat rearmed warning: %d", n)
	}
	_, _ = w.Write([]byte("child output"))
	w.check(time.Now().Add(4 * time.Minute))
	report := w.stop()
	if n := strings.Count(output.String(), "ci output idle"); n != 2 {
		t.Fatalf("warnings after output = %d", n)
	}
	if report.LastOutputAt == nil || !report.WarningEmitted {
		t.Fatalf("observation = %+v", report)
	}
}

func TestOutputWatchSilentAndNotMeasured(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, run  string
		wantStatus string
		wantOutput bool
	}{
		{"silent", "true", "succeeded", true},
		{"output", "printf hello", "succeeded", true},
		{"failure", "exit 7", "failed", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := runStep(context.Background(), root, config.CIStep{Name: tc.name, Run: tc.run}, nil, 1, "test", &bytes.Buffer{}, time.Now())
			if r.Status != tc.wantStatus || (err == nil) != (tc.wantStatus == "succeeded") {
				t.Fatalf("status=%s err=%v", r.Status, err)
			}
			if (r.OutputObservation != nil) != tc.wantOutput {
				t.Fatalf("observation=%+v", r.OutputObservation)
			}
			if tc.name == "silent" && r.OutputObservation.LastOutputAt != nil {
				t.Fatal("silent output was measured")
			}
			if tc.name == "output" && r.OutputObservation.LastOutputAt == nil {
				t.Fatal("output was not observed")
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, err := runStep(ctx, root, config.CIStep{Name: "cancel", Run: "true"}, nil, 1, "test", &bytes.Buffer{}, time.Now())
	if err == nil || r.OutputObservation != nil {
		t.Fatalf("cancel: %+v %v", r, err)
	}
}

func TestOutputObservationAbsentForSkippedReusedAndOCI(t *testing.T) {
	root, step := reuseFixture(t)
	state := t.TempDir()
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)}
	first, err := Run(context.Background(), opts)
	if err != nil || first.Steps[0].OutputObservation == nil {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := Run(context.Background(), opts)
	if err != nil || second.Steps[0].Reuse == nil || second.Steps[0].Reuse.Disposition != "reused" || second.Steps[0].OutputObservation != nil {
		t.Fatalf("reused=%+v err=%v", second, err)
	}
	step.Condition = &config.CICondition{Run: `printf '{"selected":false}'`, Timeout: "1s"}
	opts.Profile = profile(step)
	skipped, err := Run(context.Background(), opts)
	if err != nil || skipped.Steps[0].OutputObservation != nil {
		t.Fatalf("skipped=%+v err=%v", skipped, err)
	}
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	oci, _ := runStep(context.Background(), canonical, config.CIStep{Name: "oci", OCI: &config.CIOCI{}}, nil, 1, "test", &bytes.Buffer{}, time.Now())
	if oci.OutputObservation != nil {
		t.Fatalf("oci=%+v", oci)
	}
	var old StepReport
	if err := json.Unmarshal([]byte(`{"name":"legacy","status":"succeeded"}`), &old); err != nil || old.OutputObservation != nil {
		t.Fatalf("legacy=%+v err=%v", old, err)
	}
}

func TestOutputWarningDoesNotStopChild(t *testing.T) {
	var output bytes.Buffer
	w := newOutputWatch(&output, "busy", time.Minute, time.Hour)
	cmd := exec.Command("/bin/sh", "-c", "i=0; while [ $i -lt 50000 ]; do i=$((i+1)); done")
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	w.start()
	w.check(w.started.Add(2 * time.Minute))
	err := cmd.Wait()
	observation := w.stop()
	if err != nil || !observation.WarningEmitted {
		t.Fatalf("err=%v observation=%+v", err, observation)
	}
}

func TestOutputWatchRealQuietAndOutputPreservesExit(t *testing.T) {
	for _, tc := range []struct {
		name, command string
		exit          int
		hasOutput     bool
	}{
		{"quiet", "sleep 0.15; exit 7", 7, false},
		{"output", "printf child; sleep 0.15; exit 0", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			w := newOutputWatch(&lockedWriter{w: &output}, tc.name, 30*time.Millisecond, 10*time.Millisecond)
			cmd := exec.Command("/bin/sh", "-c", tc.command)
			cmd.Stdout, cmd.Stderr = w, w
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			w.start()
			err := cmd.Wait()
			observation := w.stop()
			if cmd.ProcessState.ExitCode() != tc.exit || (err == nil) != (tc.exit == 0) || !observation.WarningEmitted || (observation.LastOutputAt != nil) != tc.hasOutput {
				t.Fatalf("exit=%d err=%v observation=%+v", cmd.ProcessState.ExitCode(), err, observation)
			}
		})
	}
}

func TestOutputObservationForCanceledAndChildCI(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	r, err := runStep(ctx, root, config.CIStep{Name: "cancel-running", Run: "sleep 2"}, nil, 1, "test", &bytes.Buffer{}, time.Now())
	if err == nil || r.Status != "timed_out" || r.OutputObservation == nil {
		t.Fatalf("running cancellation: %+v err=%v", r, err)
	}
	child := compositionProcessRoot(t, profile(config.CIStep{Name: "leaf", Run: "true"}))
	parent := compositionProcessRoot(t, compositionProcessProfile(compositionProcessCommand(child), "independent-root"))
	result, err := runCompositionProcessParent(t, parent, t.TempDir(), compositionProcessProfile(compositionProcessCommand(child), "independent-root"))
	if err != nil || result.Steps[0].OutputObservation != nil {
		t.Fatalf("child ci: %+v err=%v", result, err)
	}
}
