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
