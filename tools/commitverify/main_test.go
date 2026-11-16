package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func gitFixture(t *testing.T) (string, runner) {
	t.Helper()
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}
	git("init", "-q")
	git("config", "user.name", "test")
	git("config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "file.go"), []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	git("add", "file.go")
	git("commit", "-qm", "initial")
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		return cmd.Output()
	}
	return dir, run
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %s: %v", args, out, err)
	}
}

func TestStagedGuardRealGit(t *testing.T) {
	dir, run := gitFixture(t)
	ctx := context.Background()
	if _, err := staged(ctx, run); err == nil || !strings.Contains(err.Error(), "no staged") {
		t.Fatalf("empty staged selection: %v", err)
	}
	name := "with space\nand newline.go"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("package sample\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := staged(ctx, run); err == nil || !strings.Contains(err.Error(), "untracked") {
		t.Fatalf("untracked: %v", err)
	}
	gitIn(t, dir, "add", "--", name)
	changes, err := staged(ctx, run)
	if err != nil || len(changes) != 1 || changes[0].paths[0] != name {
		t.Fatalf("staged path: %#v %v", changes, err)
	}
	if err := os.WriteFile(path, []byte("package changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := staged(ctx, run); err == nil || !strings.Contains(err.Error(), "unstaged") {
		t.Fatalf("unstaged: %v", err)
	}
}

func TestParseAndSelectConservatively(t *testing.T) {
	cases := []struct {
		data string
		all  bool
		docs bool
	}{
		{"M\x00internal/x/a.go\x00", false, false},
		{"M\x00docs/a.md\x00", false, true},
		{"M\x00skills/dva/SKILL.md\x00", true, true},
		{"D\x00docs/a.md\x00", true, true},
		{"R100\x00a.go\x00b.go\x00", true, false},
		{"T\x00a.go\x00", true, false},
		{"M\x00go.mod\x00", true, true},
	}
	for _, tc := range cases {
		changes, err := parseChanges([]byte(tc.data))
		if err != nil {
			t.Fatalf("%q: %v", tc.data, err)
		}
		sel := selectChanges(changes)
		if sel.all != tc.all || sel.docs != tc.docs {
			t.Errorf("%q: %#v", tc.data, sel)
		}
	}
	for _, raw := range []string{"M\x00a", "R100\x00a\x00", "U\x00a\x00", "M\x00\x00"} {
		if _, err := parseChanges([]byte(raw)); err == nil {
			t.Errorf("accepted malformed input %q", raw)
		}
	}
}

func TestVerificationArgumentsAndFresh(t *testing.T) {
	t.Setenv("DVA_CI_FRESH", "1")
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	listed, err := json.Marshal(goPackage{ImportPath: "example/unit", Dir: filepath.Join(root, "internal", "unit")})
	if err != nil {
		t.Fatal(err)
	}
	var calls [][]string
	run := func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{}, args...))
		if reflect.DeepEqual(args, []string{"git", "diff", "--cached", "--name-status", "-z", "--"}) {
			return []byte("M\x00internal/unit/x with space.go\x00"), nil
		}
		if reflect.DeepEqual(args, []string{"go", "list", "-json", "./..."}) {
			return listed, nil
		}
		return nil, nil
	}
	for _, phase := range []string{"integrity", "lint", "compile", "test"} {
		if err := verify(context.Background(), phase, run, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range [][]string{
		{"gofmt", "-s", "-l", "./internal/unit/x with space.go"},
		{"go", "vet", "example/unit"},
		{"go", "test", "-exec", "/usr/bin/true", "-run", "^$", "example/unit"},
		{"go", "test", "-timeout", "75s", "-count=1", "example/unit"},
	} {
		found := false
		for _, call := range calls {
			found = found || reflect.DeepEqual(call, want)
		}
		if !found {
			t.Errorf("missing argv %q in %q", want, calls)
		}
	}
}

func TestStagedRejectsHiddenIndexFlags(t *testing.T) {
	for _, flag := range []string{"--assume-unchanged", "--skip-worktree"} {
		t.Run(flag, func(t *testing.T) {
			dir, run := gitFixture(t)
			gitIn(t, dir, "update-index", flag, "file.go")
			if err := os.WriteFile(filepath.Join(dir, "file.go"), []byte("package hidden\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "other.go"), []byte("package sample\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			gitIn(t, dir, "add", "other.go")
			before, err := run(context.Background(), "git", "ls-files", "-v", "-z")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := staged(context.Background(), run); err == nil || !strings.Contains(err.Error(), "assume-unchanged or skip-worktree") {
				t.Fatalf("hidden flag accepted: %v", err)
			}
			after, err := run(context.Background(), "git", "ls-files", "-v", "-z")
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("index flags changed: %q -> %q: %v", before, after, err)
			}
		})
	}
}

func TestIndexFlagsMalformed(t *testing.T) {
	for _, raw := range []string{"H file", "H \x00", "Hfile\x00", "Z file\x00", "\x00"} {
		if err := checkIndexFlags([]byte(raw)); err == nil {
			t.Errorf("accepted malformed index flags %q", raw)
		}
	}
	if err := checkIndexFlags([]byte("H line\nname.go\x00")); err != nil {
		t.Fatalf("valid newline path rejected: %v", err)
	}
}

func TestCompileDoesNotRunInitOrTestMain(t *testing.T) {
	dir := t.TempDir()
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		"go.mod":          "module example.invalid/fixture\n\ngo 1.24\n",
		"fixture.go":      "package fixture\nimport \"os\"\nfunc init() { _ = os.WriteFile(\"init-marker\", []byte(\"bad\"), 0600) }\n",
		"fixture_test.go": "package fixture\nimport (\"os\"; \"testing\")\nfunc TestMain(m *testing.M) { _ = os.WriteFile(\"main-marker\", []byte(\"bad\"), 0600); os.Exit(m.Run()) }\nfunc TestFixture(t *testing.T) {}\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(ctx context.Context, args ...string) ([]byte, error) {
		if args[0] == "git" {
			if reflect.DeepEqual(args, []string{"git", "diff", "--cached", "--name-status", "-z", "--"}) {
				return []byte("M\x00internal/fixture/fixture.go\x00"), nil
			}
			return nil, nil
		}
		if reflect.DeepEqual(args, []string{"go", "list", "-json", "./..."}) {
			return json.Marshal(goPackage{ImportPath: "example.invalid/fixture", Dir: filepath.Join(root, "internal", "fixture")})
		}
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Dir = dir
		return cmd.CombinedOutput()
	}
	var output bytes.Buffer
	if err := verify(context.Background(), "compile", run, &output); err != nil {
		t.Fatalf("compile: %v: %s", err, output.String())
	}
	for _, name := range []string{"init-marker", "main-marker"} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("test binary executed: %s: %v", name, err)
		}
	}
}

