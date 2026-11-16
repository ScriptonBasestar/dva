// Command commitverify checks staged changes without changing the index or source.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

type runner func(context.Context, ...string) ([]byte, error)

type streamOutputKey struct{}

type change struct {
	status string
	paths  []string
}

type selection struct {
	goFiles        []string
	all            bool
	docs           bool
	productionDirs map[string]bool
	testDirs       map[string]bool
}

func main() {
	phase := flag.String("phase", "integrity", "integrity, lint, compile, or test")
	flag.Parse()
	if flag.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "commitverify: unexpected argument")
		os.Exit(2)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 110*time.Second)
	defer cancel()
	if err := verify(ctx, *phase, execCommand, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "commitverify: %v\n", err)
		os.Exit(1)
	}
}

func execCommand(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	cmd.Stderr = os.Stderr
	if out, ok := ctx.Value(streamOutputKey{}).(io.Writer); ok &&
		(args[0] == "make" || (args[0] == "go" && len(args) > 1 && (args[1] == "vet" || args[1] == "test"))) {
		cmd.Stdout = out
		return nil, cmd.Run()
	}
	return cmd.Output()
}

func verify(ctx context.Context, phase string, run runner, out io.Writer) error {
	ctx = context.WithValue(ctx, streamOutputKey{}, out)
	if phase != "integrity" && phase != "lint" && phase != "compile" && phase != "test" {
		return fmt.Errorf("unknown phase %q", phase)
	}
	changes, err := staged(ctx, run)
	if err != nil {
		return err
	}
	sel := selectChanges(changes)
	if sel.all {
		return errors.New("change requires core/full; run make ci-core or make ci-full")
	}
	if phase == "integrity" {
		if _, err := run(ctx, "git", "diff", "--cached", "--check"); err != nil {
			return fmt.Errorf("staged whitespace check: %w", err)
		}
		if len(sel.goFiles) > 0 {
			args := append([]string{"gofmt", "-s", "-l"}, sel.goFiles...)
			bad, err := run(ctx, args...)
			if err != nil {
				return fmt.Errorf("gofmt: %w", err)
			}
			if len(bad) != 0 {
				return fmt.Errorf("unformatted staged Go files: %s", bad)
			}
		}
		if sel.docs {
			output, err := run(ctx, "make", "doc-check")
			_, _ = out.Write(output)
			if err != nil {
				return fmt.Errorf("documentation check: %w", err)
			}
		}
		return nil
	}
	if !sel.all && len(sel.productionDirs) == 0 && len(sel.testDirs) == 0 {
		_, _ = fmt.Fprintf(out, "%s: not applicable (documentation-only changes)\n", phase)
		return nil
	}
	packages := []string{"./..."}
	if !sel.all {
		packages, err = affectedPackages(ctx, run, sel.productionDirs, sel.testDirs)
		if err != nil {
			return err
		}
	}
	var args []string
	switch phase {
	case "lint":
		args = []string{"go", "vet"}
	case "compile":
		// go test -run alone still executes init and TestMain. The exec wrapper
		// makes Go build and link each test binary without starting it.
		args = []string{"go", "test", "-exec", "/usr/bin/true", "-run", "^$"}
	case "test":
		args = []string{"go", "test", "-timeout", "75s"}
		if os.Getenv("DVA_CI_FRESH") == "1" {
			args = append(args, "-count=1")
		}
	}
	args = append(args, packages...)
	output, err := run(ctx, args...)
	_, _ = out.Write(output)
	if err != nil {
		return fmt.Errorf("%s: %w", phase, err)
	}
	return nil
}

func staged(ctx context.Context, run runner) ([]change, error) {
	flags, err := run(ctx, "git", "ls-files", "-v", "-z")
	if err != nil {
		return nil, fmt.Errorf("inspect index flags: %w", err)
	}
	if err := checkIndexFlags(flags); err != nil {
		return nil, err
	}
	if _, err := run(ctx, "git", "diff", "--quiet", "--"); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return nil, errors.New("unstaged tracked changes: worktree must equal index")
		}
		return nil, fmt.Errorf("inspect unstaged changes: %w", err)
	}
	unmerged, err := run(ctx, "git", "ls-files", "--unmerged", "-z")
	if err != nil {
		return nil, fmt.Errorf("inspect unmerged entries: %w", err)
	}
	if len(unmerged) != 0 {
		return nil, errors.New("unmerged entries: resolve before verification")
	}
	untracked, err := run(ctx, "git", "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return nil, fmt.Errorf("inspect untracked files: %w", err)
	}
	if len(untracked) != 0 {
		return nil, errors.New("nonignored untracked files: worktree must equal index")
	}
	raw, err := run(ctx, "git", "diff", "--cached", "--name-status", "-z", "--")
	if err != nil {
		return nil, fmt.Errorf("inspect staged changes: %w", err)
	}
	return parseChanges(raw)
}

func checkIndexFlags(raw []byte) error {
	if len(raw) == 0 {
		return nil
	}
	if raw[len(raw)-1] != 0 {
		return errors.New("unterminated index flag listing")
	}
	for record := range bytes.SplitSeq(raw[:len(raw)-1], []byte{0}) {
		if len(record) < 3 || record[1] != ' ' {
			return errors.New("malformed index flag listing")
		}
		tag := record[0]
		if tag == 'S' || (tag >= 'a' && tag <= 'z') {
			return fmt.Errorf("index has assume-unchanged or skip-worktree flag: %q", record[2:])
		}
		if !bytes.ContainsRune([]byte("HMRCK?"), rune(tag)) {
			return fmt.Errorf("unknown index flag tag %q", tag)
		}
	}
	return nil
}

