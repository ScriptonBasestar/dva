package cirun

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

func TestConditionalAdmissionPresence(t *testing.T) {
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	t.Setenv("DVA_RESOURCE_POLICY", "")
	if err := os.Unsetenv("DVA_RESOURCE_POLICY"); err != nil {
		t.Fatal(err)
	}
	root, state := t.TempDir(), t.TempDir()
	p := config.CIProfile{Timeout: "5s", MaxParallel: 1, Admission: &config.CIAdmission{Mode: "when-policy-configured", Class: "heavy", Priority: "interactive", WaitTimeout: "1s"}, Steps: []config.CIStep{{Name: "one", Run: `test -z "${DVA_RESOURCE_SUPERVISED:-}" && touch ran`}}}
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Profile: p}
	report, err := Run(context.Background(), opts)
	if err != nil || report.Admission == nil || report.Admission.Outcome != "not_configured" || report.Admission.WaitDuration != nil || report.Admission.ExecutionDuration != nil || report.Admission.ClaimID != "" || report.Admission.PolicyFingerprint != "" {
		t.Fatalf("report=%+v err=%v", report.Admission, err)
	}
	if _, err := os.Stat(filepath.Join(root, "ran")); err != nil {
		t.Fatal(err)
	}
	receipts, err := Status(state)
	if err != nil || len(receipts) != 1 || receipts[0].Admission == nil || receipts[0].Admission.Outcome != "not_configured" {
		t.Fatalf("receipts=%+v err=%v", receipts, err)
	}
}

func TestAbsentPolicyRequired(t *testing.T) {
	t.Setenv("DVA_RESOURCE_POLICY", "")
	if err := os.Unsetenv("DVA_RESOURCE_POLICY"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_CI_PARENT_RUN", "")
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	root, state := t.TempDir(), t.TempDir()
	p := config.CIProfile{Timeout: "5s", MaxParallel: 1, Admission: &config.CIAdmission{Class: "heavy", Priority: "normal", WaitTimeout: "1s"}, Steps: []config.CIStep{{Name: "one", Run: "touch ran"}}}
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Profile: p}
	report, err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "no resource policy") || report.ID == "" || report.Status != "failed" || report.Admission == nil || report.Admission.Outcome != "failed" {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	receipts, err := Status(state)
	if err != nil || len(receipts) != 1 || receipts[0].ID != report.ID || receipts[0].Status != "failed" {
		t.Fatalf("receipts=%+v err=%v", receipts, err)
	}
	if _, err := os.Stat(filepath.Join(root, "ran")); !os.IsNotExist(err) {
		t.Fatalf("step ran: %v", err)
	}
}

func TestEmptyPolicyRequired(t *testing.T) {
	t.Setenv("DVA_RESOURCE_POLICY", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	root, state := t.TempDir(), t.TempDir()
	p := config.CIProfile{Timeout: "5s", MaxParallel: 1, Admission: &config.CIAdmission{Class: "heavy", Priority: "normal", WaitTimeout: "1s"}, Steps: []config.CIStep{{Name: "one", Run: "touch ran"}}}
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Profile: p}
	report, err := Run(context.Background(), opts)
	if err == nil || !strings.Contains(err.Error(), "explicitly empty: set it to an absolute policy path") || report.ID == "" || report.Status != "failed" || report.Admission == nil || report.Admission.Outcome != "failed" {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	receipts, err := Status(state)
	if err != nil || len(receipts) != 1 || receipts[0].ID != report.ID || receipts[0].Status != "failed" {
		t.Fatalf("receipts=%+v err=%v", receipts, err)
	}
	if _, err := os.Stat(filepath.Join(root, "ran")); !os.IsNotExist(err) {
		t.Fatalf("step ran: %v", err)
	}
}

func TestAbsentPolicySupervisedConditional(t *testing.T) {
	t.Setenv("DVA_RESOURCE_POLICY", "")
	if err := os.Unsetenv("DVA_RESOURCE_POLICY"); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_CI_PARENT_RUN", "")
	t.Setenv("DVA_RESOURCE_SUPERVISED", "1")
	root, state := t.TempDir(), t.TempDir()
	p := config.CIProfile{Timeout: "5s", MaxParallel: 1, Admission: &config.CIAdmission{Mode: "when-policy-configured", Class: "heavy", Priority: "normal", WaitTimeout: "1s"}, Steps: []config.CIStep{{Name: "one", Run: "true"}}}
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Profile: p}
	report, err := Run(context.Background(), opts)
	if err != nil || report.Admission == nil || report.Admission.Outcome != "not_configured" {
		t.Fatalf("report=%+v err=%v", report.Admission, err)
	}
}

