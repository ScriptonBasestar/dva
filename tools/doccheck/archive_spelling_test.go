package main

import (
	"strings"
	"testing"
)

// TestArchiveSpelling pins the board's in-progress archive-directory migration from
// tasks/archive/ to tasks/_archive/: doccheck must judge both spellings identically everywhere
// it decides whether a path is "the archive", or the checker regresses to recognising only
// whichever spelling it hardcodes and silently stops covering cards under the other one. It
// exercises the real classification functions directly (isArchivePath, resolveCardZone) and the
// archive-frontmatter consumer. CE's selected strict dialect owns status validation.
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

	t.Run("cardstatus.go: resolveCardZone covers both spellings for identity checks", func(t *testing.T) {
		legacyZone, ok := resolveCardZone(legacyPath)
		if !ok {
			t.Fatalf("resolveCardZone(%q) resolved to no zone", legacyPath)
		}
		canonicalZone, ok := resolveCardZone(canonicalPath)
		if !ok {
			t.Fatalf("resolveCardZone(%q) resolved to no zone", canonicalPath)
		}
		if legacyZone.skip || canonicalZone.skip {
			t.Errorf("archive zones must remain eligible for identity checks: %s=%v %s=%v", legacyPath, legacyZone.skip, canonicalPath, canonicalZone.skip)
		}
	})
}