func parseChanges(raw []byte) ([]change, error) {
	if len(raw) == 0 {
		return nil, errors.New("no staged changes")
	}
	if raw[len(raw)-1] != 0 {
		return nil, errors.New("unterminated staged name-status")
	}
	parts := bytes.Split(raw[:len(raw)-1], []byte{0})
	var result []change
	for len(parts) > 0 {
		status := string(parts[0])
		parts = parts[1:]
		if !validStatus(status) {
			return nil, fmt.Errorf("unsupported staged status %q", status)
		}
		n := 1
		if status[0] == 'R' || status[0] == 'C' {
			n = 2
		}
		if len(parts) < n {
			return nil, errors.New("incomplete staged name-status")
		}
		c := change{status: status}
		for _, part := range parts[:n] {
			if len(part) == 0 {
				return nil, errors.New("empty staged path")
			}
			c.paths = append(c.paths, string(part))
		}
		result = append(result, c)
		parts = parts[n:]
	}
	return result, nil
}

func validStatus(status string) bool {
	if len(status) == 1 {
		return strings.ContainsRune("AMDT", rune(status[0]))
	}
	if len(status) < 2 || (status[0] != 'R' && status[0] != 'C') {
		return false
	}
	for _, c := range status[1:] {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func selectChanges(changes []change) selection {
	s := selection{productionDirs: make(map[string]bool), testDirs: make(map[string]bool)}
	for _, c := range changes {
		if strings.ContainsAny(c.status[:1], "DRCT") {
			s.all = true
		}
		for i, path := range c.paths {
			shared := path == "go.mod" || path == "go.sum" || path == "dva.yml" || path == "Makefile" || strings.HasPrefix(path, ".github/") || strings.HasPrefix(path, "scripts/") || strings.HasPrefix(path, "tools/") || strings.HasPrefix(path, "skills/") || path == "internal/config/schema.json" || path == "internal/cli/dva_guide_template.txt"
			if shared {
				s.all, s.docs = true, true
			}
			if strings.HasSuffix(path, ".md") {
				s.docs = true
				continue
			}
			if !strings.HasSuffix(path, ".go") {
				s.all, s.docs = true, true
				continue
			}
			// Only paths still present in the index are formatted.
			if c.status[0] != 'D' && ((c.status[0] != 'R' && c.status[0] != 'C') || i == len(c.paths)-1) {
				s.goFiles = append(s.goFiles, "./"+path)
			}
			dir := filepath.ToSlash(filepath.Dir(path))
			if strings.HasSuffix(path, "_test.go") {
				s.testDirs[dir] = true
			} else {
				s.productionDirs[dir] = true
			}
		}
	}
	return s
}

type goPackage struct {
	ImportPath      string
	Dir             string
	Imports         []string
	TestImports     []string
	XTestImports    []string
	EmbedFiles      []string
	TestEmbedFiles  []string
	XTestEmbedFiles []string
}

func affectedPackages(ctx context.Context, run runner, productionDirs, testDirs map[string]bool) ([]string, error) {
	root, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	raw, err := run(ctx, "go", "list", "-json", "./...")
	if err != nil {
		return nil, fmt.Errorf("list packages: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	var packages []goPackage
	for {
		var p goPackage
		if err := dec.Decode(&p); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("parse Go package list: %w", err)
		}
		packages = append(packages, p)
	}
	selected := make(map[string]bool)
	testPackages := make(map[string]bool)
	mapped := make(map[string]bool)
	for _, p := range packages {
		rel, err := filepath.Rel(root, p.Dir)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || p.ImportPath == "" {
			return nil, fmt.Errorf("invalid package directory %q", p.Dir)
		}
		embedded := append(append(append([]string{}, p.EmbedFiles...), p.TestEmbedFiles...), p.XTestEmbedFiles...)
		for _, path := range embedded {
			if !strings.HasSuffix(path, "_test.go") {
				continue
			}
			clean := filepath.Clean(path)
			if filepath.IsAbs(path) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("invalid embedded path %q", path)
			}
			embedDir, err := filepath.Rel(root, filepath.Dir(filepath.Join(p.Dir, clean)))
			if err != nil || embedDir == ".." || strings.HasPrefix(embedDir, ".."+string(filepath.Separator)) {
				return nil, fmt.Errorf("invalid embedded path %q", path)
			}
			if testDirs[filepath.ToSlash(embedDir)] {
				return nil, errors.New("embedded Go test input requires core/full; run make ci-core or make ci-full")
			}
		}
		dir := filepath.ToSlash(rel)
		if productionDirs[dir] {
			selected[p.ImportPath] = true
			mapped[dir] = true
		}
		if testDirs[dir] {
			testPackages[p.ImportPath] = true
			mapped[dir] = true
		}
	}
	for dir := range productionDirs {
		if !mapped[dir] {
			return nil, errors.New("unmapped Go change requires core/full; run make ci-core or make ci-full")
		}
	}
	for dir := range testDirs {
		if !mapped[dir] {
			return nil, errors.New("unmapped Go change requires core/full; run make ci-core or make ci-full")
		}
	}
	if len(selected) == 0 && len(testPackages) == 0 {
		return nil, errors.New("unmapped Go change requires core/full; run make ci-core or make ci-full")
	}
	for changed := true; changed; {
		changed = false
		for _, p := range packages {
			if selected[p.ImportPath] {
				continue
			}
			deps := append(append(append([]string{}, p.Imports...), p.TestImports...), p.XTestImports...)
			for _, dep := range deps {
				if selected[dep] {
					selected[p.ImportPath], changed = true, true
					break
				}
			}
		}
	}
	for name := range testPackages {
		selected[name] = true
	}
	result := make([]string, 0, len(selected))
	for name := range selected {
		result = append(result, name)
	}
	slices.Sort(result)
	return result, nil
}
