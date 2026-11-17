package cimetrics

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Metadata intentionally excludes command arguments, environment values and
// paths. An unknown value is not evidence of a particular cache or toolchain.
type Metadata struct {
	Revision       string            `json:"revision"`
	Dirty          string            `json:"dirty"`
	ObservedAt     *time.Time        `json:"observed_at,omitempty"`
	SourceEvidence string            `json:"source_evidence"`
	OS             string            `json:"os"`
	Arch           string            `json:"arch"`
	CPUs           int               `json:"cpus"`
	WrapperGo      string            `json:"wrapper_go"`
	GoBinaryBuild  string            `json:"go_binary_build"`
	GOROOTVersion  string            `json:"goroot_version"`
	GoPairing      string            `json:"go_pairing"`
	Fresh          string            `json:"fresh"`
	Workspace      string            `json:"workspace"`
	ConfigKeyID    string            `json:"config_key_id,omitempty"`
	ConfigHMAC     map[string]string `json:"config_hmac,omitempty"`
	ConfigStatus   string            `json:"config_fingerprint_status"`
}

// Options records recognized flags and repository packages only; raw command
// text, unknown flag values and dynamic filters never enter metrics files.
type Options struct {
	Flags        []string `json:"flags,omitempty"`
	Packages     []string `json:"packages,omitempty"`
	Filtered     bool     `json:"test_filter"`
	Unclassified bool     `json:"unclassified_options"`
	Workers      int      `json:"workers,omitempty"`
}

func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func gitDirectories(root string) (string, string, bool) {
	gitDir := filepath.Join(root, ".git")
	info, err := os.Stat(gitDir)
	if err != nil {
		return "", "", false
	}
	commonDir := gitDir
	if !info.IsDir() {
		b, e := readSmall(gitDir, 4096)
		if e != nil || !strings.HasPrefix(string(b), "gitdir: ") {
			return "", "", false
		}
		gitDir = strings.TrimSpace(strings.TrimPrefix(string(b), "gitdir: "))
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(root, gitDir)
		}
		common, e := readSmall(filepath.Join(gitDir, "commondir"), 4096)
		if e != nil {
			return "", "", false
		}
		commonPath := strings.TrimSpace(string(common))
		if filepath.IsAbs(commonPath) {
			commonDir = commonPath
		} else {
			commonDir = filepath.Join(gitDir, commonPath)
		}
	}
	gitDir, err = filepath.EvalSymlinks(gitDir)
	if err != nil {
		return "", "", false
	}
	commonDir, err = filepath.EvalSymlinks(commonDir)
	if err != nil {
		return "", "", false
	}
	for _, dir := range []string{gitDir, commonDir} {
		info, err = os.Stat(dir)
		if err != nil || !info.IsDir() {
			return "", "", false
		}
	}
	return gitDir, commonDir, true
}

func repositoryKey(root string) (string, bool) {
	_, common, ok := gitDirectories(root)
	if !ok {
		return "", false
	}
	sum := sha256.Sum256([]byte(common))
	return hex.EncodeToString(sum[:]), true
}

// observedRevision is intentionally not an attestation: it reads only bounded
// local HEAD/ref files, does not invoke Git and never records reference names.
func observedRevision(root string) string {
	gitDir, common, ok := gitDirectories(root)
	if !ok {
		return "unknown"
	}
	head, err := readSmall(filepath.Join(gitDir, "HEAD"), 4096)
	if err != nil {
		return "unknown"
	}
	value := strings.TrimSpace(string(head))
	if len(value) == 40 && isHex(value) {
		return value
	}
	if !strings.HasPrefix(value, "ref: ") {
		return "unknown"
	}
	ref := strings.TrimPrefix(value, "ref: ")
	if !safeRef(ref) {
		return "unknown"
	}
	for _, dir := range []string{gitDir, common} {
		if b, e := readSmall(filepath.Join(dir, filepath.FromSlash(ref)), 4096); e == nil {
			sha := strings.TrimSpace(string(b))
			if len(sha) == 40 && isHex(sha) {
				return sha
			}
		}
	}
	packed, err := readSmall(filepath.Join(common, "packed-refs"), 4*1024*1024)
	if err != nil {
		return "unknown"
	}
	for line := range strings.SplitSeq(string(packed), "\n") {
		sha, name, ok := strings.Cut(line, " ")
		if ok && name == ref && len(sha) == 40 && isHex(sha) {
			return sha
		}
	}
	return "unknown"
}

