package cirun

import (
	"testing"
	"time"
)

func TestAnalyzeHistoryKeepsUnknownConditionsAndStatusesSeparate(t *testing.T) {
	r := []Report{{ID: "a", Root: "/a", Profile: "full", Status: "succeeded", Duration: time.Second}, {ID: "b", Root: "/a", Profile: "full", Status: "timed_out"}, {ID: "c", Root: "/b", Profile: "full", Status: "succeeded"}}
	a := AnalyzeHistory(r, "/a", "full")
	if len(a.Runs) != 2 || a.StatusCounts["succeeded"] != 1 || a.StatusCounts["timed_out"] != 1 || a.Comparability == "" {
		t.Fatalf("unexpected analysis: %+v", a)
	}
}
