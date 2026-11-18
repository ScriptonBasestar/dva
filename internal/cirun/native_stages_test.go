//go:build darwin || linux

package cirun

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestRunCollectsNativeStagesAndProtectsActualStepName(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	step := config.CIStep{Name: "rust-test", Environment: map[string]string{"DVA_CI_STEP_NAME": "forged"}, Run: `
umask 077
start="$DVA_CI_EVENTS_DIR/native-abcdef0123456789abcdef0123456789-start.json"
finish="$DVA_CI_EVENTS_DIR/native-abcdef0123456789abcdef0123456789-finish.json"
printf '{"schema":"dva-ci-native-stage/1","run_id":"%s","parent_step":"%s","invocation_id":"abcdef0123456789abcdef0123456789","stage_name":"workspace_tests","event":"start","started_at":"2026-11-18T10:00:00Z","plan_ordinal":1,"plan_total":2}\n' "$DVA_CI_RUN_ID" "$DVA_CI_STEP_NAME" > "$start"
printf '{"schema":"dva-ci-native-stage/1","run_id":"%s","parent_step":"%s","invocation_id":"abcdef0123456789abcdef0123456789","stage_name":"workspace_tests","event":"finish","started_at":"2026-11-18T10:00:00Z","finished_at":"2026-11-18T10:00:02Z","status":"failed","exit_code":7,"plan_ordinal":1,"plan_total":2}\n' "$DVA_CI_RUN_ID" "$DVA_CI_STEP_NAME" > "$finish"
`}
	report, err := Run(context.Background(), Options{Root: root, ProfileName: "native", StateDir: state, lockDirectory: state, Env: append(os.Environ(), "DVA_CI_STEP_NAME=forged-inherited"), Profile: profile(step)})
	if err != nil || report.Status != "succeeded" {
		t.Fatalf("report status=%s err=%v", report.Status, err)
	}
	if got := report.NativeStages; got == nil || got.CollectionStatus != "available" || got.EventsSeen != 2 || got.EventsRead != 2 || got.InvalidEvents != 0 || len(got.Stages) != 1 {
		t.Fatalf("native observations: %+v", got)
	}
	stage := report.NativeStages.Stages[0]
	if stage.ParentStep != "rust-test" || stage.StageName != "workspace_tests" || stage.Status != "failed" || stage.DurationText == nil || *stage.DurationText != "2s" || stage.ExitCode == nil || *stage.ExitCode != 7 {
		t.Fatalf("stage: %+v", stage)
	}
	if report.Steps[0].Duration < 0 || report.Steps[0].Status != "succeeded" {
		t.Fatalf("native status altered parent step: %+v", report.Steps[0])
	}
	if report.Steps[0].OSWaitResources == nil || report.Steps[0].OSWaitResources.CollectionStatus != "available" {
		t.Fatalf("host command wait resources missing: %+v", report.Steps[0].OSWaitResources)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip Report
	if err := json.Unmarshal(encoded, &roundTrip); err != nil || roundTrip.NativeStages == nil || len(roundTrip.NativeStages.Stages) != 1 {
		t.Fatalf("receipt round trip: err=%v report=%+v", err, roundTrip.NativeStages)
	}
	analysis := AnalyzeHistory([]Report{roundTrip}, report.Root, report.Profile)
	if len(analysis.Observations) != 1 || analysis.Observations[0].NativeStages == nil || analysis.Observations[0].Steps[0].OSWaitResources == nil {
		t.Fatalf("history projection lost native observations: %+v", analysis.Observations)
	}
	if !strings.Contains(string(encoded), `"status":"failed"`) || report.Status == "failed" {
		t.Fatal("stage result was incorrectly promoted to DVA gate result")
	}
}

func TestOSWaitResourcesRequireWaitedProcessAndAreOmittedForOCI(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	started := time.Now()
	processReport, _ := runStep(context.Background(), root, config.CIStep{Name: "host", Run: "exit 3"}, nil, 1, "test", io.Discard, started)
	if processReport.Status != "failed" || processReport.OSWaitResources == nil {
		t.Fatalf("failed child wait not observed: %+v", processReport)
	}
	ociReport, _ := runStep(context.Background(), root, config.CIStep{Name: "container", OCI: &config.CIOCI{}}, nil, 1, "test", io.Discard, started)
	if ociReport.OSWaitResources != nil {
		t.Fatalf("OCI step reported host process stats: %+v", ociReport.OSWaitResources)
	}
}

func TestNativeStageStartOnlyIsUnknownAndLegacyReceiptStaysAbsent(t *testing.T) {
	runID := strings.Repeat("a", 32)
	events := nativeTestEventsDir(t, runID)
	writeNativeTestEvent(t, events, "native-abcdef0123456789abcdef0123456789-start.json", nativeTestEvent(runID, "one", "start"), 0600)
	got := collectNativeStages(events, runID, []string{"one"})
	if got.CollectionStatus != "available" || len(got.Stages) != 1 || got.Stages[0].Status != "unknown" || got.Stages[0].DurationText != nil {
		t.Fatalf("start only: %+v", got)
	}
	legacy := Report{ID: runID, Status: "succeeded", Duration: time.Second}
	projected := projectHistoryObservation(legacy, new(100))
	if projected.NativeStages != nil {
		t.Fatalf("legacy history invented observations: %+v", projected.NativeStages)
	}
	legacyReceipt := []byte(`{"id":"` + runID + `","status":"succeeded","duration":"1s"}`)
	var decoded Report
	if err := json.Unmarshal(legacyReceipt, &decoded); err != nil || decoded.NativeStages != nil {
		t.Fatalf("legacy receipt acquired native observations: err=%v %+v", err, decoded.NativeStages)
	}
}

func TestNativeStageFinalSnapshotIsPreservedWhenStepFails(t *testing.T) {
	root, state := t.TempDir(), t.TempDir()
	step := config.CIStep{Name: "rust-test", Run: `
umask 077
printf '{"schema":"dva-ci-native-stage/1","run_id":"%s","parent_step":"%s","invocation_id":"abcdef0123456789abcdef0123456789","stage_name":"workspace_tests","event":"start","started_at":"2026-11-18T10:00:00Z"}\n' "$DVA_CI_RUN_ID" "$DVA_CI_STEP_NAME" > "$DVA_CI_EVENTS_DIR/native-abcdef0123456789abcdef0123456789-start.json"
exit 7
`}
	report, err := Run(context.Background(), Options{Root: root, ProfileName: "native-failure", StateDir: state, lockDirectory: state, Env: os.Environ(), Profile: profile(step)})
	if err == nil || report.Status != "failed" || report.NativeStages == nil || len(report.NativeStages.Stages) != 1 || report.NativeStages.Stages[0].Status != "unknown" {
		t.Fatalf("failed run final snapshot: status=%s native=%+v err=%v", report.Status, report.NativeStages, err)
	}
}

func TestNativeStageRejectsMismatchedRunParentAndFinishIdentity(t *testing.T) {
	runID := strings.Repeat("a", 32)
	events := nativeTestEventsDir(t, runID)
	start := nativeTestEvent(runID, "one", "start")
	finish := nativeTestEvent(runID, "two", "finish")
	finish.StartedAt = start.StartedAt
	writeNativeTestEvent(t, events, "native-abcdef0123456789abcdef0123456789-start.json", start, 0600)
	writeNativeTestEvent(t, events, "native-abcdef0123456789abcdef0123456789-finish.json", finish, 0600)
	got := collectNativeStages(events, runID, []string{"one", "two"})
	if got.CollectionStatus != "partial" || len(got.Stages) != 0 || got.InvalidEvents == 0 {
		t.Fatalf("mismatched parent pair: %+v", got)
	}

	badRun := nativeTestEvent("bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "one", "start")
	data, _ := json.Marshal(badRun)
	if _, err := parseNativeStageEvent(data, runID, badRun.InvocationID, "start", map[string]struct{}{"one": {}}); err == nil {
		t.Fatal("mismatched run id accepted")
	}
}

func TestNativeParentNameContractAndDegradedProfile(t *testing.T) {
	runID := strings.Repeat("a", 32)
	events := nativeTestEventsDir(t, runID)
	if !validNativeParentName("build Rust 단계") || validNativeParentName("") || validNativeParentName("bad\nname") || validNativeParentName(strings.Repeat("x", 257)) {
		t.Fatal("parent name contract mismatch")
	}
	got := collectNativeStages(events, runID, []string{strings.Repeat("x", 257)})
	if got.CollectionStatus != "degraded" || len(got.Stages) != 0 {
		t.Fatalf("overlong actual profile step was remapped: %+v", got)
	}
	event := nativeTestEvent(runID, "build Rust 단계", "start")
	data, _ := json.Marshal(event)
	if _, err := parseNativeStageEvent(data, runID, event.InvocationID, "start", map[string]struct{}{event.ParentStep: {}}); err != nil {
		t.Fatalf("valid exact Unicode parent rejected: %v", err)
	}
	event.StageName = "phase.1"
	data, _ = json.Marshal(event)
	if _, err := parseNativeStageEvent(data, runID, event.InvocationID, "start", map[string]struct{}{event.ParentStep: {}}); err != nil {
		t.Fatalf("valid phase basename rejected: %v", err)
	}
	event.StageName = "_phase"
	data, _ = json.Marshal(event)
	if _, err := parseNativeStageEvent(data, runID, event.InvocationID, "start", map[string]struct{}{event.ParentStep: {}}); err == nil {
		t.Fatal("unsafe leading underscore stage name accepted")
	}
}

func nativeTestEventsDir(t *testing.T, runID string) string {
	t.Helper()
	parent := filepath.Join(t.TempDir(), "events")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(parent, runID)
	if err := os.Mkdir(runDir, 0700); err != nil {
		t.Fatal(err)
	}
	return runDir
}

func nativeTestEvent(runID, parent, kind string) nativeStageEvent {
	e := nativeStageEvent{Schema: nativeStageSchema, RunID: runID, ParentStep: parent, InvocationID: "abcdef0123456789abcdef0123456789", StageName: "compile", Event: kind, StartedAt: "2026-11-18T10:00:00Z"}
	if kind == "finish" {
		e.FinishedAt, e.Status = "2026-11-18T10:00:01Z", "succeeded"
	}
	return e
}

func writeNativeTestEvent(t *testing.T, dir, name string, event nativeStageEvent, mode os.FileMode) {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), data, mode); err != nil {
		t.Fatal(err)
	}
}
