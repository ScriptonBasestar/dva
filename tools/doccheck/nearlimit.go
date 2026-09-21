package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// nearLimitEntry is one size-enforced document past 80% of either limit.
type nearLimitEntry struct {
	path   string
	lines  int
	nbytes int
}

// collectNearLimit returns the headroom documents in the inventory, sorted by
// bytes descending (path ascending on ties) so the listing is reproducible.
func collectNearLimit(root string, inv []InventoryEntry) ([]nearLimitEntry, error) {
	var out []nearLimitEntry
	for _, e := range inv {
		if !isMarkdownPath(e.Path) || !sizeEnforced(e.Path) || isSymlinkMode(e.Mode) {
			continue
		}
		full := filepath.Join(root, filepath.FromSlash(e.Path))
		data, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("%s: read: %w", e.Path, err)
		}
		lines := countLines(string(data))
		if isHeadroom(lines, len(data)) {
			out = append(out, nearLimitEntry{path: e.Path, lines: lines, nbytes: len(data)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].nbytes != out[j].nbytes {
			return out[i].nbytes > out[j].nbytes
		}
		return out[i].path < out[j].path
	})
	return out, nil
}

// runNearLimit prints the reproducible near-limit listing (TASK-405).
// Informational only: always exits 0, never fails the gate.
func runNearLimit(root string) error {
	inv, err := LoadInventory(root)
	if err != nil {
		return err
	}
	entries, err := collectNearLimit(root, inv)
	if err != nil {
		return err
	}
	for _, e := range entries {
		fmt.Printf("%s: %d lines, %d bytes\n", e.path, e.lines, e.nbytes)
	}
	fmt.Printf("near_limit_docs: %d (over 80%% of %d lines / %d bytes, under both limits)\n",
		len(entries), maxDocLines, maxDocBytes)
	return nil
}
