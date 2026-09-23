// Command changelogcheck fails the build when a backlog of user-visible code has
// landed without a single line reaching CHANGELOG.md's `## [Unreleased]` section.
//
// The drift this catches is not hypothetical. Between v0.2.0 and the commit that
// introduced this gate, 66 commits touched internal/ or cmd/ — 48 of them feat, fix
// or refactor, 92 files, +12700/-530 — while `## [Unreleased]` stayed empty. TASK-369
// had already fixed the same symptom five days earlier by hand and added no guard, so
// it simply happened again. The existing release tooling cannot catch it: releasecheck
// and releaseworkflow need a RELEASE_TAG and therefore only ever run at release time,
// by which point the backlog is already whatever it is. Nothing measured the gap
// continuously, so nothing reported it growing.
//
// The rule deliberately has a threshold rather than demanding an entry per commit. A
// gate that fires on the first unlogged fix would be routed around within a week — it
// would force a CHANGELOG edit onto every one-line change, including the test-only and
// refactor churn that has nothing to tell a user. drainThreshold is where "ordinary
// work in flight" stops and "a release's worth of undocumented change" starts: it is
// far enough above a normal working batch to stay quiet during one, and far enough
// below 66 that the drift is named while it is still an afternoon's writing rather
// than an archaeology project.
//
// When the thing being measured is not there — no release tag reachable from HEAD, no
// git, a shallow clone whose tags were not fetched — this program prints an explicit
// unavailable line and exits 0. It does not exit nonzero, because it has not found a
// defect; and it does not print the OK line, because it has not verified anything
// either. A checker that cannot measure must not claim a verdict in either direction:
// in this repository a broken gate is worse than a red board.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
)

// drainThreshold is the number of qualifying commits at which an empty `## [Unreleased]`
// becomes a defect rather than a normal in-flight state. See the package comment for why
// it is not 1. Lowering it toward 1 makes the gate noisy on single-commit work; raising it
// lets the backlog grow past the point where it can be written up from memory.
const drainThreshold = 10

// codePaths are the trees whose commits are user-visible enough to owe a CHANGELOG line.
// Everything else — tools/, docs/, tasks/, workflows/ — is repository housekeeping that a
// user of the dva binary never observes.
var codePaths = []string{"internal/", "cmd/"}

// codePathsLabel names codePaths the way a person would read them in a report line.
func codePathsLabel() string { return strings.Join(codePaths, " or ") }

// loggableTypes are the conventional-commit types that change what the binary does. test,
// docs, chore, style and build commits are excluded: requiring a CHANGELOG entry for them
// would make the threshold meaningless, since they are the bulk of the churn.
var loggableTypes = []string{"feat", "fix", "refactor"}

// subjectType matches the `type(scope):` prefix that commitcheck already enforces on this
// repository's subjects, so the type is a parse and not a guess.
var subjectRe = regexp.MustCompile(`^([a-z]+)(\([^)]*\))?!?:`)

// isLoggable reports whether a commit subject names a change a reader of CHANGELOG.md
// would expect to find there.
func isLoggable(subject string) bool {
	m := subjectRe.FindStringSubmatch(strings.TrimSpace(subject))
	if m == nil {
		return false
	}
	return slices.Contains(loggableTypes, m[1])
}

// countLoggable counts the subjects that owe a CHANGELOG line.
func countLoggable(subjects []string) int {
	n := 0
	for _, s := range subjects {
		if isLoggable(s) {
			n++
		}
	}
	return n
}

// unreleasedState is what the CHANGELOG could be made to say about its own top section.
type unreleasedState int

const (
	unreleasedMissing unreleasedState = iota // no `## [Unreleased]` heading at all
	unreleasedEmpty                          // heading present, nothing but blank lines under it
	unreleasedFilled                         // heading present with content
)

