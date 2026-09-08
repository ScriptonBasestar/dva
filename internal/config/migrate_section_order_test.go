package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCanonicalSectionOrderPreservesComments is the card's binding acceptance test:
// a file with a header comment, a section banner, an inline comment and blank-line
// separators comes out reordered with every comment still attached to the same
// section and every separator intact.
func TestCanonicalSectionOrderPreservesComments(t *testing.T) {
	src := `# dva.yml - project config
# generated for testing
interaction:
  test:
    command: make test  # run unit tests

version: "0.1.44"  # pinned version

# ---- stack ----
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [compose.yml]

plans:
  full:
    entries:
      - name: compose

`
	want := `# dva.yml - project config
# generated for testing
version: "0.1.44"  # pinned version

# ---- stack ----
stack:
  compose:
    default_runner: compose
    runners:
      compose:
        files: [compose.yml]

plans:
  full:
    entries:
      - name: compose

interaction:
  test:
    command: make test  # run unit tests
`

	out, report, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%s\nwant\n%s", out, want)
	}
	if err := VerifyMigrated(out); err != nil {
		t.Fatalf("reordered config does not load: %v\n%s", err, out)
	}
	if len(report.Changes) != 1 || !strings.Contains(report.Changes[0], "section order:") {
		t.Fatalf("report.Changes = %v, want one 'section order:' line", report.Changes)
	}
	const wantOrder = "version → stack → plans → interaction"
	if !strings.Contains(report.Changes[0], wantOrder) {
		t.Errorf("report.Changes[0] = %q, want it to name %q", report.Changes[0], wantOrder)
	}
}

// TestMigrateSectionOrderNonCanonicalKeyKeepsSlot proves an unknown top-level key is
// neither pushed to the front nor the back: it keeps the exact slot it started in
// while the canonical keys around it move into canonical order.
func TestMigrateSectionOrderNonCanonicalKeyKeepsSlot(t *testing.T) {
	src := "plans: a\nx_custom_extension: mystery\nversion: b\n"
	want := "version: b\nx_custom_extension: mystery\nplans: a\n"

	out, report, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%s\nwant\n%s", out, want)
	}
	if len(report.Changes) != 1 {
		t.Fatalf("report.Changes = %v, want exactly one line", report.Changes)
	}
}

// TestMigrateSectionOrderCommentAboveBlankStaysWithPreviousSection covers the case
// where a trailing comment on one section is separated from the *next* section's key
// by a blank line: it must move with the section above it, not the one below.
func TestMigrateSectionOrderCommentAboveBlankStaysWithPreviousSection(t *testing.T) {
	src := "plans:\n  x: 1\n# leftover note about plans\n\nversion: b\n"
	want := "version: b\nplans:\n  x: 1\n# leftover note about plans\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%s\nwant\n%s", out, want)
	}
}

// TestMigrateSectionOrderAlreadyCanonicalUnchanged proves a file already in
// canonical order is left byte-identical and produces no report line — this is what
// keeps `dva config migrate` printing "nothing to convert" for such a file.
func TestMigrateSectionOrderAlreadyCanonicalUnchanged(t *testing.T) {
	src := []byte("version: a\nstack: b\nplans: c\n")

	out, report, err := MigrateSectionOrder(src)
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if !bytes.Equal(out, src) {
		t.Errorf("expected unchanged output, got:\n%s", out)
	}
	if len(report.Changes) != 0 {
		t.Errorf("report.Changes = %v, want none for an already-ordered file", report.Changes)
	}
}

// TestMigrateSectionOrderIdempotent proves running the rewrite a second time is a
// no-op: the second pass sees a canonical file and changes nothing.
func TestMigrateSectionOrderIdempotent(t *testing.T) {
	src := []byte("plans: a\nversion: b\n")

	out1, report1, err := MigrateSectionOrder(src)
	if err != nil {
		t.Fatalf("first MigrateSectionOrder() error = %v", err)
	}
	if len(report1.Changes) == 0 {
		t.Fatal("first pass reported no change for an unordered file")
	}

	out2, report2, err := MigrateSectionOrder(out1)
	if err != nil {
		t.Fatalf("second MigrateSectionOrder() error = %v", err)
	}
	if !bytes.Equal(out1, out2) {
		t.Errorf("second pass changed the output:\nfirst:\n%s\nsecond:\n%s", out1, out2)
	}
	if len(report2.Changes) != 0 {
		t.Errorf("second pass report.Changes = %v, want none", report2.Changes)
	}
}

// TestMigrateSectionOrderClearsTheValidateWarning proves the rewrite actually
// satisfies the check that motivated it: validateCanonicalOrder's "section order:"
// warning fires on the input and is gone from the output.
func TestMigrateSectionOrderClearsTheValidateWarning(t *testing.T) {
	src := `version: "0.1.29"
interaction:
  test:
    command: make test
stack:
  compose:
    order: 10
`
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(src), 0644); err != nil {
		t.Fatal(err)
	}
	before := validateCanonicalOrder(path)
	if len(before) != 1 || !strings.Contains(before[0], "section order") {
		t.Fatalf("expected the input to trip the section-order warning, got %v", before)
	}

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if err := os.WriteFile(path, out, 0644); err != nil {
		t.Fatal(err)
	}

	after := validateCanonicalOrder(path)
	if len(after) != 0 {
		t.Errorf("expected no section-order warning after the rewrite, got %v:\n%s", after, out)
	}
}
