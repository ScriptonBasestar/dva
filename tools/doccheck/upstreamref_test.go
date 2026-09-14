package main

import (
	"strings"
	"testing"
)

// The three axes TASK-395 requires, plus the two shape tests around them: what the heading
// tokens classify, and that the sweep's own failure is an error rather than a clean zero.
// Fixtures go through cardFixture (whole gate) so the advisory's wiring is tested with its
// classification, the same reasoning cardstatus_test.go states.

func TestUpstreamRefCountsUnreportedUpstreamCard(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/030-x.md",
		body: "---\nid: ISSUE-030\nstatus: todo\n---\n\n## 소유권 — 상류다\n",
	})
	if res.UpstreamUnrefed != 1 || res.UpstreamOwned != 1 {
		t.Fatalf("owned/unrefed = %d/%d, want 1/1 — an upstream-owned card with no upstream-ref: must be counted", res.UpstreamOwned, res.UpstreamUnrefed)
	}
	if !containsAny(res.UpstreamDetail, "030-x.md") {
		t.Fatalf("report must name the unreported card, got %v", res.UpstreamDetail)
	}
	// Advisory: counted, never red — the meter is the line in the report, not a gate failure.
	if !res.OK {
		t.Fatalf("unreported upstream card must not fail the gate (advisory), errors: %v", res.Errors)
	}
}

func TestUpstreamRefCountsEmptyRefAsUnreported(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/031-x.md",
		body: "---\nid: ISSUE-031\nstatus: todo\nupstream-ref: \"\"\n---\n\n## 소유권 — 상류다\n",
	})
	if res.UpstreamUnrefed != 1 {
		t.Fatalf("unrefed = %d, want 1 — an empty upstream-ref: value is a seat with nobody in it", res.UpstreamUnrefed)
	}
}

func TestUpstreamRefAcceptsReportedCard(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/032-x.md",
		body: "---\nid: ISSUE-032\nstatus: todo\nupstream-ref: ce-agent-kit#12\n---\n\n## 소유권 — 상류다\n",
	})
	if res.UpstreamOwned != 1 || res.UpstreamUnrefed != 0 {
		t.Fatalf("owned/unrefed = %d/%d, want 1/0 — a non-empty value is a report trail, whatever channel shaped it", res.UpstreamOwned, res.UpstreamUnrefed)
	}
	if len(res.UpstreamDetail) != 0 {
		t.Fatalf("reported card must produce no detail line, got %v", res.UpstreamDetail)
	}
}

func TestUpstreamRefExemptsSelfOwnedCard(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/033-x.md",
		body: "---\nid: ISSUE-033\nstatus: todo\n---\n\n## 소유권 — 이 저장소다\n",
	})
	if res.UpstreamOwned != 0 || res.UpstreamUnrefed != 0 || res.OwnershipUnmarked != 0 {
		t.Fatalf("owned/unrefed/unmarked = %d/%d/%d, want 0/0/0 — a this-repo card owes no report trail", res.UpstreamOwned, res.UpstreamUnrefed, res.OwnershipUnmarked)
	}
}

func TestUpstreamRefCountsUnmarkedCard(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/034-x.md",
		body: "---\nid: ISSUE-034\nstatus: todo\n---\n\n# No ownership section\n",
	})
	if res.OwnershipUnmarked != 1 {
		t.Fatalf("unmarked = %d, want 1 — criterion 1's meter: silence pools with the upstream majority", res.OwnershipUnmarked)
	}
	// An unmarked card is not pooled into owned: the two meters stay separate axes.
	if res.UpstreamOwned != 0 || res.UpstreamUnrefed != 0 {
		t.Fatalf("unmarked card must not count as owned/unrefed, got %d/%d", res.UpstreamOwned, res.UpstreamUnrefed)
	}
}

func TestUpstreamRefSplitCardStillOwesARef(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/035-x.md",
		body: "---\nid: ISSUE-035\nstatus: todo\n---\n\n## 소유권 — 갈린다\n",
	})
	if res.UpstreamOwned != 1 || res.UpstreamUnrefed != 1 {
		t.Fatalf("owned/unrefed = %d/%d, want 1/1 — a split card's upstream half still owes the report trail", res.UpstreamOwned, res.UpstreamUnrefed)
	}
}

func TestUpstreamRefVacuousSweepIsAnError(t *testing.T) {
	// A file the inventory claims but the disk does not have: the sweep saw the zone, read
	// nothing, and would otherwise report a clean zero. Called directly (not through cardFixture)
	// because Check cannot build this state from real files.
	root := t.TempDir()
	inv := []InventoryEntry{{Path: "tasks/issue/036-x.md", Mode: modeRegular}}
	seen, read, _, _, _, _, errs := checkUpstreamRefs(root, inv)
	if seen != 1 || read != 0 {
		t.Fatalf("seen/read = %d/%d, want 1/0", seen, read)
	}
	if len(errs) != 1 || !strings.Contains(errs[0], "036-x.md") {
		t.Fatalf("unreadable card must surface as an error naming it, got %v", errs)
	}
}

func TestUpstreamRefIgnoresOtherZones(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "tasks/todo/001-x.md", "---\nid: TASK-001\nstatus: todo\n---\n\n# No section\n")
	seen, _, _, _, _, _, _ := checkUpstreamRefs(root, mustInventory(t, root, "tasks/todo/001-x.md"))
	if seen != 0 {
		t.Fatalf("seen = %d, want 0 — the meter reads tasks/issue/ only", seen)
	}
}

// Guard against the heading matcher drifting into fenced examples: a ``` block quoting the
// heading is documentation about the section, not the section.
func TestUpstreamRefHeadingInsideFenceIsNotMarked(t *testing.T) {
	body := "---\nid: ISSUE-037\nstatus: todo\n---\n\n```\n## 소유권 — 상류다\n```\n"
	self, marked := ownershipOf(body)
	if marked {
		t.Fatalf("fenced heading must not mark ownership (self=%v)", self)
	}
}
