package main

import (
	"reflect"
	"strings"
	"testing"
)

// TestArchiveSpelling pins the board's in-progress archive-directory migration from
// tasks/archive/ to tasks/_archive/: doccheck must judge both spellings identically everywhere
// it decides whether a path is "the archive", or the checker regresses to recognising only
// whichever spelling it hardcodes and silently stops covering cards under the other one. It
// exercises the real classification functions directly (isArchivePath, resolveCardZone) and the
// two call sites that consume them (checkArchiveFrontmatter via archiveFixture,
// checkCardStatus via cardFixture) rather than re-implementing their logic here.
func TestArchiveSpelling(t *testing.T) {
	const (
		legacyPath    = "tasks/archive/001-x.md"
		canonicalPath = "tasks/_archive/001-x.md"
	)

	t.Run("archive.go: isArchivePath accepts both spellings", func(t *testing.T) {
		if !isArchivePath(legacyPath) {
			t.Errorf("isArchivePath(%q) = false, want true", legacyPath)
		}
		if !isArchivePath(canonicalPath) {
			t.Errorf("isArchivePath(%q) = false, want true", canonicalPath)
		}
	})

	t.Run("archive.go: checkArchiveFrontmatter flags both spellings identically", func(t *testing.T) {
		// Same malformed body (carries neither id: nor type:) under each spelling. If either
		// site regressed to a single hardcoded prefix, the file under the other spelling would
		// stop being swept at all — ArchiveFilesSeen would drop and no message would appear for
		// it, rather than the message differing.
		body := "---\ntitle: \"no id or type\"\n---\n\n# Body\n"
		res := archiveFixture(t,
			archiveCard{path: legacyPath, body: body},
			archiveCard{path: canonicalPath, body: body},
		)
		if res.ArchiveFilesSeen != 2 {
			t.Fatalf("ArchiveFilesSeen = %d, want 2 — both %s and %s must be swept", res.ArchiveFilesSeen, legacyPath, canonicalPath)
		}
		if res.ArchiveMissing != 2 {
			t.Fatalf("ArchiveMissing = %d, want 2 — both cards carry neither id: nor type:", res.ArchiveMissing)
		}
		for _, path := range []string{legacyPath, canonicalPath} {
			if !containsAny(res.ArchiveDetail, strings.ToLower(path+": frontmatter carries neither")) {
				t.Errorf("ArchiveDetail = %v, want an entry for %s flagging the missing id/type field", res.ArchiveDetail, path)
			}
		}

		// A well-formed card (carries id:) under each spelling must be swept and read as a card,
		// and must not be flagged.
		goodBody := "---\nid: TASK-001\nstatus: done\n---\n\n# Body\n"
		resGood := archiveFixture(t,
			archiveCard{path: legacyPath, body: goodBody},
			archiveCard{path: canonicalPath, body: goodBody},
		)
		if resGood.ArchiveFilesSeen != 2 || resGood.ArchiveCards != 2 {
			t.Fatalf("ArchiveFilesSeen=%d ArchiveCards=%d, want 2/2 — a well-formed card under either spelling must be swept and read",
				resGood.ArchiveFilesSeen, resGood.ArchiveCards)
		}
		if resGood.ArchiveMissing != 0 {
			t.Fatalf("ArchiveMissing = %d, want 0: %v", resGood.ArchiveMissing, resGood.ArchiveDetail)
		}
	})

	t.Run("cardstatus.go: resolveCardZone resolves both spellings to the same permitted set", func(t *testing.T) {
		legacyZone, ok := resolveCardZone(legacyPath)
		if !ok {
			t.Fatalf("resolveCardZone(%q) resolved to no zone", legacyPath)
		}
		canonicalZone, ok := resolveCardZone(canonicalPath)
		if !ok {
			t.Fatalf("resolveCardZone(%q) resolved to no zone", canonicalPath)
		}
		if legacyZone.skip != canonicalZone.skip {
			t.Errorf("skip differs: %s=%v %s=%v", legacyPath, legacyZone.skip, canonicalPath, canonicalZone.skip)
		}
		if !reflect.DeepEqual(legacyZone.permitted, canonicalZone.permitted) {
			t.Errorf("permitted differs: %s=%v %s=%v", legacyPath, legacyZone.permitted, canonicalPath, canonicalZone.permitted)
		}
	})

	t.Run("cardstatus.go: checkCardStatus judges both spellings identically", func(t *testing.T) {
		// status: superseded is permitted only in an archive zone (see archiveCardStatuses).
		// Both spellings must accept it.
		resOK := cardFixture(t,
			archiveCard{path: legacyPath, body: "---\nid: TASK-001\nstatus: superseded\n---\n\n# Body\n"},
			archiveCard{path: canonicalPath, body: "---\nid: TASK-002\nstatus: superseded\n---\n\n# Body\n"},
		)
		if resOK.StatusMismatches != 0 {
			t.Fatalf("StatusMismatches = %d, want 0 — status: superseded is valid under either archive spelling: %v",
				resOK.StatusMismatches, resOK.CardStatusDetail)
		}

		// An impermissible status must be rejected identically under either spelling.
		resBad := cardFixture(t,
			archiveCard{path: legacyPath, body: "---\nid: TASK-001\nstatus: bogus\n---\n\n# Body\n"},
			archiveCard{path: canonicalPath, body: "---\nid: TASK-002\nstatus: bogus\n---\n\n# Body\n"},
		)
		if resBad.StatusMismatches != 2 {
			t.Fatalf("StatusMismatches = %d, want 2 — status: bogus is invalid under either archive spelling: %v",
				resBad.StatusMismatches, resBad.CardStatusDetail)
		}
		for _, path := range []string{legacyPath, canonicalPath} {
			if !containsAny(resBad.CardStatusDetail, strings.ToLower(path+": zone")) {
				t.Errorf("CardStatusDetail = %v, want an entry for %s", resBad.CardStatusDetail, path)
			}
		}
	})
}
