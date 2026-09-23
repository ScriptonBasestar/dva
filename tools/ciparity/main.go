// Command ciparity keeps Makefile's doc-check target and dva.yml's ci.profiles
// commit "docs" step running the exact same leaf tools.
//
// TASK-413: dva.yml's `docs` CI step and Makefile's `doc-check` target each hand-list
// the same `go run ./tools/...` gates as independent copies (Make aliases must not call
// back into `dva ci`, and `make doc-check` must stay a fast, standalone path — see
// skills/dva-ci/SKILL.md and docs/53-ci-profiles.md). Nothing compared the two lists
// before this: TASK-412's gap (two doc gates that existed only in `make doc-check`, never
// added to dva.yml's `docs` step) was invisible to every existing gate. This program
// extracts both lists and fails when they disagree.
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// ciparity does not require dva.yml to also run itself as a "docs" tool: it is the
// comparator, not one of the compared gates. Without this exclusion, adding
// `go run ./tools/ciparity` to Makefile's doc-check target would make the tool
// permanently report itself as an "extra" Makefile tool.
const selfToolName = "ciparity"

func main() {
	root := "."
	if len(os.Args) > 1 {
		root = os.Args[1]
	}
	mfPath := filepath.Join(root, "Makefile")
	dvaPath := filepath.Join(root, "dva.yml")

	mfTools, err := docCheckToolsFromMakefile(mfPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ciparity: Makefile: %v\n", err)
		os.Exit(2)
	}
	commitTools, err := docsStepToolsFromDvaYAML(dvaPath, "commit")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ciparity: dva.yml: %v\n", err)
		os.Exit(2)
	}
	// full reuses commit's docs step via a YAML anchor (&ci-docs / *ci-docs) today, but an
	// anchor is an authoring convenience, not a guarantee — a future edit could replace it
	// with a literal, independently-drifting copy. Compare both, not just commit's.
	fullTools, err := docsStepToolsFromDvaYAML(dvaPath, "full")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ciparity: dva.yml: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("Makefile doc-check tools (%d): %s\n", len(mfTools), strings.Join(sortedKeys(mfTools), ", "))
	fmt.Printf("dva.yml ci.profiles.commit docs step tools (%d): %s\n", len(commitTools), strings.Join(sortedKeys(commitTools), ", "))
	fmt.Printf("dva.yml ci.profiles.full docs step tools (%d): %s\n", len(fullTools), strings.Join(sortedKeys(fullTools), ", "))

	var errs []string
	errs = append(errs, diffToolSets("Makefile doc-check", mfTools, "dva.yml's commit/docs step", commitTools)...)
	errs = append(errs, diffToolSets("Makefile doc-check", mfTools, "dva.yml's full/docs step", fullTools)...)

	if len(errs) == 0 {
		fmt.Println("ciparity: OK — Makefile doc-check, dva.yml commit/docs and dva.yml full/docs all run the same tools")
		return
	}
	for _, e := range errs {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", e)
	}
	fmt.Fprintf(os.Stderr, "hint: keep Makefile's doc-check recipe and dva.yml's ci.profiles \"docs\" step run: lines listing the same ./tools/* commands\n")
	os.Exit(1)
}

var (
	reMakeTarget = regexp.MustCompile(`^([A-Za-z0-9_.-]+):`)
	reGoRunTool  = regexp.MustCompile(`go run \./tools/(\S+)`)
)

// docCheckToolsFromMakefile extracts the `go run ./tools/*` gates run by the
// Makefile's doc-check recipe. It scans only that target's recipe lines (tab-indented,
// stopping at the next target or a blank line) so an unrelated target's tool commands
// are never counted.
func docCheckToolsFromMakefile(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	out := map[string]bool{}
	sc := bufio.NewScanner(f)
	inTarget := false
	for sc.Scan() {
		line := sc.Text()
		if inTarget {
			if strings.HasPrefix(line, "\t") {
				if m := reGoRunTool.FindStringSubmatch(line); m != nil && m[1] != selfToolName {
					out[m[1]] = true
				}
				continue
			}
			if strings.TrimSpace(line) == "" {
				continue
			}
			inTarget = false
		}
		if m := reMakeTarget.FindStringSubmatch(line); m != nil && m[1] == "doc-check" {
			inTarget = true
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no doc-check target with go run ./tools/* recipe lines found in %s", path)
	}
	return out, nil
}

// docsStepDvaStepName is the ci.profiles step name this tool compares — the tool's
// whole purpose is checking that profile's docs gate against Makefile's doc-check, so
// this is a fixed fact about what ciparity does, not a caller-supplied choice.
const docsStepDvaStepName = "docs"

// docsStepToolsFromDvaYAML extracts the `go run ./tools/*` gates declared in the named
// ci.profiles's "docs" step run: command, using the project's own config loader (the
// same decoder dva ci runs against) rather than a hand-rolled YAML scan.
func docsStepToolsFromDvaYAML(path, profileName string) (map[string]bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg, _, err := config.ValidateConfigBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if cfg.CI == nil {
		return nil, fmt.Errorf("no ci.profiles declared")
	}
	profile, ok := cfg.CI.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("ci profile %q is not declared", profileName)
	}
	for _, step := range profile.Steps {
		if step.Name != docsStepDvaStepName {
			continue
		}
		matches := reGoRunTool.FindAllStringSubmatch(step.Run, -1)
		if len(matches) == 0 {
			return nil, fmt.Errorf("ci profile %q step %q has no go run ./tools/* commands", profileName, docsStepDvaStepName)
		}
		out := make(map[string]bool, len(matches))
		for _, m := range matches {
			out[m[1]] = true
		}
		return out, nil
	}
	return nil, fmt.Errorf("ci profile %q has no step named %q", profileName, docsStepDvaStepName)
}

// diffToolSets reports, as human-readable messages, the tools present in only one of
// the two named sets. It returns nil when the sets are identical.
func diffToolSets(aName string, a map[string]bool, bName string, b map[string]bool) []string {
	var onlyInA, onlyInB []string
	for t := range a {
		if !b[t] {
			onlyInA = append(onlyInA, t)
		}
	}
	for t := range b {
		if !a[t] {
			onlyInB = append(onlyInB, t)
		}
	}
	sort.Strings(onlyInA)
	sort.Strings(onlyInB)

	var out []string
	if len(onlyInA) > 0 {
		out = append(out, fmt.Sprintf("%s runs tools %s does not: %s", aName, bName, strings.Join(onlyInA, ", ")))
	}
	if len(onlyInB) > 0 {
		out = append(out, fmt.Sprintf("%s runs tools %s does not: %s", bName, aName, strings.Join(onlyInB, ", ")))
	}
	return out
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
