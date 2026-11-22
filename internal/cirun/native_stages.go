package cirun

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const (
	nativeStageSchema       = "dva-ci-native-stage/1"
	maxNativeStageFileBytes = 8192
	maxNativeStageFiles     = 1000
	maxNativeStageTotalSize = 8 << 20
)

// NativeStageObservation contains bounded, diagnostic observations from native
// CI producers. It is not evidence that a gate passed or that a command ran.
type NativeStageObservation struct {
	ObservedAt       *time.Time           `json:"observed_at,omitempty"`
	CollectionStatus string               `json:"collection_status"`
	EventsSeen       int                  `json:"events_seen"`
	EventsRead       int                  `json:"events_read"`
	EventsOmitted    int                  `json:"events_omitted"`
	InvalidEvents    int                  `json:"invalid_events"`
	Stages           []NativeStageSummary `json:"stages"`
}

// NativeStageSummary keeps child stage times separate from parent step times.
// Duration is present only when a matching start and finish pair was observed.
type NativeStageSummary struct {
	ParentStep   string     `json:"parent_step"`
	InvocationID string     `json:"invocation_id"`
	StageName    string     `json:"stage_name"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	FinishedAt   *time.Time `json:"finished_at,omitempty"`
	Status       string     `json:"status"`
	ExitCode     *int       `json:"exit_code,omitempty"`
	PlanOrdinal  *int       `json:"plan_ordinal,omitempty"`
	PlanTotal    *int       `json:"plan_total,omitempty"`
	DurationText *string    `json:"duration,omitempty"`
}

type nativeStageEvent struct {
	Schema       string `json:"schema"`
	RunID        string `json:"run_id"`
	ParentStep   string `json:"parent_step"`
	InvocationID string `json:"invocation_id"`
	StageName    string `json:"stage_name"`
	Event        string `json:"event"`
	StartedAt    string `json:"started_at"`
	FinishedAt   string `json:"finished_at,omitempty"`
	Status       string `json:"status,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
	PlanOrdinal  *int   `json:"plan_ordinal,omitempty"`
	PlanTotal    *int   `json:"plan_total,omitempty"`
}

type nativeStagePair struct {
	start  *nativeStageEvent
	finish *nativeStageEvent
}

func ciProfileStepNames(profile config.CIProfile) []string {
	names := make([]string, 0, len(profile.Steps))
	for _, step := range profile.Steps {
		names = append(names, step.Name)
	}
	return names
}

var nativeInvocationPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
var nativeStageNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$`)

func collectNativeStages(eventsDir, runID string, steps []string) (result *NativeStageObservation) {
	defer func() {
		observedAt := time.Now().UTC()
		result.ObservedAt = &observedAt
	}()
	if eventsDir == "" {
		return &NativeStageObservation{CollectionStatus: "unavailable", Stages: []NativeStageSummary{}}
	}
	parentSet := make(map[string]struct{}, len(steps))
	for _, step := range steps {
		if validNativeParentName(step) {
			parentSet[step] = struct{}{}
		}
	}
	dir, err := openNativeEventsDir(filepath.Dir(eventsDir), filepath.Base(eventsDir))
	if err != nil {
		if nativeStagePlatformUnsupported(err) {
			return &NativeStageObservation{CollectionStatus: "unsupported", Stages: []NativeStageSummary{}}
		}
		return &NativeStageObservation{CollectionStatus: "unavailable", Stages: []NativeStageSummary{}}
	}
	result = &NativeStageObservation{CollectionStatus: "available", Stages: []NativeStageSummary{}}
	defer func() {
		if dir.Close() != nil {
			setNativeCollectionPartial(result)
		}
	}()
	for _, step := range steps {
		if !validNativeParentName(step) {
			result.CollectionStatus = "degraded"
			break
		}
	}
	entries, err := dir.ReadDir(maxNativeStageFiles + 1)
	if err != nil && err != io.EOF {
		setNativeCollectionPartial(result)
		return result
	}
	if len(entries) > maxNativeStageFiles {
		result.EventsOmitted += len(entries) - maxNativeStageFiles
		entries = entries[:maxNativeStageFiles]
		setNativeCollectionPartial(result)
	}
	pairs := map[string]*nativeStagePair{}
	usedBytes := 0
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "native-") {
			continue
		}
		result.EventsSeen++
		invocation, eventKind, ok := nativeEventFilename(name)
		if !ok {
			result.InvalidEvents++
			setNativeCollectionPartial(result)
			continue
		}
		if usedBytes+maxNativeStageFileBytes > maxNativeStageTotalSize {
			result.EventsOmitted++
			setNativeCollectionPartial(result)
			continue
		}
		data, readErr := readNativeEvent(dir, name)
		if readErr != nil {
			result.InvalidEvents++
			setNativeCollectionPartial(result)
			continue
		}
		usedBytes += len(data)
		event, parseErr := parseNativeStageEvent(data, runID, invocation, eventKind, parentSet)
		if parseErr != nil {
			result.InvalidEvents++
			setNativeCollectionPartial(result)
			continue
		}
		result.EventsRead++
		pair := pairs[invocation]
		if pair == nil {
			pair = &nativeStagePair{}
			pairs[invocation] = pair
		}
		if event.Event == "start" {
			if pair.start != nil {
				result.InvalidEvents++
				setNativeCollectionPartial(result)
				continue
			}
			pair.start = &event
		} else {
			if pair.finish != nil {
				result.InvalidEvents++
				setNativeCollectionPartial(result)
				continue
			}
			pair.finish = &event
		}
	}
	for invocation, pair := range pairs {
		stage, valid := summarizeNativeStage(invocation, pair)
		if !valid {
			result.InvalidEvents++
			setNativeCollectionPartial(result)
			continue
		}
		result.Stages = append(result.Stages, stage)
	}
	sort.Slice(result.Stages, func(i, j int) bool {
		a, b := result.Stages[i], result.Stages[j]
		if a.ParentStep != b.ParentStep {
			return a.ParentStep < b.ParentStep
		}
		if a.StageName != b.StageName {
			return a.StageName < b.StageName
		}
		return a.InvocationID < b.InvocationID
	})
	return result
}

func nativeEventFilename(name string) (invocation, kind string, ok bool) {
	if !strings.HasPrefix(name, "native-") || !strings.HasSuffix(name, ".json") {
		return "", "", false
	}
	body := strings.TrimSuffix(strings.TrimPrefix(name, "native-"), ".json")
	parts := strings.Split(body, "-")
	if len(parts) != 2 || !nativeInvocationPattern.MatchString(parts[0]) || (parts[1] != "start" && parts[1] != "finish") {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func parseNativeStageEvent(data []byte, runID, invocation, eventKind string, parents map[string]struct{}) (nativeStageEvent, error) {
	var event nativeStageEvent
	if !utf8.Valid(data) {
		return event, fmt.Errorf("invalid UTF-8")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&event); err != nil {
		return event, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return event, fmt.Errorf("trailing JSON")
	}
	if event.Schema != nativeStageSchema || event.RunID != runID || event.InvocationID != invocation || event.Event != eventKind || !nativeInvocationPattern.MatchString(event.InvocationID) || !nativeStageNamePattern.MatchString(event.StageName) {
		return event, fmt.Errorf("invalid native stage identity")
	}
	if _, ok := parents[event.ParentStep]; !ok {
		return event, fmt.Errorf("unknown parent step")
	}
	if !validNativeParentName(event.ParentStep) {
		return event, fmt.Errorf("invalid parent step")
	}
	if _, err := time.Parse(time.RFC3339Nano, event.StartedAt); err != nil {
		return event, fmt.Errorf("invalid start time")
	}
	if (event.PlanOrdinal == nil) != (event.PlanTotal == nil) {
		return event, fmt.Errorf("incomplete plan position")
	}
	if event.PlanOrdinal != nil && (*event.PlanOrdinal < 1 || *event.PlanTotal < 1 || *event.PlanOrdinal > *event.PlanTotal) {
		return event, fmt.Errorf("invalid plan position")
	}
	if event.Event == "start" {
		if event.FinishedAt != "" || event.Status != "" || event.ExitCode != nil {
			return event, fmt.Errorf("start event has finish fields")
		}
	} else {
		if event.FinishedAt == "" || !validNativeStatus(event.Status) {
			return event, fmt.Errorf("invalid finish event")
		}
		finished, err := time.Parse(time.RFC3339Nano, event.FinishedAt)
		started, _ := time.Parse(time.RFC3339Nano, event.StartedAt)
		if err != nil || finished.Before(started) {
			return event, fmt.Errorf("invalid finish time")
		}
	}
	return event, nil
}

func validNativeStatus(status string) bool {
	switch status {
	case "succeeded", "failed", "timed_out", "canceled", "unknown":
		return true
	default:
		return false
	}
}

func summarizeNativeStage(invocation string, pair *nativeStagePair) (NativeStageSummary, bool) {
	base := pair.start
	if base == nil {
		base = pair.finish
	}
	if base == nil {
		return NativeStageSummary{}, false
	}
	started, _ := time.Parse(time.RFC3339Nano, base.StartedAt)
	stage := NativeStageSummary{ParentStep: base.ParentStep, InvocationID: invocation, StageName: base.StageName, StartedAt: &started, Status: "unknown", PlanOrdinal: base.PlanOrdinal, PlanTotal: base.PlanTotal}
	if pair.start != nil && pair.finish != nil {
		finish := pair.finish
		if !sameNativeStage(pair.start, finish) {
			return NativeStageSummary{}, false
		}
		finished, _ := time.Parse(time.RFC3339Nano, finish.FinishedAt)
		stage.FinishedAt, stage.Status, stage.ExitCode = &finished, finish.Status, finish.ExitCode
		duration := finished.Sub(started)
		text := duration.String()
		stage.DurationText = &text
	} else if pair.finish != nil {
		finish := pair.finish
		finished, _ := time.Parse(time.RFC3339Nano, finish.FinishedAt)
		stage.FinishedAt, stage.ExitCode = &finished, finish.ExitCode
		// A finish without its matching start is incomplete; do not assert status.
	}
	return stage, true
}

func sameNativeStage(start, finish *nativeStageEvent) bool {
	return start.RunID == finish.RunID && start.ParentStep == finish.ParentStep && start.InvocationID == finish.InvocationID && start.StageName == finish.StageName && start.StartedAt == finish.StartedAt && sameOptionalInt(start.PlanOrdinal, finish.PlanOrdinal) && sameOptionalInt(start.PlanTotal, finish.PlanTotal)
}

func sameOptionalInt(a, b *int) bool { return a == nil && b == nil || a != nil && b != nil && *a == *b }

func validNativeParentName(name string) bool {
	if name == "" || len(name) > 256 || !utf8.ValidString(name) {
		return false
	}
	for _, r := range name {
		if r <= 0x1f || r == 0x7f {
			return false
		}
	}
	return true
}

func setNativeCollectionPartial(result *NativeStageObservation) {
	if result.CollectionStatus != "degraded" {
		result.CollectionStatus = "partial"
	}
}
