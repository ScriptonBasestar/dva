package config

import (
	"bytes"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// migrateAndLoad runs the migration and proves the result is something Load()
// would accept, which is the only outcome that matters: a migration that emits
// unloadable YAML is worse than no migration at all.
func migrateAndLoad(t *testing.T, src string) string {
	t.Helper()

	out, migrated, err := MigrateLegacyCompose([]byte(src))
	if err != nil {
		t.Fatalf("MigrateLegacyCompose() error = %v", err)
	}
	if len(migrated) == 0 {
		t.Fatalf("MigrateLegacyCompose() reported no change for a legacy config:\n%s", src)
	}
	if err := VerifyMigrated(out); err != nil {
		t.Fatalf("migrated config does not load: %v\n%s", err, out)
	}
	return string(out)
}

func TestMigrateAutoInferredCompose(t *testing.T) {
	got := migrateAndLoad(t, `version: "0.1.44"
stack:
  compose:
    order: 10
    files: [docker-compose.yml]
    project_name: app
`)

	for _, want := range []string{"default_runner: compose", "runners:", "files: [docker-compose.yml]"} {
		if !strings.Contains(got, want) {
			t.Errorf("migrated config missing %q:\n%s", want, got)
		}
	}
	// Anchored to line start: entry-level keys sit at 4 spaces, runner keys at 8.
	if !strings.Contains(got, "\n    order: 10") {
		t.Errorf("entry-level order: must stay on the entry:\n%s", got)
	}
	if strings.Contains(got, "\n    files:") {
		t.Errorf("files: must move under runners.compose, not stay at entry level:\n%s", got)
	}
}

func TestMigrateFlatComposePluginDropsPluginKey(t *testing.T) {
	got := migrateAndLoad(t, `version: "0.1.44"
stack:
  core:
    plugin: compose
    files: [compose.yml]
`)

	// plugin: compose is the exact key schema.json rejects; default_runner
	// carries the same meaning in the supported shape.
	if strings.Contains(got, "plugin: compose") {
		t.Errorf("plugin: compose must be dropped, not carried over:\n%s", got)
	}
	if !strings.Contains(got, "default_runner: compose") {
		t.Errorf("migrated entry needs default_runner: compose:\n%s", got)
	}
}

func TestMigrateNestedCompose(t *testing.T) {
	got := migrateAndLoad(t, `version: "0.1.44"
stack:
  core:
    order: 5
    compose:
      files: [compose.yml]
`)

	if strings.Contains(got, "\n    compose:\n") {
		t.Errorf("nested compose: sub-key must be replaced by runners.compose:\n%s", got)
	}
	if !strings.Contains(got, "runners:") || !strings.Contains(got, "files: [compose.yml]") {
		t.Errorf("nested compose config must land under runners.compose:\n%s", got)
	}
}

// TestMigrateDuplicatesTags pins the one lossy-looking choice in the transform.
// `tags` is the only key both LifecycleEntry and ComposePluginConfig declare, so
// a legacy flat entry fed stack-entry filtering and compose service-filter
// defaults from a single key. Moving it would silently drop one of the two.
func TestMigrateDuplicatesTags(t *testing.T) {
	got := migrateAndLoad(t, `version: "0.1.44"
stack:
  compose:
    tags: [infra]
    files: [compose.yml]
`)

	if n := strings.Count(got, "tags: [infra]"); n != 2 {
		t.Fatalf("tags must appear at entry level and under runners.compose, got %d:\n%s", n, got)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(got), cfg); err != nil {
		t.Fatal(err)
	}
	entry := cfg.Stack["compose"]
	entry.Name = "compose"
	if err := entry.ResolvePluginFromName(); err != nil {
		t.Fatal(err)
	}
	if len(entry.Tags) != 1 || entry.Tags[0] != "infra" {
		t.Errorf("entry tags = %v, want [infra]", entry.Tags)
	}
	if cc := entry.ComposeConfig(); cc == nil || len(cc.Tags) != 1 || cc.Tags[0] != "infra" {
		t.Errorf("compose runner tags = %v, want [infra]", cc)
	}
}

// TestMigratePreservesEverythingElse is the safety property that justifies
// splicing line spans instead of re-encoding the document: bytes outside a
// migrated entry, including comments and blank lines, must survive untouched.
func TestMigratePreservesEverythingElse(t *testing.T) {
	src := `version: "0.1.44"

# top-level comment survives
environment:
  FOO: bar

stack:
  compose:
    files: [compose.yml]   # trailing comment

interaction:
  hello:
    command: echo hi
`
	got := migrateAndLoad(t, src)

	for _, want := range []string{
		"# top-level comment survives",
		"version: \"0.1.44\"\n\n# top-level comment",
		"interaction:\n  hello:\n    command: echo hi",
		"# trailing comment",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("migration disturbed content outside the entry, missing %q:\n%s", want, got)
		}
	}
}

