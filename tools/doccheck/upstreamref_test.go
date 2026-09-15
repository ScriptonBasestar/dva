package main

import (
	"os"
	"path/filepath"
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
		body: "---\nid: ISSUE-030\nstatus: todo\nownership: upstream\n---\n\n## 소유권 — 상류다\n",
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
		body: "---\nid: ISSUE-031\nstatus: todo\nownership: upstream\nupstream-ref: \"\"\n---\n\n## 소유권 — 상류다\n",
	})
	if res.UpstreamUnrefed != 1 {
		t.Fatalf("unrefed = %d, want 1 — an empty upstream-ref: value is a seat with nobody in it", res.UpstreamUnrefed)
	}
}

func TestUpstreamRefAcceptsReportedCard(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/032-x.md",
		body: "---\nid: ISSUE-032\nstatus: todo\nownership: upstream\nupstream-ref: ce-agent-kit#12\n---\n\n## 소유권 — 상류다\n",
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
		body: "---\nid: ISSUE-033\nstatus: todo\nownership: local\n---\n\n## 소유권 — 이 저장소다\n",
	})
	if res.UpstreamOwned != 0 || res.UpstreamUnrefed != 0 || res.OwnershipUnclassified != 0 {
		t.Fatalf("owned/unrefed/unclassified = %d/%d/%d, want 0/0/0 — a this-repo card owes no report trail", res.UpstreamOwned, res.UpstreamUnrefed, res.OwnershipUnclassified)
	}
}

func TestUpstreamRefCountsUnclassifiedCard(t *testing.T) {
	// Every way an `ownership:` value can fail to exist. All four are the same defect — the
	// board cannot say who owns this card — and all four are fatal, because classifying is an
	// in-repo edit with no external dependency. If it were optional nobody would write
	// `upstream` and UpstreamUnrefed would read zero forever.
	for _, tc := range []struct {
		name string
		body string
	}{
		{"absent", "---\nid: ISSUE-034\nstatus: todo\n---\n\n## 소유권 — 상류다\n"},
		{"empty", "---\nid: ISSUE-034\nstatus: todo\nownership:\n---\n\n## 소유권 — 상류다\n"},
		{"unrecognized", "---\nid: ISSUE-034\nstatus: todo\nownership: theirs\n---\n\n## 소유권 — 상류다\n"},
		{"no frontmatter", "## 소유권 — 상류다\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := cardFixture(t, archiveCard{path: "tasks/issue/034-x.md", body: tc.body})
			if res.OwnershipUnclassified != 1 {
				t.Fatalf("unclassified = %d, want 1 — %s must not pass for a classification", res.OwnershipUnclassified, tc.name)
			}
			// Unclassified is not pooled into owned: the two meters stay separate axes.
			if res.UpstreamOwned != 0 || res.UpstreamUnrefed != 0 {
				t.Fatalf("unclassified card must not count as owned/unrefed, got %d/%d", res.UpstreamOwned, res.UpstreamUnrefed)
			}
			if res.OK {
				t.Fatalf("an unclassified issue card must fail the gate — that is what makes the other meter honest")
			}
		})
	}
}

// The cross-check TASK-398 added. The heading is the reason the field says what it says; when
// they disagree one of the two was edited and the other was not, and nothing can pick a winner
// automatically. Both directions, so the check is not accidentally one-way.
func TestUpstreamRefFlagsHeadingFieldDisagreement(t *testing.T) {
	for _, tc := range []struct{ name, body string }{
		{"field local, heading upstream", "---\nid: ISSUE-038\nstatus: todo\nownership: local\n---\n\n## 소유권 — 상류다\n"},
		{"field upstream, heading local", "---\nid: ISSUE-038\nstatus: todo\nownership: upstream\n---\n\n## 소유권 — 이 저장소다\n"},
		{"field upstream, heading split", "---\nid: ISSUE-038\nstatus: todo\nownership: upstream\n---\n\n## 소유권 — 갈린다\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := cardFixture(t, archiveCard{path: "tasks/issue/038-x.md", body: tc.body})
			if res.OwnershipMismatched != 1 {
				t.Fatalf("mismatched = %d, want 1 — %s", res.OwnershipMismatched, tc.name)
			}
			if res.OK {
				t.Fatalf("a stale ownership heading must fail the gate; errors: %v", res.Errors)
			}
		})
	}
}

