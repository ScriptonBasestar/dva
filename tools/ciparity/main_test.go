package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocCheckToolsFromMakefile(t *testing.T) {
	dir := t.TempDir()
	mfPath := filepath.Join(dir, "Makefile")
	writeFile(t, mfPath, `
build:
	go build ./...

doc-check:
	go run ./tools/doccheck
	go run ./tools/cilabels
	go run ./tools/yamlcheck

commit-check:
	go run ./tools/commitcheck
`)
	got, err := docCheckToolsFromMakefile(mfPath)
	if err != nil {
		t.Fatal(err)
	}
	assertToolSet(t, got, "doccheck", "cilabels", "yamlcheck")
}

func TestDocCheckToolsFromMakefileExcludesSelf(t *testing.T) {
	dir := t.TempDir()
	mfPath := filepath.Join(dir, "Makefile")
	writeFile(t, mfPath, `
doc-check:
	go run ./tools/doccheck
	go run ./tools/ciparity
`)
	got, err := docCheckToolsFromMakefile(mfPath)
	if err != nil {
		t.Fatal(err)
	}
	// ciparity is the comparator, not a compared gate — it must never appear here,
	// or this tool would permanently flag itself as an "extra" Makefile tool.
	assertToolSet(t, got, "doccheck")
}

func TestDocCheckToolsFromMakefileStopsAtNextTarget(t *testing.T) {
	dir := t.TempDir()
	mfPath := filepath.Join(dir, "Makefile")
	writeFile(t, mfPath, `
doc-check:
	go run ./tools/doccheck

commit-check:
	go run ./tools/commitcheck
`)
	got, err := docCheckToolsFromMakefile(mfPath)
	if err != nil {
		t.Fatal(err)
	}
	assertToolSet(t, got, "doccheck")
}

func TestDocCheckToolsFromMakefileMissingTargetErrors(t *testing.T) {
	dir := t.TempDir()
	mfPath := filepath.Join(dir, "Makefile")
	writeFile(t, mfPath, `
build:
	go build ./...
`)
	if _, err := docCheckToolsFromMakefile(mfPath); err == nil {
		t.Fatal("expected error for a Makefile with no doc-check recipe")
	}
}

func TestDocsStepToolsFromDvaYAML(t *testing.T) {
	dir := t.TempDir()
	dvaPath := filepath.Join(dir, "dva.yml")
	writeFile(t, dvaPath, `
version: "0.1.44"
ci:
  profiles:
    commit:
      timeout: 10m
      steps:
        - name: format
          run: gofmt -s -l .
        - name: docs
          run: go run ./tools/doccheck && go run ./tools/cilabels && go run ./tools/yamlcheck
          depends_on: [format]
`)
	got, err := docsStepToolsFromDvaYAML(dvaPath, "commit")
	if err != nil {
		t.Fatal(err)
	}
	assertToolSet(t, got, "doccheck", "cilabels", "yamlcheck")
}

func TestDocsStepToolsFromDvaYAMLMissingStepErrors(t *testing.T) {
	dir := t.TempDir()
	dvaPath := filepath.Join(dir, "dva.yml")
	writeFile(t, dvaPath, `
version: "0.1.44"
ci:
  profiles:
    commit:
      timeout: 10m
      steps:
        - name: format
          run: gofmt -s -l .
`)
	if _, err := docsStepToolsFromDvaYAML(dvaPath, "commit"); err == nil {
		t.Fatal("expected error when the commit profile has no docs step")
	}
}

// TestRepoDocCheckAndDocsStepAgree is the actual regression guard: it runs the same
// extraction this program's main() runs, against this checkout's real Makefile and
// dva.yml, so `go test ./tools/ciparity/...` (part of `go test ./...`) fails the moment
// the two lists drift — not only the separate `go run ./tools/ciparity` invocation
// wired into `make doc-check`.
func TestRepoDocCheckAndDocsStepAgree(t *testing.T) {
	root := repoRoot(t)
	mfTools, err := docCheckToolsFromMakefile(filepath.Join(root, "Makefile"))
	if err != nil {
		t.Fatalf("Makefile: %v", err)
	}
	dvaTools, err := docsStepToolsFromDvaYAML(filepath.Join(root, "dva.yml"), "commit")
	if err != nil {
		t.Fatalf("dva.yml: %v", err)
	}
	for tool := range mfTools {
		if !dvaTools[tool] {
			t.Errorf("Makefile doc-check runs %q; dva.yml ci.profiles.commit docs step does not", tool)
		}
	}
	for tool := range dvaTools {
		if !mfTools[tool] {
			t.Errorf("dva.yml ci.profiles.commit docs step runs %q; Makefile doc-check does not", tool)
		}
	}
}

// TestRepoFullProfileDocsStepAgreesWithCommit guards the full profile's reuse of
// commit's docs step: today it is the same YAML anchor (&ci-docs / *ci-docs), but an
// anchor is an authoring convenience, not an enforced invariant — a future edit could
// silently replace it with a literal, independently-drifting step list.
func TestRepoFullProfileDocsStepAgreesWithCommit(t *testing.T) {
	root := repoRoot(t)
	commitTools, err := docsStepToolsFromDvaYAML(filepath.Join(root, "dva.yml"), "commit")
	if err != nil {
		t.Fatalf("dva.yml commit profile: %v", err)
	}
	fullTools, err := docsStepToolsFromDvaYAML(filepath.Join(root, "dva.yml"), "full")
	if err != nil {
		t.Fatalf("dva.yml full profile: %v", err)
	}
	for tool := range commitTools {
		if !fullTools[tool] {
			t.Errorf("dva.yml ci.profiles.commit docs step runs %q; full profile's docs step does not", tool)
		}
	}
	for tool := range fullTools {
		if !commitTools[tool] {
			t.Errorf("dva.yml ci.profiles.full docs step runs %q; commit profile's docs step does not", tool)
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Dir(filepath.Dir(wd)) // tools/ciparity -> tools -> repo root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertToolSet(t *testing.T, got map[string]bool, want ...string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("tools = %v, want %v", sortedKeys(got), want)
	}
	for _, name := range want {
		if !got[name] {
			t.Errorf("missing tool %q in %v", name, sortedKeys(got))
		}
	}
}
