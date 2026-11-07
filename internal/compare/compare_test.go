package compare

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// testRepo creates a git repository with one commit on main and returns its directory. The
// gitignore line decides whether snapshot paths are ignored; empty writes no .gitignore.
func testRepo(t *testing.T, gitignore string) string {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", "-b", "main")
	if gitignore != "" {
		writeFile(t, filepath.Join(dir, ".gitignore"), gitignore+"\n")
	}
	writeFile(t, filepath.Join(dir, "dva.yml"), "version: \"0.1.0\"\n")
	commitAll(t, dir, "initial")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com"}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func commitAll(t *testing.T, dir, msg string) string {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-q", "-m", msg)
	return runGit(t, dir, "rev-parse", "HEAD")
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustContain(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want substring %q", err, want)
	}
}

func TestRecordRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if _, err := ReadRecord(dir, "web"); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("missing record: err = %v, want fs.ErrNotExist", err)
	}
	rec := &Record{Name: "web", Variants: []Variant{{
		Name: "a", Mode: ModeSnapshot, Source: "main", Commit: "abc", Path: "/p", ConfigDir: "/p",
		Ports: map[string]int{"WEB_PORT": 3001},
	}}}
	if err := WriteRecord(dir, rec); err != nil {
		t.Fatalf("write: %v", err)
	}
	got, err := ReadRecord(dir, "web")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Version != recordVersion || got.Variant("a") == nil || got.Variant("a").Ports["WEB_PORT"] != 3001 {
		t.Fatalf("round trip = %+v", got)
	}
	recDir := filepath.Join(dir, config.DotDirName, config.CompareDirName)
	entries, err := os.ReadDir(recDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "web.json" {
		t.Fatalf("record dir holds %v, want only web.json (no leftover temp file)", entries)
	}

	writeFile(t, filepath.Join(recDir, "web.json"), `{"version": 9, "name": "web"}`)
	_, err = ReadRecord(dir, "web")
	mustContain(t, err, "has version 9")
	writeFile(t, filepath.Join(recDir, "web.json"), `{"version": 1, "name": "other"}`)
	_, err = ReadRecord(dir, "web")
	mustContain(t, err, `names comparison "other"`)

	if err := WriteRecord(dir, &Record{Name: "../x"}); err == nil {
		t.Fatal("record name with a separator accepted")
	}
	err = WriteRecord(dir, &Record{Name: "web", Variants: []Variant{{Name: "a/b"}}})
	mustContain(t, err, `variant name "a/b"`)
}

func TestSnapshotCreateReuseAndRefuse(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t, config.DotDirName+"/*")
	first := runGit(t, repo, "rev-parse", "HEAD")
	branchesBefore := runGit(t, repo, "branch", "--list")

	v, reused, err := EnsureSnapshot(ctx, repo, "web", "a", "main", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if reused || v.Commit != first || v.Mode != ModeSnapshot || v.Source != "main" {
		t.Fatalf("created = %+v reused=%v", v, reused)
	}
	want := filepath.Join(repo, config.DotDirName, config.VariantsDirName, "web", "a")
	if v.Path != want || v.ConfigDir != want {
		t.Fatalf("path = %s config_dir = %s, want %s", v.Path, v.ConfigDir, want)
	}
	if head := runGit(t, v.Path, "rev-parse", "HEAD"); head != first {
		t.Fatalf("snapshot HEAD = %s, want %s", head, first)
	}
	if sym, err := exec.Command("git", "-C", v.Path, "symbolic-ref", "-q", "HEAD").Output(); err == nil {
		t.Fatalf("snapshot is on branch %s, want detached HEAD", sym)
	}
	if after := runGit(t, repo, "branch", "--list"); after != branchesBefore {
		t.Fatalf("branches changed: %q -> %q", branchesBefore, after)
	}

	again, reused, err := EnsureSnapshot(ctx, repo, "web", "a", "main", &v)
	if err != nil || !reused || again.Path != v.Path {
		t.Fatalf("reuse: v=%+v reused=%v err=%v", again, reused, err)
	}

	_, _, err = EnsureSnapshot(ctx, repo, "web", "a", "main", nil)
	mustContain(t, err, "does not attribute to source")
	other := v
	other.Source = "feat"
	_, _, err = EnsureSnapshot(ctx, repo, "web", "a", "main", &other)
	mustContain(t, err, "does not attribute to source")

	writeFile(t, filepath.Join(repo, "next.txt"), "x")
	second := commitAll(t, repo, "second")
	_, _, err = EnsureSnapshot(ctx, repo, "web", "a", "main", &v)
	mustContain(t, err, "now resolves to "+second)

	_, _, err = EnsureSnapshot(ctx, repo, "web", "b", "no-such-ref", nil)
	mustContain(t, err, `ref "no-such-ref" does not name a commit`)

	plain := filepath.Join(repo, config.DotDirName, config.VariantsDirName, "web", "c")
	writeFile(t, filepath.Join(plain, "keep.txt"), "mine")
	_, _, err = EnsureSnapshot(ctx, repo, "web", "c", "main", nil)
	mustContain(t, err, "is not a git worktree of this repository")
	if _, err := os.Stat(filepath.Join(plain, "keep.txt")); err != nil {
		t.Fatalf("refused path was modified: %v", err)
	}
}