func TestUpstreamRefAcceptsAgreementBetweenFieldAndHeading(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/039-x.md",
		body: "---\nid: ISSUE-039\nstatus: todo\nownership: split\n---\n\n## 소유권 — 갈린다 (2026-09-15 명시)\n",
	})
	if res.OwnershipMismatched != 0 || res.OwnershipUnclassified != 0 {
		t.Fatalf("mismatched/unclassified = %d/%d, want 0/0 — a dated heading suffix is still agreement", res.OwnershipMismatched, res.OwnershipUnclassified)
	}
}

func TestUpstreamRefSplitCardStillOwesARef(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/issue/035-x.md",
		body: "---\nid: ISSUE-035\nstatus: todo\nownership: split\n---\n\n## 소유권 — 갈린다\n",
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
	c := checkUpstreamRefs(root, inv)
	if c.Seen != 1 || c.Read != 0 {
		t.Fatalf("seen/read = %d/%d, want 1/0", c.Seen, c.Read)
	}
	if len(c.Errs) != 1 || !strings.Contains(c.Errs[0], "036-x.md") {
		t.Fatalf("unreadable card must surface as an error naming it, got %v", c.Errs)
	}
}

func TestUpstreamRefIgnoresOtherZones(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "tasks/todo/001-x.md", "---\nid: TASK-001\nstatus: todo\n---\n\n# No section\n")
	c := checkUpstreamRefs(root, mustInventory(t, root, "tasks/todo/001-x.md"))
	if c.Seen != 0 {
		t.Fatalf("seen = %d, want 0 — the meter reads tasks/issue/ only", c.Seen)
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

// repoRoot resolves the repository root from the package directory. Tests run with the package
// as cwd, so the root is two levels up (tools/doccheck) — resolved rather than assumed, so the
// test fails loudly if the package is ever moved instead of sweeping an empty tree.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("repo root %s has no go.mod: %v", root, err)
	}
	return root
}

// TestUpstreamRefsSweepsTheRealCorpus is the binding on the live board, not a fixture: the
// property only matters where the cards actually are. It also carries this check's exit
// condition, because a stage that never names its exit is how ISSUE-023's advisory became
// unread. When the last upstream-owned card gains a reference, this test fails and says what
// to do — so the promotion announces itself instead of waiting to be remembered.
func TestUpstreamRefsSweepsTheRealCorpus(t *testing.T) {
	root := repoRoot(t)
	inv, err := LoadInventory(root)
	if err != nil {
		t.Fatalf("inventory: %v", err)
	}
	c := checkUpstreamRefs(root, inv)
	if c.Seen == 0 || c.Read == 0 {
		t.Fatalf("seen/read = %d/%d — the sweep must reach the real tasks/issue/ zone", c.Seen, c.Read)
	}
	if c.Unclassified != 0 {
		t.Fatalf("unclassified = %d on the real board; every issue card must carry ownership: — %v", c.Unclassified, c.Msgs)
	}
	if c.Mismatched != 0 {
		t.Fatalf("mismatched = %d on the real board; ownership: and its heading must agree — %v", c.Mismatched, c.Msgs)
	}
	if c.Unrefed == 0 {
		t.Fatalf("upstream_unref=0 across %d issue card(s) — the advisory stage is now due to end: "+
			"promote UpstreamUnrefed to res.Errors in check.go, then replace this guard with "+
			"an assertion that it stays 0", c.Read)
	}
	t.Logf("swept %d issue card(s) from %d file(s) under %s: %d owned, %d unrefed", c.Read, c.Seen, issueZonePrefix, c.Owned, c.Unrefed)
}
