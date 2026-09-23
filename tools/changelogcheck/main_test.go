package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsLoggable(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    bool
	}{
		{"feat with scope", "feat(lifecycle): optional entry, post_build, primary", true},
		{"fix with scope", "fix(cli): omit scripts from logs targets", true},
		{"refactor with scope", "refactor(config): use slices.Contains in AppliesToPlan", true},
		{"feat without scope", "feat: add a thing", true},
		{"breaking marker", "feat(config)!: drop a field", true},
		{"test is not loggable", "test(cli): range the field sequence the lint gate asks for", false},
		{"docs is not loggable", "docs(cli): record what the two contract tests cannot show", false},
		{"chore is not loggable", "chore(tasks): complete plan profile validation", false},
		{"merge subject", "Merge branch 'master' into task-354", false},
		{"no type prefix", "just some words", false},
		{"type-like prefix without colon", "fixture(cli) something", false},
		{"leading whitespace tolerated", "  fix(cli): bound the git call", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLoggable(tt.subject); got != tt.want {
				t.Errorf("isLoggable(%q) = %v, want %v", tt.subject, got, tt.want)
			}
		})
	}
}

func TestCountLoggable(t *testing.T) {
	subjects := []string{
		"feat(a): one",
		"test(a): two",
		"fix(a): three",
		"chore(a): four",
		"refactor(a): five",
	}
	if got := countLoggable(subjects); got != 3 {
		t.Errorf("countLoggable = %d, want 3", got)
	}
	if got := countLoggable(nil); got != 0 {
		t.Errorf("countLoggable(nil) = %d, want 0", got)
	}
}

func TestInspectUnreleased(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    unreleasedState
	}{
		{
			name:    "no heading at all",
			content: "# Changelog\n\n## [0.2.0] - 2026-09-09\n\n### Added\n- a thing\n",
			want:    unreleasedMissing,
		},
		{
			name:    "heading followed immediately by the next version",
			content: "# Changelog\n\n## [Unreleased]\n\n## [0.2.0] - 2026-09-09\n\n### Added\n- a thing\n",
			want:    unreleasedEmpty,
		},
		{
			name:    "heading at EOF with nothing under it",
			content: "# Changelog\n\n## [Unreleased]\n\n\n",
			want:    unreleasedEmpty,
		},
		{
			name:    "heading with a bullet",
			content: "## [Unreleased]\n\n### Added\n- a thing\n\n## [0.2.0] - 2026-09-09\n",
			want:    unreleasedFilled,
		},
		{
			name:    "sub-heading alone still counts as content",
			content: "## [Unreleased]\n\n### Added\n\n## [0.2.0] - 2026-09-09\n",
			want:    unreleasedFilled,
		},
		{
			name:    "content of the NEXT section must not leak in",
			content: "## [Unreleased]\n\n## [0.2.0] - 2026-09-09\n\n### Added\n- a thing\n",
			want:    unreleasedEmpty,
		},
		{
			name:    "a deeper heading does not terminate the section",
			content: "## [Unreleased]\n\n#### note\n\n## [0.2.0] - 2026-09-09\n",
			want:    unreleasedFilled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := inspectUnreleased(tt.content); got != tt.want {
				t.Errorf("inspectUnreleased = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDrainThresholdIsDeliberate pins the threshold against an accidental edit: the value is
// an argued choice (see the package comment), not a tuning knob, so changing it should mean
// changing this test on purpose.
func TestDrainThresholdIsDeliberate(t *testing.T) {
	if drainThreshold != 10 {
		t.Errorf("drainThreshold = %d, want 10 — if this changed deliberately, update the package comment too", drainThreshold)
	}
}

func writeSourceVersion(t *testing.T, root, body string) {
	t.Helper()
	path := filepath.Join(root, "internal", "config", "version.go")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestPendingReleaseCandidate(t *testing.T) {
	const candidate = "# Changelog\n\n## [Unreleased]\n\n## [0.3.0] - 2026-09-23\n\n### Added\n- release content\n\n## [0.2.0] - 2026-09-09\n"
	tests := []struct {
		name, source, changelog, tag string
		want                         bool
	}{
		{"current newer source and immediate populated dated section", "package config\nvar Version = \"0.3.0\"\n", candidate, "v0.2.0", true},
		{"same source version as tag", "package config\nvar Version = \"0.2.0\"\n", candidate, "v0.2.0", false},
		{"older source version than tag", "package config\nvar Version = \"0.1.9\"\n", candidate, "v0.2.0", false},
		{"candidate heading does not match source", "package config\nvar Version = \"0.3.1\"\n", candidate, "v0.2.0", false},
		{"source version is not stable semver", "package config\nvar Version = \"0.3.0-rc.1\"\n", candidate, "v0.2.0", false},
		{"source version has a leading zero identifier", "package config\nvar Version = \"0.03.0\"\n", "## [Unreleased]\n\n## [0.03.0] - 2026-09-23\n\n- content\n", "v0.2.0", false},
		{"candidate heading has no date", "package config\nvar Version = \"0.3.0\"\n", "## [Unreleased]\n\n## [0.3.0]\n\n- content\n", "v0.2.0", false},
		{"candidate heading has invalid date", "package config\nvar Version = \"0.3.0\"\n", "## [Unreleased]\n\n## [0.3.0] - 2026-02-30\n\n- content\n", "v0.2.0", false},
		{"candidate section is empty", "package config\nvar Version = \"0.3.0\"\n", "## [Unreleased]\n\n## [0.3.0] - 2026-09-23\n\n## [0.2.0] - 2026-09-09\n", "v0.2.0", false},
		{"arbitrary immediate section cannot bypass", "package config\nvar Version = \"0.3.0\"\n", "## [Unreleased]\n\n## Notes\n\n- content\n\n## [0.3.0] - 2026-09-23\n\n- release content\n", "v0.2.0", false},
		{"malformed reachable tag", "package config\nvar Version = \"0.3.0\"\n", candidate, "release-0.2.0", false},
		{"reachable tag has a leading zero identifier", "package config\nvar Version = \"0.3.0\"\n", candidate, "v0.02.0", false},
		{"candidate heading has a leading zero identifier", "package config\nvar Version = \"0.3.0\"\n", "## [Unreleased]\n\n## [0.03.0] - 2026-09-23\n\n- content\n", "v0.2.0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeSourceVersion(t, root, tt.source)
			_, got := pendingReleaseCandidate(root, tt.changelog, tt.tag)
			if got != tt.want {
				t.Errorf("pendingReleaseCandidate() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestParseReleaseVersionRejectsLeadingZeroIdentifiers(t *testing.T) {
	for _, value := range []string{"00.3.0", "0.03.0", "0.3.00"} {
		t.Run(value, func(t *testing.T) {
			if _, err := parseReleaseVersion(value); err == nil {
				t.Errorf("parseReleaseVersion(%q) succeeded, want leading-zero rejection", value)
			}
		})
	}
}

func TestSourceVersionRejectsMalformedDeclarations(t *testing.T) {
	tests := []struct {
		name, body string
	}{
		{"missing", "package config\nvar Commit = \"dev\"\n"},
		{"not a literal", "package config\nvar Version = buildVersion()\n"},
		{"duplicate", "package config\nvar Version = \"0.3.0\"\nvar Version = \"0.3.1\"\n"},
		{"syntax error", "package config\nvar Version = \"0.3.0\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			writeSourceVersion(t, root, tt.body)
			if _, err := sourceVersion(root); err == nil {
				t.Error("sourceVersion() succeeded on malformed declaration")
			}
		})
	}
}
