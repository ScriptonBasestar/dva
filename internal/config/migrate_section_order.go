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
// extension), so a trailing blank-line separator travels with the block above it. The
// very first block in the file is the one exception: it is never extended upward.
// Whatever sits above it — a leading `---`, `%YAML` directives, a file header comment
// — is a document preamble, not that key's section banner, and it must stay fixed at
// the top of the file even when the key below it is relocated elsewhere. Treating it
// as an ordinary block would instead drag it away with that key.
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
	end[n-1] = len(lines)

	preamble := lines[:start[0]-1]

	blockText := make([]string, n)
	for i := range n {
		blockText[i] = strings.Join(lines[start[i]-1:end[i]], "\n")
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

	var out strings.Builder
	out.WriteString(strings.Join(preamble, "\n"))
	if len(preamble) > 0 {
		out.WriteString("\n")
	}
	for i, t := range newBlockText {
		out.WriteString(t)
		if i < n-1 {
			out.WriteString("\n")
		}
	}
	// The new last block's own trailing content decides whether this already ends
	// in a newline; only add one when the original file had one and this block's
	// text did not already supply it, so a moved separator is never doubled.
	if trailingNewline && !strings.HasSuffix(out.String(), "\n") {
		out.WriteString("\n")
	}

	report.Changes = []string{
		fmt.Sprintf("section order: reordered to %s", strings.Join(newKeys, " → ")),
	}
	return []byte(out.String()), report, nil
}

// commentExtendedStart returns the first line of the comment block directly above
// keyLine, or keyLine itself if there is none. "Directly above" means contiguous: a
// blank line stops the walk, so a comment separated from the key by one belongs to
// whatever block precedes it instead.
func commentExtendedStart(lines []string, keyLine int) int {
	start := keyLine
	for i := keyLine - 1; i >= 1; i-- {
		line := lines[i-1]
		if strings.TrimSpace(line) == "" {
			break
		}
		if !strings.HasPrefix(strings.TrimLeft(line, " \t"), "#") {
			break
		}
		start = i
	}
	return start
}
