package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
// not judge uniqueness — its stated scope is plan/child consistency. Card hygiene lives here,
// beside the zone/status sweep, which is why the check is in doccheck rather than there.
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
		id, found := frontmatterField(frontmatter, "id")
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
