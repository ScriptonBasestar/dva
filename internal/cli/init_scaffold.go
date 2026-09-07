package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

var errComposeFileNotFound = errors.New("no Docker Compose file detected")

// discoveryOutcome classifies what verified evidence scaffoldDvaYml found in a
// directory before it generates anything (TASK-250 / TASK-249 decided contract).
// The generator never invents a native run/build command, so evidence of a
// language manifest alone ("native") never grows a stack entry — only a Compose
// file does. See generateNativeOnlyConfigIn for what "native" evidence still
// buys the user.
type discoveryOutcome int

const (
	outcomeNoDiscovery discoveryOutcome = iota
	outcomeComposeOnly
	outcomeNativeOnly
	outcomeHybrid
)

// classifyDiscovery inspects dir for verified, self-contained evidence: Compose
// files (sufficient to generate a compose stack entry) and language manifests
// (identity evidence only — never a source for a guessed native runner).
func classifyDiscovery(dir string) (outcome discoveryOutcome, composeFiles []string, nativeLang string) {
	composeFiles = detectComposeFilesIn(dir)
	nativeLang, nativeFound := detectNativeMarkerIn(dir)

	switch {
	case len(composeFiles) > 0 && nativeFound:
		return outcomeHybrid, composeFiles, nativeLang
	case len(composeFiles) > 0:
		return outcomeComposeOnly, composeFiles, ""
	case nativeFound:
		return outcomeNativeOnly, nil, nativeLang
	default:
		return outcomeNoDiscovery, nil, ""
	}
}

// detectNativeMarkerIn reports a verified language manifest in dir, if any.
// Unlike detectTemplateIn it never falls back to "minimal" — absence of
// evidence must be reported as absence, not silently coerced into a guess.
func detectNativeMarkerIn(dir string) (lang string, ok bool) {
	indicators := []struct {
		file string
		lang string
	}{
		{"Gemfile", "rails"},
		{"package.json", "node"},
		{"requirements.txt", "python"},
		{"Pipfile", "python"},
		{"pyproject.toml", "python"},
		{"go.mod", "go"},
		// TASK-322 gap 3: a Go workspace root carries go.work and delegates
		// every go.mod to a member directory, so the root of such a repository
		// read as "no recognized language manifest".
		{"go.work", "go"},
	}
	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(dir, ind.file)); err == nil {
			return ind.lang, true
		}
	}
	// Weaker, but still verified and self-contained: a tool-version manifest
	// names the language outright. TASK-322 — a workspace root that keeps every
	// package manifest one level down (scripton-dashboard: dashboard-webui/
	// package.json) declares its language only here, so without this the whole
	// repository looked manifest-less.
	return detectToolManifestLangIn(dir)
}

// toolManifestLangs maps a tool-version manifest entry to the template language
// it evidences. Only tools DVA actually has a template for are listed; every
// other entry (linters, LSP servers, package managers) is not language
// evidence and must not classify a directory.
var toolManifestLangs = map[string]string{
	"node":   "node",
	"nodejs": "node",
	"python": "python",
	"go":     "go",
	"golang": "go",
	"ruby":   "rails",
}

// detectToolManifestLangIn reads a mise or asdf tool-version manifest in dir and
// reports the first language it declares. It reads only mise's tools table — a
// tool pinned through a backend prefix ("npm:typescript") names a package, not a
// runtime, so those keys are skipped.
//
// Trade-off (TASK-322, deliberate): a tool pin is weaker evidence than a package
// manifest. A repository that pins `python` only for its pre-commit hooks now
// classifies as native-only python and gets python-worded output. The bound on
// that misfire is narrow — a native-only result never authors a stack entry, so
// the worst case is a comment-only dva.yml naming the wrong language, which the
// user edits or deletes, instead of the exit-1 refusal that produced nothing at
// all. Widening it (guessing a run command from the pin) is what the contract
// forbids, and this does not do that.
func detectToolManifestLangIn(dir string) (string, bool) {
	for _, name := range []string{"mise.toml", ".mise.toml", ".tool-versions"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		// .tool-versions has no tables; every line is a tool entry.
		inTools := name == ".tool-versions"
		tableSeen := false
		for line := range strings.SplitSeq(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if strings.HasPrefix(line, "[") {
				inTools = isToolsTableHeader(line)
				tableSeen = true
				continue
			}
			key, _, _ := strings.Cut(line, "=")
			fields := strings.Fields(key)
			if len(fields) == 0 {
				continue
			}
			key = strings.Trim(fields[0], `"'`)
			// A TOML dotted key names its table inline: `tools.node = "24"` at
			// the top of the file is the same pin as `node = "24"` under
			// [tools], and mise accepts both. It only means that before any
			// table header — inside [settings] it would be settings.tools.node.
			if rest, dotted := strings.CutPrefix(key, "tools."); dotted && !tableSeen {
				key = strings.Trim(rest, `"'`)
			} else if !inTools {
				continue
			}
			if key == "" || strings.Contains(key, ":") {
				continue
			}
			if lang, found := toolManifestLangs[key]; found {
				return lang, true
			}
		}
	}
	return "", false
}