func TestMigrateLeavesModernConfigByteIdentical(t *testing.T) {
	src := []byte(`version: "0.1.44"
stack:
  core:
    default_runner: compose
    runners:
      compose:
        files: [compose.yml]
`)
	out, migrated, err := MigrateLegacyCompose(src)
	if err != nil {
		t.Fatalf("MigrateLegacyCompose() error = %v", err)
	}
	if len(migrated) != 0 {
		t.Errorf("modern config reported as migrated: %v", migrated)
	}
	if string(out) != string(src) {
		t.Errorf("modern config was rewritten:\n%s", out)
	}
}

func TestMigrateRefusesAmbiguousEntry(t *testing.T) {
	_, _, err := MigrateLegacyCompose([]byte(`version: "0.1.44"
stack:
  core:
    plugin: compose
    files: [legacy.yml]
    runners:
      compose:
        files: [modern.yml]
`))
	if err == nil {
		t.Fatal("expected an error when both shapes declare compose")
	}
	if !strings.Contains(err.Error(), "authoritative") {
		t.Errorf("error = %v, want it to explain why migration cannot decide", err)
	}
}

// This fixture deliberately activates five pipeline steps in one run: legacy compose,
// applications, stack order, modes, and final section order. Testing the full chain is
// the contract: step-local CRLF preservation would still allow a later encoder to turn
// the file into LF or a mixture of both.
const combinedMigrationFixture = `plans:
  release:
    entries:
      - name: compose
      - name: api
modes:
  local:
    stack: [compose, api]
applications:
  api:
    run: "./api"
stack:
  compose:
    order: 10
    files: [compose.yml]
version: "0.1.44"
`

func migrateCombinedCRLFFixture(t *testing.T) []byte {
	t.Helper()
	src := bytes.ReplaceAll([]byte(combinedMigrationFixture), []byte("\n"), []byte("\r\n"))
	out, report, err := Migrate(src)
	if err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := VerifyMigrated(out); err != nil {
		t.Fatalf("migrated config does not load: %v\n%s", err, out)
	}

	changes := strings.Join(report.Changes, "\n")
	for _, want := range []string{
		"stack.compose → runners.compose",
		"applications.api → stack.api",
		"stack.compose.order: 10 → plans.release.entries[compose].order",
		"modes.local → plans.local",
		"section order: reordered",
	} {
		if !strings.Contains(changes, want) {
			t.Errorf("combined migration did not activate %q:\n%s", want, changes)
		}
	}
	assertUniformCRLF(t, out)
	return out
}

func assertUniformCRLF(t *testing.T, content []byte) {
	t.Helper()
	if !bytes.Contains(content, []byte("\r\n")) {
		t.Fatal("output has no CRLF line ending")
	}
	withoutCRLF := bytes.ReplaceAll(content, []byte("\r\n"), nil)
	if bytes.ContainsAny(withoutCRLF, "\r\n") {
		t.Fatalf("output contains a bare CR or LF: %q", content)
	}
}

func TestMigratePipelinePreservesUniformCRLFAcrossCombinedSteps(t *testing.T) {
	out := migrateCombinedCRLFFixture(t)
	for _, want := range [][]byte{
		[]byte("version: \"0.1.44\"\r\nstack:"),
		[]byte("default_runner: compose\r\n"),
		[]byte("run: \"./api\"\r\n"),
		[]byte("order: 10\r\n"),
		[]byte("  local:\r\n"),
	} {
		if !bytes.Contains(out, want) {
			t.Errorf("combined migration output missing %q:\n%s", want, out)
		}
	}
}

// The boundary check belongs in Migrate, before MigrateLegacyCompose: that first step
// already combines yaml.v3 line numbers with strings.Split indexes and used to panic on
// an all-lone-CR legacy config before MigrateSectionOrder could report its own guard.
func TestMigratePipelineRefusesUnsupportedLineEndingsBeforeLineIndexing(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "lone CR",
			src:  "version: \"0.1.44\"\rstack:\r  core:\r    plugin: compose\r    files: [compose.yml]\r",
		},
		{
			name: "mixed LF and CRLF",
			src:  "version: \"0.1.44\"\r\nstack:\n  core:\r\n    plugin: compose\n    files: [compose.yml]\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, report, err := Migrate([]byte(tt.src))
			if err != nil {
				t.Fatalf("Migrate() error = %v", err)
			}
			if string(out) != tt.src {
				t.Fatalf("unsupported input was rewritten:\ngot  %q\nwant %q", out, tt.src)
			}
			if len(report.Changes) != 0 {
				t.Errorf("unsupported input reported conversions: %v", report.Changes)
			}
			blocked := strings.Join(report.Blocked, "\n")
			if !strings.Contains(blocked, "uniform LF or CRLF") {
				t.Errorf("Blocked does not explain the supported styles: %q", blocked)
			}
		})
	}
}
