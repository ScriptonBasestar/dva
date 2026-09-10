package main

import (
	"strings"
	"testing"
)

// TestDuplicateCardIDsAreReported pins the invariant the tree lost: two cards claiming one id.
// The real collision was tasks/todo/331-enforce-… and tasks/todo/331-plan-scoped-… , both
// `id: TASK-331`, both on master, with every gate green — links resolve by id (TASK-143), so
// the ambiguity is invisible until a human reads the directory.
func TestDuplicateCardIDsAreReported(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/331-first.md", body: "---\nid: TASK-331\nstatus: todo\n---\n\n# First\n"},
		archiveCard{path: "tasks/todo/331-second.md", body: "---\nid: TASK-331\nstatus: todo\n---\n\n# Second\n"},
	)
	if res.DuplicateCardIDs != 1 {
		t.Fatalf("duplicate_card_ids = %d, want 1", res.DuplicateCardIDs)
	}
	if res.OK {
		t.Fatal("Check reported OK with two cards claiming TASK-331")
	}
	joined := strings.Join(res.DuplicateIDDetail, "\n")
	for _, want := range []string{"TASK-331", "tasks/todo/331-first.md", "tasks/todo/331-second.md"} {
		if !strings.Contains(joined, want) {
			t.Errorf("detail %q does not name %q — a report that does not say which files collide cannot be acted on", joined, want)
		}
	}
}

// TestDuplicateCardIDsAcrossZones covers the shape that actually hides longest: the same id in
// two different zones, where a directory listing of either zone alone looks correct.
func TestDuplicateCardIDsAcrossZones(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/010-open.md", body: "---\nid: TASK-010\nstatus: todo\n---\n\n# Open\n"},
		archiveCard{path: "tasks/done/010-closed.md", body: "---\nid: TASK-010\nstatus: done\n---\n\n# Closed\n"},
	)
	if res.DuplicateCardIDs != 1 {
		t.Fatalf("duplicate_card_ids = %d, want 1 for one id spanning todo/ and done/", res.DuplicateCardIDs)
	}
}

// TestDistinctCardIDsPass is the control: the check must not fire on a well-formed tree, and
// must count the ids it saw. Without the count a broken walk would report zero duplicates and
// look identical to a clean tree.
func TestDistinctCardIDsPass(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/010-a.md", body: "---\nid: TASK-010\nstatus: todo\n---\n\n# A\n"},
		archiveCard{path: "tasks/todo/011-b.md", body: "---\nid: TASK-011\nstatus: todo\n---\n\n# B\n"},
	)
	if res.DuplicateCardIDs != 0 {
		t.Fatalf("duplicate_card_ids = %d, want 0; detail: %v", res.DuplicateCardIDs, res.DuplicateIDDetail)
	}
	if res.CardIDsSeen != 2 {
		t.Fatalf("card_ids_seen = %d, want 2 — a walk that reads no ids reports zero duplicates and looks clean", res.CardIDsSeen)
	}
}

// TestPlanFilesAreNotCardIDs keeps tasks/plan/ out of the id space. Plans carry their own
// PLAN-N ids and are excluded from the card sweep; counting them here would make a plan and a
// card that happened to share a number report as a collision.
func TestPlanFilesAreNotCardIDs(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/006-card.md", body: "---\nid: TASK-006\nstatus: todo\n---\n\n# Card\n"},
		archiveCard{path: "tasks/plan/006-plan.md", body: "---\nid: TASK-006\n---\n\n# Plan\n"},
	)
	if res.DuplicateCardIDs != 0 {
		t.Fatalf("duplicate_card_ids = %d, want 0 — tasks/plan/ is not a card zone; detail: %v", res.DuplicateCardIDs, res.DuplicateIDDetail)
	}
}

// TestCardsWithoutAnIDAreNotDuplicates: a missing id: is not a collision. Two such cards must
// not be folded together under an empty key, which would report a duplicate that does not exist.
func TestCardsWithoutAnIDAreNotDuplicates(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/010-a.md", body: "---\nstatus: todo\n---\n\n# A\n"},
		archiveCard{path: "tasks/todo/011-b.md", body: "---\nstatus: todo\n---\n\n# B\n"},
	)
	if res.DuplicateCardIDs != 0 {
		t.Fatalf("duplicate_card_ids = %d, want 0; detail: %v", res.DuplicateCardIDs, res.DuplicateIDDetail)
	}
}