// isToolsTableHeader reports whether a TOML table header names mise's tools
// table. Exact-string matching on "[tools]" gave false negatives on the spacing
// and trailing comments real files carry, and a false negative here is not
// cosmetic: it puts the directory back to "no recognized language manifest".
func isToolsTableHeader(line string) bool {
	if idx := strings.Index(line, "#"); idx >= 0 {
		line = line[:idx]
	}
	line = strings.TrimSpace(line)
	line = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
	return strings.Trim(strings.TrimSpace(line), `"'`) == "tools"
}

// scaffoldDvaYml creates a dva.yml in the given directory if one doesn't exist.
// Returns true if a file was created. Generation goes through one canonical
// path shared by `dva init`, `dva config init`, the top-level alias, and
// --recursive sub-project scaffolding.
func scaffoldDvaYml(dir, tmpl string) (bool, error) {
	target := filepath.Join(dir, config.FileName)
	if _, err := os.Stat(target); err == nil {
		fmt.Printf("⏭  dva.yml already exists in %s (skipped)\n", dir)
		return false, nil
	}

	outcome, _, nativeLang := classifyDiscovery(dir)

	if outcome == outcomeNoDiscovery {
		return false, fmt.Errorf(`%w in %s; dva.yml was not created
  DVA init also found no recognized language manifest, so it has no verified
  evidence to scaffold from.
  For non-standard or multi-project layouts, inspect the project first:
    am run dva-discover
  If a full rewrite is explicitly intended, run:
    am run dva-improve -p mode=rewrite
  Or create dva.yml manually, then run:
    dva config validate`, errComposeFileNotFound, dir)
	}

	if outcome == outcomeNativeOnly {
		effectiveTmpl := tmpl
		if effectiveTmpl == "" {
			effectiveTmpl = nativeLang
		}
		content := generateNativeOnlyConfigIn(effectiveTmpl)
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return false, fmt.Errorf("failed to write %s: %w", target, err)
		}
		fmt.Printf("✅ Created %s (no Compose file; %s manifest detected, no stack entry generated — DVA does not guess native run/build commands)\n", target, nativeLang)
		if updated, err := ensureGitignore(dir); err == nil && updated {
			fmt.Printf("📎 Updated .gitignore to ignore %s/\n", config.DotDirName)
		}
		return true, nil
	}

	if outcome == outcomeHybrid {
		fmt.Printf("ℹ️  Detected both a Compose file and a %s project manifest in %s; using the Compose stack (add a native runner manually if you also want one)\n", nativeLang, dir)
	}

	if tmpl == "" {
		tmpl = detectTemplateIn(dir)
	}

	content := generateConfigIn(dir, tmpl)
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", target, err)
	}

	fmt.Printf("✅ Created %s (template: %s)\n", target, tmpl)

	if updated, err := ensureGitignore(dir); err == nil && updated {
		fmt.Printf("📎 Updated .gitignore to ignore %s/\n", config.DotDirName)
	}

	return true, nil
}

// generateNativeOnlyConfigIn produces a minimal, self-contained dva.yml for a
// directory with verified language-manifest evidence but no Compose file. It
// deliberately omits `stack:` — DVA does not guess a native run/build command,
// so a stack entry without verified evidence would be an unverified placeholder,
// which the decided TASK-249 contract forbids. The comment tells a human exactly
// what evidence was insufficient and how to add a native runner by hand.
func generateNativeOnlyConfigIn(tmpl string) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "version: \"%s\"\n\n", config.MinScaffoldVersion)
	b.WriteString("# No Compose file was found, so no `stack:` entry was generated.\n")
	if tmpl != "" {
		_, _ = fmt.Fprintf(&b, "# A %s project manifest was detected, but DVA does not guess a native\n", tmpl)
	} else {
		b.WriteString("# DVA does not guess a native\n")
	}
	b.WriteString("# run/build command, so it cannot fill in stack.<name>.runners.native on its own.\n")
	b.WriteString("# Add one yourself, for example:\n")
	b.WriteString("#\n")
	b.WriteString("# stack:\n")
	b.WriteString("#   app:\n")
	b.WriteString("#     default_runner: native\n")
	b.WriteString("#     runners:\n")
	b.WriteString("#       native:\n")
	b.WriteString("#         run: \"<your run command>\"\n")
	b.WriteString("#\n")
	b.WriteString("# Then run `dva config validate` and `dva up`.\n")
	return b.String()
}
