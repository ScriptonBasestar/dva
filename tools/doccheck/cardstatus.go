package main

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
)

// cardZone describes one task-tracker directory and the status values a card is permitted to
// declare while it sits there.
//
// Resolved by longest matching path prefix — never by splitting the path on "/" and indexing a
// segment. tasks/archive/done/ exists, so an nth-segment reader classifies an archived done-card
// by the "done" segment instead of the "archive" one, checking it against the wrong permitted
// set. A peer session made exactly that mistake on 2026-09-03 and reported two closed cards
// (superseded, valid under archive/) as dangling because it compared them to the done/ zone's
// stricter set (TASK-287).
type cardZone struct {
	prefix    string
	permitted []string
	skip      bool // true for tasks/plan/: not a card, no status: field to check
}

// archiveCardStatuses are the status values permitted under an archive zone, whichever spelling
// it uses. `superseded` is permitted only here because every superseded card lives in the
// archive and is closed work; done/ is for cards that were actually completed in place.
var archiveCardStatuses = []string{"done", "superseded"}

// cardZones is the zone table TASK-287 froze. The board is mid-migration from `tasks/archive/`
// to `tasks/_archive/`, so cardZones carries one entry per spelling in archivePrefixes (defined
// in archive.go) rather than a single hardcoded prefix — both entries share archiveCardStatuses
// so the two spellings are judged identically, and resolveCardZone's longest-prefix rule still
// applies cleanly since the two prefixes never nest inside each other.
var cardZones = buildCardZones()

func buildCardZones() []cardZone {
	zones := make([]cardZone, 0, len(archivePrefixes)+8)
	for _, prefix := range archivePrefixes {
		zones = append(zones, cardZone{prefix: prefix, permitted: archiveCardStatuses})
	}
	return append(zones,
		cardZone{prefix: "tasks/done/", permitted: []string{"done"}},
		cardZone{prefix: "tasks/todo/", permitted: []string{"todo"}},
		cardZone{prefix: "tasks/issue/", permitted: []string{"todo"}},
		cardZone{prefix: "tasks/review/", permitted: []string{"review"}},
		cardZone{prefix: "tasks/doing/", permitted: []string{"doing"}},
		cardZone{prefix: "tasks/blocked/", permitted: []string{"blocked"}},
		// tasks/backlog/ is declared permitted, not skip, on purpose (TASK-414): skip would also
		// exempt backlog cards from checkDuplicateCardIDs and checkDuplicateFilenameNumbers, which
		// is the exact blind spot this task closes for tasks/review/. BACKLOG-009 was given
		// `status: backlog` to satisfy this.
		cardZone{prefix: "tasks/backlog/", permitted: []string{"backlog"}},
		cardZone{prefix: "tasks/plan/", skip: true},
	)
}

// declaredBoardDirs returns the top-level tasks/ directory names cardZones declares, derived
// from each zone's prefix rather than kept as a second hand-written list — a list separate from
// cardZones is exactly how tasks/review/ and tasks/backlog/ went undeclared before TASK-414.
func declaredBoardDirs() map[string]struct{} {
	dirs := make(map[string]struct{}, len(cardZones))
	for _, z := range cardZones {
		name := strings.TrimSuffix(strings.TrimPrefix(z.prefix, "tasks/"), "/")
		dirs[name] = struct{}{}
	}
	return dirs
}

// checkUndeclaredBoardDirectories reports every top-level tasks/ directory not declared in
// cardZones. A card sitting under such a directory is invisible to every zone-gated check —
// checkCardStatus, checkDuplicateCardIDs, checkDuplicateFilenameNumbers alike — which is exactly
// how tasks/review/ and tasks/backlog/ went unguarded until a duplicate id crossing review/ and
// todo/ was measured live (TASK-414). This guard makes the next such gap fail the gate instead of
// waiting for another live measurement to notice it.
//
// dirsSeen counts distinct top-level directories found under tasks/ in the inventory; a file
// written directly at tasks/ (e.g. tasks/README.md) is not a board directory and is not counted.
func checkUndeclaredBoardDirectories(inv []InventoryEntry) (dirsSeen, undeclared int, msgs []string) {
	declared := declaredBoardDirs()
	seen := map[string]struct{}{}
	for _, e := range inv {
		if isSymlinkMode(e.Mode) {
			continue
		}
		rest, ok := strings.CutPrefix(e.Path, "tasks/")
		if !ok {
			continue
		}
		dir, _, ok := strings.Cut(rest, "/")
		if !ok || dir == "" {
			continue
		}
		seen[dir] = struct{}{}
	}
	dirsSeen = len(seen)
	names := make([]string, 0, len(seen))
	for dir := range seen {
		names = append(names, dir)
	}
	sort.Strings(names)
	for _, dir := range names {
		if _, ok := declared[dir]; ok {
			continue
		}
		undeclared++
		msgs = append(msgs, fmt.Sprintf(
			"tasks/%s/: undeclared board directory — add it to cardZones (permitted or skip) before its cards are covered by any zone-gated check",
			dir))
	}
	return dirsSeen, undeclared, msgs
}

