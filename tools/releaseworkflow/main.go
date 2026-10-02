// releaseworkflow validates the manual release boundary without publishing or deleting remote state.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const defaultRepo = "ScriptonBasestar/dva"

var (
	fullSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
	assets  = map[string]bool{
		"checksums.txt":          true,
		"dva_linux_amd64.tar.gz": true, "dva_linux_arm64.tar.gz": true,
		"dva_darwin_amd64.tar.gz": true, "dva_darwin_arm64.tar.gz": true,
		"dva_windows_amd64.zip": true, "dva_windows_arm64.zip": true,
	}
)

func main() {
	if len(os.Args) < 2 {
		fail(errors.New("usage: releaseworkflow <preflight|postflight|clean> [flags]"))
	}
	var err error
	switch os.Args[1] {
	case "preflight":
		err = preflight(os.Args[2:])
	case "postflight":
		err = postflight(os.Args[2:])
	case "clean":
		err = clean(os.Args[2:])
	default:
		err = fmt.Errorf("unknown command %q (want preflight, postflight, or clean)", os.Args[1])
	}
	fail(err)
}

type common struct {
	tag, commit, repo string
	cleanupPaths      []string
}

func commonFlags(f *flag.FlagSet) *common {
	c := &common{repo: defaultRepo}
	f.StringVar(&c.tag, "tag", "", "exact v-prefixed release tag")
	f.StringVar(&c.commit, "commit", "", "exact 40-hex release commit")
	f.Var((*stringList)(&c.cleanupPaths), "cleanup-path", "path that must not exist after the check (repeatable)")
	return c
}

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

func validateCommon(c *common) error {
	if !strings.HasPrefix(c.tag, "v") || len(c.tag) == 1 {
		return fmt.Errorf("--tag %q must be v-prefixed", c.tag)
	}
	if !fullSHA.MatchString(c.commit) {
		return fmt.Errorf("--commit %q is not a full 40-hex SHA", c.commit)
	}
	if strings.Count(c.repo, "/") != 1 || strings.HasPrefix(c.repo, "/") || strings.HasSuffix(c.repo, "/") {
		return fmt.Errorf("--repo %q must be owner/repository", c.repo)
	}
	return nil
}

func run(name string, args ...string) ([]byte, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

func runStatus(name string, args ...string) ([]byte, int, error) {
	return commandStatus(exec.Command(name, args...))
}

func runGH(args ...string) ([]byte, error) {
	out, code, err := runGHStatus(args...)
	if err != nil {
		return out, err
	}
	if code != 0 {
		return out, fmt.Errorf("gh %s: exit status %d: %s", strings.Join(args, " "), code, redactCredential(strings.TrimSpace(string(out))))
	}
	return out, nil
}

func runGHStatus(args ...string) ([]byte, int, error) {
	cmd := exec.Command("gh", args...)
	env := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "GH_TOKEN=") &&
			!strings.HasPrefix(entry, "GITHUB_TOKEN=") &&
			!strings.HasPrefix(entry, "GH_HOST=") &&
			!strings.HasPrefix(entry, "GH_ENTERPRISE_TOKEN=") &&
			!strings.HasPrefix(entry, "GITHUB_ENTERPRISE_TOKEN=") {
			env = append(env, entry)
		}
	}
	token := os.Getenv("GITHUB_TOKEN")
	cmd.Env = append(env, "GH_HOST=github.com", "GH_TOKEN="+token, "GITHUB_TOKEN="+token)
	return commandStatus(cmd)
}

func commandStatus(cmd *exec.Cmd) ([]byte, int, error) {
	out, err := cmd.CombinedOutput()
	if err == nil {
		return out, 0, nil
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		return out, exitErr.ExitCode(), nil
	}
	return out, -1, fmt.Errorf("start %s: %w", cmd.Path, err)
}

type release struct {
	TagName    string `json:"tagName"`
	Target     string `json:"targetCommitish"`
	Draft      bool   `json:"isDraft"`
	Prerelease bool   `json:"isPrerelease"`
	Assets     []struct {
		Name  string `json:"name"`
		State string `json:"state"`
		Size  int64  `json:"size"`
	} `json:"assets"`
}

