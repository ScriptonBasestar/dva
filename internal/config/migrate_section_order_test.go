package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
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
// by a blank line: it must move with the section above it, not the one below. The
// blank itself does not move — it separates slot 0 from slot 1 before and after.
func TestMigrateSectionOrderCommentAboveBlankStaysWithPreviousSection(t *testing.T) {
	src := "plans:\n  x: 1\n# leftover note about plans\n\nversion: b\n"
	want := "version: b\n\nplans:\n  x: 1\n# leftover note about plans\n"

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

// TestMigrateSectionOrderKeepsIndentedHashInsideBlockScalar pins the column-0 rule in
// commentExtendedStart. The `#` line here is the last line of a literal block scalar —
// shell text, not a comment — and an indented-tolerant upward walk files it above
// `version` as that key's banner, deleting a line from the user's script. The result
// still loads as valid YAML, so neither VerifyMigrated nor `dva validate` catches it;
// only this test does.
func TestMigrateSectionOrderKeepsIndentedHashInsideBlockScalar(t *testing.T) {
	src := "interaction:\n  seed:\n    command: |\n      set -e\n      # TODO: keep this in the script\nversion: \"1\"\n"
	want := "version: \"1\"\ninteraction:\n  seed:\n    command: |\n      set -e\n      # TODO: keep this in the script\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
}

// TestMigrateSectionOrderBailsOnUnrepresentableShapes covers the two document shapes a
// line-range partition cannot express. Both must return the source untouched — and,
// for the duplicate key, must not panic: Migrate runs before VerifyMigrated, so the
// duplicate reaches this code on exactly the broken file `dva config migrate` exists
// to repair.
func TestMigrateSectionOrderBailsOnUnrepresentableShapes(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			// Two slots for one name, one canonical position to move it to.
			name: "duplicate top-level key",
			src:  "stack:\n  db: 1\nversion: \"a\"\nversion: \"b\"\n",
		},
		{
			// No line belongs to one key alone, so no key has a range of its own.
			name: "flow-style root mapping",
			src:  "{stack: b, version: a}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, report, err := MigrateSectionOrder([]byte(tt.src))
			if err != nil {
				t.Fatalf("MigrateSectionOrder() error = %v", err)
			}
			if string(out) != tt.src {
				t.Errorf("expected the source back untouched, got:\n%q", out)
			}
			if len(report.Changes) != 0 {
				t.Errorf("report.Changes = %v, want none", report.Changes)
			}
		})
	}
}

func TestMigrateSectionOrderBlocksDuplicateKeyReason(t *testing.T) {
	src := "stack:\n  db: 1\nversion: \"a\"\nversion: \"b\"\n"
	wantBlocked := "section order: duplicate top-level key \"version\" — reordering is undefined; fix it by hand"

	out, report, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != src {
		t.Errorf("expected the source back untouched, got:\n%q", out)
	}
	if len(report.Blocked) != 1 || report.Blocked[0] != wantBlocked {
		t.Errorf("report.Blocked = %q, want %q", report.Blocked, wantBlocked)
	}
}

func TestMigrateSectionOrderBlocksFlowStyleRootReason(t *testing.T) {
	src := "{stack: b, version: a}\n"
	wantBlocked := "section order: flow-style root mapping puts two keys on one line — reorder by hand"

	out, report, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != src {
		t.Errorf("expected the source back untouched, got:\n%q", out)
	}
	if len(report.Blocked) != 1 || report.Blocked[0] != wantBlocked {
		t.Errorf("report.Blocked = %q, want %q", report.Blocked, wantBlocked)
	}
}

