package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// Provision markers record that a profile has already been run, so `dva up` can stop
// suggesting it. They are named after the profile and live in the dot directory, which is
// also where a user's own modules live — and that shared namespace is the whole reason the
// naming below is more careful than "prefix plus profile".
const (
	provisionMarkerPrefix = "provisioned-"

	// provisionMarkerExt is what keeps a marker distinguishable from a module. Both sit at the
	// same level of the dot directory and both carry a name a person chose — a profile here, a
	// module there — so without an extension `provisioned-*` names the user's own
	// `provisioned-base.yml` as readily as DVA's `provisioned-default`. That collision was not
	// theoretical: it is what provisionMarkers matched on, so `dva down --purge` deleted the
	// module.
	provisionMarkerExt = ".marker"

	// moduleExt is the one spelling the module loader reads (see internal/config).
	moduleExt = ".yml"
)

// authoredExts are the extensions that say a file in the dot directory is something a person
// wrote. Nothing here is deletable by --purge and nothing here is transient state doctor should
// report, whatever its name looks like.
//
// It is wider than moduleExt on purpose, and the extra two are not hypothetical. The loader reads
// only `.yml`, so a module misspelled `.yaml` — the commoner spelling everywhere else — is already
// inert, and a person debugging that has DVA quietly deleting the file when they run `--purge` to
// reset state. `.md` is the note somebody leaves next to their config. Both carry the marker
// prefix as readily as `provisioned-base.yml` did, which is the collision that started all of
// this, one letter apart.
//
// A legacy marker for a profile named `setup-v1.0` has extension `.0`, is in none of these, and
// is still cleared — which is the case a positive "looks like a marker" rule strands.
var authoredExts = []string{moduleExt, ".yaml", ".md"}

// authoredFile reports whether a name in the dot directory is content rather than a marker. It is
// the one predicate both the deletion in this file and doctor's pathspec in gitignore.go are
// built from, so the two cannot disagree about what a marker is.
func authoredFile(name string) bool {
	return slices.Contains(authoredExts, filepath.Ext(name))
}

// provisionMarkerName is the file name recording that a profile has been provisioned.
//
// The marker stays in the *invoked* project's dot-directory even for an imported profile:
// it answers "has this project been provisioned", which is the question `dva up` asks of
// the config it was run against, not of the child that owns the steps.
func provisionMarkerName(profile string) string {
	return legacyProvisionMarkerName(profile) + provisionMarkerExt
}

// legacyProvisionMarkerName is the extensionless spelling written before the extension
// existed. It is read and never written: a repository that provisioned under the old name
// must not be told to provision all over again, and nobody chose to pay for a rename in
// re-run provision steps. Nothing migrates the files — the old name keeps working until
// `--purge` clears it and the next run writes the new one.
//
// The slash replacement is what makes an imported name usable as a file name at all.
// Imports register canonically as `child/profile`, and filepath.Join then read that slash
// as a directory component: MkdirAll had created only the dot-directory, so the write
// failed with ENOENT and every imported provision run ended on a warning. The literal "/"
// is the separator applySubprojectImports writes, on every platform (TASK-264).
func legacyProvisionMarkerName(profile string) string {
	return provisionMarkerPrefix + strings.ReplaceAll(profile, "/", "__")
}

// provisionMarkerExists answers whether a profile has been provisioned, under either
// spelling. Callers ask this instead of stat-ing a name they built themselves, so that the
// legacy spelling cannot be honoured in one place and forgotten in another.
func provisionMarkerExists(configDir, profile string) bool {
	markerDir := filepath.Join(configDir, config.DotDirName)
	for _, name := range []string{provisionMarkerName(profile), legacyProvisionMarkerName(profile)} {
		if _, err := os.Stat(filepath.Join(markerDir, name)); err == nil {
			return true
		}
	}
	return false
}

// isProvisionMarker decides what clearProvisionMarkers is allowed to delete, and it is
// written as an exclusion rather than as a shape: every non-directory carrying the prefix
// is a marker except authored content. A positive rule — "ends in .marker, or has no
// extension" — reads tighter but quietly abandons a legacy marker for a profile whose name
// contains a dot, which would then survive `--purge` and suppress provisioning forever.
//
// Directories are not markers whatever they are called. DVA writes markers as flat files, so
// a `provisioned-*` directory is someone else's, and deleting into it is not this command's
// business.
func isProvisionMarker(e fs.DirEntry) bool {
	return !e.IsDir() &&
		strings.HasPrefix(e.Name(), provisionMarkerPrefix) &&
		!authoredFile(e.Name())
}

// provisionMarkers lists what clearProvisionMarkers would delete, as full paths.
//
// The probe-only half of the same walk, in the shape of portOwnerPIDs against reclaimPort:
// `dva down --purge --dry-run` needs to name the files without removing them, and a preview
// that re-derived the rule itself would be free to drift from the deletion it claims to
// describe. Returns nil for an unreadable directory, which is the same silence
// clearProvisionMarkers has always kept — a missing .sb/dva is the ordinary case on a
// project that has never provisioned. TASK-166.
func provisionMarkers(configDir string) []string {
	markerDir := filepath.Join(configDir, config.DotDirName)
	entries, err := os.ReadDir(markerDir)
	if err != nil {
		return nil
	}
	var found []string
	for _, e := range entries {
		if isProvisionMarker(e) {
			found = append(found, filepath.Join(markerDir, e.Name()))
		}
	}
	return found
}

// clearProvisionMarkers removes all provision marker files from .sb/dva/.
// Called by `dva down --purge` so that provision suggestions reappear after a reset.
func clearProvisionMarkers(configDir string) {
	for _, m := range provisionMarkers(configDir) {
		_ = os.Remove(m)
	}
}

// writeProvisionMarker creates a marker file indicating that a provision
// profile has been run. Used by `dva up` to skip provision suggestions.
func writeProvisionMarker(configDir, profile string) {
	markerDir := filepath.Join(configDir, config.DotDirName)
	if err := os.MkdirAll(markerDir, 0755); err != nil {
		return
	}
	markerFile := filepath.Join(markerDir, provisionMarkerName(profile))
	if err := os.WriteFile(markerFile, []byte(""), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "[warn] could not write provision marker: %v\n", err)
	}
}
