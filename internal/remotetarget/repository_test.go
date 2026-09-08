package remotetarget

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestOriginForms(t *testing.T) {
	for _, raw := range []string{"git@github.com:owner/repo.git", "https://github.com/owner/repo.git", "ssh://git@github.com/owner/repo"} {
		got, err := ParseOrigin(raw)
		if err != nil || got != "owner/repo" {
			t.Fatalf("%s: %q, %v", raw, got, err)
		}
	}
	for _, raw := range []string{"https://token@github.com/owner/repo", "https://github.com.evil.test/owner/repo", "git@alias:owner/repo", "/local/repo", "https://github.com/owner/../repo", "https://github.com/owner/repo?token=secret", "https://github.com/owner%2frepo", "https://github.com:443/owner/repo", "ssh://evil@github.com/owner/repo"} {
		if _, err := ParseOrigin(raw); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

func TestValidateIgnoresParentGitDirectory(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	for dir, repo := range map[string]string{root: "owner/actual", other: "owner/elsewhere"} {
		for _, args := range [][]string{{"init", dir}, {"-C", dir, "remote", "add", "origin", "https://github.com/" + repo + ".git"}} {
			if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
				t.Fatalf("git %v: %v %s", args, err, out)
			}
		}
	}
	t.Setenv("GIT_DIR", filepath.Join(other, ".git"))
	if err := Validate(t.Context(), root, "owner/actual"); err != nil {
		t.Fatal(err)
	}
	if err := Validate(t.Context(), root, "owner/elsewhere"); err == nil {
		t.Fatal("parent GIT_DIR redirected target")
	}
}

func TestValidateUsesLiteralOrigin(t *testing.T) {
	root := t.TempDir()
	for _, args := range [][]string{
		{"init", root},
		{"-C", root, "remote", "add", "origin", "https://github.com/owner/actual.git"},
		{"-C", root, "config", "url.https://github.com/elsewhere/repo.git.insteadOf", "https://github.com/owner/actual.git"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if err := Validate(t.Context(), root, "owner/actual"); err != nil {
		t.Fatal(err)
	}
	if err := Validate(t.Context(), root, "elsewhere/repo"); err == nil {
		t.Fatal("insteadOf redirected the trust anchor")
	}
}