func TestSnapshotRefusesUnignoredPath(t *testing.T) {
	repo := testRepo(t, "")
	_, _, err := EnsureSnapshot(context.Background(), repo, "web", "a", "main", nil)
	mustContain(t, err, "is not ignored by git")
	if _, err := os.Stat(filepath.Join(repo, config.DotDirName, config.VariantsDirName)); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("variants directory created for a refused snapshot: %v", err)
	}
}

// TestSnapshotConfigDirBelowRepoRoot: a dva.yml below the repository root runs from the same
// relative directory inside the snapshot, which is where its pid files are written.
func TestSnapshotConfigDirBelowRepoRoot(t *testing.T) {
	repo := testRepo(t, config.DotDirName+"/*")
	sub := filepath.Join(repo, "app")
	writeFile(t, filepath.Join(sub, "dva.yml"), "version: \"0.1.0\"\n")
	commitAll(t, repo, "app")
	// The root rule contains a slash, so git anchors it at the root and it does not reach
	// app/.sb/dva. The guard has to see that rather than trust the root file.
	_, _, err := EnsureSnapshot(context.Background(), sub, "web", "a", "main", nil)
	mustContain(t, err, "is not ignored by git")
	writeFile(t, filepath.Join(sub, ".gitignore"), config.DotDirName+"/*\n")
	commitAll(t, repo, "app ignore")
	v, _, err := EnsureSnapshot(context.Background(), sub, "web", "a", "main", nil)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if v.ConfigDir != filepath.Join(v.Path, "app") {
		t.Fatalf("config_dir = %s, want %s", v.ConfigDir, filepath.Join(v.Path, "app"))
	}
	if _, err := os.Stat(filepath.Join(v.ConfigDir, "dva.yml")); err != nil {
		t.Fatalf("snapshot config dir has no dva.yml: %v", err)
	}
}