// TestMigrateSectionOrderBailsOnAnchorBelowAlias keeps the original bytes whenever
// YAML references appear anywhere in the document. Moving whole top-level blocks
// can put an anchor below its alias, which makes an otherwise valid document fail
// to parse. An unused anchor gets the same conservative treatment because the
// rewriter does not have a safe way to distinguish it from a future reference.
func TestMigrateSectionOrderBailsOnAnchorBelowAlias(t *testing.T) {
	const wantBlocked = "section order: not reordered — YAML anchors or aliases can depend on section order; reorder by hand"
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "anchor moves below alias",
			src:  "environment: &e\n  A: \"1\"\nversion: \"1\"\nvars: *e\n",
		},
		{
			name: "anchor without an alias",
			src:  "environment: &defaults\n  A: \"1\"\nversion: \"1\"\nvars:\n  B: \"2\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, report, err := MigrateSectionOrder([]byte(tt.src))
			if err != nil {
				t.Fatalf("MigrateSectionOrder() error = %v", err)
			}
			if string(out) != tt.src {
				t.Fatalf("expected the source back untouched, got:\n%q", out)
			}
			if len(report.Changes) != 0 {
				t.Errorf("report.Changes = %v, want none", report.Changes)
			}
			if len(report.Blocked) != 1 || report.Blocked[0] != wantBlocked {
				t.Errorf("report.Blocked = %q, want %q", report.Blocked, wantBlocked)
			}

			var probe yaml.Node
			if err := yaml.Unmarshal(out, &probe); err != nil {
				t.Fatalf("output does not parse: %v", err)
			}
		})
	}
}

// TestMigrateSectionOrderAlreadyCanonicalWithAnchorKeepsNoOpContract makes the
// anchor guard conditional on a needed rewrite. A canonical document must retain
// MigrateSectionOrder's established byte-identical, empty-report fast path.
func TestMigrateSectionOrderAlreadyCanonicalWithAnchorKeepsNoOpContract(t *testing.T) {
	src := []byte("version: \"1\"\nvars: &defaults\n  A: \"1\"\nenvironment: *defaults\n")

	out, report, err := MigrateSectionOrder(src)
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if !bytes.Equal(out, src) {
		t.Errorf("expected unchanged output, got:\n%q", out)
	}
	if len(report.Changes) != 0 {
		t.Errorf("report.Changes = %v, want none", report.Changes)
	}
	if len(report.Blocked) != 0 {
		t.Errorf("report.Blocked = %v, want none", report.Blocked)
	}
}

// TestMigrateSectionOrderStopsAtDocumentBoundary is the silent-data-loss regression.
// keys/keyLines describe the first document only, so letting the last block run to EOF
// carries whatever follows a `...` or `---` along as that key's content. Hoisted to the
// top with the key, a boundary marker turns every section below it into a second
// document that the loader ignores: the file still parses and `dva validate` still
// passes while the config is gone.
func TestMigrateSectionOrderStopsAtDocumentBoundary(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "explicit end marker",
			src:  "stack:\n  db:\n    plugin: compose\nversion: \"1\"\n...\n",
			want: "version: \"1\"\nstack:\n  db:\n    plugin: compose\n...\n",
		},
		{
			name: "second document",
			src:  "plans:\n  dev: {}\nversion: \"1\"\n---\nother: doc\n",
			want: "version: \"1\"\nplans:\n  dev: {}\n---\nother: doc\n",
		},
		// The two above with CRLF endings. The split leaves a `\r` on every line, so the
		// boundary test compared "...\r" against "..." and never matched: the terminator
		// was hoisted with its key and everything below it became a second document the
		// loader drops. It reproduced end to end — `dva validate` green, `dva run` on the
		// vanished interaction "not recognized" — so LF-only coverage was not coverage.
		{
			name: "explicit end marker with CRLF endings",
			src:  "stack:\r\n  db:\r\n    plugin: compose\r\nversion: \"1\"\r\n...\r\n",
			want: "version: \"1\"\r\nstack:\r\n  db:\r\n    plugin: compose\r\n...\r\n",
		},
		{
			name: "second document with CRLF endings",
			src:  "plans:\r\n  dev: {}\r\nversion: \"1\"\r\n---\r\nother: doc\r\n",
			want: "version: \"1\"\r\nplans:\r\n  dev: {}\r\n---\r\nother: doc\r\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := MigrateSectionOrder([]byte(tt.src))
			if err != nil {
				t.Fatalf("MigrateSectionOrder() error = %v", err)
			}
			if string(out) != tt.want {
				t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, tt.want)
			}
		})
	}
}

