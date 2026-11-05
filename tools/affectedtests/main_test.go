package main

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestAffectedAddsReverseTestDependencies(t *testing.T) {
	packages := []packageInfo{
		{path: "example/internal/a", dir: "internal/a"},
		{path: "example/internal/b", dir: "internal/b", deps: []string{"example/internal/a"}},
		{path: "example/cmd/app", dir: "cmd/app", deps: []string{"example/internal/b"}},
	}
	got, full := affected(packages, []string{"internal/a/a.go"})
	if full || len(got) != 3 || got[0] != "example/cmd/app" || got[2] != "example/internal/b" {
		t.Fatalf("affected = %v, full = %v", got, full)
	}
}

func TestAffectedWidensSharedAndUnknownInputs(t *testing.T) {
	packages := []packageInfo{{path: "example/internal/a", dir: "internal/a"}}
	for _, path := range []string{"go.mod", "tools/check/main.go", "config/unknown.yaml", "docs/guide.md", "internal/missing/new.go"} {
		if _, full := affected(packages, []string{path}); !full {
			t.Errorf("%s did not widen", path)
		}
	}
}

func TestChangedPathsRejectsDeletionAndRename(t *testing.T) {
	for _, raw := range [][]byte{[]byte("D\x00old.go\x00"), []byte("R100\x00old.go\x00new.go\x00")} {
		if _, deleted, err := changedPaths(true, raw); err != nil || !deleted {
			t.Errorf("changedPaths(%q) = deleted %v, err %v", raw, deleted, err)
		}
	}
}

func TestChangedPathsRejectsMalformedStatusStreams(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte("\x00file.go\x00"),
		[]byte("M\x00file.go"),
		[]byte("R100\x00old.go\x00"),
		[]byte("Rxx\x00old.go\x00new.go\x00"),
		[]byte("M\x00\x00"),
	} {
		if _, _, err := changedPaths(true, raw); err == nil {
			t.Errorf("changedPaths(%q) unexpectedly succeeded", raw)
		}
	}
}

func TestPackagesAcceptsEmptyImportColumn(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	s := selector{run: func(args ...string) ([]byte, error) {
		return []byte("example/internal/empty\t" + filepath.Join(root, "internal", "empty") + "\t  \n"), nil
	}}
	packages, err := s.packages()
	if err != nil || len(packages) != 1 || len(packages[0].deps) != 0 {
		t.Fatalf("packages = %#v, err = %v", packages, err)
	}
}

func TestSelectPackagesRefusesNoChanges(t *testing.T) {
	s := selector{run: func(args ...string) ([]byte, error) {
		switch strings.Join(args[:2], " ") {
		case "git rev-parse":
			return []byte("base\n"), nil
		case "git merge-base":
			return []byte("merge-base\n"), nil
		case "git diff", "git ls-files":
			return nil, nil
		default:
			t.Fatalf("unexpected command %q", args)
			return nil, nil
		}
	}}
	_, err := s.selectPackages("master")
	if err == nil || !strings.Contains(err.Error(), "no tracked or untracked changes") {
		t.Fatalf("selectPackages error = %v", err)
	}
}

func TestSelectPackagesUnionsEveryGitInputAndReverseDependencies(t *testing.T) {
	root, err := filepath.Abs(".")
	if err != nil {
		t.Fatal(err)
	}
	s := selector{run: func(args ...string) ([]byte, error) {
		switch {
		case strings.Join(args[:2], " ") == "git rev-parse":
			return []byte("base\n"), nil
		case strings.Join(args[:2], " ") == "git merge-base":
			return []byte("merge-base\n"), nil
		case strings.Join(args[:3], " ") == "git diff --name-status" && strings.Contains(strings.Join(args, " "), "merge-base HEAD"):
			return []byte("M\x00internal/a/a.go\x00"), nil
		case strings.Join(args[:3], " ") == "git diff --name-status" && strings.Contains(strings.Join(args, " "), "--cached"):
			return []byte("M\x00internal/b/b.go\x00"), nil
		case strings.Join(args[:3], " ") == "git diff --name-status":
			return []byte("M\x00cmd/app/main.go\x00"), nil
		case strings.Join(args[:2], " ") == "git ls-files":
			return []byte("internal/new/new.go\x00"), nil
		case strings.Join(args[:2], " ") == "go list":
			return []byte(strings.Join([]string{
				"example/internal/a\t" + filepath.Join(root, "internal", "a") + "\t",
				"example/internal/b\t" + filepath.Join(root, "internal", "b") + "\texample/internal/a",
				"example/cmd/app\t" + filepath.Join(root, "cmd", "app") + "\texample/internal/b",
				"example/internal/new\t" + filepath.Join(root, "internal", "new") + "\t",
			}, "\n") + "\n"), nil
		default:
			t.Fatalf("unexpected command %q", args)
			return nil, nil
		}
	}}
	got, err := s.selectPackages("master")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"example/cmd/app", "example/internal/a", "example/internal/b", "example/internal/new"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("packages = %v, want %v", got, want)
	}
}

func TestSelectPackagesReturnsGitFailure(t *testing.T) {
	s := selector{run: func(args ...string) ([]byte, error) {
		if strings.Join(args[:2], " ") == "git rev-parse" {
			return []byte("base\n"), nil
		}
		if strings.Join(args[:2], " ") == "git merge-base" {
			return []byte("merge-base\n"), nil
		}
		return nil, errors.New("fixture failure")
	}}
	_, err := s.selectPackages("master")
	if err == nil || !strings.Contains(err.Error(), "git diff") || !strings.Contains(err.Error(), "fixture failure") {
		t.Fatalf("selectPackages error = %v", err)
	}
}