// resolveCardZone returns the zone governing path, chosen by the longest matching prefix among
// cardZones. ok is false when path is not under any declared zone (e.g. tasks/README.md).
func resolveCardZone(path string) (zone cardZone, ok bool) {
	bestLen := -1
	for _, z := range cardZones {
		if strings.HasPrefix(path, z.prefix) && len(z.prefix) > bestLen {
			zone = z
			bestLen = len(z.prefix)
			ok = true
		}
	}
	return zone, ok
}

// frontmatterField reads a top-level scalar value out of frontmatter. Only a top-level key
// counts — an indented key belongs to the mapping above it, the same reasoning
// hasCanonicalField applies to id:/type: in archive.go. A repeated key is an error instead of
// silently choosing one value, matching YAML's duplicate-mapping-key rule.
func frontmatterField(frontmatter, want string) (value string, found bool, err error) {
	for line := range strings.SplitSeq(stripFencedRegions(frontmatter), "\n") {
		if line == "" || line[0] == ' ' || line[0] == '\t' || line[0] == '#' || line[0] == '-' {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if unquoteKey(strings.TrimSpace(key)) != want {
			continue
		}
		if found {
			return "", false, fmt.Errorf("repeated frontmatter key %q", want)
		}
		value = frontmatterValue(val)
		found = true
	}
	return value, found, nil
}

// frontmatterValue removes an inline YAML comment from a plain scalar, then unwraps exactly one
// matching outer quote pair. A hash inside a quoted scalar is data, not a comment.
func frontmatterValue(raw string) string {
	value := strings.TrimSpace(raw)
	if len(value) == 0 {
		return value
	}
	if value[0] == '#' {
		return ""
	}
	quote := value[0]
	if quote != '"' && quote != '\'' {
		return stripFrontmatterComment(value)
	}
	end, ok := matchingQuoteEnd(value, quote)
	if !ok {
		// Leave malformed or mismatched quoted text intact. This small reader is not a YAML
		// parser, so it must not guess where its scalar ends.
		return value
	}
	if tail := strings.TrimSpace(value[end+1:]); tail == "" || strings.HasPrefix(tail, "#") {
		return value[1:end]
	}
	// The closing quote is not an outer pair when ordinary text follows it. Treat this as a
	// plain scalar for comment stripping, but preserve its quotes.
	return stripFrontmatterComment(value)
}

// matchingQuoteEnd returns the closing quote for a scalar whose first byte is quote. Double
// quotes may escape a byte with backslash; YAML single-quote doubling is skipped as content.
func matchingQuoteEnd(value string, quote byte) (int, bool) {
	for i := 1; i < len(value); i++ {
		ch := value[i]
		if quote == '"' && ch == '\\' && i+1 < len(value) {
			i++
			continue
		}
		if quote == '\'' && ch == '\'' && i+1 < len(value) && value[i+1] == '\'' {
			i++
			continue
		}
		if ch == quote {
			return i, true
		}
	}
	return 0, false
}

func stripFrontmatterComment(value string) string {
	for i := 1; i < len(value); i++ {
		if value[i] != '#' {
			continue
		}
		if value[i-1] == ' ' || value[i-1] == '\t' {
			return strings.TrimSpace(value[:i])
		}
	}
	return value
}

// cardStatus reads the top-level `status:` value out of frontmatter.
func cardStatus(frontmatter string) (value string, found bool, err error) {
	return frontmatterField(frontmatter, "status")
}

// checkCardStatus reports every task card whose status: is not permitted in the zone it sits in.
//
// seen counts every non-symlink file resolved to a card zone (tasks/plan/ excluded, since it is
// not a card zone), regardless of whether it is markdown or readable; checked counts cards
// actually evaluated. The pair lets the caller distinguish "no card zone exists in this tree" —
// seen==0, a legitimate state for a fixture that builds no tasks/ directory at all, mirroring
// ArchiveFilesSeen's reasoning in archive.go — from "the zone walk found files but the sweep
// never evaluated any of them", seen>0 && checked==0, which is the walk being broken (TASK-287).
func checkCardStatus(root string, inv []InventoryEntry) (seen, checked, mismatches int, msgs, errs []string) {
	for _, e := range inv {
		if isSymlinkMode(e.Mode) {
			continue
		}
		zone, ok := resolveCardZone(e.Path)
		if !ok || zone.skip {
			continue
		}
		seen++
		if !isMarkdownPath(e.Path) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: read: %v", e.Path, err))
			continue
		}
		checked++
		frontmatter, state := splitFrontmatter(string(data))
		permitted := strings.Join(zone.permitted, "|")
		if state != frontmatterOK {
			mismatches++
			msgs = append(msgs, fmt.Sprintf("%s: zone %s permits status: %s, but %s",
				e.Path, zone.prefix, permitted, malformedFrontmatterReason(state)))
			continue
		}
		status, found, err := cardStatus(frontmatter)
		if err != nil {
			mismatches++
			msgs = append(msgs, fmt.Sprintf("%s: %v", e.Path, err))
			continue
		}
		if !found {
			mismatches++
			msgs = append(msgs, fmt.Sprintf("%s: zone %s requires a status: field (permitted: %s), but frontmatter carries none",
				e.Path, zone.prefix, permitted))
			continue
		}
		if !slices.Contains(zone.permitted, status) {
			mismatches++
			msgs = append(msgs, fmt.Sprintf("%s: zone %s permits status: %s, found %q",
				e.Path, zone.prefix, permitted, status))
		}
	}
	return seen, checked, mismatches, msgs, errs
}
