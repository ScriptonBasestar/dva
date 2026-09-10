package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// checkDuplicateCardIDs reports every task id claimed by more than one card file.
//
// A card's id is a key, not a label: doccheck resolves task links by id and ignores the
// directory the link was written against (TASK-143), so two cards claiming one id make every
// link to that id resolve to whichever the walk happened to reach — and stay green either way.
// The tree carried `TASK-331` twice for a full day without a single check noticing, and a
// worker numbering a new card from the highest id it could see picked ids already taken on
// master. Both failures are the same missing invariant.
//
// planprogress already collects the paths per id (its taskRecord.paths) but deliberately does
// not judge uniqueness — its stated scope is plan/child consistency. The deciding reason is
// narrower than that, though: the two tools key on different things. planprogress indexes by
// the leading filename digits off a filesystem walk; this indexes by the frontmatter id off the
// git inventory. Putting the check there would have built a filename-collision check, which is
// a different check — and one that is still missing (TASK-343). The bug being fixed here is
// that doccheck's own link resolution keys on the frontmatter id, so doccheck owns that space.
//
// Note what this does *not* separate: PLAN-00n and TASK-00n never collide even where their
// filename numbers match, because the key is the id string. tasks/plan/'s skip is a second,
// independent reason for those — the namespaces would stay distinct without it.
//
// The inventory (inventory.go:82-85) merges tracked files with untracked non-ignored ones, so
// an unstaged scratch copy of a card fails the gate repo-wide with no waiver path. That reach
// is inherited, not new — checkCardStatus already fails on an untracked malformed card through
// the same inventory — and it is kept deliberately: a duplicate that only appears once staged
// is a duplicate someone has already started building on.
//
// Only files under a declared card zone are considered, and tasks/plan/ is excluded, matching
// checkCardStatus. Cards whose frontmatter declares no id are skipped: a missing id: is not a
// duplicate, and the zone sweep already has its own opinion about malformed frontmatter.
func checkDuplicateCardIDs(root string, inv []InventoryEntry) (idsSeen, duplicates int, msgs, errs []string) {
	paths := map[string][]string{}
	for _, e := range inv {
		if isSymlinkMode(e.Mode) {
			continue
		}
		zone, ok := resolveCardZone(e.Path)
		if !ok || zone.skip || !isMarkdownPath(e.Path) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: read: %v", e.Path, err))
			continue
		}
		frontmatter, state := splitFrontmatter(string(data))
		if state != frontmatterOK {
			continue
		}
		id, found, err := frontmatterField(frontmatter, "id")
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Path, err))
			continue
		}
		if !found || id == "" {
			continue
		}
		paths[id] = append(paths[id], e.Path)
	}

	idsSeen = len(paths)
	ids := make([]string, 0, len(paths))
	for id := range paths {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if len(paths[id]) < 2 {
			continue
		}
		duplicates++
		where := append([]string(nil), paths[id]...)
		sort.Strings(where)
		msgs = append(msgs, fmt.Sprintf("%s is claimed by %d cards: %s",
			id, len(where), strings.Join(where, ", ")))
	}
	return idsSeen, duplicates, msgs, errs
}

var cardFilenameNumberRE = regexp.MustCompile(`^0*(\d+)-.+\.md$`)

// checkDuplicateFilenameNumbers reports filename-number collisions within one card namespace.
//
// A filename number is the sequence a person or an agent sees when choosing the next card. It
// is intentionally distinct from the full frontmatter id: existing cards may use a filename
// number that differs from the id's numeric suffix, and enforcing equality would be a new
// compatibility rule rather than a collision guard. The frontmatter id's prefix separates
// namespaces, so ISSUE-001 may coexist with historical TASK-001, while two TASK cards named
// 331-... remain an actionable collision even if their complete ids differ.
//
// A plan is not a task card. That applies to both tasks/plan/ and archived plans under a
// `plan` path component; otherwise PLAN-006 and TASK-006 would produce a false collision.
// Missing or non-namespaced ids are left to the card-frontmatter checks that own that contract.
func checkDuplicateFilenameNumbers(root string, inv []InventoryEntry) (numbersSeen, duplicates int, msgs, errs []string) {
	paths := map[string][]string{}
	for _, e := range inv {
		if isSymlinkMode(e.Mode) || !isMarkdownPath(e.Path) || hasPlanPathComponent(e.Path) {
			continue
		}
		zone, ok := resolveCardZone(e.Path)
		if !ok || zone.skip {
			continue
		}
		match := cardFilenameNumberRE.FindStringSubmatch(path.Base(e.Path))
		if match == nil {
			continue
		}
		number, err := strconv.ParseUint(match[1], 10, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: invalid filename number %q: %v", e.Path, match[1], err))
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(e.Path)))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: read: %v", e.Path, err))
			continue
		}
		frontmatter, state := splitFrontmatter(string(data))
		if state != frontmatterOK {
			continue
		}
		id, found, err := frontmatterField(frontmatter, "id")
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", e.Path, err))
			continue
		}
		namespace, ok := cardIDNamespace(id, found)
		if !ok {
			continue
		}
		key := fmt.Sprintf("%s-%d", namespace, number)
		paths[key] = append(paths[key], e.Path)
	}

	numbersSeen = len(paths)
	keys := make([]string, 0, len(paths))
	for key := range paths {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if len(paths[key]) < 2 {
			continue
		}
		duplicates++
		where := append([]string(nil), paths[key]...)
		sort.Strings(where)
		msgs = append(msgs, fmt.Sprintf("filename number %s is claimed by %d cards: %s",
			key, len(where), strings.Join(where, ", ")))
	}
	return numbersSeen, duplicates, msgs, errs
}

func hasPlanPathComponent(cardPath string) bool {
	return strings.Contains("/"+path.Clean(cardPath)+"/", "/plan/")
}

func cardIDNamespace(id string, found bool) (string, bool) {
	if !found || id == "" {
		return "", false
	}
	namespace, _, ok := strings.Cut(id, "-")
	if !ok || namespace == "" {
		return "", false
	}
	return namespace, true
}
