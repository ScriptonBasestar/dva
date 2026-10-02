package config

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

func TestVersionCompatibility(t *testing.T) {
	tests := []struct {
		required   string
		compatible bool
	}{
		{"0.1.0", true},
		{"0.0.9", true},
		{"1.0.0", false},
	}

	for _, tt := range tests {
		got, err := isVersionCompatible(tt.required)
		if err != nil {
			t.Errorf("isVersionCompatible(%s) error = %v, want nil", tt.required, err)
			continue
		}
		if got != tt.compatible {
			t.Errorf("isVersionCompatible(%s) = %v, want %v", tt.required, got, tt.compatible)
		}
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.2.3", [3]int{1, 2, 3}},
		{"0.1", [3]int{0, 1, 0}},
		{"v1.0.0", [3]int{1, 0, 0}},
	}
	for _, tt := range tests {
		got, err := parseVersion(tt.input)
		if err != nil {
			t.Errorf("parseVersion(%s) error = %v, want nil", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseVersion(%s) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// TestParseVersionRejectsMalformed locks TASK-070. This table used to assert
// {"invalid", [3]int{0,0,0}} and {"", [3]int{0,0,0}} — it pinned the defect rather
// than the behavior. parseVersion discarded fmt.Sscanf's error, so anything
// unparseable stayed [0,0,0], isVersionCompatible then asked whether the running DVA
// was at least 0.0.0, and the answer was always yes. A misspelled `version: O.2.0`
// (letter O) was written to require 0.2.0 and instead required nothing, silently.
//
// The empty case is not malformed and is not tested here: `version:` is optional, and
// checkConfigVersion returns before parsing (see TestCheckConfigVersion).
func TestParseVersionRejectsMalformed(t *testing.T) {
	malformed := []string{
		"invalid",
		"",           // parseVersion itself is strict; the empty gate lives in checkConfigVersion
		"O.2.0",      // letter O for zero — the typo that motivated this task
		"0.1.44-rc1", // fmt.Sscanf accepted this: it does not error on trailing input
		"1",
		"1.2.3.4",
		"1.2.x",
		"latest",
		"V1.0.0",                   // only a lowercase v is a prefix; the old TrimPrefix was case-sensitive too
		" 0.1.0",                   // Sscanf skipped leading space for %d; the pattern does not
		"0.1.0\n",                  // Go's RE2 `$` does not admit a trailing newline, unlike Perl's
		"99999999999999999999.0.0", // digits, but wider than an int — see TestVersionPatternMatchesSchema
	}
	for _, v := range malformed {
		if _, err := parseVersion(v); err == nil {
			t.Errorf("parseVersion(%q) error = nil, want a rejection", v)
		}
	}

	// The error must name the offending value, since the whole point of the check is
	// that a misspelled version is otherwise invisible.
	_, err := parseVersion("O.2.0")
	if err == nil {
		t.Fatal("parseVersion(\"O.2.0\") error = nil, want a rejection")
	}
	if !strings.Contains(err.Error(), `"O.2.0"`) {
		t.Errorf("error = %q, want it to quote the offending value", err)
	}

	// Two segments must keep parsing: an unquoted `version: 0.1` is a YAML number that
	// yaml.v3 coerces into the Go string "0.1", so rejecting it would break configs
	// that load today.
	got, err := parseVersion("0.1")
	if err != nil {
		t.Fatalf("parseVersion(\"0.1\") error = %v, want nil", err)
	}
	if got != [3]int{0, 1, 0} {
		t.Errorf("parseVersion(\"0.1\") = %v, want [0 1 0]", got)
	}
}

func TestCheckConfigVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		wantErr string // substring, empty means no error
	}{
		{"absent", "", ""},
		{"satisfied", "0.1.0", ""},
		{"equal to running", Version, ""},
		{"too new", "9.9.9", "requires minimum version"},
		{"malformed", "O.2.0", "is not a version"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkConfigVersion(&Config{Version: tt.version})
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("checkConfigVersion(%q) error = %v, want nil", tt.version, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("checkConfigVersion(%q) error = nil, want %q", tt.version, tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}

	// A too-new version must report the incompatibility, not a parse error — the
	// strictness added by TASK-070 must not swallow the check it guards.
	err := checkConfigVersion(&Config{Version: "9.9.9"})
	if err == nil {
		t.Fatal("checkConfigVersion(\"9.9.9\") error = nil, want an incompatibility")
	}
	if strings.Contains(err.Error(), "is not a version") {
		t.Errorf("error = %q, want an incompatibility report, not a parse failure", err)
	}
}

// TestUnreadableBinaryVersionBlamesTheBuild covers the second parseVersion call in
// isVersionCompatible, which no test reached before: Version is injected by ldflags from
// a literal in version.go, so the branch is only reachable through a build that overrides
// it. Reachable-only-by-mistake is exactly the branch whose message goes unread, and this
// one used to end with "Omit `version:` entirely" — advice about a file the user cannot
// fix, for a fault in the binary reading it.
//
// Mutating the package var is safe here because no test in this package calls t.Parallel.
func TestUnreadableBinaryVersionBlamesTheBuild(t *testing.T) {
	original := Version
	Version = "bogus"
	t.Cleanup(func() { Version = original })

	err := checkConfigVersion(&Config{Version: "0.1.0"})
	if err == nil {
		t.Fatal("checkConfigVersion error = nil, want a build complaint")
	}
	if !strings.Contains(err.Error(), "this dva binary") {
		t.Errorf("error = %q, want it to name the binary as the faulty side", err)
	}
	if strings.Contains(err.Error(), "Omit `version:`") {
		t.Errorf("error = %q, must not advise a config edit for a build defect", err)
	}
}

// TestVersionPatternMatchesSchema keeps the two statements of the accepted `version:`
// shape in agreement. They must be two copies: schema.json's pattern runs only under
// `dva validate` (Config.Validate has one call site) while versionPattern runs on every
// Load, so neither can be derived from the other without making Load parse the schema.
// A test that fails on divergence is the cheaper half of that trade.
func TestVersionPatternMatchesSchema(t *testing.T) {
	raw, err := embeddedSchema.ReadFile("schema.json")
	if err != nil {
		t.Fatalf("read embedded schema.json: %v", err)
	}
	var doc struct {
		Properties struct {
			Version struct {
				Pattern string `json:"pattern"`
			} `json:"version"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("parse schema.json: %v", err)
	}
	if doc.Properties.Version.Pattern == "" {
		t.Fatal("schema.json properties.version has no pattern; the Go gate and the schema have diverged")
	}
	schemaPattern, err := regexp.Compile(doc.Properties.Version.Pattern)
	if err != nil {
		t.Fatalf("compile schema pattern %q: %v", doc.Properties.Version.Pattern, err)
	}

	for _, v := range []string{"0.1.44", "0.1", "v1.0.0", "10.20.30", "invalid", "O.2.0", "0.1.44-rc1", "1", "1.2.3.4", "", "V1.0.0", " 0.1.0", "0.1.0\n"} {
		_, goErr := parseVersion(v)
		goAccepts := goErr == nil
		if schemaAccepts := schemaPattern.MatchString(v); schemaAccepts != goAccepts {
			t.Errorf("%q: schema accepts = %v, Go accepts = %v — the two rules have diverged", v, schemaAccepts, goAccepts)
		}
	}

	// One asymmetry is intentional, and is asserted rather than left for someone to
	// rediscover as a spurious divergence: a segment can be all digits and still not fit
	// in an int. The schema constrains shape only, so it accepts; parseVersion also has
	// to represent the value, so strconv.Atoi's range error rejects. Do not "fix" this by
	// adding the value to the loop above — the loop would then fail correctly.
	const tooWide = "99999999999999999999.0.0"
	if !schemaPattern.MatchString(tooWide) {
		t.Errorf("%q: schema pattern rejects it; this test's premise no longer holds", tooWide)
	}
	if _, err := parseVersion(tooWide); err == nil {
		t.Errorf("parseVersion(%q) error = nil, want an int-range rejection", tooWide)
	}
}
