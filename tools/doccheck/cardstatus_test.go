package main

import (
	"slices"
	"strings"
	"testing"
)

// cardFixture runs the whole gate over a synthetic tasks/ tree. Going through Check rather than
// calling checkCardStatus directly keeps the wiring under test too, the same reasoning
// archiveFixture applies in archive_test.go.
func cardFixture(t *testing.T, cards ...archiveCard) Result {
	t.Helper()
	root := t.TempDir()
	writeFile(t, root, "docs/a.md", "# A\n\nSee [self](a.md).\n")
	paths := []string{"docs/a.md"}
	for _, c := range cards {
		writeFile(t, root, c.path, c.body)
		paths = append(paths, c.path)
	}
	return Check(CheckInput{Root: root, Inventory: mustInventory(t, root, paths...)})
}

// TestZoneResolvesArchiveDoneAsArchive pins the longest-prefix rule against the exact collision
// TASK-287 names: tasks/archive/done/ exists, and an nth-segment reader takes its "done"
// segment as the zone instead of "archive", checking a closed card against the stricter done/
// set. A card here declares status: superseded, valid only under archive/; if the zone were
// misresolved as done/ (permits only "done") this would wrongly fail.
func TestZoneResolvesArchiveDoneAsArchive(t *testing.T) {
	zone, ok := resolveCardZone("tasks/archive/done/001-x.md")
	if !ok {
		t.Fatal("tasks/archive/done/001-x.md resolved to no zone")
	}
	if zone.prefix != "tasks/archive/" {
		t.Fatalf("zone prefix = %q, want tasks/archive/ — an archived done-card must not be classified by its done/ segment", zone.prefix)
	}

	res := cardFixture(t, archiveCard{
		path: "tasks/archive/done/001-x.md",
		body: "---\nid: TASK-001\nstatus: superseded\n---\n\n# Body\n",
	})
	if res.CardsChecked != 1 {
		t.Fatalf("cards_checked=%d, want 1", res.CardsChecked)
	}
	if res.StatusMismatches != 0 {
		t.Errorf("status_mismatches=%d on a card valid under its real zone (archive/); detail=%v", res.StatusMismatches, res.CardStatusDetail)
	}
	if !res.OK {
		t.Errorf("Check reported FAIL on a card whose status is permitted in its longest-matching zone; errors=%v", res.Errors)
	}
}

// TestCardStatusMismatchFailsWithNamedFile pins that a card whose status: is not permitted in
// its zone is reported by file, found status, and the permitted set, and that Check exits
// non-OK.
func TestCardStatusMismatchFailsWithNamedFile(t *testing.T) {
	const path = "tasks/done/002-x.md"
	res := cardFixture(t, archiveCard{
		path: path,
		body: "---\nid: TASK-002\nstatus: todo\n---\n\n# Body\n",
	})
	if res.CardsChecked != 1 {
		t.Fatalf("cards_checked=%d, want 1", res.CardsChecked)
	}
	if res.StatusMismatches != 1 {
		t.Fatalf("status_mismatches=%d, want 1; detail=%v", res.StatusMismatches, res.CardStatusDetail)
	}
	if res.OK {
		t.Error("Check reported OK on a card whose status: is not permitted in its zone")
	}
	if !containsAny(res.CardStatusDetail, strings.ToLower(path)) {
		t.Errorf("detail %v does not name the file %q", res.CardStatusDetail, path)
	}
	if !containsAny(res.CardStatusDetail, `found "todo"`) {
		t.Errorf("detail %v does not name the found status", res.CardStatusDetail)
	}
	if !containsAny(res.CardStatusDetail, "permits status: done") {
		t.Errorf("detail %v does not name the permitted set for tasks/done/", res.CardStatusDetail)
	}
}

// TestMissingStatusIsErrorAndPlansAreSkipped covers both halves of the completion criterion in
// one test: a card zone missing status: entirely is a mismatch, and tasks/plan/ is skipped
// without being counted as a card at all.
func TestMissingStatusIsErrorAndPlansAreSkipped(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/003-x.md", body: "---\nid: TASK-003\ntitle: \"no status field\"\n---\n\n# Body\n"},
		archiveCard{path: "tasks/plan/004-x.md", body: "---\nid: PLAN-004\ntype: plan\nprogress: 3/10\n---\n\n# Plan\n"},
	)
	if res.CardsChecked != 1 {
		t.Fatalf("cards_checked=%d, want 1 — the plan card must not be counted", res.CardsChecked)
	}
	if res.StatusMismatches != 1 {
		t.Fatalf("status_mismatches=%d, want 1; detail=%v", res.StatusMismatches, res.CardStatusDetail)
	}
	if !containsAny(res.CardStatusDetail, "tasks/todo/003-x.md") {
		t.Errorf("detail %v does not name the card missing status:", res.CardStatusDetail)
	}
	if containsAny(res.CardStatusDetail, "tasks/plan/004-x.md") {
		t.Errorf("detail %v mentions the plan file, which should be skipped entirely", res.CardStatusDetail)
	}
	if res.OK {
		t.Error("Check reported OK on a card zone missing status: entirely")
	}
}

