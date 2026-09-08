package config

import (
	"fmt"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// MigrateSectionOrder rewrites dva.yml's top-level keys into canonicalSectionOrder —
// the order validateCanonicalOrder's "section order:" warning asks for — without
// re-encoding a single value.
//
// The document is partitioned into contiguous line-range blocks, one per top-level
// key, and the blocks are permuted: nothing inside a block is touched, so every
// comment, inline comment and blank-line separator a block carries moves with it
// verbatim. This is the same technique MigrateLegacyCompose and MigrateStackOrder use
// for the reason their own comments give — a whole-document yaml.v3 round-trip does
// not model blank lines, so it would strip every separator in the file.
//
// Only keys present in canonicalSectionOrder are candidates for relocation. A key
// that is not on that list (unknown today, or a future addition) keeps its own slot
// — position and content both — so this command never has to guess where such a key
// belongs. The canonical keys then fill the remaining slots in canonical order.
//
// Block boundaries: a block starts at its key's own line, extended upward across
// *contiguous* comment lines — a comment separated from the key by a blank line
// belongs to the previous block instead, which matches yaml.v3's own HeadComment
// attachment. A block ends the line before the next block starts (after that
// extension), and the blank lines at that seam are then split back off as the slot's
// separator rather than moved as the block's content — see slotSeparator below.
//
// The document's two ends are fixed, and for the same reason. The first block is never
// extended upward: whatever sits above it — a leading `---`, a file header comment —
// is a document preamble, not that key's section banner, so it stays at the top even
// when the key below it moves. The last block stops symmetrically, at a document
// boundary or a blank-separated footer comment rather than at EOF. Treating either end
// as ordinary block content drags it away with a key, and in the boundary case that
// silently discards the config (see the end[n-1] comment below).
func MigrateSectionOrder(src []byte) ([]byte, MigrationReport, error) {
	var report MigrationReport

	var doc yaml.Node
	if err := yaml.Unmarshal(src, &doc); err != nil {
		return nil, report, fmt.Errorf("parse: %w", err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return src, report, nil
	}
	root := doc.Content[0]
	n := len(root.Content) / 2
	if n < 2 {
		return src, report, nil
	}

	keys := make([]string, n)
	keyLines := make([]int, n)
	for i := range n {
		keys[i] = root.Content[2*i].Value
		keyLines[i] = root.Content[2*i].Line
	}

	// Two shapes this line-range partition cannot represent. Both return the source
	// untouched rather than cutting the file at a boundary that does not exist — the
	// other Migrate steps still run, and the loader still reports what is wrong.
	//
	// A duplicate top-level key: yaml.Node keeps both halves, so the same name occupies
	// two slots while there is only one canonical position to move it to. Reordering is
	// not merely hard here but ill-defined, and `dva validate` already rejects the file
	// with "mapping key ... already defined". Migrate runs before VerifyMigrated, so
	// without this the command panics on exactly the broken file it exists to repair.
	//
	// Keys sharing a line: a flow-style mapping ({a: 1, b: 2}) gives no line that
	// belongs to one key alone, and the comment-extension below can then push a later
	// key's start above an earlier one's, inverting a slice bound.
	seen := make(map[string]bool, n)
	for i, k := range keys {
		if seen[k] {
			return src, report, nil
		}
		seen[k] = true
		if i > 0 && keyLines[i] <= keyLines[i-1] {
			return src, report, nil
		}
	}

	// Relative order of the canonical keys already present is the same check
	// validateCanonicalOrder runs. A file that passes it here produces
	// byte-identical output — nothing to reorder means nothing gets touched.
	var canonicalPresent []string
	for _, k := range keys {
		if _, ok := canonicalOrderIndex[k]; ok {
			canonicalPresent = append(canonicalPresent, k)
		}
	}
	inOrder := true
	for i := 1; i < len(canonicalPresent); i++ {
		if canonicalOrderIndex[canonicalPresent[i]] < canonicalOrderIndex[canonicalPresent[i-1]] {
			inOrder = false
			break
		}
	}
	if inOrder {
		return src, report, nil
	}

	// strings.Split turns a trailing "\n" into a synthetic empty final element that
	// is not an authored line — it just records that the file ends in a newline. Left
	// in, it gets folded into whichever block happens to be last and, once that
	// block is no longer last after reordering, resurfaces as a blank line in the
	// middle of the file. Trimming it here and re-adding exactly one trailing
	// newline at the very end (once the new last block's own content does not
	// already supply one) keeps every remaining blank line authored rather than
	// artifactual.
	text := string(src)
	trailingNewline := strings.HasSuffix(text, "\n")
	lines := strings.Split(strings.TrimSuffix(text, "\n"), "\n")

	// start/end are 1-based inclusive line ranges. Block 0 never absorbs a leading
	// comment — see the preamble note in the doc comment above — every later block's
	// start does.
	start := make([]int, n)
	start[0] = keyLines[0]
	for i := 1; i < n; i++ {
		start[i] = commentExtendedStart(lines, keyLines[i])
	}
	end := make([]int, n)
	for i := 0; i < n-1; i++ {
		end[i] = start[i+1] - 1
	}

	// The last block runs to a document boundary, not to EOF. yaml.Unmarshal decodes
	// only the first document, so keys/keyLines describe that document alone; letting
	// the last block absorb everything below it would carry a `...` terminator — or a
	// whole second document — along as if it were that key's content. Relocated to the
	// top, a terminator turns every section under it into a second document, which the
	// loader then ignores: the file still parses, `dva validate` still passes, and the
	// config is gone. Everything from the boundary on is a fixed postamble instead.
	end[n-1] = len(lines)
	postambleStart := 0
	for i := keyLines[n-1] + 1; i <= len(lines); i++ {
		if isDocumentBoundary(lines[i-1]) {
			end[n-1] = i - 1
			postambleStart = i
			break
		}
	}

	// A blank-separated comment run at EOF is a file footer and stays put, for the same
	// reason the header above the first key does. The tail is otherwise the one place
	// this walk is asymmetric: a licence footer or a `# vim:` line would be the last
	// block's content, so it rides that block to wherever the block lands — for a file
	// whose last section belongs first, that is the top of the document.
	if postambleStart == 0 {
		i := len(lines)
		for i > keyLines[n-1] && strings.HasPrefix(lines[i-1], "#") {
			i--
		}
		if i < len(lines) && i > keyLines[n-1] && strings.TrimSpace(lines[i-1]) == "" {
			end[n-1] = i
			postambleStart = i + 1
		}
	}

	preamble := lines[:start[0]-1]

	// A blank-line separator belongs to the slot, not to the block that happens to sit
	// in it. The last slot has no separator by construction and the others carry
	// whatever the author wrote, so a block that moves out of the last slot arrives
	// somewhere in the middle with nothing after it — gluing its successor onto its
	// final line — while the block that takes its place brings a now-trailing blank.
	// Splitting each block into content plus its slot's separator and permuting only
	// the content leaves the document's vertical rhythm exactly as authored.
	blockText := make([]string, n)
	slotSeparator := make([]int, n)
	for i := range n {
		body := lines[start[i]-1 : end[i]]
		blanks := 0
		for len(body)-blanks > 0 && strings.TrimSpace(body[len(body)-blanks-1]) == "" {
			blanks++
		}
		blockText[i] = strings.Join(body[:len(body)-blanks], "\n")
		slotSeparator[i] = blanks
	}

	// Canonical keys fill the slots canonical keys currently occupy, in canonical
	// order. A non-canonical key's slot is simply skipped here, so its block text
	// (assigned below via the copy) is never overwritten.
	var canonicalSlots []int
	for i, k := range keys {
		if _, ok := canonicalOrderIndex[k]; ok {
			canonicalSlots = append(canonicalSlots, i)
		}
	}
	var orderedCanonical []string
	for _, s := range canonicalSectionOrder {
		if slices.Contains(canonicalPresent, s) {
			orderedCanonical = append(orderedCanonical, s)
		}
	}

	newBlockText := slices.Clone(blockText)
	newKeys := slices.Clone(keys)
	for j, slot := range canonicalSlots {
		name := orderedCanonical[j]
		idx := slices.Index(keys, name)
		newBlockText[slot] = blockText[idx]
		newKeys[slot] = name
	}

	// Assembled as lines rather than as a string so the separators stay countable:
	// every blank line in the output is one an author wrote, in the slot they wrote it.
	outLines := slices.Clone(preamble)
	for i, t := range newBlockText {
		outLines = append(outLines, strings.Split(t, "\n")...)
		for range slotSeparator[i] {
			outLines = append(outLines, "")
		}
	}
	if postambleStart > 0 {
		outLines = append(outLines, lines[postambleStart-1:]...)
	}
	out := strings.Join(outLines, "\n")
	// No line carries its own newline, so the trailing one is re-added exactly when
	// the original file had it — the mirror of the TrimSuffix above.
	if trailingNewline {
		out += "\n"
	}

	report.Changes = []string{
		fmt.Sprintf("section order: reordered to %s", strings.Join(newKeys, " → ")),
	}
	return []byte(out), report, nil
}

// commentExtendedStart returns the first line of the comment block directly above
// keyLine, or keyLine itself if there is none. "Directly above" means contiguous: a
// blank line stops the walk, so a comment separated from the key by one belongs to
// whatever block precedes it instead.
//
// Only a `#` in column 0 counts. A top-level key sits in column 0, so its banner does
// too; an indented `#` is inside the previous block — most dangerously the last line
// of a literal block scalar, where it is script text and not a comment at all. Left
// indented-tolerant, this walk tears `# TODO: ...` out of an `interaction.*.command: |`
// body and files it above the next key, and the result still loads as valid YAML, so
// VerifyMigrated and `dva validate` both pass while the user's script is a line short.
func commentExtendedStart(lines []string, keyLine int) int {
	start := keyLine
	for i := keyLine - 1; i >= 1; i-- {
		line := lines[i-1]
		if strings.TrimSpace(line) == "" {
			break
		}
		if !strings.HasPrefix(line, "#") {
			break
		}
		start = i
	}
	return start
}

// isDocumentBoundary reports whether line opens or closes a YAML document at the top
// level: `---`, `...`, or either followed by content on the same line.
func isDocumentBoundary(line string) bool {
	for _, marker := range []string{"---", "..."} {
		if line == marker || strings.HasPrefix(line, marker+" ") || strings.HasPrefix(line, marker+"\t") {
			return true
		}
	}
	return false
}
