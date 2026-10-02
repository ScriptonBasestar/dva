package config

import (
	"bytes"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

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

// TestMigrateSectionOrderPreservesCRLFSeparator proves that the blank line owned
// by a slot retains the CRLF style of a uniformly CRLF source. The separator is
// created during output assembly, rather than copied from block text, so it needs
// the same newline choice as every authored source line.
func TestMigrateSectionOrderPreservesCRLFSeparator(t *testing.T) {
	src := "plans:\r\n  dev: {}\r\n\r\nversion: \"1\"\r\n"
	want := "version: \"1\"\r\n\r\nplans:\r\n  dev: {}\r\n"

	out, report, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
	if len(report.Blocked) != 0 {
		t.Errorf("report.Blocked = %v, want none", report.Blocked)
	}
	if bytes.Contains(out, []byte("\r\n\n")) {
		t.Errorf("output contains a bare-LF separator: %q", out)
	}
}

// TestMigrateSectionOrderPreservesLFSeparator keeps the existing LF-only
// rendering contract explicit while the CRLF path chooses its own separator.
func TestMigrateSectionOrderPreservesLFSeparator(t *testing.T) {
	src := "plans:\n  dev: {}\n\nversion: \"1\"\n"
	want := "version: \"1\"\n\nplans:\n  dev: {}\n"

	out, report, err := MigrateSectionOrder([]byte(src))
	if err != nil {
		t.Fatalf("MigrateSectionOrder() error = %v", err)
	}
	if string(out) != want {
		t.Fatalf("MigrateSectionOrder() =\n%q\nwant\n%q", out, want)
	}
	if len(report.Blocked) != 0 {
		t.Errorf("report.Blocked = %v, want none", report.Blocked)
	}
	if bytes.Contains(out, []byte("\r\n")) {
		t.Errorf("LF output unexpectedly contains CRLF: %q", out)
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