func safeRef(ref string) bool {
	if !strings.HasPrefix(ref, "refs/") || len(ref) > 1024 || strings.Contains(ref, "..") || strings.Contains(ref, "//") || strings.HasSuffix(ref, "/") {
		return false
	}
	for _, ch := range ref {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') && (ch < '0' || ch > '9') && !strings.ContainsRune("/._-", ch) {
			return false
		}
	}
	return true
}

func collectMetadata(root, dir string) Metadata {
	m := unknownMetadata()
	observed := time.Now().UTC()
	m.ObservedAt = &observed
	m.SourceEvidence = "asynchronous_unattested"
	m.Workspace = "auto_unknown"
	if v := os.Getenv("GOWORK"); v == "off" {
		m.Workspace = "off"
	} else if v != "" {
		m.Workspace = "explicit_unknown"
	}
	if fresh := os.Getenv("DVA_CI_FRESH"); fresh == "1" {
		m.Fresh = "true"
	} else if fresh == "0" {
		m.Fresh = "false"
	}
	m.Revision = observedRevision(root)
	// Inspect a real selected binary without running a version manager or
	// causing toolchain downloads. A shim or automatic toolchain is unknown.
	if path, err := exec.LookPath("go"); err == nil {
		if resolved, e := filepath.EvalSymlinks(path); e == nil {
			if info, e := buildinfo.ReadFile(resolved); e == nil && safeGoVersion(info.GoVersion) {
				m.GoBinaryBuild = info.GoVersion
			}
		}
	}
	if rootEnv := os.Getenv("GOROOT"); rootEnv != "" {
		if b, e := readSmall(filepath.Join(rootEnv, "VERSION"), 1024); e == nil {
			v, _, _ := strings.Cut(string(b), "\n")
			if safeGoVersion(v) {
				m.GOROOTVersion = v
			}
		}
	}
	if os.Getenv("GOTOOLCHAIN") == "local" && m.GoBinaryBuild != "unknown" && m.GOROOTVersion != "unknown" {
		if m.GoBinaryBuild == m.GOROOTVersion {
			m.GoPairing = "matched_build_versions"
		} else {
			m.GoPairing = "different_build_versions"
		}
	}
	if key, e := fingerprintKey(dir); e == nil {
		id := sha256.Sum256(key)
		m.ConfigKeyID = hex.EncodeToString(id[:8])
		m.ConfigHMAC = make(map[string]string)
		paths := map[string]string{".mise.toml": filepath.Join(root, ".mise.toml"), "go.mod": filepath.Join(root, "go.mod"), "go.sum": filepath.Join(root, "go.sum"), "dva.yml": filepath.Join(root, "dva.yml"), "workspace_candidate": filepath.Join(root, "go.work")}
		if v := os.Getenv("GOWORK"); v != "" && v != "off" && filepath.IsAbs(v) {
			paths["workspace_explicit"] = v
		}
		for name, path := range paths {
			f, e := openNoFollow(path)
			if e != nil {
				continue
			}
			info, e := f.Stat()
			if e != nil || !info.Mode().IsRegular() || info.Size() > 2*1024*1024 {
				_ = f.Close()
				continue
			}
			b, e := io.ReadAll(io.LimitReader(f, 2*1024*1024+1))
			_ = f.Close()
			if e != nil || len(b) > 2*1024*1024 {
				continue
			}
			h := hmac.New(sha256.New, key)
			_, _ = h.Write(b)
			m.ConfigHMAC[name] = hex.EncodeToString(h.Sum(nil))
		}
		m.ConfigStatus = "complete"
		for _, name := range []string{".mise.toml", "go.mod", "go.sum", "dva.yml"} {
			if _, ok := m.ConfigHMAC[name]; !ok {
				m.ConfigStatus = "partial"
			}
		}
	}
	return m
}

var goVersionPattern = regexp.MustCompile(`^go1\.[0-9]{1,3}(\.[0-9]{1,3})?((rc|beta)[0-9]{1,3})?$`)

func safeGoVersion(v string) bool { return goVersionPattern.MatchString(v) }