// TestZeroCardsSweptIsFailure pins the vacuous-run guard: files that resolve to a task-card zone
// but yield zero checked cards must fail the gate as a broken walk, not pass as "nothing to
// check" — the same seen-vs-checked distinction ArchiveFilesSeen/ArchiveCards already draws for
// the archive guard.
func TestZeroCardsSweptIsFailure(t *testing.T) {
	// An empty inventory resolves nothing to a zone at all — not a broken walk, since no fixture
	// here builds a tasks/ tree unrelated to this file.
	seen, checked, mismatches, msgs, errs := checkCardStatus(t.TempDir(), nil)
	if seen != 0 || checked != 0 || mismatches != 0 || len(msgs) != 0 || len(errs) != 0 {
		t.Fatalf("checkCardStatus on empty inventory = (%d, %d, %d, %v, %v), want all zero", seen, checked, mismatches, msgs, errs)
	}

	// A non-markdown file under a card zone: seen>0 (the walk reached the zone) but checked==0
	// (nothing there could be read as a card) — the broken-walk shape this guard exists to catch.
	root := t.TempDir()
	writeFile(t, root, "docs/a.md", "# A\n\nSee [self](a.md).\n")
	writeFile(t, root, "tasks/todo/notes.txt", "not a card\n")
	res := Check(CheckInput{Root: root, Inventory: mustInventory(t, root, "docs/a.md", "tasks/todo/notes.txt")})
	if res.CardsSeen == 0 {
		t.Fatal("cards_seen=0, want 1 — the zone prefix matched nothing")
	}
	if res.CardsChecked != 0 {
		t.Fatalf("cards_checked=%d, want 0", res.CardsChecked)
	}
	if res.OK {
		t.Error("Check reported OK with files seen in a card zone but zero read as cards")
	}
	if !containsAny(res.Errors, "vacuous") {
		t.Errorf("errors %v do not name the vacuous sweep", res.Errors)
	}
}

// TestCardStatus_acceptsPermittedStatusPerZone is the accepting half: every zone's permitted
// status values must pass cleanly, so the guard is not accidentally stricter than the table it
// was built from.
func TestCardStatus_acceptsPermittedStatusPerZone(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/010-a.md", body: "---\nid: TASK-010\nstatus: todo\n---\n\n# A\n"},
		archiveCard{path: "tasks/done/011-b.md", body: "---\nid: TASK-011\nstatus: done\n---\n\n# B\n"},
		archiveCard{path: "tasks/issue/012-c.md", body: "---\nid: TASK-012\nstatus: todo\n---\n\n# C\n"},
		archiveCard{path: "tasks/doing/013-d.md", body: "---\nid: TASK-013\nstatus: doing\n---\n\n# D\n"},
		archiveCard{path: "tasks/blocked/014-e.md", body: "---\nid: TASK-014\nstatus: blocked\n---\n\n# E\n"},
		archiveCard{path: "tasks/archive/015-f.md", body: "---\nid: TASK-015\nstatus: done\n---\n\n# F\n"},
		archiveCard{path: "tasks/archive/016-g.md", body: "---\nid: TASK-016\nstatus: superseded\n---\n\n# G\n"},
	)
	if res.CardsChecked != 7 {
		t.Fatalf("cards_checked=%d, want 7", res.CardsChecked)
	}
	if res.StatusMismatches != 0 {
		t.Errorf("status_mismatches=%d on all-permitted cards; detail=%v", res.StatusMismatches, res.CardStatusDetail)
	}
}

