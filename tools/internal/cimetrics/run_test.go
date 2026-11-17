//go:build darwin || linux || freebsd || netbsd || openbsd

package cimetrics

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPreserveCancellation(t *testing.T) {
	childFailure := errors.New("child failed")
	for _, tc := range []struct {
		name                                       string
		runErr, cancellation                       error
		wantCancellation, wantWaitDelay, wantChild bool
	}{
		{"cancel after success", nil, context.Canceled, true, false, false},
		{"cancel with pipe wait", exec.ErrWaitDelay, context.Canceled, true, true, false},
		{"deadline with pipe wait", exec.ErrWaitDelay, context.DeadlineExceeded, true, true, false},
		{"child failure unchanged", childFailure, context.Canceled, false, false, true},
		{"completed pipe failure unchanged", exec.ErrWaitDelay, nil, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := preserveCancellation(tc.runErr, tc.cancellation)
			if errors.Is(got, tc.cancellation) != tc.wantCancellation || errors.Is(got, exec.ErrWaitDelay) != tc.wantWaitDelay || errors.Is(got, childFailure) != tc.wantChild {
				t.Fatalf("preserveCancellation(%v, %v) = %v", tc.runErr, tc.cancellation, got)
			}
		})
	}
}

func awaitRecord(t *testing.T, state string) (Record, []byte) {
	t.Helper()
	root, ok := findProjectRoot()
	if !ok {
		t.Fatal("project root not found")
	}
	key, ok := repositoryKey(root)
	if !ok {
		t.Fatal("repository identity unavailable")
	}
	var names []string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		names, _ = filepath.Glob(filepath.Join(state, key, "*.json"))
		if len(names) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if len(names) != 1 {
		var observed []string
		_ = filepath.WalkDir(state, func(path string, entry os.DirEntry, err error) error {
			if err == nil {
				observed = append(observed, path)
			}
			return err
		})
		t.Fatalf("expected one record, got %v; state entries: %v", names, observed)
	}
	b, err := os.ReadFile(names[0])
	if err != nil {
		t.Fatal(err)
	}
	var rec Record
	if err := json.Unmarshal(b, &rec); err != nil {
		t.Fatal(err)
	}
	return rec, b
}