// TestMigrateSectionOrderKeepsSlotSeparators pins the separator-belongs-to-the-slot
// rule on the shape that exposes it: the block that has to leave the last slot is the
// only one carrying no trailing blank, so moving separators with blocks glues the
// section that follows onto its final line. examples/modules/main.yml is exactly this
// file — `provision` last, `modules` bound for the tail — and it came out one line
// shorter with `modules:` welded to the end of the last provision step.
func TestMigrateSectionOrderKeepsSlotSeparators(t *testing.T) {
	src := "modules:\n  - sast\n\nversion: \"1\"\n\nprovision:\n  - step: build\n"
	want := "version: \"1\"\n\nprovision:\n  - step: build\n\nmodules:\n  - sast\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
	if got, wantN := strings.Count(string(out), "\n"), strings.Count(src, "\n"); got != wantN {
		t.Errorf("line count changed: %d → %d", wantN, got)
	}
}

// TestMigrateSectionOrderKeepsKeepChompedTrailingBlank proves a blank line at
// the end of a keep-chomped block scalar remains part of the decoded value when
// its top-level section moves. A valid output is not enough here: dropping the
// blank still produces valid YAML while silently changing the command.
func TestMigrateSectionOrderKeepsKeepChompedTrailingBlank(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		indicator string
	}{
		{name: "literal", indicator: "|+"},
		{name: "folded", indicator: ">+"},
		{name: "anchor before literal", prefix: "&script ", indicator: "|+"},
		{name: "tag before folded", prefix: "!!str ", indicator: ">+"},
		{name: "anchor and tag before literal", prefix: "&script !!str ", indicator: "|+"},
		{name: "tag and anchor before folded", prefix: "!!str &script ", indicator: ">+"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "interaction:\n  seed:\n    command: " + tt.prefix + tt.indicator + "\n      hi\n\nversion: \"1\"\n"

			out, _, err := MigrateSectionOrder([]byte(src))
			if err != nil {
				t.Fatalf("MigrateSectionOrder() error = %v", err)
			}

			type document struct {
				Interaction map[string]struct {
					Command string `yaml:"command"`
				} `yaml:"interaction"`
			}
			decodeCommand := func(raw []byte) string {
				t.Helper()
				var doc document
				if err := yaml.Unmarshal(raw, &doc); err != nil {
					t.Fatalf("yaml.Unmarshal() error = %v\n%s", err, raw)
				}
				return doc.Interaction["seed"].Command
			}

			before := decodeCommand([]byte(src))
			after := decodeCommand(out)
			if before != "hi\n\n" {
				t.Fatalf("test fixture decoded command = %q, want %q", before, "hi\n\n")
			}
			if after != before {
				t.Fatalf("decoded command changed: before %q, after %q\n%s", before, after, out)
			}
		})
	}
}

// TestMigrateSectionOrderKeepsExplicitIndentKeepChompedTrailingBlank proves
// that an explicit indentation indicator, in either legal modifier order, sets
// the scalar's required indentation. The first content line may be deeper than
// that baseline without causing a later baseline-indented line to end the scalar.
func TestMigrateSectionOrderKeepsExplicitIndentKeepChompedTrailingBlank(t *testing.T) {
	tests := []struct {
		name      string
		prefix    string
		indicator string
	}{
		{name: "literal with anchor", prefix: "&script ", indicator: "|2+"},
		{name: "folded with tag", prefix: "!!str ", indicator: ">+2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "interaction:\n  seed:\n    command: " + tt.prefix + tt.indicator +
				"\n        deeper\n      baseline\n\nversion: \"1\"\n"

			out, _, err := MigrateSectionOrder([]byte(src))
			if err != nil {
				t.Fatalf("MigrateSectionOrder() error = %v", err)
			}

			type document struct {
				Interaction map[string]struct {
					Command string `yaml:"command"`
				} `yaml:"interaction"`
			}
			decodeCommand := func(raw []byte) string {
				t.Helper()
				var doc document
				if err := yaml.Unmarshal(raw, &doc); err != nil {
					t.Fatalf("yaml.Unmarshal() error = %v\n%s", err, raw)
				}
				return doc.Interaction["seed"].Command
			}

			before := decodeCommand([]byte(src))
			after := decodeCommand(out)
			if !strings.HasSuffix(before, "\n\n") {
				t.Fatalf("test fixture decoded command = %q, want two trailing newlines", before)
			}
			if after != before {
				t.Fatalf("decoded command changed: before %q, after %q\n%s", before, after, out)
			}
		})
	}
}