func TestRemoveSnapshot(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t, config.DotDirName+"/*")
	a, _, err := EnsureSnapshot(ctx, repo, "web", "a", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := EnsureSnapshot(ctx, repo, "web", "b", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	rec := &Record{Name: "web", Variants: []Variant{a, b}}

	mustContain(t, RemoveSnapshot(ctx, repo, rec, "zzz"), `variant "zzz" is not in compare record`)

	writeFile(t, filepath.Join(b.Path, "scratch.txt"), "unsaved")
	mustContain(t, RemoveSnapshot(ctx, repo, rec, "b"), "has uncommitted or untracked changes")
	if _, err := os.Stat(filepath.Join(b.Path, "scratch.txt")); err != nil {
		t.Fatalf("dirty snapshot touched: %v", err)
	}

	moved := &Record{Name: "web", Variants: []Variant{{Name: "a", Mode: ModeSnapshot, Path: repo}}}
	mustContain(t, RemoveSnapshot(ctx, repo, moved, "a"), "not at dva's snapshot path")
	worktreeMode := &Record{Name: "web", Variants: []Variant{{Name: "a", Mode: "worktree", Path: a.Path}}}
	mustContain(t, RemoveSnapshot(ctx, repo, worktreeMode, "a"), "removes only snapshots it created")

	if err := RemoveSnapshot(ctx, repo, rec, "a"); err != nil {
		t.Fatalf("remove clean snapshot: %v", err)
	}
	if _, err := os.Stat(a.Path); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("snapshot path still exists: %v", err)
	}
	if strings.Contains(runGit(t, repo, "worktree", "list", "--porcelain"), "/web/a\n") {
		t.Fatal("removed snapshot still registered")
	}
	mustContain(t, RemoveSnapshot(ctx, repo, rec, "a"), "is not a git worktree of this repository")
}

func TestCheckReportsStaleWithoutRepair(t *testing.T) {
	ctx := context.Background()
	repo := testRepo(t, config.DotDirName+"/*")
	a, _, err := EnsureSnapshot(ctx, repo, "web", "a", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := EnsureSnapshot(ctx, repo, "web", "b", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	c, _, err := EnsureSnapshot(ctx, repo, "web", "c", "main", nil)
	if err != nil {
		t.Fatal(err)
	}
	gone, _, err := EnsureSnapshot(ctx, repo, "web", "d", "main", nil)
	if err != nil {
		t.Fatal(err)
	}

	// a: a live process, so not stale. b: the record names a commit the checkout is not at.
	// c: a pid that has exited. d: the checkout was deleted behind dva's back.
	pidDir := func(v Variant) string { return filepath.Join(v.ConfigDir, config.DotDirName, config.PidsDirName) }
	writeFile(t, filepath.Join(pidDir(a), "web.pid"), strconv.Itoa(os.Getpid()))
	b.Commit = strings.Repeat("0", 40)
	done := exec.Command("true")
	if err := done.Run(); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(pidDir(c), "web.pid"), strconv.Itoa(done.Process.Pid))
	if err := os.RemoveAll(gone.Path); err != nil {
		t.Fatal(err)
	}

	rec := &Record{Name: "web", Variants: []Variant{a, b, c, gone}}
	if err := WriteRecord(repo, rec); err != nil {
		t.Fatal(err)
	}
	recPath, _ := RecordPath(repo, "web")
	before, err := os.ReadFile(recPath)
	if err != nil {
		t.Fatal(err)
	}

	states, err := Check(ctx, repo, rec, []string{"web", "api"})
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	byName := map[string]VariantState{}
	for _, s := range states {
		byName[s.Name] = s
	}
	if s := byName["a"]; s.Stale {
		t.Errorf("a stale: %v", s.Reasons)
	}
	wantReason := map[string]string{
		"b": "record says " + b.Commit,
		"c": `entry "web" pid ` + strconv.Itoa(done.Process.Pid) + " is not running",
		"d": "does not exist",
	}
	for name, want := range wantReason {
		s := byName[name]
		if !s.Stale || !strings.Contains(strings.Join(s.Reasons, "; "), want) {
			t.Errorf("%s: stale=%v reasons=%v, want %q", name, s.Stale, s.Reasons, want)
		}
	}

	after, err := os.ReadFile(recPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Error("Check rewrote the record")
	}
	if _, err := os.Stat(c.Path); err != nil {
		t.Errorf("Check removed stale snapshot c: %v", err)
	}
	if _, err := os.Stat(filepath.Join(pidDir(c), "web.pid")); err != nil {
		t.Errorf("Check removed stale pid file: %v", err)
	}
}