func TestMetricsRunPreservesCheckFailureWhenStorageFails(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", link)
	var out, stderr bytes.Buffer
	err := RunWithStdin(context.Background(), "unit", []string{"/bin/sh", "-c", `printf 'live output'; printf 'child error' >&2; exit 42`}, nil, &out, &stderr)
	var child *exec.ExitError
	if !errors.As(err, &child) || child.ExitCode() != 42 {
		t.Fatalf("child exit lost: %v", err)
	}
	if out.String() != "live output" || !strings.Contains(stderr.String(), "child error") {
		t.Fatalf("output lost: %q / %q", out.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "collection partial") {
		t.Fatalf("storage failure hidden: %q", stderr.String())
	}
	if entries, err := os.ReadDir(target); err != nil || len(entries) != 0 {
		t.Fatalf("wrote through state symlink: %v %v", entries, err)
	}
}

func TestMetricsMalformedProgressDoesNotSuppressOutputOrLeakSecrets(t *testing.T) {
	state := t.TempDir()
	if err := os.Chmod(state, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", state)
	secret := "sensitive-example-for-fixture"
	line := fmt.Sprintf(`{"Action":"run","Package":%q,"Test":%q}`, module+"/tools/internal/cimetrics", "TestMetricsMalformedProgressDoesNotSuppressOutputOrLeakSecrets/"+secret)
	output := fmt.Sprintf(`{"Action":"output","Package":%q,"Output":%q}`, module+"/tools/internal/cimetrics", secret)
	args := []string{"/bin/sh", "-c", `printf '%s\n' "$1" "$2" "$3"`, "sh", "not-json", line, output, "-json"}
	var out, stderr bytes.Buffer
	if err := RunWithStdin(context.Background(), "unit", args, nil, &out, &stderr); err != nil {
		t.Fatalf("successful check turned failure: %v", err)
	}
	if got := out.String(); got != "" || strings.Contains(got, secret) || strings.Contains(got, "not-json") {
		t.Fatalf("raw, malformed or unfinished JSON output escaped: %q", got)
	}
	if !strings.Contains(stderr.String(), "collection partial") {
		t.Fatalf("malformed input not reported: %q", stderr.String())
	}
	rec, b := awaitRecord(t, state)
	if rec.Outcome != "succeeded" || rec.Status != "partial" || !rec.OutputDegraded {
		t.Fatalf("result/coverage conflated: %+v", rec)
	}
	if strings.Contains(string(b), secret) || strings.Contains(string(b), state) {
		t.Fatalf("sensitive input appeared in summary: %s", b)
	}
	paths, err := filepath.Glob(filepath.Join(state, "*", "*.events.jsonl"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("events: %v %v", paths, err)
	}
	progress, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(progress), secret) || strings.Contains(string(progress), state) {
		t.Fatalf("sensitive input appeared in progress: %s", progress)
	}
}

func TestMetricsRunDoesNotSuppressOriginalOutputFailure(t *testing.T) {
	state := t.TempDir()
	if err := os.Chmod(state, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", state)
	var stderr bytes.Buffer
	err := RunWithStdin(context.Background(), "unit", []string{"/bin/sh", "-c", "printf hello"}, nil, failOutput{}, &stderr)
	if err == nil || !strings.Contains(err.Error(), "fixture output failure") {
		t.Fatalf("original stdout error hidden: %v", err)
	}
}

func TestMetricsRunWithoutJSONMarksTestCoveragePartial(t *testing.T) {
	state := t.TempDir()
	if err := os.Chmod(state, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", state)
	var out, stderr bytes.Buffer
	if err := RunWithStdin(context.Background(), "unit", []string{"/bin/sh", "-c", "printf plain-output"}, nil, &out, &stderr); err != nil {
		t.Fatal(err)
	}
	if out.String() != "plain-output" {
		t.Fatalf("non-JSON output changed: %q", out.String())
	}
	rec, _ := awaitRecord(t, state)
	if rec.Outcome != "succeeded" || rec.Status != "partial" || rec.Reason != "structured test events unavailable" || rec.OutputDegraded {
		t.Fatalf("missing structured events misclassified: %+v", rec)
	}
	if !strings.Contains(stderr.String(), "collection partial") {
		t.Fatalf("missing structured events not reported: %q", stderr.String())
	}
}

func TestMetricsActualGoFixtureHidesPassingLogOnPackageFailure(t *testing.T) {
	root, ok := findProjectRoot()
	if !ok {
		t.Fatal("project root not found")
	}
	tmp := filepath.Join(root, "tmp")
	if err := os.MkdirAll(tmp, 0o700); err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(tmp, "cimetrics-output-fixture-")
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(dir, "fixture_test.go")
	t.Cleanup(func() {
		entries, e := os.ReadDir(dir)
		if e != nil {
			t.Errorf("fixture inspection: %v", e)
			return
		}
		if len(entries) != 1 || entries[0].Name() != "fixture_test.go" {
			t.Errorf("unexpected fixture files; preserved: %v", entries)
			return
		}
		if e = os.Remove(file); e != nil {
			t.Errorf("fixture file cleanup: %v", e)
			return
		}
		if e = os.Remove(dir); e != nil {
			t.Errorf("fixture directory cleanup: %v", e)
		}
	})
	source := `package fixture
import "testing"
func TestPass(t *testing.T) { t.Log("passing-fixture-secret") }
func TestFail(t *testing.T) { t.Error("expected-fixture-diagnostic") }
`
	f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(f, source); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	state := t.TempDir()
	if err := os.Chmod(state, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_PROJECT_CI_METRICS_DIR", state)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOTOOLCHAIN", "local")
	t.Setenv("GOPROXY", "off")
	t.Setenv("GOSUMDB", "off")
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var out, stderr bytes.Buffer
	err = RunWithStdin(ctx, "unit", []string{"go", "test", "-json", "-count=1", "-timeout=15s", dir}, nil, &out, &stderr)
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() == 0 {
		t.Fatalf("fixture must fail: %v; %s", err, stderr.String())
	}
	if strings.Contains(out.String(), "passing-fixture-secret") || !strings.Contains(out.String(), "expected-fixture-diagnostic") {
		t.Fatalf("Go fixture output failed privacy/diagnostics contract: %q", out.String())
	}
	rec, b := awaitRecord(t, state)
	if rec.Outcome != "failed" || strings.Contains(string(b), "passing-fixture-secret") {
		t.Fatalf("fixture summary lost result or leaked: %+v %s", rec, b)
	}
	paths, err := filepath.Glob(filepath.Join(state, "*", "*.events.jsonl"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("progress files: %v %v", paths, err)
	}
	progress, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(progress, []byte("passing-fixture-secret")) {
		t.Fatal("progress leaked passing fixture output")
	}
}