func TestExecCommandStreamsBeforeExit(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "make"), []byte("#!/bin/sh\nprintf 'progress\\n'\nwhile :; do :; done\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	progress := make(chan string, 1)
	go func() {
		line, _ := bufio.NewReader(reader).ReadString('\n')
		progress <- line
	}()
	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), streamOutputKey{}, writer))
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := execCommand(ctx, "make", "doc-check")
		done <- err
	}()
	select {
	case got := <-progress:
		if got != "progress\n" {
			t.Fatalf("got output %q", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("progress was not streamed before command exit")
	}
	cancel()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("cancelled process succeeded")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled command did not exit")
	}
}

func TestGitErrorIsNotClean(t *testing.T) {
	run := func(context.Context, ...string) ([]byte, error) { return nil, errors.New("git unavailable") }
	if _, err := staged(context.Background(), run); err == nil || !strings.Contains(err.Error(), "git unavailable") {
		t.Fatalf("got %v", err)
	}
}

func TestAffectedPackagesIncludesReverseTestDependencies(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	packages := []goPackage{
		{ImportPath: "example/a", Dir: filepath.Join(root, "a")},
		{ImportPath: "example/b", Dir: filepath.Join(root, "b"), TestImports: []string{"example/a"}},
		{ImportPath: "example/c", Dir: filepath.Join(root, "c"), Imports: []string{"example/b"}},
		{ImportPath: "example/other", Dir: filepath.Join(root, "other")},
	}
	var raw bytes.Buffer
	for _, pkg := range packages {
		if err := json.NewEncoder(&raw).Encode(pkg); err != nil {
			t.Fatal(err)
		}
	}
	run := func(_ context.Context, args ...string) ([]byte, error) {
		if !reflect.DeepEqual(args, []string{"go", "list", "-json", "./..."}) {
			return nil, errors.New("unexpected command: " + strings.Join(args, " "))
		}
		return raw.Bytes(), nil
	}
	got, err := affectedPackages(context.Background(), run, map[string]bool{"a": true}, nil)
	want := []string{"example/a", "example/b", "example/c"}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q, want %q: %v", got, want, err)
	}
	got, err = affectedPackages(context.Background(), run, map[string]bool{"missing": true}, nil)
	if err == nil || !strings.Contains(err.Error(), "requires core/full") || len(got) != 0 {
		t.Fatalf("missing package must require core/full: %q, %v", got, err)
	}
}

