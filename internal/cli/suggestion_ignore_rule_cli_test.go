package cli

import (
	"strings"
	"testing"
)

func TestSuggestionIgnoreObjectAttributesReasonAndBoundsActualSuppression(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.4.2"
suggestion_ignore:
  - pattern: "test-*"
    reason: CI matrix only
    max_matches: 1
`, map[string]string{
		"Makefile": "test-unit: ## Unit\ntest-integration: ## Integration\n",
	}))

	warnings, suppressed := detectConfigSuggestionWarningsWithSuppressions(c)
	if len(warnings) != 0 {
		t.Fatalf("bounded ignore should hide both ordinary suggestions: %v", warnings)
	}
	if got := suppressed.count(suppressedSuggestion); got != 2 {
		t.Fatalf("suppressed suggestions = %d, want 2", got)
	}
	if lines := strings.Join(suppressed.showIgnoredLines(), "\n"); !strings.Contains(lines, "suggestion_ignore: test-* — CI matrix only") {
		t.Fatalf("object reason missing from attribution: %s", lines)
	}
	if got := strings.Join(suppressed.overbroad, "\n"); !strings.Contains(got, "suppresses 2 uncovered suggestion(s), exceeding max_matches 1") {
		t.Fatalf("over-broad bounded rule was not diagnosed: %s", got)
	}
}

func TestSuggestionIgnoreObjectBoundCountsOnlyFirstMatchingRule(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.4.2"
suggestion_ignore:
  - "test-unit"
  - pattern: "test-*"
    reason: CI matrix only
    max_matches: 1
`, map[string]string{
		"Makefile": "test-unit: ## Unit\ntest-integration: ## Integration\n",
	}))

	_, suppressed := detectConfigSuggestionWarningsWithSuppressions(c)
	if len(suppressed.overbroad) != 0 {
		t.Fatalf("later rule must count only the one candidate it actually hides, got %v", suppressed.overbroad)
	}
}

func TestSuggestionIgnoreObjectBoundExcludesAlreadyCoveredTargets(t *testing.T) {
	c := loadSuppressionConfig(t, suppressionFixture(t, `version: "0.4.2"
suggestion_ignore:
  - pattern: "test-*"
    reason: CI matrix only
    max_matches: 1
interaction:
  test-unit:
    runner: local
    command: make test-unit
`, map[string]string{
		"Makefile": "test-unit: ## Unit\ntest-integration: ## Integration\n",
	}))

	_, suppressed := detectConfigSuggestionWarningsWithSuppressions(c)
	if len(suppressed.overbroad) != 0 {
		t.Fatalf("already-covered target must not consume max_matches: %v", suppressed.overbroad)
	}
}

func TestValidateSuggestionIgnoreBoundRendersReasonAndFailsOnlyUnderStrict(t *testing.T) {
	configPath := suppressionFixture(t, `version: "0.4.2"
suggestion_ignore:
  - pattern: "test-*"
    reason: CI matrix only
    max_matches: 1
`, map[string]string{
		"Makefile": "test-unit: ## Unit\ntest-integration: ## Integration\n",
	})

	plain := runValidateCommandForTest(t, configPath, "validate")
	if plain.err != "" {
		t.Fatalf("plain validate failed: %s", plain.err)
	}
	if !strings.Contains(plain.stdout, "[warn] overbroad ignore: suggestion_ignore[0]") {
		t.Fatalf("plain validate did not render cap warning:\n%s", plain.stdout)
	}
	shown := runValidateCommandForTest(t, configPath, "validate", "--show-ignored")
	if !strings.Contains(shown.stdout, "suggestion_ignore: test-* — CI matrix only") {
		t.Fatalf("--show-ignored did not render object reason:\n%s", shown.stdout)
	}
	if strict := runValidateCommandForTest(t, configPath, "validate", "--strict"); strict.err == "" {
		t.Fatal("--strict succeeded with an over-broad bounded ignore")
	}

	withinCap := suppressionFixture(t, `version: "0.4.2"
suggestion_ignore:
  - pattern: "test-*"
    reason: CI matrix only
    max_matches: 2
`, map[string]string{
		"Makefile": "test-unit: ## Unit\ntest-integration: ## Integration\n",
	})
	if strict := runValidateCommandForTest(t, withinCap, "validate", "--strict"); strict.err != "" {
		t.Fatalf("--strict failed within declared cap: %s", strict.err)
	}
}
