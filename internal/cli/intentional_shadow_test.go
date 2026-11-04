package cli

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateIntentionalShadowJSONRemainsVisible(t *testing.T) {
	path := suppressionFixture(t, "version: \"0.1.0\"\ninteraction:\n  engine:test:\n    runner: local\n    command: \"true\"\n    intentional_shadow: true\nsubprojects:\n  engine:\n    path: engine\n", map[string]string{"engine/dva.yml": "version: \"0.1.0\"\n"})
	oldJSON := jsonOutput
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = oldJSON })
	result := runValidateCommandForTest(t, path, "validate", "--strict")
	if result.err != "" {
		t.Fatalf("acknowledged shadow failed strict: %+v", result)
	}
	var report validateReport
	if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v: %s", err, result.stdout)
	}
	if !report.Valid || len(report.Suppressed) != 1 {
		t.Fatalf("acknowledgment must remain visible: %+v", report)
	}
	item := report.Suppressed[0]
	if item.Kind != string(suppressedShadow) || item.Name != "engine:test" || !strings.Contains(item.Reason, "intentional_shadow: true") {
		t.Fatalf("unexpected acknowledgment: %+v", item)
	}
}

func TestValidateStrictIntentionalShadow(t *testing.T) {
	for _, tc := range []struct {
		name, key, declaration, finding string
		fail                            bool
	}{
		{name: "unacknowledged collision", key: "engine:test", finding: "literal key takes precedence", fail: true},
		{name: "acknowledged collision", key: "engine:test", declaration: "    intentional_shadow: true\n"},
		{name: "explicit false", key: "engine:test", declaration: "    intentional_shadow: false\n", finding: "literal key takes precedence", fail: true},
		{name: "stale declaration", key: "other:test", declaration: "    intentional_shadow: true\n", finding: "stale declaration", fail: true},
		{name: "reserved key still rejected", key: "build", declaration: "    intentional_shadow: true\n", finding: "reserved command conflict", fail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := suppressionFixture(t, "version: \"0.1.0\"\ninteraction:\n  "+tc.key+":\n    runner: local\n    command: \"true\"\n"+tc.declaration+"subprojects:\n  engine:\n    path: engine\n",
				map[string]string{"engine/dva.yml": "version: \"0.1.0\"\n"})
			result := runValidateCommandForTest(t, path, "validate", "--strict", "--show-ignored")
			if (result.err != "") != tc.fail {
				t.Fatalf("strict result error=%q stdout=%s stderr=%s", result.err, result.stdout, result.stderr)
			}
			if !tc.fail && strings.Contains(result.stdout+result.stderr, "literal key takes precedence") {
				t.Fatalf("acknowledged collision still reported: %+v", result)
			}
			if !tc.fail && (!strings.Contains(result.stdout, "1 shadow key ignored") || !strings.Contains(result.stdout, "intentional_shadow: true")) {
				t.Fatalf("acknowledged shadow was not counted and explained: %+v", result)
			}
			if tc.finding != "" && !strings.Contains(result.stdout+result.stderr+result.err, tc.finding) {
				t.Fatalf("expected finding %q absent from %+v", tc.finding, result)
			}
		})
	}
}