func finalRelease(repo, tag, commit string) error {
	out, err := runGH("release", "view", tag, "--repo", repo, "--json", "tagName,targetCommitish,isDraft,isPrerelease,assets")
	if err != nil {
		return err
	}
	var r release
	if err := json.Unmarshal(out, &r); err != nil {
		return fmt.Errorf("parse GitHub release response: %w", err)
	}
	if r.TagName != tag || r.Target != commit || r.Draft || r.Prerelease {
		return fmt.Errorf("release must be final and target %s (tag=%q target=%q draft=%t prerelease=%t)", commit, r.TagName, r.Target, r.Draft, r.Prerelease)
	}
	got := map[string]bool{}
	for _, a := range r.Assets {
		if a.State != "uploaded" || a.Size <= 0 {
			return fmt.Errorf("release asset %s must be uploaded and non-empty (state=%q size=%d)", a.Name, a.State, a.Size)
		}
		got[a.Name] = true
	}
	if len(r.Assets) != len(assets) || len(got) != len(assets) {
		return fmt.Errorf("release assets = %v, want exactly seven expected assets", sorted(got))
	}
	for a := range assets {
		if !got[a] {
			return fmt.Errorf("release assets missing %s", a)
		}
	}
	return verifyReleaseDownloads(repo, tag)
}

func verifyReleaseDownloads(repo, tag string) error {
	dir, err := os.MkdirTemp("", "dva-release-postflight-")
	if err != nil {
		return fmt.Errorf("create release verification directory: %w", err)
	}
	defer func() {
		_ = os.RemoveAll(dir)
	}()

	names := sorted(assets)
	args := []string{"release", "download", tag, "--repo", repo, "--dir", dir}
	for _, name := range names {
		args = append(args, "--pattern", name)
	}
	if _, err := runGH(args...); err != nil {
		return fmt.Errorf("download release assets for checksum verification: %w", err)
	}
	return verifyDownloadedChecksums(dir)
}

func verifyDownloadedChecksums(dir string) error {
	b, err := os.ReadFile(filepath.Join(dir, "checksums.txt"))
	if err != nil {
		return fmt.Errorf("read downloaded checksums.txt: %w", err)
	}
	want := make(map[string]string, len(assets)-1)
	for lineNo, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(fields[0]) {
			return fmt.Errorf("checksums.txt line %d is not '<sha256> <asset>'", lineNo+1)
		}
		name := strings.TrimPrefix(fields[1], "*")
		if name == "checksums.txt" || !assets[name] {
			return fmt.Errorf("checksums.txt names unexpected asset %q", name)
		}
		if _, exists := want[name]; exists {
			return fmt.Errorf("checksums.txt repeats asset %q", name)
		}
		want[name] = fields[0]
	}
	if len(want) != len(assets)-1 {
		return fmt.Errorf("checksums.txt covers %d archives, want %d", len(want), len(assets)-1)
	}
	for name, digest := range want {
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read downloaded asset %s: %w", name, err)
		}
		got := sha256.Sum256(b)
		if hex.EncodeToString(got[:]) != digest {
			return fmt.Errorf("downloaded asset %s SHA-256 does not match checksums.txt", name)
		}
	}
	return nil
}

func clean(args []string) error {
	f := flag.NewFlagSet("clean", flag.ContinueOnError)
	f.SetOutput(io.Discard)
	if err := f.Parse(args); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("clean accepts no arguments")
	}
	if err := checkRepositoryRoot(); err != nil {
		return err
	}
	if err := checkOrigin(); err != nil {
		return err
	}
	paths := []string{"dist", "bin", "tmp"}
	for _, name := range paths {
		info, err := os.Lstat(name)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect cleanup path %s: %w", name, err)
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return fmt.Errorf("refusing cleanup path %s: expected a real directory", name)
		}
	}
	for _, name := range paths {
		if err := os.RemoveAll(name); err != nil {
			return fmt.Errorf("remove cleanup path %s: %w", name, err)
		}
	}
	fmt.Println("releaseworkflow: removed repository-local dist, bin, and tmp outputs")
	return nil
}

func checkRepositoryRoot() error {
	out, err := run("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	root, err := filepath.EvalSymlinks(strings.TrimSpace(string(out)))
	if err != nil {
		return fmt.Errorf("resolve repository root path: %w", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve current directory: %w", err)
	}
	cwd, err = filepath.EvalSymlinks(cwd)
	if err != nil {
		return fmt.Errorf("resolve current directory path: %w", err)
	}
	if cwd != root {
		return fmt.Errorf("refusing cleanup outside repository root (cwd=%s root=%s)", cwd, root)
	}
	return nil
}
func sorted(m map[string]bool) []string {
	r := make([]string, 0, len(m))
	for k := range m {
		r = append(r, k)
	}
	sort.Strings(r)
	return r
}
func checkCleanup(paths []string) error {
	for _, p := range paths {
		if _, err := os.Lstat(filepath.Clean(p)); err == nil {
			return fmt.Errorf("cleanup path still exists: %s", p)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect cleanup path %s: %w", p, err)
		}
	}
	return nil
}
func fail(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "releaseworkflow: ERROR: %v\n", err)
		os.Exit(1)
	}
}