// TestCardZonesDeclareReviewAndBacklog pins the two zones TASK-414 found missing from cardZones:
// a card sitting under either directory used to resolve to no zone at all (resolveCardZone
// ok=false), which silently excluded it from checkCardStatus, checkDuplicateCardIDs and
// checkDuplicateFilenameNumbers alike. tasks/review/ permits status: review; tasks/backlog/
// permits status: backlog (declared, not skipped — see buildCardZones for why skip was rejected).
func TestCardZonesDeclareReviewAndBacklog(t *testing.T) {
	zone, ok := resolveCardZone("tasks/review/410-x.md")
	if !ok || zone.prefix != "tasks/review/" {
		t.Fatalf("resolveCardZone(tasks/review/410-x.md) = (%+v, %t), want tasks/review/ zone", zone, ok)
	}
	if !slices.Contains(zone.permitted, "review") {
		t.Fatalf("tasks/review/ zone permits %v, want it to include \"review\"", zone.permitted)
	}

	zone, ok = resolveCardZone("tasks/backlog/009-x.md")
	if !ok || zone.prefix != "tasks/backlog/" {
		t.Fatalf("resolveCardZone(tasks/backlog/009-x.md) = (%+v, %t), want tasks/backlog/ zone", zone, ok)
	}
	if zone.skip {
		t.Fatal("tasks/backlog/ must not be skip: skip would also exempt it from the duplicate-id and duplicate-filename-number sweeps")
	}
	if !slices.Contains(zone.permitted, "backlog") {
		t.Fatalf("tasks/backlog/ zone permits %v, want it to include \"backlog\"", zone.permitted)
	}

	res := cardFixture(t,
		archiveCard{path: "tasks/review/020-r.md", body: "---\nid: TASK-020\nstatus: review\n---\n\n# R\n"},
		archiveCard{path: "tasks/backlog/021-b.md", body: "---\nid: BACKLOG-021\nstatus: backlog\n---\n\n# B\n"},
	)
	if res.CardsChecked != 2 {
		t.Fatalf("cards_checked=%d, want 2", res.CardsChecked)
	}
	if res.StatusMismatches != 0 {
		t.Errorf("status_mismatches=%d on review/backlog cards using their permitted status; detail=%v", res.StatusMismatches, res.CardStatusDetail)
	}
	if !res.OK {
		t.Errorf("Check reported FAIL on review/backlog cards using their permitted status; errors=%v", res.Errors)
	}
}

// TestCardZonesDeclareDoingAndBlocked keeps both active lifecycle zones in the same status
// sweep as todo/review/done. They must reject a status valid only in the other zone.
func TestCardZonesDeclareDoingAndBlocked(t *testing.T) {
	for _, tt := range []struct {
		path, permitted, invalid string
	}{
		{"tasks/doing/411-x.md", "doing", "blocked"},
		{"tasks/blocked/412-x.md", "blocked", "doing"},
	} {
		t.Run(tt.permitted, func(t *testing.T) {
			zone, ok := resolveCardZone(tt.path)
			if !ok || zone.prefix != "tasks/"+tt.permitted+"/" {
				t.Fatalf("resolveCardZone(%s) = (%+v, %t), want tasks/%s/ zone", tt.path, zone, ok, tt.permitted)
			}
			if !slices.Contains(zone.permitted, tt.permitted) {
				t.Fatalf("tasks/%s/ zone permits %v, want it to include %q", tt.permitted, zone.permitted, tt.permitted)
			}

			res := cardFixture(t, archiveCard{path: tt.path, body: "---\nid: TASK-411\nstatus: " + tt.invalid + "\n---\n\n# Wrong state\n"})
			if res.StatusMismatches != 1 || res.OK {
				t.Fatalf("status_mismatches=%d, OK=%t; want one rejected status in tasks/%s/: %v", res.StatusMismatches, res.OK, tt.permitted, res.CardStatusDetail)
			}
			if !containsAny(res.CardStatusDetail, `found "`+tt.invalid+`"`) {
				t.Errorf("detail %v does not name rejected status %q", res.CardStatusDetail, tt.invalid)
			}
		})
	}
}

