package main

import (
	"fmt"
	"sort"
	"strings"
)

// cardZone describes one task-tracker directory that identity checks cover.
//
// CE owns the status contract through the repository's strict-status dialect
// (TASK-424). Doccheck keeps this table solely because duplicate-card-id and
// duplicate-filename-number checks need the same stable card scope. The
// longest-prefix rule matters for archived cards under paths such as
// tasks/_archive/done/: storage still belongs to the archive scope rather than
// the nested workflow directory.
type cardZone struct {
	prefix string
	skip   bool // true for tasks/plan/: plans are outside task-card identity checks
}

var cardZones = buildCardZones()

func buildCardZones() []cardZone {
	zones := make([]cardZone, 0, len(archivePrefixes)+8)
	for _, prefix := range archivePrefixes {
		zones = append(zones, cardZone{prefix: prefix})
	}
	return append(zones,
		cardZone{prefix: "tasks/done/"},
		cardZone{prefix: "tasks/todo/"},
		cardZone{prefix: "tasks/issue/"},
		cardZone{prefix: "tasks/review/"},
		cardZone{prefix: "tasks/doing/"},
		cardZone{prefix: "tasks/blocked/"},
		cardZone{prefix: "tasks/backlog/"},
		cardZone{prefix: "tasks/plan/", skip: true},
	)
}

// declaredBoardDirs returns the top-level tasks/ directory names cardZones declares, derived
// from each zone's prefix rather than kept as a second hand-written list. A board directory
// outside this table would otherwise bypass duplicate-card-id and duplicate-filename-number
// checks.
func declaredBoardDirs() map[string]struct{} {
	dirs := make(map[string]struct{}, len(cardZones))
	for _, z := range cardZones {
		name := strings.TrimSuffix(strings.TrimPrefix(z.prefix, "tasks/"), "/")
		dirs[name] = struct{}{}
	}
	return dirs
}

// checkUndeclaredBoardDirectories reports every top-level tasks/ directory not declared in
// cardZones. The status contract is intentionally not duplicated here; this is an identity-sweep
// boundary only.
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
			"tasks/%s/: undeclared board directory — add it to cardZones before its cards are covered by identity checks",
			dir))
	}
	return dirsSeen, undeclared, msgs
}

// resolveCardZone returns the identity scope governing path, chosen by the longest matching
// prefix among cardZones. ok is false when path is not under any declared zone (e.g.
// tasks/README.md).
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
		return value
	}
	if tail := strings.TrimSpace(value[end+1:]); tail == "" || strings.HasPrefix(tail, "#") {
		return value[1:end]
	}
	return stripFrontmatterComment(value)
}

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