// inspectUnreleased classifies the `## [Unreleased]` section of a CHANGELOG body. Content
// is anything non-blank between that heading and the next `## ` heading (or EOF); a section
// holding only sub-headings with no bullets still counts as filled, because someone wrote
// them on purpose and this gate is not a style checker.
func inspectUnreleased(content string) unreleasedState {
	lines := strings.Split(content, "\n")
	start := -1
	for i, l := range lines {
		if strings.TrimSpace(l) == "## [Unreleased]" {
			start = i
			break
		}
	}
	if start < 0 {
		return unreleasedMissing
	}
	for _, l := range lines[start+1:] {
		if strings.HasPrefix(l, "## ") {
			break
		}
		if strings.TrimSpace(l) != "" {
			return unreleasedFilled
		}
	}
	return unreleasedEmpty
}

// releaseVersion is the deliberately narrow stable release version the release workflow uses:
// three non-negative decimal components, with no pre-release or build suffix. Changelogcheck
// only recognizes this exact shape for a pending candidate, so a free-form heading cannot turn
// an otherwise empty Unreleased section into a pass.
type releaseVersion struct {
	major uint64
	minor uint64
	patch uint64
}

var releaseVersionRE = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
var releaseHeadingRE = regexp.MustCompile(`^## \[((?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*))\] - (\d{4}-\d{2}-\d{2})$`)

func parseReleaseVersion(value string) (releaseVersion, error) {
	match := releaseVersionRE.FindStringSubmatch(value)
	if match == nil {
		return releaseVersion{}, fmt.Errorf("must be a stable X.Y.Z version, got %q", value)
	}
	parts := [3]uint64{}
	for i := range parts {
		parsed, err := strconv.ParseUint(match[i+1], 10, 64)
		if err != nil {
			return releaseVersion{}, fmt.Errorf("parse version %q: %w", value, err)
		}
		parts[i] = parsed
	}
	return releaseVersion{major: parts[0], minor: parts[1], patch: parts[2]}, nil
}

func (v releaseVersion) newerThan(other releaseVersion) bool {
	if v.major != other.major {
		return v.major > other.major
	}
	if v.minor != other.minor {
		return v.minor > other.minor
	}
	return v.patch > other.patch
}

// sourceVersion reads the current source declaration rather than importing config.Version.
// changelogcheck accepts an optional root for fixture and detached-worktree checks; importing
// this tool's compiled package would inspect the checkout that compiled the tool, not that root.
// AST parsing also ensures a comment or lookalike string cannot masquerade as the declaration.
func sourceVersion(root string) (string, error) {
	path := filepath.Join(root, "internal", "config", "version.go")
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return "", fmt.Errorf("parse source Version: %w", err)
	}

	var value string
	found := false
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.VAR {
			continue
		}
		for _, spec := range gen.Specs {
			values, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range values.Names {
				if name.Name != "Version" {
					continue
				}
				if found || i >= len(values.Values) {
					return "", fmt.Errorf("source Version must have exactly one string literal declaration")
				}
				literal, ok := values.Values[i].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return "", fmt.Errorf("source Version must be a string literal")
				}
				value, err = strconv.Unquote(literal.Value)
				if err != nil {
					return "", fmt.Errorf("unquote source Version: %w", err)
				}
				found = true
			}
		}
	}
	if !found {
		return "", fmt.Errorf("source Version declaration not found")
	}
	return value, nil
}

