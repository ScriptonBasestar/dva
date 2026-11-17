package cirun

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
	"github.com/ScriptonBasestar/dva/internal/resourceadmission"
)

func TestAdmissionWaitingReceiptAndCancellation(t *testing.T) {
	root, state, resource, p, digest := bridgeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	held, err := resourceadmission.Acquire(ctx, resource, p, digest, "heavy", "normal")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = held.Complete(context.Background()) }()
	runCtx, stop := context.WithCancel(context.Background())
	done := make(chan Report, 1)
	go func() {
		r, _ := Run(runCtx, Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: config.CIProfile{Timeout: "1s", MaxParallel: 1, Admission: &config.CIAdmission{Class: "heavy", Priority: "normal", WaitTimeout: "2s"}, Steps: []config.CIStep{{Name: "x", Run: "true"}}}})
		done <- r
	}()
	var path string
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		files, _ := filepath.Glob(filepath.Join(state, "*.json"))
		if len(files) == 1 {
			path = files[0]
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if path == "" {
		stop()
		t.Fatal("no waiting receipt")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err = json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["status"] != "waiting" || wire["duration"] != nil {
		t.Fatalf("waiting=%s", raw)
	}
	stop()
	select {
	case got := <-done:
		if got.ID == "" || got.Status != "canceled" || got.Admission == nil || got.Admission.Outcome != "canceled" || got.Admission.ExecutionDuration != nil {
			t.Fatalf("got=%+v", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancel blocked")
	}
	raw, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var terminal Report
	if err = json.Unmarshal(raw, &terminal); err != nil {
		t.Fatal(err)
	}
	if terminal.Status != "canceled" || terminal.Admission.WaitDuration == nil {
		t.Fatalf("terminal=%s", raw)
	}
}

func TestAdmissionBusyDoesNotOverwriteOwnerReceipt(t *testing.T) {
	root, state, resource, _, _ := bridgeFixture(t)
	canonical, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	ownerID := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	owner := Report{ID: ownerID, Root: canonical, Profile: "old", Status: "succeeded", StartedAt: time.Now(), Duration: time.Second}
	if err = writeReport(state, owner); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(state, ownerID+".json"))
	if err != nil {
		t.Fatal(err)
	}
	locks, err := acquire(state, canonical, ownerID)
	if err != nil {
		t.Fatal(err)
	}
	defer locks.release()
	report, err := Run(context.Background(), Options{Root: root, ProfileName: "full", StateDir: state, lockDirectory: state, resourceDirectory: resource, Env: []string{"DVA_RESOURCE_POLICY=" + filepath.Join(root, "policy.json")}, Profile: config.CIProfile{Timeout: "1s", MaxParallel: 1, Admission: &config.CIAdmission{Class: "heavy", Priority: "normal", WaitTimeout: "1s"}, Steps: []config.CIStep{{Name: "x", Run: "true"}}}})
	if err == nil || report.ID != ownerID || report.Status != "busy" {
		t.Fatalf("report=%+v err=%v", report, err)
	}
	after, err := os.ReadFile(filepath.Join(state, ownerID+".json"))
	if err != nil || string(after) != string(before) {
		t.Fatalf("owner receipt changed: %v", err)
	}
	files, _ := filepath.Glob(filepath.Join(state, "*.json"))
	if len(files) != 2 {
		t.Fatalf("files=%v", files)
	}
}

func TestAdmissionWaitingHistoryHasUnknownDuration(t *testing.T) {
	wait := Report{ID: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Root: "/fixture", Profile: "full", Status: "waiting", StartedAt: time.Now(), Admission: &AdmissionObservation{Class: "heavy", Priority: "normal", Outcome: "waiting"}}
	raw, err := json.Marshal(wait)
	if err != nil {
		t.Fatal(err)
	}
	var round Report
	if err = json.Unmarshal(raw, &round); err != nil {
		t.Fatal(err)
	}
	analysis := AnalyzeHistory([]Report{round}, "/fixture", "full")
	if analysis.StatusCounts["waiting"] != 1 || len(analysis.Runs) != 1 || analysis.Runs[0].Duration != nil {
		t.Fatalf("analysis=%+v", analysis)
	}
}
