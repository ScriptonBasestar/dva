package main

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

func preflight(args []string) error {
	f := flag.NewFlagSet("preflight", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	c := commonFlags(f)
	notes := f.String("release-notes", "", "reviewed release notes file")
	notesSHA := f.String("release-notes-sha256", "", "expected SHA-256 of release notes")
	miseFile := f.String("mise-file", ".mise.toml", "mise tool pin file")
	if err := f.Parse(args); err != nil {
		return err
	}
	if err := validateCommon(c); err != nil {
		return err
	}
	if *notes == "" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(*notesSHA) {
		return errors.New("--release-notes and lowercase 64-hex --release-notes-sha256 are required")
	}
	if err := requireCredential(); err != nil {
		return err
	}
	if err := checkDetachedClean(); err != nil {
		return err
	}
	if err := checkOrigin(); err != nil {
		return err
	}
	if err := checkLocalTag(c.tag, c.commit); err != nil {
		return err
	}
	if err := checkVersion(c.tag); err != nil {
		return err
	}
	if err := checkNotes(*notes, *notesSHA); err != nil {
		return err
	}
	if err := checkGoReleaser(*miseFile); err != nil {
		return err
	}
	if err := remoteTagAbsent(c.tag); err != nil {
		return err
	}
	if err := releaseAbsent(c.repo, c.tag); err != nil {
		return err
	}
	if err := capabilityProbe(c.repo, c.tag, c.commit); err != nil {
		return err
	}
	if err := checkCleanup(c.cleanupPaths); err != nil {
		return err
	}
	fmt.Printf("releaseworkflow: preflight passed for %s at %s; no remote state was created\n", c.tag, c.commit)
	return nil
}

func postflight(args []string) error {
	f := flag.NewFlagSet("postflight", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	c := commonFlags(f)
	if err := f.Parse(args); err != nil {
		return err
	}
	if err := validateCommon(c); err != nil {
		return err
	}
	if err := requireCredential(); err != nil {
		return err
	}
	if err := checkOrigin(); err != nil {
		return err
	}
	if err := remoteTagTarget(c.tag, c.commit); err != nil {
		return err
	}
	if err := finalRelease(c.repo, c.tag, c.commit); err != nil {
		return err
	}
	if err := checkCleanup(c.cleanupPaths); err != nil {
		return err
	}
	fmt.Printf("releaseworkflow: postflight passed for %s at %s with the exact seven assets\n", c.tag, c.commit)
	return nil
}

func redactCredential(s string) string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return strings.ReplaceAll(s, token, "[REDACTED]")
	}
	return s
}

func requireCredential() error {
	if os.Getenv("GITHUB_TOKEN") == "" {
		return errors.New("caller must provide a non-empty command-scoped GITHUB_TOKEN; its value is never printed")
	}
	return nil
}

func checkDetachedClean() error {
	out, code, err := runStatus("git", "symbolic-ref", "-q", "HEAD")
	if err != nil {
		return err
	}
	if code == 0 {
		return errors.New("refusing a branch checkout; create a clean detached worktree at the release tag")
	}
	if code != 1 {
		return fmt.Errorf("cannot determine whether HEAD is detached (git symbolic-ref exit %d: %s)", code, strings.TrimSpace(string(out)))
	}
	out, err = run("git", "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(out)) != "" {
		return errors.New("worktree is not clean")
	}
	return nil
}

func checkLocalTag(tag, commit string) error {
	ref := "refs/tags/" + tag
	kind, err := run("git", "cat-file", "-t", ref)
	if err != nil {
		return fmt.Errorf("local tag %s: %w", tag, err)
	}
	if strings.TrimSpace(string(kind)) != "commit" {
		return fmt.Errorf("local tag %s must resolve directly to a commit", tag)
	}
	got, err := run("git", "rev-list", "-n1", ref)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(got)) != commit {
		return fmt.Errorf("local tag %s = %s, want %s", tag, strings.TrimSpace(string(got)), commit)
	}
	head, err := run("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(head)) != commit {
		return fmt.Errorf("HEAD = %s, want release commit %s", strings.TrimSpace(string(head)), commit)
	}
	return nil
}

func checkOrigin() error {
	out, err := run("git", "remote", "get-url", "origin")
	if err != nil {
		return err
	}
	got := strings.TrimSpace(string(out))
	switch got {
	case "git@github.com:ScriptonBasestar/dva.git", "https://github.com/ScriptonBasestar/dva.git", "ssh://git@github.com/ScriptonBasestar/dva.git":
		return nil
	default:
		return fmt.Errorf("origin %q is not the fixed publication repository %s", got, defaultRepo)
	}
}

func checkVersion(tag string) error {
	_, err := run("go", "run", "./tools/releasecheck", "version", "--tag", tag)
	if err != nil {
		return fmt.Errorf("release version identity: %w", err)
	}
	return nil
}

func checkNotes(path, want string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read release notes %s: %w", path, err)
	}
	got := sha256.Sum256(b)
	if hex.EncodeToString(got[:]) != want {
		return fmt.Errorf("release notes SHA-256 = %x, want %s", got, want)
	}
	return nil
}

func checkGoReleaser(miseFile string) error {
	b, err := os.ReadFile(miseFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", miseFile, err)
	}
	m := regexp.MustCompile(`(?m)^goreleaser\s*=\s*"([^"]+)"\s*$`).FindStringSubmatch(string(b))
	if len(m) != 2 {
		return fmt.Errorf("%s does not pin goreleaser", miseFile)
	}
	out, err := run("goreleaser", "--version")
	if err != nil {
		return err
	}
	if !strings.Contains(string(out), m[1]) {
		return fmt.Errorf("goreleaser --version does not contain pinned version %q", m[1])
	}
	return nil
}

func remoteTagAbsent(tag string) error {
	out, code, err := runStatus("git", "ls-remote", "--exit-code", "--tags", "origin", "refs/tags/"+tag)
	if err != nil {
		return err
	}
	if code == 0 {
		return fmt.Errorf("remote tag %s already exists", tag)
	}
	if code == 2 && strings.TrimSpace(string(out)) == "" {
		return nil
	}
	return fmt.Errorf("cannot determine whether remote tag %s exists (git ls-remote exit %d: %s)", tag, code, strings.TrimSpace(string(out)))
}
func releaseAbsent(repo, tag string) error {
	out, code, err := runGHStatus("api", "--include", "repos/"+repo+"/releases/tags/"+tag)
	if err != nil {
		return err
	}
	if code == 0 {
		return fmt.Errorf("GitHub release %s already exists", tag)
	}
	if code != 0 && regexp.MustCompile(`(?m)^HTTP/\S+ 404(?: |\r?$)`).Match(out) {
		return nil
	}
	return fmt.Errorf("cannot determine whether GitHub release %s exists (gh api exit %d: %s)", tag, code, redactCredential(strings.TrimSpace(string(out))))
}

func capabilityProbe(repo, tag, commit string) error {
	_, err := runGH("api", "--method", "POST", "repos/"+repo+"/releases/generate-notes", "-f", "tag_name="+tag, "-f", "target_commitish="+commit)
	if err != nil {
		return fmt.Errorf("non-persisting GitHub generate-notes write-capability probe failed: %w", err)
	}
	return nil
}

func remoteTagTarget(tag, commit string) error {
	out, err := run("git", "ls-remote", "--tags", "origin", "refs/tags/"+tag, "refs/tags/"+tag+"^{}")
	if err != nil {
		return err
	}
	for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == commit {
			return nil
		}
	}
	return fmt.Errorf("remote tag %s does not target %s", tag, commit)
}
