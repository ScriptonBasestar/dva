//go:build !darwin && !linux

package cirun

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNativeStageOtherPlatformReportsUnsupportedAndKeepsWireContract(t *testing.T) {
	runID := strings.Repeat("a", 32)
	observation := collectNativeStages(filepath.Join(t.TempDir(), runID), runID, []string{"native-step"})
	if observation.CollectionStatus != "unsupported" || len(observation.Stages) != 0 {
		t.Fatalf("unsupported platform used insecure collection or hid status: %+v", observation)
	}

	event := nativeStageEvent{
		Schema:       nativeStageSchema,
		RunID:        runID,
		ParentStep:   "build Rust 단계",
		InvocationID: "abcdef0123456789abcdef0123456789",
		StageName:    "phase.1",
		Event:        "start",
		StartedAt:    "2026-11-18T10:00:00Z",
	}
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	parents := map[string]struct{}{event.ParentStep: {}}
	if _, err := parseNativeStageEvent(data, runID, event.InvocationID, "start", parents); err != nil {
		t.Fatalf("portable wire contract rejected valid event: %v", err)
	}
	event.StageName = "_phase"
	data, _ = json.Marshal(event)
	if _, err := parseNativeStageEvent(data, runID, event.InvocationID, "start", parents); err == nil {
		t.Fatal("wire contract accepted an unsafe stage name")
	}

	legacyReceipt := []byte(`{"id":"` + runID + `","status":"succeeded","duration":"1s"}`)
	var legacy Report
	if err := json.Unmarshal(legacyReceipt, &legacy); err != nil || legacy.NativeStages != nil {
		t.Fatalf("legacy receipt acquired native observations: err=%v report=%+v", err, legacy.NativeStages)
	}
	legacy = Report{ID: runID, Status: "succeeded", Duration: time.Second}
	if projected := projectHistoryObservation(legacy, new(100)); projected.NativeStages != nil {
		t.Fatalf("legacy history invented native observations: %+v", projected.NativeStages)
	}
}