// TestMigrateSectionOrderNestedSequenceExplicitIndentKeepsSlotSeparator proves
// that an inline mapping inside a sequence takes its indentation base from the
// mapping key, not from the spaces before the sequence marker. With |2+ below,
// scalar content starts at eight spaces; the six-space comment ends the scalar,
// so the final blank remains the slot separator after reordering.
func TestMigrateSectionOrderNestedSequenceExplicitIndentKeepsSlotSeparator(t *testing.T) {
	src := "interaction:\n  seeds:\n    - command: |2+\n        hi\n      # seed tail\n\nversion: \"1\"\n"
	want := "version: \"1\"\n\ninteraction:\n  seeds:\n    - command: |2+\n        hi\n      # seed tail\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
}

// TestMigrateSectionOrderExplicitKeyIndentKeepsTrailingBlank proves an explicit
// mapping key does not move the scalar indentation base after the `? ` prefix.
// The mapping starts at four spaces, so |2+ accepts both the deeper eight-space
// first line and the later six-space line, including the final blank in its value.
func TestMigrateSectionOrderExplicitKeyIndentKeepsTrailingBlank(t *testing.T) {
	src := "interaction:\n  seed:\n    ? command\n    : |2+\n        deeper\n      baseline\n\nversion: \"1\"\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}

	type document struct {
		Interaction map[string]struct {
			Command string `yaml:"command"`
		} `yaml:"interaction"`
	}
	decodeCommand := func(raw []byte) string {
		t.Helper()
		var doc document
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			t.Fatalf("yaml.Unmarshal() error = %v\n%s", err, raw)
		}
		return doc.Interaction["seed"].Command
	}

	before := decodeCommand([]byte(src))
	after := decodeCommand(out)
	if !strings.HasSuffix(before, "\n\n") {
		t.Fatalf("test fixture decoded command = %q, want two trailing newlines", before)
	}
	if after != before {
		t.Fatalf("decoded command changed: before %q, after %q\n%s", before, after, out)
	}
}

// TestMigrateSectionOrderKeepChompedScalarBeforeTailKeepsSlotSeparator proves
// that finding |+ inside a top-level block is insufficient to claim the block's
// final blank. A later field or YAML comment ends the scalar, so the blank after
// that tail remains attached to the slot and still separates the reordered keys.
func TestMigrateSectionOrderKeepChompedScalarBeforeTailKeepsSlotSeparator(t *testing.T) {
	tests := []struct {
		name string
		tail string
	}{
		{name: "later field", tail: "    shell: bash\n"},
		{name: "later comment", tail: "    # seed command note\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := "interaction:\n  seed:\n    command: |+\n      hi\n" + tt.tail + "\nversion: \"1\"\n"
			want := "version: \"1\"\n\ninteraction:\n  seed:\n    command: |+\n      hi\n" + tt.tail

			out, _, err := MigrateSectionOrder([]byte(src))
			if err != nil {
				t.Fatalf("MigrateSectionOrder() error = %v", err)
			}
			if string(out) != want {
				t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
			}
		})
	}
}

// TestMigrateSectionOrderKeepMarkerInCommentDoesNotClaimSeparator pins the
// lexical boundary of the header check. The parsed scalar is literal, but the +
// appears only in its inline comment; treating that text as a keep indicator
// would move the slot's blank line to EOF.
func TestMigrateSectionOrderKeepMarkerInCommentDoesNotClaimSeparator(t *testing.T) {
	src := "interaction:\n  seed:\n    command: | # documentation mentions |+\n      hi\n\nversion: \"1\"\n"
	want := "version: \"1\"\n\ninteraction:\n  seed:\n    command: | # documentation mentions |+\n      hi\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
}