func TestTestOnlyAndMixedChangesSelectPackageGraph(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	packages := []goPackage{
		{ImportPath: "example/a", Dir: filepath.Join(root, "a")},
		{ImportPath: "example/b", Dir: filepath.Join(root, "b"), TestImports: []string{"example/a"}},
		{ImportPath: "example/c", Dir: filepath.Join(root, "c"), Imports: []string{"example/b"}},
		{ImportPath: "example/d", Dir: filepath.Join(root, "d"), XTestImports: []string{"example/a"}},
		{ImportPath: "example/e", Dir: filepath.Join(root, "e")},
	}
	var raw bytes.Buffer
	for _, pkg := range packages {
		if err := json.NewEncoder(&raw).Encode(pkg); err != nil {
			t.Fatal(err)
		}
	}
	run := func(_ context.Context, args ...string) ([]byte, error) {
		if !reflect.DeepEqual(args, []string{"go", "list", "-json", "./..."}) {
			return nil, errors.New("unexpected command: " + strings.Join(args, " "))
		}
		return raw.Bytes(), nil
	}
	for _, tc := range []struct {
		name    string
		paths   []string
		want    []string
		wantErr bool
	}{
		{"internal test", []string{"a/a_test.go"}, []string{"example/a"}, false},
		{"external test", []string{"a/external_test.go"}, []string{"example/a"}, false},
		{"production", []string{"a/a.go"}, []string{"example/a", "example/b", "example/c", "example/d"}, false},
		{"mixed same directory", []string{"a/a.go", "a/a_test.go"}, []string{"example/a", "example/b", "example/c", "example/d"}, false},
		{"mixed directories", []string{"a/a_test.go", "b/b.go"}, []string{"example/a", "example/b", "example/c"}, false},
		{"two tests", []string{"a/a_test.go", "b/b_test.go"}, []string{"example/a", "example/b"}, false},
		{"production plus unrelated test", []string{"a/a.go", "e/e_test.go"}, []string{"example/a", "example/b", "example/c", "example/d", "example/e"}, false},
		{"missing production", []string{"missing/a.go"}, nil, true},
		{"missing test", []string{"missing/a_test.go"}, nil, true},
		{"valid test plus missing production", []string{"a/a_test.go", "missing/a.go"}, nil, true},
		{"valid production plus missing test", []string{"a/a.go", "missing/a_test.go"}, nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var changes []change
			for _, path := range tc.paths {
				changes = append(changes, change{status: "M", paths: []string{path}})
			}
			sel := selectChanges(changes)
			got, err := affectedPackages(context.Background(), run, sel.productionDirs, sel.testDirs)
			if (err != nil) != tc.wantErr || !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %q, want %q: %v", got, tc.want, err)
			}
		})
	}
}

func TestTestOnlySelectionUsedInAllGoPhases(t *testing.T) {
	t.Setenv("DVA_CI_FRESH", "0")
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var listing bytes.Buffer
	for _, pkg := range []goPackage{
		{ImportPath: "example/a", Dir: filepath.Join(root, "a")},
		{ImportPath: "example/b", Dir: filepath.Join(root, "b"), XTestImports: []string{"example/a"}},
	} {
		if err := json.NewEncoder(&listing).Encode(pkg); err != nil {
			t.Fatal(err)
		}
	}
	var calls [][]string
	run := func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{}, args...))
		switch {
		case reflect.DeepEqual(args, []string{"git", "diff", "--cached", "--name-status", "-z", "--"}):
			return []byte("M\x00a/external_test.go\x00"), nil
		case reflect.DeepEqual(args, []string{"go", "list", "-json", "./..."}):
			return listing.Bytes(), nil
		default:
			return nil, nil
		}
	}
	for _, phase := range []string{"integrity", "lint", "compile", "test"} {
		if err := verify(context.Background(), phase, run, &bytes.Buffer{}); err != nil {
			t.Fatal(err)
		}
	}
	for _, want := range [][]string{
		{"gofmt", "-s", "-l", "./a/external_test.go"},
		{"go", "vet", "example/a"},
		{"go", "test", "-exec", "/usr/bin/true", "-run", "^$", "example/a"},
		{"go", "test", "-timeout", "75s", "example/a"},
	} {
		found := false
		for _, call := range calls {
			found = found || reflect.DeepEqual(call, want)
		}
		if !found {
			t.Errorf("missing %q in %q", want, calls)
		}
	}
}

