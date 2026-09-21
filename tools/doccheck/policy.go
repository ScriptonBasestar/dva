package main

// Size policy for option B (TASK-090): size limits apply only under docs/ and
// workflows/. Relative link checking is repository-wide (every inventory .md).
const (
	maxDocLines = 500
	maxDocBytes = 10240 // 10 KiB
)

// Headroom thresholds (TASK-405): a size-enforced document past 80% of either
// limit is reported as a warning before the hard gate fails. Advisory only —
// oversize stays fatal and headroom never flips the outcome.
const (
	headroomLineWarn = 400  // 80% of maxDocLines
	headroomByteWarn = 8192 // 80% of maxDocBytes
)

// isHeadroom reports whether a size-enforced document is past 80% of either
// limit while still under both hard limits. Oversized documents are not
// headroom: they fail the gate instead of warning.
func isHeadroom(lines, nbytes int) bool {
	if lines > maxDocLines || nbytes > maxDocBytes {
		return false
	}
	return lines > headroomLineWarn || nbytes > headroomByteWarn
}

// Git ls-files modes (high bits of the 6-digit octal mode).
const (
	modeRegular = 0o100644
	modeSymlink = 0o120000
	modeMask    = 0o170000
)

func isSymlinkMode(mode int) bool {
	return mode&modeMask == modeSymlink
}

func isMarkdownPath(path string) bool {
	return len(path) >= 3 && (path[len(path)-3:] == ".md" || path[len(path)-3:] == ".MD")
}

// sizeEnforced reports whether path is under the option-B size gate
// (docs/, workflows/). There is no per-file escape hatch: a document that
// cannot meet the limits is split, not exempted.
// Lookup manuals (USAGE.md, skills/*/references/, library/) are outside
// these prefixes and are not size-checked — they are still link-checked.
func sizeEnforced(path string) bool {
	return hasPathPrefix(path, "docs/") || hasPathPrefix(path, "workflows/")
}

func hasPathPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}
