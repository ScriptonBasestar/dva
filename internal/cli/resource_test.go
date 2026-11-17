package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestResourceRunPreservesChildExitAndReport(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native resource admission not available")
	}
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	old := resourceStateDirectory
	resourceStateDirectory = func() (string, error) { return filepath.Join(dir, "state"), nil }
	t.Cleanup(func() { resourceStateDirectory = old })
	policy := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(policy, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	report := filepath.Join(dir, "report.json")
	cmd := newResourceCommand()
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"run", "--policy", policy, "--class", "light", "--execution-timeout", "2s", "--report-file", report, "--", "/bin/sh", "-c", "printf '{\"ok\":true}'; exit 17"})
	err := cmd.Execute()
	var child *resourceChildExit
	if !errors.As(err, &child) || child.code != 17 {
		t.Fatalf("unexpected error %v", err)
	}
	if out.String() != `{"ok":true}` {
		t.Fatalf("stdout %q", out.String())
	}
	b, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var r resourceReport
	if err = json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	if r.Schema != "dva-resource-run/1" || r.ExitCode == nil || *r.ExitCode != 17 || r.ExecutionDuration == nil || r.WaitDuration == nil || r.ClaimID == "" {
		t.Fatalf("report %+v", r)
	}
}

func TestResourceRunTimeoutAndSignal(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native resource admission not available")
	}
	for _, tc := range []struct {
		name string
		args []string
		code int
	}{
		{name: "timeout", args: []string{"/bin/sleep", "1"}, code: 124},
		{name: "signal", args: []string{"/bin/sh", "-c", "kill -TERM $$"}, code: 143},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DVA_RESOURCE_SUPERVISED", "")
			t.Setenv("DVA_CI_PARENT_RUN", "")
			dir := t.TempDir()
			if err := os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			old := resourceStateDirectory
			resourceStateDirectory = func() (string, error) { return filepath.Join(dir, "state"), nil }
			t.Cleanup(func() { resourceStateDirectory = old })
			policy := filepath.Join(dir, "policy.json")
			if err := os.WriteFile(policy, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := newResourceCommand()
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			cmd.SetArgs(append([]string{"run", "--policy", policy, "--class", "light", "--execution-timeout", "100ms", "--"}, tc.args...))
			err := cmd.Execute()
			var child *resourceChildExit
			if !errors.As(err, &child) || child.code != tc.code {
				t.Fatalf("exit=%v stderr=%s", err, stderr.String())
			}
		})
	}
}

func TestResourceRunDoesNotSucceedWithLiveGroup(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native resource admission not available")
	}
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	old := resourceStateDirectory
	resourceStateDirectory = func() (string, error) { return filepath.Join(dir, "state"), nil }
	t.Cleanup(func() { resourceStateDirectory = old })
	policy := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(policy, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer devnull.Close()
	report := filepath.Join(dir, "report.json")
	cmd := newResourceCommand()
	cmd.SetOut(devnull)
	cmd.SetErr(devnull)
	cmd.SetArgs([]string{"run", "--policy", policy, "--class", "light", "--execution-timeout", "2s", "--report-file", report, "--", "/bin/sh", "-c", "sleep 1 & exit 0"})
	err = cmd.Execute()
	if err == nil {
		t.Fatal("live process group falsely succeeded")
	}
	b, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var r resourceReport
	if err = json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	if r.Outcome != "quarantined" || r.ClaimID == "" || r.ExitCode == nil || *r.ExitCode != 0 {
		t.Fatalf("bad report %+v", r)
	}
}

func TestResourceReportWriteFailurePreservesChildResult(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native resource admission not available")
	}
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	for _, code := range []int{0, 17} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			dir := t.TempDir()
			if err := os.Chmod(dir, 0700); err != nil {
				t.Fatal(err)
			}
			oldDir := resourceStateDirectory
			resourceStateDirectory = func() (string, error) { return filepath.Join(dir, "state"), nil }
			t.Cleanup(func() { resourceStateDirectory = oldDir })
			oldSync := resourceReportSync
			resourceReportSync = func(*os.File) error { return errors.New("report write failed") }
			t.Cleanup(func() { resourceReportSync = oldSync })
			policy := filepath.Join(dir, "policy.json")
			if err := os.WriteFile(policy, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := newResourceCommand()
			var out, stderr bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			cmd.SetArgs([]string{"run", "--policy", policy, "--class", "light", "--execution-timeout", "2s", "--report-file", filepath.Join(dir, "report"), "--", "/bin/sh", "-c", fmt.Sprintf("exit %d", code)})
			err := cmd.Execute()
			if code == 0 && err != nil {
				t.Fatalf("success changed: %v", err)
			}
			if code != 0 {
				var child *resourceChildExit
				if !errors.As(err, &child) || child.code != code {
					t.Fatalf("child exit changed: %v", err)
				}
			}
			if !strings.Contains(stderr.String(), "resource report failed") {
				t.Fatalf("missing diagnostic: %s", stderr.String())
			}
		})
	}
}

func TestResourceRunTimeoutStopsOwnedChildGroup(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("native resource admission not available")
	}
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	old := resourceStateDirectory
	resourceStateDirectory = func() (string, error) { return filepath.Join(dir, "state"), nil }
	t.Cleanup(func() { resourceStateDirectory = old })
	policy := filepath.Join(dir, "policy.json")
	if err := os.WriteFile(policy, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := newResourceCommand()
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"run", "--policy", policy, "--class", "light", "--execution-timeout", "100ms", "--", "/bin/sh", "-c", "sleep 20 & wait"})
	start := time.Now()
	err := cmd.Execute()
	var child *resourceChildExit
	if !errors.As(err, &child) || child.code != 124 {
		t.Fatalf("timeout exit=%v stderr=%s", err, stderr.String())
	}
	if time.Since(start) > 4*time.Second {
		t.Fatalf("timeout cleanup was not bounded: %s", time.Since(start))
	}
}