// TestMigrateSectionOrderKeepsFooterCommentAtEOF is the tail half of the preamble
// symmetry. A licence or `# vim:` footer separated from the last section by a blank
// line is document furniture, not that section's content; absorbed into the block it
// follows, it rides that block to the block's new slot — which for a last section
// that sorts first is the top of the file.
func TestMigrateSectionOrderKeepsFooterCommentAtEOF(t *testing.T) {
	src := "plans:\n  x: 1\n\nversion: \"1\"\n\n# vim: set ft=yaml:\n"
	want := "version: \"1\"\n\nplans:\n  x: 1\n\n# vim: set ft=yaml:\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
}

// TestMigrateSectionOrderKeepsEveryTrailingCommentParagraph ensures every
// blank-separated EOF comment paragraph remains file furniture. Each paragraph
// must stay after the reordered sections rather than travelling with the last
// source section.
func TestMigrateSectionOrderKeepsEveryTrailingCommentParagraph(t *testing.T) {
	src := "plans:\n  x: 1\n\nversion: \"1\"\n\n# licence notice\n\n# vim: set ft=yaml:\n"
	want := "version: \"1\"\n\nplans:\n  x: 1\n\n# licence notice\n\n# vim: set ft=yaml:\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
}

// TestMigrateSectionOrderDoesNotTreatQuotedScalarContentAsFooter ensures the
// EOF scan does not mistake a column-zero # line inside a double-quoted scalar
// for a comment paragraph. The scalar deliberately crosses a blank line, the
// same shape that a trailing footer uses.
func TestMigrateSectionOrderDoesNotTreatQuotedScalarContentAsFooter(t *testing.T) {
	src := "plans:\n  x: 1\n\nversion: \"value\n\n# ending quote\"\n"
	want := "version: \"value\n\n# ending quote\"\n\nplans:\n  x: 1\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}

	type document struct {
		Plans struct {
			X int `yaml:"x"`
		} `yaml:"plans"`
		Version string `yaml:"version"`
	}
	var before, after document
	if err := yaml.Unmarshal([]byte(src), &before); err != nil {
		t.Fatalf("input does not parse: %v", err)
	}
	if err := yaml.Unmarshal(out, &after); err != nil {
		t.Fatalf("output does not parse: %v", err)
	}
	if after != before {
		t.Fatalf("decoded config changed: got %#v, want %#v", after, before)
	}
}

// TestMigrateSectionOrderTrustsHeadCommentOverColumnZero pins the two shapes that the
// column-0 heuristic gets wrong and yaml.v3's own HeadComment gets right. A double-quoted
// scalar on a top-level key continues at column 0, and that continuation may begin with
// `#` — which is a character in the value, not a comment. The heuristic filed such a line
// above the next key and split the scalar; the output no longer parsed, so VerifyMigrated
// refused the whole migration of a file that had been valid going in.
//
// The second case is why the empty check alone is not enough: the key really does have a
// banner, so the walk runs, and unbounded it takes the scalar's tail with it. The walk is
// bounded by the number of lines yaml reported in the HeadComment.
func TestMigrateSectionOrderTrustsHeadCommentOverColumnZero(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want string
	}{
		{
			name: "column-zero hash inside a quoted scalar is not a banner",
			src:  "stack: \"x\n#y\"\nversion: \"1\"\n",
			want: "version: \"1\"\nstack: \"x\n#y\"\n",
		},
		{
			name: "walk stops at the banner yaml reported, not at the scalar above it",
			src:  "stack: \"x\n#not a comment\"\n# real banner\nversion: \"1\"\n",
			want: "# real banner\nversion: \"1\"\nstack: \"x\n#not a comment\"\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, _, err := MigrateSectionOrder([]byte(tt.src))
			if err != nil {
				t.Fatalf("MigrateSectionOrder() error = %v", err)
			}
			if string(out) != tt.want {
				t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, tt.want)
			}
			// The failure this guards is "the output does not parse at all", so assert
			// that directly rather than only on the bytes.
			var probe yaml.Node
			if err := yaml.Unmarshal(out, &probe); err != nil {
				t.Fatalf("output does not parse: %v", err)
			}
		})
	}
}