func TestAdmissionInvalidPolicyNeverRuns(t *testing.T) {
	t.Setenv("DVA_RESOURCE_SUPERVISED", "")
	t.Setenv("DVA_CI_PARENT_RUN", "")
	root, state := t.TempDir(), t.TempDir()
	malformed := filepath.Join(root, "bad.json")
	if err := os.WriteFile(malformed, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	valid := filepath.Join(root, "valid.json")
	if err := os.WriteFile(valid, []byte(`{"capacity":4,"max_running":3,"max_heavy":1,"weights":{"heavy":2,"light":1},"aging_interval":"5m"}`), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DVA_RESOURCE_POLICY", valid)
	for _, mode := range []string{"", "required", "when-policy-configured"} {
		for _, env := range [][]string{{"DVA_RESOURCE_POLICY="}, {"DVA_RESOURCE_POLICY=relative"}, {"DVA_RESOURCE_POLICY=/missing-policy"}, {"DVA_RESOURCE_POLICY=" + malformed}, {"DVA_RESOURCE_POLICY=" + malformed, "DVA_RESOURCE_POLICY="}} {
			p := config.CIProfile{Timeout: "5s", MaxParallel: 1, Admission: &config.CIAdmission{Mode: mode, Class: "heavy", Priority: "normal", WaitTimeout: "1s"}, Steps: []config.CIStep{{Name: "one", Run: "touch ran"}}}
			report, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, Env: env, Profile: p})
			if err == nil {
				t.Fatalf("mode=%q env=%v accepted", mode, env)
			}
			if report.ID == "" || report.Status != "failed" || report.Admission == nil || report.Admission.Outcome != "failed" {
				t.Fatalf("missing failed attempt: %+v", report)
			}
			if _, err := os.Stat(filepath.Join(root, "ran")); !os.IsNotExist(err) {
				t.Fatalf("step ran: %v", err)
			}
		}
	}
}

func TestConditionalAdmissionWaitCancelAndRelease(t *testing.T) {
	root, state, resource, policy, digest := bridgeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	held, err := resourceadmission.Acquire(ctx, resource, policy, digest, "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	p := config.CIProfile{Timeout: "5s", MaxParallel: 1, Admission: &config.CIAdmission{Mode: "when-policy-configured", Class: "heavy", Priority: "normal", WaitTimeout: "2s"}, Steps: []config.CIStep{{Name: "one", Run: "true"}}}
	opts := Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: p}
	waitCtx, stop := context.WithCancel(context.Background())
	done := make(chan Report, 1)
	go func() { r, _ := Run(waitCtx, opts); done <- r }()
	deadline := time.Now().Add(time.Second)
	for {
		files, _ := filepath.Glob(filepath.Join(state, "*.json"))
		if len(files) > 0 {
			break
		}
		if time.Now().After(deadline) {
			stop()
			t.Fatal("no waiting receipt")
		}
		time.Sleep(10 * time.Millisecond)
	}
	stop()
	r := <-done
	if r.Admission == nil || r.Admission.Outcome != "canceled" || r.Admission.WaitDuration == nil {
		t.Fatalf("cancellation: %+v", r)
	}
	if err := held.Complete(ctx); err != nil {
		t.Fatal(err)
	}
	r, err = Run(ctx, opts)
	if err != nil || r.Admission == nil || r.Admission.Outcome != "completed" {
		t.Fatalf("execution: %+v %v", r, err)
	}
	entries, err := resourceadmission.Status(ctx, resource, digest)
	if err != nil || len(entries) != 0 {
		t.Fatalf("lease remains: %+v %v", entries, err)
	}
}
