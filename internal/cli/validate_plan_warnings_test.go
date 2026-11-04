package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

func TestValidationDiagnosticNoPlansStrictOutputCompatibility(t *testing.T) {
	path := writeValidateConfigForTest(t, "version: \"0.1.0\"\nstack:\n  task:\n    script:\n      up: \"true\"\n")
	plain := runValidateCommandForTest(t, path, "validate", "--strict")
	if plain.err != "" || !strings.Contains(plain.stdout, "[warn] semantic: ℹ No 'plans' defined") {
		t.Fatalf("informational strict output changed: %+v", plain)
	}
	oldJSON := jsonOutput
	jsonOutput = true
	t.Cleanup(func() { jsonOutput = oldJSON })
	result := runValidateCommandForTest(t, path, "validate", "--strict")
	var report validateReport
	if err := json.Unmarshal([]byte(result.stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v: %s", err, result.stdout)
	}
	if result.err != "" || !report.Valid || len(report.Warnings) != 1 || report.Warnings[0].Category != "semantic" || report.Warnings[0].Message != "ℹ No 'plans' defined — consider adding execution plans for 'dva up <name>' support" {
		t.Fatalf("informational JSON contract changed: result=%+v report=%+v", result, report)
	}
	if strings.Contains(result.stdout, `"severity"`) {
		t.Fatalf("internal severity leaked into the JSON contract: %s", result.stdout)
	}
}

// TestValidatePlanDriftWarningsRenderAndPromoteUnderStrict is the end-to-end coverage
// TASK-244 criterion 4 asks for: D6 (duplicate plan declarations) and D7 (multiple
// plans without default_plan) must render under the existing "semantic" text prefix
// and JSON category — no new category — stay non-fatal by default, and be promoted
// to a non-zero exit by --strict exactly like every other semantic warning.
//
// The fixture below fires D6 and D7 together: "alpha" and "beta" declare identical
// fields (D6), and neither config sets default_plan across two-or-more plans (D7).
func TestValidatePlanDriftWarningsRenderAndPromoteUnderStrict(t *testing.T) {
	configPath := writeValidateConfigForTest(t, `version: "0.1.44"
plans:
  alpha:
    entries: []
  beta:
    entries: []
`)

	defaultRun := runValidateCommandForTest(t, configPath, "validate")
	if defaultRun.err != "" {
		t.Fatalf("default run returned an error: %s", defaultRun.err)
	}
	// validateNoticeWriter routes these through stdout in this harness (root.go writes
	// notices to os.Stdout, which runValidateCommandForTest captures as .stdout — see
	// its sibling assertions in this package for the same convention).
	if !strings.Contains(defaultRun.stdout, "[warn] semantic: plans \"alpha\" and \"beta\" declare equal") {
		t.Errorf("default run stdout missing D6 warning under [warn] semantic: prefix:\n%s", defaultRun.stdout)
	}
	if !strings.Contains(defaultRun.stdout, "[warn] semantic: 2 plans are defined (alpha, beta) but default_plan is not set") {
		t.Errorf("default run stdout missing D7 warning under [warn] semantic: prefix:\n%s", defaultRun.stdout)
	}

	strictRun := runValidateCommandForTest(t, configPath, "validate", "--strict")
	if strictRun.err == "" {
		t.Fatal("--strict run succeeded, want non-zero exit on plan drift warnings")
	}
}

// TestValidatePlanDriftWarningsUseSemanticJSONCategory pins the JSON side of the same
// contract: both new warnings land in the existing "semantic" category, not a new one.
func TestValidatePlanDriftWarningsUseSemanticJSONCategory(t *testing.T) {
	body := `version: "0.1.44"
plans:
  alpha:
    entries: []
  beta:
    entries: []
`
	out, _, err := runValidate(t, body, true)
	if err != nil {
		t.Fatalf("json run returned an error: %v", err)
	}

	doc := decodeOneDocument(t, out)
	var sawD6, sawD7 bool
	for _, w := range doc.Warnings {
		if w.Category != "semantic" {
			continue
		}
		if strings.Contains(w.Message, "declare equal environment, site, vars, endpoint_tags, entries, and composes") {
			sawD6 = true
		}
		if strings.Contains(w.Message, "but default_plan is not set") {
			sawD7 = true
		}
	}
	if !sawD6 {
		t.Errorf("no semantic-category warning carries the D6 message; got %+v", doc.Warnings)
	}
	if !sawD7 {
		t.Errorf("no semantic-category warning carries the D7 message; got %+v", doc.Warnings)
	}
}

func TestStrictIgnoresInformationalHints(t *testing.T) {
	info := config.ValidationDiagnostic{Severity: config.SeverityInfo, Message: "No plans defined"}
	warning := config.ValidationDiagnostic{Severity: config.SeverityWarning, Message: "ℹ Still an actionable warning"}
	if hasActionableWarning([]config.ValidationDiagnostic{info}) {
		t.Fatal("an informational hint must not fail --strict")
	}
	if !hasActionableWarning([]config.ValidationDiagnostic{info, warning}) {
		t.Fatal("a warning must fail --strict even with an informational glyph")
	}
	if !hasActionableWarning([]config.ValidationDiagnostic{{Severity: config.DiagnosticSeverity(99), Message: "ℹ Unknown classification"}}) {
		t.Fatal("an unknown severity must not bypass --strict")
	}
	if hasActionableWarning(nil) {
		t.Fatal("no diagnostics must not fail --strict")
	}
}

func TestValidateStrictFailsOnUnresolvedEnvFilePath(t *testing.T) {
	configPath := writeValidateConfigForTest(t, "version: \"0.1.44\"\nenv_file:\n  - .env.${DVA_TEST_UNSET_X}\n")
	if run := runValidateCommandForTest(t, configPath, "validate", "--strict"); run.err == "" {
		t.Fatal("--strict succeeded on an unresolved env_file path")
	}
}