func TestEmbeddedTestFilesRequireCore(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		embed   goPackage
		dir     string
		wantErr bool
	}{
		{"production embed same package", goPackage{EmbedFiles: []string{"fixture_test.go"}}, "a", true},
		{"test embed same package", goPackage{TestEmbedFiles: []string{"fixture_test.go"}}, "a", true},
		{"external test embed same package", goPackage{XTestEmbedFiles: []string{"fixture_test.go"}}, "a", true},
		{"parent embeds child file", goPackage{EmbedFiles: []string{"child/fixture_test.go"}}, "a/child", true},
		{"parent test embeds child file", goPackage{TestEmbedFiles: []string{"child/fixture_test.go"}}, "a/child", true},
		{"parent external test embeds child file", goPackage{XTestEmbedFiles: []string{"child/fixture_test.go"}}, "a/child", true},
		{"ordinary data unaffected", goPackage{EmbedFiles: []string{"fixture.json"}}, "a", false},
		{"unrelated test directory unaffected", goPackage{EmbedFiles: []string{"fixture_test.go"}}, "a/child", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.embed.ImportPath = "example/a"
			tc.embed.Dir = filepath.Join(root, "a")
			var listing bytes.Buffer
			for _, pkg := range []goPackage{tc.embed, {ImportPath: "example/a/child", Dir: filepath.Join(root, "a", "child")}} {
				if err := json.NewEncoder(&listing).Encode(pkg); err != nil {
					t.Fatal(err)
				}
			}
			run := func(_ context.Context, args ...string) ([]byte, error) {
				if !reflect.DeepEqual(args, []string{"go", "list", "-json", "./..."}) {
					return nil, errors.New("unexpected command: " + strings.Join(args, " "))
				}
				return listing.Bytes(), nil
			}
			got, err := affectedPackages(context.Background(), run, nil, map[string]bool{tc.dir: true})
			if tc.wantErr {
				if err == nil || !strings.Contains(err.Error(), "embedded Go test input requires core/full") {
					t.Fatalf("got %q, %v; want core/full", got, err)
				}
			} else if err != nil || len(got) != 1 {
				t.Fatalf("got %q, %v; want one test package", got, err)
			}
		})
	}
}

func TestBroadChangesFailBeforeExpensiveChecks(t *testing.T) {
	for _, input := range []string{"M\x00go.mod\x00", "D\x00internal/a/file.go\x00", "D\x00internal/a/file_test.go\x00", "R100\x00a.go\x00b.go\x00", "R100\x00a_test.go\x00b_test.go\x00", "M\x00tools/shared/check_test.go\x00", "M\x00skills/dva/SKILL.md\x00"} {
		for _, phase := range []string{"integrity", "lint", "compile", "test"} {
			run := func(_ context.Context, args ...string) ([]byte, error) {
				if args[0] != "git" {
					t.Fatalf("broad %q phase %s ran expensive command %q", input, phase, args)
				}
				if reflect.DeepEqual(args, []string{"git", "diff", "--cached", "--name-status", "-z", "--"}) {
					return []byte(input), nil
				}
				return nil, nil
			}
			err := verify(context.Background(), phase, run, &bytes.Buffer{})
			if err == nil || !strings.Contains(err.Error(), "requires core/full; run make ci-core or make ci-full") {
				t.Fatalf("broad %q phase %s: %v", input, phase, err)
			}
		}
	}
}

func TestDocumentationOnlySkipsGo(t *testing.T) {
	var output bytes.Buffer
	run := func(_ context.Context, args ...string) ([]byte, error) {
		if args[0] == "go" {
			t.Fatal("Go command executed for documentation only")
		}
		if len(args) > 2 && args[1] == "diff" && args[2] == "--cached" {
			return []byte("M\x00docs/a.md\x00"), nil
		}
		return nil, nil
	}
	if err := verify(context.Background(), "test", run, &output); err != nil || !strings.Contains(output.String(), "not applicable") {
		t.Fatalf("output=%q error=%v", output.String(), err)
	}
}
