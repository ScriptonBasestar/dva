package cimetrics

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMetricsMetadataHashesConfigsWithoutContentsOrPaths(t *testing.T) {
	if !groupSupported() {
		t.Skip("private Unix metrics storage unavailable")
	}
	root := t.TempDir()
	state := t.TempDir()
	secret := "private-token-for-fixture"
	for name, body := range map[string]string{".mise.toml": "token = '" + secret + "'\n", "go.mod": "module " + module + "\n", "dva.yml": "fixture: " + secret + "\n"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("GOROOT", "")
	t.Setenv("GOWORK", "")
	t.Setenv("DVA_CI_FRESH", "")
	first := collectMetadata(root, state)
	if first.ConfigKeyID == "" || first.ConfigHMAC[".mise.toml"] == "" {
		t.Fatalf("fingerprint unavailable: %+v", first)
	}
	if err := os.WriteFile(filepath.Join(root, ".mise.toml"), []byte("changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	second := collectMetadata(root, state)
	if first.ConfigKeyID != second.ConfigKeyID || first.ConfigHMAC[".mise.toml"] == second.ConfigHMAC[".mise.toml"] {
		t.Fatal("fingerprints did not use a stable private key and reflect content")
	}
	b, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secret) || strings.Contains(string(b), root) || strings.Contains(string(b), state) {
		t.Fatalf("metadata leaked input: %s", b)
	}
	info, err := os.Stat(filepath.Join(state, ".fingerprint-key"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("private key permissions: %v %v", info, err)
	}
}

func TestMetricsOptionsAllowOnlyKnownFlagsAndPackages(t *testing.T) {
	allowed := map[string]map[string]bool{module + "/internal/cli": {}}
	t.Setenv("GOFLAGS", "-p=2")
	t.Setenv("DVA_CI_JOBS", "2")
	valid := safeOptions([]string{"go", "test", "-json", "-race", "-cover", "-timeout", "75s", "-count=1", module + "/internal/cli"}, allowed)
	if valid.Unclassified || len(valid.Packages) != 1 || valid.Packages[0] != module+"/internal/cli" || valid.Workers != 2 {
		t.Fatalf("safe flags/packages lost: %+v", valid)
	}
	if !strings.Contains(strings.Join(valid.Flags, " "), "GOFLAGS:-p=2") {
		t.Fatalf("worker flag missing: %+v", valid)
	}
	secret := "private-token-for-fixture"
	t.Setenv("GOFLAGS", "-p=2 -modfile=/"+secret)
	unsafe := safeOptions([]string{"go", "test", "-run", secret, "-tags=" + secret, "./../" + secret, module + "/" + secret}, allowed)
	if !unsafe.Unclassified || !unsafe.Filtered || len(unsafe.Packages) != 0 {
		t.Fatalf("unknown inputs misclassified: %+v", unsafe)
	}
	b, err := json.Marshal(unsafe)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), secret) {
		t.Fatalf("secret appeared in options: %s", b)
	}
}

func TestMetricsFingerprintRejectsSymlinkKey(t *testing.T) {
	if !groupSupported() {
		t.Skip("private Unix metrics storage unavailable")
	}
	dir := t.TempDir()
	target := filepath.Join(t.TempDir(), "target")
	if err := os.WriteFile(target, []byte(strings.Repeat("x", 32)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(dir, ".fingerprint-key")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := fingerprintKey(dir); err == nil {
		t.Fatal("accepted symlink key")
	}
	if b, err := os.ReadFile(target); err != nil || string(b) != strings.Repeat("x", 32) {
		t.Fatalf("target changed: %q %v", b, err)
	}
}

func TestMetricsRevisionReadsWorktreeAndPackedRefsWithoutGit(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "checkout")
	common := filepath.Join(base, "repo.git")
	gitDir := filepath.Join(common, "worktrees", "checkout")
	if err := os.MkdirAll(gitDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(common, "refs", "heads", "dev"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(root, ".git"):                              "gitdir: " + gitDir + "\n",
		filepath.Join(gitDir, "commondir"):                       "../..\n",
		filepath.Join(gitDir, "HEAD"):                            "ref: refs/heads/dev/metrics\n",
		filepath.Join(common, "refs", "heads", "dev", "metrics"): strings.Repeat("a", 40) + "\n",
	}
	for name, body := range files {
		if err := os.WriteFile(name, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if got := observedRevision(root); got != strings.Repeat("a", 40) {
		t.Fatalf("loose ref: %s", got)
	}
	if err := os.Remove(filepath.Join(common, "refs", "heads", "dev", "metrics")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(common, "packed-refs"), []byte(strings.Repeat("b", 40)+" refs/heads/dev/metrics\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := observedRevision(root); got != strings.Repeat("b", 40) {
		t.Fatalf("packed ref: %s", got)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/../../secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := observedRevision(root); got != "unknown" {
		t.Fatalf("unsafe ref accepted: %s", got)
	}
}