// TestDuplicateFilenameNumbersAreReported closes the second half of TASK-331: a writer may
// choose a filename from the highest number on disk while using a different frontmatter id.
// The ids stay distinct, so the id guard is correctly silent; the filename-number guard must
// still fail and name both files.
func TestDuplicateFilenameNumbersAreReported(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/331-first.md", body: "---\nid: TASK-900\nstatus: todo\n---\n\n# First\n"},
		archiveCard{path: "tasks/done/331-second.md", body: "---\nid: TASK-901\nstatus: done\n---\n\n# Second\n"},
	)
	if res.DuplicateCardIDs != 0 {
		t.Fatalf("duplicate_card_ids = %d, want 0: ids are deliberately distinct", res.DuplicateCardIDs)
	}
	if res.DuplicateFilenameNums != 1 {
		t.Fatalf("duplicate_filename_numbers = %d, want 1; detail: %v", res.DuplicateFilenameNums, res.DuplicateFilenameDetail)
	}
	joined := strings.Join(res.DuplicateFilenameDetail, "\n")
	for _, want := range []string{"TASK-331", "tasks/todo/331-first.md", "tasks/done/331-second.md"} {
		if !strings.Contains(joined, want) {
			t.Errorf("detail %q does not name %q", joined, want)
		}
	}
	if res.OK {
		t.Fatal("Check reported OK with two TASK cards sharing filename number 331")
	}
}

// TestLeadingZeroFilenameNumbersCollide pins the normalization used for a filename number.
// A worker can see either spelling while scanning a mixed historical tree, but both 001-* and
// 1-* claim the same TASK namespace slot and must not be allowed to coexist.
func TestLeadingZeroFilenameNumbersCollide(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/001-first.md", body: "---\nid: TASK-900\nstatus: todo\n---\n\n# First\n"},
		archiveCard{path: "tasks/done/1-second.md", body: "---\nid: TASK-901\nstatus: done\n---\n\n# Second\n"},
	)
	if res.DuplicateFilenameNums != 1 {
		t.Fatalf("duplicate_filename_numbers = %d, want 1 for 001-* and 1-*; detail: %v", res.DuplicateFilenameNums, res.DuplicateFilenameDetail)
	}
	if !strings.Contains(strings.Join(res.DuplicateFilenameDetail, "\n"), "TASK-1") {
		t.Errorf("detail %v does not report normalized TASK-1", res.DuplicateFilenameDetail)
	}
}

// TestPlanAndTaskNamespacesDoNotCollideByNumber preserves both plan exclusions: active plans
// live in tasks/plan/, while completed plans live under tasks/_archive/plan/. Either may have
// the same filename number as a TASK card without entering the task filename namespace.
func TestPlanAndTaskNamespacesDoNotCollideByNumber(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/006-task.md", body: "---\nid: TASK-900\nstatus: todo\n---\n\n# Task\n"},
		archiveCard{path: "tasks/plan/006-active-plan.md", body: "---\nid: PLAN-006\n---\n\n# Active plan\n"},
		archiveCard{path: "tasks/_archive/plan/006-archived-plan.md", body: "---\nid: PLAN-006\nstatus: done\n---\n\n# Archived plan\n"},
	)
	if res.DuplicateFilenameNums != 0 {
		t.Fatalf("duplicate_filename_numbers = %d, want 0; detail: %v", res.DuplicateFilenameNums, res.DuplicateFilenameDetail)
	}
	if !res.OK {
		t.Fatalf("Check reported FAIL for a task and excluded plan files: errors=%v", res.Errors)
	}
}

// TestFilenameNumbersAreScopedByIDNamespace keeps ISSUE and TASK number histories separate.
// The repository intentionally has ISSUE-001 beside historical TASK-001, so applying a
// global number rule would reject its existing tracker layout.
func TestFilenameNumbersAreScopedByIDNamespace(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/issue/001-issue.md", body: "---\nid: ISSUE-001\nstatus: todo\n---\n\n# Issue\n"},
		archiveCard{path: "tasks/_archive/001-task.md", body: "---\nid: TASK-001\nstatus: done\n---\n\n# Task\n"},
	)
	if res.DuplicateFilenameNums != 0 {
		t.Fatalf("duplicate_filename_numbers = %d, want 0; detail: %v", res.DuplicateFilenameNums, res.DuplicateFilenameDetail)
	}
}
