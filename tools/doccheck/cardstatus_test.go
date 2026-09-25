package main

import (
	"strings"
	"testing"
)

// cardFixture runs the whole gate over a synthetic tasks/ tree. Identity tests use Check rather
// than individual duplicate sweeps so the zone table's wiring stays covered.
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

func TestZoneResolvesArchiveDoneAsArchive(t *testing.T) {
	zone, ok := resolveCardZone("tasks/_archive/done/001-x.md")
	if !ok {
		t.Fatal("tasks/_archive/done/001-x.md resolved to no zone")
	}
	if zone.prefix != "tasks/_archive/" {
		t.Fatalf("zone prefix = %q, want tasks/_archive/ — archived identity must not be classified by its done segment", zone.prefix)
	}
}

func TestCardIdentityZonesCoverLiveAndArchivedCards(t *testing.T) {
	for _, path := range []string{
		"tasks/todo/001-x.md",
		"tasks/doing/001-x.md",
		"tasks/review/001-x.md",
		"tasks/blocked/001-x.md",
		"tasks/done/001-x.md",
		"tasks/issue/001-x.md",
		"tasks/backlog/001-x.md",
		"tasks/archive/001-x.md",
		"tasks/_archive/001-x.md",
	} {
		zone, ok := resolveCardZone(path)
		if !ok || zone.skip {
			t.Fatalf("resolveCardZone(%q) = (%+v, %t), want a non-skipped identity zone", path, zone, ok)
		}
	}
	zone, ok := resolveCardZone("tasks/plan/001-x.md")
	if !ok || !zone.skip {
		t.Fatalf("resolveCardZone(tasks/plan/001-x.md) = (%+v, %t), want skipped plan zone", zone, ok)
	}
}

func TestUndeclaredBoardDirectoryFailsTheGate(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/a.md", "# A\n\nSee [self](a.md).\n")
	writeFile(t, root, "tasks/staging/900-x.md", "---\nid: TASK-900\nstatus: staging\n---\n\n# X\n")
	res := Check(CheckInput{Root: root, Inventory: mustInventory(t, root, "docs/a.md", "tasks/staging/900-x.md")})

	if res.UndeclaredBoardDirs != 1 {
		t.Fatalf("undeclared_board_dirs=%d, want 1; errors=%v", res.UndeclaredBoardDirs, res.Errors)
	}
	if res.OK {
		t.Error("Check reported OK with an undeclared tasks/*/ directory")
	}
	if !containsAny(res.UndeclaredBoardDirDetail, "tasks/staging/") {
		t.Errorf("detail %v does not name the undeclared directory tasks/staging/", res.UndeclaredBoardDirDetail)
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

func TestCanonicalFieldSharesIdentityParser(t *testing.T) {
	frontmatter := "status: todo # accepted comment\nid: TASK-331\nid: TASK-332\n"
	status, found, err := frontmatterField(frontmatter, "status")
	if err != nil || !found || status != "todo" {
		t.Fatalf("frontmatterField(status) = (%q, %t, %v), want (todo, true, nil)", status, found, err)
	}
	hasCanonical, err := hasCanonicalField(frontmatter)
	if hasCanonical || err == nil || !strings.Contains(err.Error(), `repeated frontmatter key "id"`) {
		t.Fatalf("hasCanonicalField = (%t, %v), want repeated id error", hasCanonical, err)
	}
}
