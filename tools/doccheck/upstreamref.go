package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// The advisory TASK-395 added. An issue card whose fix lives upstream (ce-agent-kit) is only
// honest about that while the board can see whether it was ever reported: on 2026-09-14 every
// one of the upstream-owned cards carried zero trace of a report, and "보고하지 않은 것" and
// "보고했는데 안 고쳐진 것" looked identical — both todo, both green.
//
// Two meters, both counted rather than failed. `## 소유권` is the per-card ownership verdict
// this board already writes: a heading containing "이 저장소" exempts the card; every other
// form ("상류다", "갈린다") owes an `upstream-ref:` frontmatter value naming where it was
// reported. And a card with no ownership heading at all is counted as unmarked, because the
// default for silence is to be pooled with the upstream-owned majority (the failure mode
// ISSUE-021's section was written to stop).
//
// Why advisory and not red: the act this measures happens outside this repository, so a hard
// gate here could only be satisfied by writing a reference this repository cannot verify —
// StaleLinkPaths stays a maintenance signal for the same reason. The count is the meter: an
// unreported card stops hiding the day the upstream_unref line reads zero, which is all the
// card asks ("보고 안 됨이 초록 뒤에 숨지 못한다"). When `ce task lint` grows the same field,
// this widens check retires in favor of it (the same handoff rule as the zone check).
const (
	ownershipHeadingPrefix = "## 소유권"
	selfOwnedMarker        = "이 저장소"
	upstreamRefField       = "upstream-ref"
	issueZonePrefix        = "tasks/issue/"
)

// ownershipOf reads a card's ownership verdict from its `## 소유권` heading line. marked is
// false when no such heading exists. Any heading that does not name "이 저장소" counts as
// upstream-owned — including "갈린다", whose this-repo half still owes the upstream half a
// report trail, and including an unrecognized marker, which reads as upstream rather than
// silently exempting the card.
func ownershipOf(body string) (selfOwned bool, marked bool) {
	for line := range strings.SplitSeq(stripFencedRegions(body), "\n") {
		if !strings.HasPrefix(line, ownershipHeadingPrefix) {
			continue
		}
		return strings.Contains(line, selfOwnedMarker), true
	}
	return false, false
}

// checkUpstreamRefs counts issue cards by ownership marking and report trail.
//
// seen counts every non-symlink markdown file under tasks/issue/; read counts those whose body
// was read (frontmatter state included). The pair carries the same "walk broke, not clean"
// distinction checkCardStatus draws: seen>0 with read==0 means the sweep stopped reaching the
// zone, an error rather than a clean zero. unmarked counts read cards with no `## 소유권`
// heading; owned counts read, marked, non-self cards; unrefed counts those among owned whose
// `upstream-ref:` is absent or empty. msgs carries one line per counted card for the report.
func checkUpstreamRefs(root string, inv []InventoryEntry) (seen, read, unmarked, owned, unrefed int, msgs, errs []string) {
	for _, e := range inv {
		if isSymlinkMode(e.Mode) || !strings.HasPrefix(e.Path, issueZonePrefix) || !isMarkdownPath(e.Path) {
			continue
		}
		seen++
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: read: %v", e.Path, err))
			continue
		}
		read++
		body := string(data)
		frontmatter, state := splitFrontmatter(body)
		if state != frontmatterOK {
			errs = append(errs, fmt.Sprintf("%s: %s", e.Path, malformedFrontmatterReason(state)))
			continue
		}
		self, hasHeading := ownershipOf(body)
		if !hasHeading {
			unmarked++
			msgs = append(msgs, fmt.Sprintf("%s: no %s heading — ownership unmarked (TASK-395 criterion 1)", e.Path, ownershipHeadingPrefix))
			continue
		}
		if self {
			continue
		}
		owned++
		ref, found, err := frontmatterField(frontmatter, upstreamRefField)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Path, err))
			continue
		}
		absence := "empty"
		if !found {
			absence = "absent"
		}
		if !found || strings.TrimSpace(ref) == "" {
			unrefed++
			msgs = append(msgs, fmt.Sprintf("%s: upstream-owned but %s: is %s — no report trail (TASK-395)", e.Path, upstreamRefField, absence))
		}
	}
	return seen, read, unmarked, owned, unrefed, msgs, errs
}
