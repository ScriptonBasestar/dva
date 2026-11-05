// Command affectedtests prints Go packages whose tests cover the current change.
//
// It considers the merge-base delta plus staged, unstaged, deleted, renamed, and
// untracked files. Unknown or shared inputs intentionally widen to ./...; the
// selector is edit feedback, never a replacement for commit or full CI.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

type packageInfo struct {
	path string
	dir  string
	deps []string
}

type selector struct {
	run func(args ...string) ([]byte, error)
}

func main() {
	base := flag.String("base", "", "required source ref")
	flag.Parse()
	if *base == "" || flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "usage: affectedtests --base <source-ref>")
		os.Exit(2)
	}
	s := selector{run: gitOrGo}
	packages, err := s.selectPackages(*base)
	if err != nil {
		fmt.Fprintf(os.Stderr, "affectedtests: %v\n", err)
		os.Exit(2)
	}
	for _, pkg := range packages {
		fmt.Println(pkg)
	}
}

func gitOrGo(args ...string) ([]byte, error) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	return cmd.Output()
}

func (s selector) selectPackages(base string) ([]string, error) {
	baseCommit, err := s.run("git", "rev-parse", "--verify", base+"^{commit}")
	if err != nil {
		return nil, fmt.Errorf("cannot resolve base %q: %w", base, err)
	}
	mergeBase, err := s.run("git", "merge-base", strings.TrimSpace(string(baseCommit)), "HEAD")
	if err != nil {
		return nil, fmt.Errorf("cannot determine merge-base with %q: %w", base, err)
	}

	changes := map[string]bool{}
	for _, args := range [][]string{
		{"git", "diff", "--name-status", "-z", strings.TrimSpace(string(mergeBase)), "HEAD"},
		{"git", "diff", "--name-status", "-z", "--cached"},
		{"git", "diff", "--name-status", "-z"},
		{"git", "ls-files", "-z", "--others", "--exclude-standard"},
	} {
		output, runErr := s.run(args...)
		if runErr != nil {
			return nil, fmt.Errorf("%s: %w", strings.Join(args[:2], " "), runErr)
		}
		paths, deleted, parseErr := changedPaths(args[1] == "diff", output)
		if parseErr != nil {
			return nil, parseErr
		}
		if deleted {
			return []string{"./..."}, nil
		}
		for _, path := range paths {
			changes[path] = true
		}
	}
	if len(changes) == 0 {
		return nil, errors.New("no tracked or untracked changes; refusing an empty selection")
	}

	packages, err := s.packages()
	if err != nil {
		return nil, err
	}
	selected, full := affected(packages, mapKeys(changes))
	if full {
		return []string{"./..."}, nil
	}
	if len(selected) == 0 {
		return nil, errors.New("no Go package is affected by this change")
	}
	return selected, nil
}

func (s selector) packages() ([]packageInfo, error) {
	output, err := s.run("go", "list", "-f", "{{.ImportPath}}\t{{.Dir}}\t{{join .Imports \" \"}} {{join .TestImports \" \"}} {{join .XTestImports \" \"}}", "./...")
	if err != nil {
		return nil, fmt.Errorf("go list packages: %w", err)
	}
	root, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("read working directory: %w", err)
	}
	var result []packageInfo
	for line := range strings.SplitSeq(strings.TrimSuffix(string(output), "\n"), "\n") {
		parts := strings.Split(line, "\t")
		if len(parts) != 3 || parts[0] == "" || parts[1] == "" {
			return nil, fmt.Errorf("unexpected go list output %q", line)
		}
		rel, relErr := filepath.Rel(root, parts[1])
		if relErr != nil || strings.HasPrefix(rel, "..") {
			return nil, fmt.Errorf("package %s lies outside repository", parts[0])
		}
		result = append(result, packageInfo{path: parts[0], dir: filepath.ToSlash(rel), deps: strings.Fields(parts[2])})
	}
	return result, nil
}

func changedPaths(status bool, output []byte) ([]string, bool, error) {
	if len(output) == 0 {
		return nil, false, nil
	}
	if output[len(output)-1] != 0 {
		return nil, false, errors.New("git name-status output is not NUL terminated")
	}
	fields := bytes.Split(output[:len(output)-1], []byte{0})
	var paths []string
	widen := false
	for len(fields) > 0 {
		if !status {
			if len(fields[0]) == 0 {
				return nil, false, errors.New("git path output contains an empty path")
			}
			paths, fields = append(paths, string(fields[0])), fields[1:]
			continue
		}
		statusField := string(fields[0])
		fields = fields[1:]
		if len(fields) == 0 || !validStatus(statusField) {
			return nil, false, errors.New("malformed git name-status output")
		}
		pathsNeeded := 1
		if statusField[0] == 'R' || statusField[0] == 'C' {
			pathsNeeded = 2
		}
		if len(fields) < pathsNeeded {
			return nil, false, errors.New("malformed git name-status output")
		}
		for _, path := range fields[:pathsNeeded] {
			if len(path) == 0 {
				return nil, false, errors.New("git name-status output contains an empty path")
			}
		}
		if statusField[0] == 'D' || statusField[0] == 'R' || statusField[0] == 'C' || statusField[0] == 'T' {
			widen = true
		}
		for _, path := range fields[:pathsNeeded] {
			paths = append(paths, string(path))
		}
		fields = fields[pathsNeeded:]
	}
	return paths, widen, nil
}

func validStatus(status string) bool {
	if status == "" {
		return false
	}
	if len(status) == 1 {
		return strings.ContainsRune("ADMTUXB", rune(status[0]))
	}
	if (status[0] != 'R' && status[0] != 'C') || len(status) == 1 {
		return false
	}
	for _, r := range status[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func affected(packages []packageInfo, paths []string) ([]string, bool) {
	byPath := map[string]packageInfo{}
	byDir := map[string]string{}
	for _, pkg := range packages {
		byPath[pkg.path] = pkg
		byDir[pkg.dir] = pkg.path
	}
	seeds := map[string]bool{}
	for _, path := range paths {
		path = filepath.ToSlash(path)
		if sharedInput(path) {
			return nil, true
		}
		if !strings.HasSuffix(path, ".go") {
			return nil, true
		}
		dir := filepath.ToSlash(filepath.Dir(path))
		pkg, ok := byDir[dir]
		if !ok {
			return nil, true
		}
		seeds[pkg] = true
	}
	if len(seeds) == 0 {
		return nil, false
	}
	reverse := map[string][]string{}
	for _, pkg := range packages {
		for _, dep := range pkg.deps {
			if _, exists := byPath[dep]; exists {
				reverse[dep] = append(reverse[dep], pkg.path)
			}
		}
	}
	queue := mapKeys(seeds)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		for _, dependent := range reverse[current] {
			if !seeds[dependent] {
				seeds[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}
	return mapKeys(seeds), false
}

func sharedInput(path string) bool {
	return path == "go.mod" || path == "go.sum" || path == "dva.yml" || path == "Makefile" ||
		strings.HasPrefix(path, ".github/") || strings.HasPrefix(path, "scripts/") ||
		strings.HasPrefix(path, "tools/") || strings.HasPrefix(path, "internal/cirun/")
}

func mapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for value := range values {
		keys = append(keys, value)
	}
	slices.Sort(keys)
	return keys
}