// pendingReleaseCandidate recognizes only the release shape the manual runbook requires:
// empty Unreleased, then the current source version's immediately following dated and populated
// section. The source version must be strictly newer than the reachable release tag. This is not
// a generic nonempty-section escape hatch: mismatched, old, undated, or empty sections fail.
func pendingReleaseCandidate(root, content, tag string) (version string, ok bool) {
	source, err := sourceVersion(root)
	if err != nil {
		return "", false
	}
	sourceParsed, err := parseReleaseVersion(source)
	if err != nil {
		return "", false
	}
	if !strings.HasPrefix(tag, "v") {
		return "", false
	}
	tagParsed, err := parseReleaseVersion(strings.TrimPrefix(tag, "v"))
	if err != nil || !sourceParsed.newerThan(tagParsed) {
		return "", false
	}

	lines := strings.Split(content, "\n")
	start := -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "## [Unreleased]" {
			start = i
			break
		}
	}
	if start < 0 {
		return "", false
	}

	heading := -1
	for i := start + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "## ") {
			heading = i
			break
		}
		if strings.TrimSpace(lines[i]) != "" {
			return "", false
		}
	}
	if heading < 0 {
		return "", false
	}
	match := releaseHeadingRE.FindStringSubmatch(lines[heading])
	if match == nil || match[1] != source {
		return "", false
	}
	if _, err := time.Parse("2006-01-02", match[2]); err != nil {
		return "", false
	}
	for _, line := range lines[heading+1:] {
		if strings.HasPrefix(line, "## ") {
			break
		}
		if strings.TrimSpace(line) != "" {
			return source, true
		}
	}
	return "", false
}

// git runs a git command in dir. Every failure here means "cannot measure", never "measured
// zero", so callers degrade rather than report a clean run.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %v: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(string(out)), nil
}

// latestTag returns the most recent release tag reachable from HEAD.
func latestTag(dir string) (string, error) {
	return git(dir, "describe", "--tags", "--abbrev=0", "--match", "v*", "HEAD")
}

// loggableSince returns the subjects of the commits after tag that touch codePaths.
func loggableSince(dir, tag string) ([]string, error) {
	args := []string{"log", "--no-merges", "--format=%s", tag + "..HEAD", "--"}
	args = append(args, codePaths...)
	out, err := git(dir, args...)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}

	changelog := filepath.Join(root, "CHANGELOG.md")
	body, err := os.ReadFile(changelog)
	if err != nil {
		fmt.Fprintf(os.Stderr, "changelogcheck: %v\n", err)
		os.Exit(2)
	}

	tag, err := latestTag(root)
	if err != nil {
		fmt.Printf("changelogcheck: unavailable — cannot resolve a release tag reachable from HEAD (%v); "+
			"nothing was measured and no verdict is claimed\n", err)
		return
	}

	subjects, err := loggableSince(root, tag)
	if err != nil {
		fmt.Printf("changelogcheck: unavailable — cannot read history since %s (%v); "+
			"nothing was measured and no verdict is claimed\n", tag, err)
		return
	}
	n := countLoggable(subjects)

	switch inspectUnreleased(string(body)) {
	case unreleasedMissing:
		fmt.Fprintf(os.Stderr, "ERROR: %s has no '## [Unreleased]' section; "+
			"this gate and the release runbook both require it\n", changelog)
		fmt.Println("changelogcheck: FAIL")
		os.Exit(1)
	case unreleasedEmpty:
		if version, ok := pendingReleaseCandidate(root, string(body), tag); ok {
			fmt.Printf("changelogcheck: OK — %d loggable commit(s) touching %s since %s; "+
				"Unreleased is drained into the populated pending %s release section\n",
				n, codePathsLabel(), tag, version)
			return
		}
		if n >= drainThreshold {
			fmt.Fprintf(os.Stderr, "ERROR: %d commit(s) touching %s since %s are feat/fix/refactor, "+
				"but '## [Unreleased]' in %s is empty (threshold %d). Write them up before the backlog "+
				"outgrows anyone's memory of it — see docs/52-manual-release-runbook.md\n",
				n, codePathsLabel(), tag, changelog, drainThreshold)
			fmt.Println("changelogcheck: FAIL")
			os.Exit(1)
		}
		fmt.Printf("changelogcheck: OK — %d loggable commit(s) touching %s since %s, below the %d threshold "+
			"that requires an '## [Unreleased]' entry\n", n, codePathsLabel(), tag, drainThreshold)
	case unreleasedFilled:
		fmt.Printf("changelogcheck: OK — %d loggable commit(s) touching %s since %s, and '## [Unreleased]' "+
			"has content\n", n, codePathsLabel(), tag)
	}
}
