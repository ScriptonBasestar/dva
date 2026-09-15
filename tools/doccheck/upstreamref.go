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
//
// TASK-398 moved the verdict off the heading. The heading stays, but as the *reason*: a prose
// line drifts one syllable ("— 상류입니다") and a substring match goes quietly green, which is
// the drift review-395 filed as F1 and the shape ISSUE-022 already is. So `ownership:` in
// frontmatter is now the machine axis and carries a closed value set; the heading is re-read as
// a cross-check against it. Disagreement between the two is fatal — it means one of them was
// edited and the other was not, and there is no honest way to pick a winner automatically.
const (
	ownershipHeadingPrefix = "## 소유권"
	selfOwnedMarker        = "이 저장소"
	splitMarker            = "갈린다"
	ownershipField         = "ownership"
	upstreamRefField       = "upstream-ref"
	issueZonePrefix        = "tasks/issue/"
)

// ownershipValues is the closed set `ownership:` may carry. `split` is upstream-owing like
// `upstream` — its this-repo half being handled does not discharge the upstream half — but the
// two are kept distinct because the distinction decides whether a card is finished when the
// upstream fix lands. See TASK-395 결정 1 for the rule separating them: a local half that was
// resolved and left scope reads `upstream`, one that was only worked around reads `split`.
var ownershipValues = map[string]bool{"local": true, "upstream": true, "split": true}

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

// headingOwnership maps a `## 소유권` heading to the ownership: value it asserts, for the
// cross-check. It reports the same three values the field carries so the comparison is a plain
// equality rather than a second, differently-shaped judgement.
func headingOwnership(body string) (value string, marked bool) {
	for line := range strings.SplitSeq(stripFencedRegions(body), "\n") {
		if !strings.HasPrefix(line, ownershipHeadingPrefix) {
			continue
		}
		switch {
		case strings.Contains(line, selfOwnedMarker):
			return "local", true
		case strings.Contains(line, splitMarker):
			return "split", true
		default:
			return "upstream", true
		}
	}
	return "", false
}

// upstreamRefCounts is what one sweep of tasks/issue/ measured. It is a struct rather than a
// row of ints because the two severities below are easy to swap by accident at a call site:
// Unclassified and Mismatched fail the gate, Unrefed does not.
type upstreamRefCounts struct {
	Seen         int // non-symlink markdown files under tasks/issue/
	Read         int // of those, ones whose body was read
	Unclassified int // read cards with no usable `ownership:` value — FATAL
	Mismatched   int // `ownership:` and the `## 소유권` heading disagree — FATAL
	Owned        int // classified upstream or split
	Unrefed      int // of Owned, those with no `upstream-ref:` value — advisory
	Msgs         []string
	Errs         []string
}

// checkUpstreamRefs counts issue cards by ownership classification and report trail.
//
// Seen counts every non-symlink markdown file under tasks/issue/; Read counts those whose body
// was read (frontmatter state included). The pair carries the same "walk broke, not clean"
// distinction checkCardStatus draws: Seen>0 with Read==0 means the sweep stopped reaching the
// zone, an error rather than a clean zero.
//
// Classifying is not optional, and that is the load-bearing asymmetry. If a card could stay
// unclassified at no cost, nobody would ever write `upstream` and Unrefed would read zero
// forever — a meter that measures nothing, which is exactly the pathology this check exists to
// end. So Unclassified is fatal. Unrefed is not, because the act it measures (filing the
// upstream issue) happens outside this repository and a hard gate here could only be satisfied
// by writing a reference this repository cannot verify.
func checkUpstreamRefs(root string, inv []InventoryEntry) upstreamRefCounts {
	var c upstreamRefCounts
	for _, e := range inv {
		if isSymlinkMode(e.Mode) || !strings.HasPrefix(e.Path, issueZonePrefix) || !isMarkdownPath(e.Path) {
			continue
		}
		c.Seen++
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			c.Errs = append(c.Errs, fmt.Sprintf("%s: read: %v", e.Path, err))
			continue
		}
		c.Read++
		body := string(data)
		frontmatter, state := splitFrontmatter(body)
		if state != frontmatterOK {
			c.Unclassified++
			c.Msgs = append(c.Msgs, fmt.Sprintf("%s: %s — ownership unclassified (TASK-398)", e.Path, malformedFrontmatterReason(state)))
			continue
		}
		owner, found, err := frontmatterField(frontmatter, ownershipField)
		switch {
		case err != nil:
			c.Unclassified++
			c.Msgs = append(c.Msgs, fmt.Sprintf("%s: %v — ownership unclassified (TASK-398)", e.Path, err))
			continue
		case !found:
			c.Unclassified++
			c.Msgs = append(c.Msgs, fmt.Sprintf("%s: no %s: field — ownership unclassified (TASK-398)", e.Path, ownershipField))
			continue
		case !ownershipValues[owner]:
			c.Unclassified++
			c.Msgs = append(c.Msgs, fmt.Sprintf("%s: %s: %q is not local|upstream|split — ownership unclassified (TASK-398)", e.Path, ownershipField, owner))
			continue
		}
		// The heading is the reason the field says what it says. Reading it back catches the
		// case where one of the two was edited and the other was not — the drift that made a
		// heading-only verdict unsafe in the first place.
		if heading, marked := headingOwnership(body); marked && heading != owner {
			c.Mismatched++
			c.Msgs = append(c.Msgs, fmt.Sprintf("%s: %s: %s but the %s heading says %s — one of the two is stale (TASK-398)", e.Path, ownershipField, owner, ownershipHeadingPrefix, heading))
		}
		if owner == "local" {
			continue
		}
		c.Owned++
		ref, found, err := frontmatterField(frontmatter, upstreamRefField)
		if err != nil {
			c.Errs = append(c.Errs, fmt.Sprintf("%s: %v", e.Path, err))
			continue
		}
		absence := "empty"
		if !found {
			absence = "absent"
		}
		if !found || strings.TrimSpace(ref) == "" {
			c.Unrefed++
			c.Msgs = append(c.Msgs, fmt.Sprintf("%s: %s: %s but %s: is %s — no report trail (TASK-395)", e.Path, ownershipField, owner, upstreamRefField, absence))
		}
	}
	return c
}