func fingerprintKey(dir string) ([]byte, error) {
	path := filepath.Join(dir, ".fingerprint-key")
	read := func() ([]byte, error) {
		f, err := openNoFollow(path)
		if err != nil {
			return nil, err
		}
		defer func() { _ = f.Close() }()
		info, err := f.Stat()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() || !ownedByUser(info) || info.Mode().Perm()&0o077 != 0 || info.Size() != 32 {
			return nil, os.ErrInvalid
		}
		b, err := io.ReadAll(io.LimitReader(f, 33))
		if err != nil || len(b) != 32 {
			return nil, os.ErrInvalid
		}
		return b, nil
	}
	if b, err := read(); err == nil {
		return b, nil
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	var key [32]byte
	if _, err := rand.Read(key[:]); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(dir, ".key-*")
	if err != nil {
		return nil, err
	}
	name := f.Name()
	defer func() { _ = os.Remove(name) }()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return nil, err
	}
	if _, err := f.Write(key[:]); err != nil {
		_ = f.Close()
		return nil, err
	}
	if err := f.Close(); err != nil {
		return nil, err
	}
	if err := os.Link(name, path); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return read()
}

func safeOptions(argv []string, allowed map[string]map[string]bool) Options {
	o := Options{}
	for i := 0; i < len(argv); i++ {
		a := argv[i]
		if i == 0 || a == "test" || a == "--" {
			continue
		}
		if a == "-json" || a == "-race" || a == "-cover" {
			o.Flags = append(o.Flags, a)
			continue
		}
		if a == "-run" || a == "-bench" || a == "-fuzz" {
			o.Filtered = true
			if i+1 < len(argv) {
				i++
			} else {
				o.Unclassified = true
			}
			continue
		}
		if strings.HasPrefix(a, "-run=") || strings.HasPrefix(a, "-bench=") || strings.HasPrefix(a, "-fuzz=") {
			o.Filtered = true
			continue
		}
		if a == "-timeout" || a == "-count" || a == "-p" || a == "-tags" {
			if i+1 >= len(argv) {
				o.Unclassified = true
				continue
			}
			i++
			a += "=" + argv[i]
		}
		if value, ok := strings.CutPrefix(a, "-timeout="); ok {
			d, err := time.ParseDuration(value)
			if err == nil && d > 0 && d <= 24*time.Hour {
				o.Flags = append(o.Flags, "-timeout="+d.String())
			} else {
				o.Unclassified = true
			}
			continue
		}
		if strings.HasPrefix(a, "-count=") || strings.HasPrefix(a, "-p=") {
			k, v, _ := strings.Cut(a, "=")
			n, e := strconv.Atoi(v)
			if e == nil && n >= 0 && n <= 10000 {
				o.Flags = append(o.Flags, k+"="+strconv.Itoa(n))
			} else {
				o.Unclassified = true
			}
			continue
		}
		if a == "-tags=integration" {
			o.Flags = append(o.Flags, a)
			continue
		}
		if a == "./..." {
			o.Packages = append(o.Packages, a)
			continue
		}
		if strings.HasPrefix(a, module+"/") {
			base := strings.TrimSuffix(a, "/...")
			if _, ok := allowed[base]; ok && !strings.Contains(strings.TrimPrefix(base, module), "..") {
				o.Packages = append(o.Packages, a)
			} else {
				o.Unclassified = true
			}
			continue
		}
		if value, ok := strings.CutPrefix(a, "./"); ok {
			base := strings.TrimSuffix(value, "/...")
			if _, ok := allowed[module+"/"+base]; ok && !strings.Contains(base, "..") {
				o.Packages = append(o.Packages, a)
			} else {
				o.Unclassified = true
			}
			continue
		}
		o.Unclassified = true
	}
	if n, e := strconv.Atoi(os.Getenv("DVA_CI_JOBS")); e == nil && n > 0 && n <= 10000 {
		o.Workers = n
	}
	fields := strings.Fields(os.Getenv("GOFLAGS"))
	for i := 0; i < len(fields); i++ {
		a := fields[i]
		if a == "-p" && i+1 < len(fields) {
			i++
			a = "-p=" + fields[i]
		}
		if value, ok := strings.CutPrefix(a, "-p="); ok {
			n, e := strconv.Atoi(value)
			if e == nil && n > 0 && n <= 10000 {
				o.Flags = append(o.Flags, "GOFLAGS:-p="+strconv.Itoa(n))
				continue
			}
		}
		o.Unclassified = true
	}
	return o
}

func readSmall(path string, limit int64) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > limit {
		return nil, os.ErrInvalid
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	b, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > limit {
		return nil, os.ErrInvalid
	}
	return b, nil
}

func isHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}