// TestUndeclaredBoardDirectoryFailsTheGate pins the persisting rule TASK-414's Design section
// calls for: adding two more prefixes was not enough, since the same silent gap recurs the next
// time the board grows a directory nobody adds to cardZones. A tasks/*/ directory that cardZones
// does not declare — permitted or skip — must fail the gate instead of being silently invisible
// to checkCardStatus, checkDuplicateCardIDs and checkDuplicateFilenameNumbers.
func TestUndeclaredBoardDirectoryFailsTheGate(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/a.md", "# A\n\nSee [self](a.md).\n")
	writeFile(t, root, "tasks/staging/900-x.md", "---\nid: TASK-900\nstatus: staging\n---\n\n# X\n")
	res := Check(CheckInput{Root: root, Inventory: mustInventory(t, root, "docs/a.md", "tasks/staging/900-x.md")})

	if res.UndeclaredBoardDirs != 1 {
		t.Fatalf("undeclared_board_dirs=%d, want 1; errors=%v", res.UndeclaredBoardDirs, res.Errors)
	}
	if res.OK {
		t.Error("Check reported OK with an undeclared tasks/*/ directory on disk")
	}
	if !containsAny(res.UndeclaredBoardDirDetail, "tasks/staging/") {
		t.Errorf("detail %v does not name the undeclared directory tasks/staging/", res.UndeclaredBoardDirDetail)
	}

	// The control: every currently declared board directory must not trip this guard.
	clean := cardFixture(t,
		archiveCard{path: "tasks/todo/001-a.md", body: "---\nid: TASK-001\nstatus: todo\n---\n\n# A\n"},
		archiveCard{path: "tasks/done/002-b.md", body: "---\nid: TASK-002\nstatus: done\n---\n\n# B\n"},
		archiveCard{path: "tasks/issue/003-c.md", body: "---\nid: TASK-003\nstatus: todo\n---\n\n# C\n"},
		archiveCard{path: "tasks/review/004-d.md", body: "---\nid: TASK-004\nstatus: review\n---\n\n# D\n"},
		archiveCard{path: "tasks/doing/005-e.md", body: "---\nid: TASK-005\nstatus: doing\n---\n\n# E\n"},
		archiveCard{path: "tasks/blocked/006-f.md", body: "---\nid: TASK-006\nstatus: blocked\n---\n\n# F\n"},
		archiveCard{path: "tasks/backlog/007-g.md", body: "---\nid: BACKLOG-007\nstatus: backlog\n---\n\n# G\n"},
		archiveCard{path: "tasks/archive/008-h.md", body: "---\nid: TASK-008\nstatus: done\n---\n\n# H\n"},
		archiveCard{path: "tasks/plan/009-i.md", body: "---\nid: PLAN-009\ntype: plan\n---\n\n# I\n"},
	)
	if clean.UndeclaredBoardDirs != 0 {
		t.Fatalf("undeclared_board_dirs=%d, want 0 on an all-declared board; errors=%v", clean.UndeclaredBoardDirs, clean.Errors)
	}
}

func TestFrontmatterValueDropsTrailingComment(t *testing.T) {
	res := cardFixture(t,
		archiveCard{path: "tasks/todo/331-comment.md", body: "---\nid: TASK-331 # duplicate\nstatus: todo\n---\n\n# Comment\n"},
		archiveCard{path: "tasks/todo/331-plain.md", body: "---\nid: TASK-331\nstatus: todo\n---\n\n# Plain\n"},
	)
	if res.DuplicateCardIDs != 1 {
		t.Fatalf("duplicate_card_ids = %d, want 1 after removing a trailing id comment; detail=%v", res.DuplicateCardIDs, res.DuplicateIDDetail)
	}
}

func TestQuotedValueKeepsItsHash(t *testing.T) {
	value, found, err := frontmatterField("title: \"release #1\" # catalog\n", "title")
	if err != nil || !found || value != "release #1" {
		t.Fatalf("frontmatterField quoted hash = (%q, %t, %v), want (release #1, true, nil)", value, found, err)
	}
}

func TestFrontmatterValueHandlesQuoteEdges(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "plain embedded apostrophe", raw: "O'Reilly # catalog", want: "O'Reilly"},
		{name: "comment only scalar", raw: "# catalog", want: ""},
		{name: "single quote doubling", raw: "'O''Reilly #1' # catalog", want: "O''Reilly #1"},
		{name: "unterminated double quote", raw: "\"release #1", want: "\"release #1"},
		{name: "mismatched quote", raw: "\"release' # catalog", want: "\"release' # catalog"},
		{name: "non-outer closing quote", raw: "\"release\" draft # catalog", want: "\"release\" draft"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := frontmatterValue(tt.raw); got != tt.want {
				t.Errorf("frontmatterValue(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestRepeatedFrontmatterKeyIsReported(t *testing.T) {
	res := cardFixture(t, archiveCard{
		path: "tasks/todo/331-repeated.md",
		body: "---\nid: TASK-331\nid: TASK-332\nstatus: todo\n---\n\n# Repeated\n",
	})
	if res.OK {
		t.Fatal("Check reported OK with a repeated id: frontmatter key")
	}
	if !containsAny(res.Errors, `repeated frontmatter key "id"`) {
		t.Errorf("errors %v do not report the repeated id key", res.Errors)
	}
}

func TestCardStatusAndCanonicalFieldShareParser(t *testing.T) {
	frontmatter := "status: todo # accepted comment\nid: TASK-331\nid: TASK-332\n"
	status, found, err := cardStatus(frontmatter)
	if err != nil || !found || status != "todo" {
		t.Fatalf("cardStatus = (%q, %t, %v), want (todo, true, nil)", status, found, err)
	}
	hasCanonical, err := hasCanonicalField(frontmatter)
	if hasCanonical || err == nil || !strings.Contains(err.Error(), `repeated frontmatter key "id"`) {
		t.Fatalf("hasCanonicalField = (%t, %v), want repeated id error", hasCanonical, err)
	}
}