// TestMigrateSectionOrderBoundsTheBannerWalkOnCRLFToo pins the CRLF half of the bound M7
// added. Trusting yaml.v3's HeadComment was the right move, but the line count derived
// from that string is a second heuristic, and on CRLF input the string it reads is wrong:
// for a three-line banner yaml.v3 reports the second line onward, padded with empty
// entries and a trailing newline, so the count came out one too high. One extra step is
// all it takes — the walk reaches the continuation line of the quoted scalar above the
// banner, files it under the next key, and the output stops parsing. Same failure as M7,
// on the axis M7 was not measured against.
//
// The fix hands the parser CRLF-normalized bytes and keeps the original bytes for output.
// `want` is CRLF throughout, so the byte comparison already pins the preservation half —
// a separate "no bare LF" assertion was written here and removed as unreachable: the
// Fatalf above it ends the function on any mismatch, and TestMigrateSectionOrder-
// StopsAtDocumentBoundary's two CRLF subtests fail independently on that regression.
func TestMigrateSectionOrderBoundsTheBannerWalkOnCRLFToo(t *testing.T) {
	src := "stack: \"x\r\n#cont\"\r\n# b1\r\n# b2\r\n# b3\r\nversion: \"1\"\r\n"
	want := "# b1\r\n# b2\r\n# b3\r\nversion: \"1\"\r\nstack: \"x\r\n#cont\"\r\n"

	out, _, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
	// The observed failure was "found unexpected end of stream" — the scalar was torn in
	// half — so assert parseability directly rather than only on bytes.
	var probe yaml.Node
	if err := yaml.Unmarshal(out, &probe); err != nil {
		t.Fatalf("output does not parse: %v", err)
	}
}

// TestMigrateSectionOrderRefusesWhenLineBreaksDisagree pins the guard on the one
// assumption every index in this function rests on: that yaml.v3 and strings.Split agree
// on where the lines are. On a file whose breaks are lone CRs they do not — yaml.v3 reads
// "\r" as a break and Split on "\n" does not — and the three panics below came out of
// files `dva config migrate` exists to repair, which is the worst possible place to die.
//
// The fourth case is the one that motivated reporting rather than silently returning src.
// A stray lone CR among LF lines does not panic; before the guard it prepended a blank
// line and reordered nothing, so the file was rewritten and the warning it was rewritten
// to clear survived. Mixed-newline files are not hypothetical: before Migrate normalized
// its working copy at the pipeline boundary, earlier steps could produce them. This
// function also remains callable directly, so its own guard must reject such input.
//
// Asserting the invariant rather than enumerating newline conventions is the point. A
// fourth convention would break an enumeration; it cannot break this.
func TestMigrateSectionOrderRefusesWhenLineBreaksDisagree(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"lone CR", "plans:\r  x: 1\rversion: \"1\"\r"},
		{"lone CR with three keys", "plans:\r  x: 1\rstack:\r  db: 1\rversion: \"1\"\r"},
		{"lone CR with a banner", "plans:\r  x: 1\r# banner\rversion: \"1\"\r"},
		{"one lone CR among LF lines", "plans:\r  x: 1\nversion: \"1\"\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A panic here is the failure being guarded, so no recover: let it fail loudly.
			out, report, err := MigrateSectionOrder([]byte(tt.src))
			if err != nil {
				t.Fatalf("MigrateSectionOrder() error = %v", err)
			}
			if string(out) != tt.src {
				t.Fatalf("file was rewritten despite not being reordered:\ngot  %q\nwant %q", out, tt.src)
			}
			if len(report.Blocked) == 0 {
				t.Fatal("returned the source unchanged and said nothing: the reader is left " +
					"with the same `section order:` warning and no reason it was not fixed")
			}
			// The reason has to name a fix, not just a refusal -- this is the bail-out
			// SHOULD-FIX 5 would otherwise have to come back for.
			if !strings.Contains(report.Blocked[0], "LF or CRLF") {
				t.Errorf("Blocked reason does not tell the reader what to do: %q", report.Blocked[0])
			}
		})
	}
}
