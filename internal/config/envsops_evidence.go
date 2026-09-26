package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// NewForSopsDetection returns a bare Config rooted at dir, for evidence-only
// detection before a dva.yml exists — dva init's own scan (TASK-441). It
// declares no secrets.sources and no env_file, so DetectSopsEvidence sees
// exactly what a fresh directory shows: the naming-convention scan and the
// .sops.yaml check, nothing a not-yet-written config could have declared.
func NewForSopsDetection(dir string) *Config {
	return &Config{filePath: filepath.Join(dir, FileName)}
}

// SopsEvidence is what a project directory shows about sops use without anyone
// having declared it in env_file.
//
// It exists because sops_source is invisible to the load path (TASK-245 §2-1):
// a config that omits it runs `up` and `run` without complaint, and the omission
// only surfaces when `dva config env` refuses. Evidence lets the refusal, and
// doctor before it, name the file the author most likely meant instead of
// describing the field in the abstract.
type SopsEvidence struct {
	// CreationRules reports a .sops.yaml at the config root.
	CreationRules bool
	// Candidates are config-root-relative paths that look like sops-encrypted
	// files, sorted, deduplicated. secrets.sources entries come first in the
	// sense that they are declared intent; directory matches are a guess.
	Candidates []string
}

// Found reports whether there is any sign of sops use at all.
func (e SopsEvidence) Found() bool { return e.CreationRules || len(e.Candidates) > 0 }

// DetectSopsEvidence inspects the config root (one level, no recursion) and the
// secrets.sources declarations. It reads names only, never file contents: an
// encrypted file's bytes are not needed to recognise it, and a misnamed
// plaintext secret must not be opened by a diagnostic.
func (c *Config) DetectSopsEvidence() SopsEvidence {
	var ev SopsEvidence
	// A declared plaintext path is never a source candidate, whatever its name:
	// suggesting it would render {path: X, sops_source: X}, which every bridge
	// command refuses as source_is_target.
	seen := map[string]bool{}
	for _, e := range c.AllEnvFileConfigs() {
		seen[e.Path] = true
	}
	add := func(p string) {
		if p != "" && !seen[p] {
			seen[p] = true
			ev.Candidates = append(ev.Candidates, p)
		}
	}

	if c.Secrets != nil {
		for _, name := range sortedKeysOf(c.Secrets.Sources) {
			add(c.Secrets.Sources[name].Sops)
		}
	}

	entries, err := os.ReadDir(c.FileDir())
	if err == nil {
		var found []string
		for _, e := range entries {
			if !e.Type().IsRegular() {
				continue
			}
			name := e.Name()
			if name == ".sops.yaml" || name == ".sops.yml" {
				ev.CreationRules = true
				continue
			}
			if looksSopsEncrypted(name) {
				found = append(found, name)
			}
		}
		slices.Sort(found)
		for _, f := range found {
			add(f)
		}
	}
	return ev
}

// looksSopsEncrypted matches the naming conventions sops users actually use for
// dotenv payloads: a trailing .enc or .sops, or a .sops. / .enc. infix
// (secrets.sops.env, .env.enc.yaml).
func looksSopsEncrypted(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, ".enc") || strings.HasSuffix(lower, ".sops") ||
		strings.Contains(lower, ".sops.") || strings.Contains(lower, ".enc.")
}
