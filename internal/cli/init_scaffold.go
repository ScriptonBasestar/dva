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
func classifyDiscovery(dir string) (outcome discoveryOutcome, composeFiles []string, nativeLang string, nativeEvidence langEvidence) {
	composeFiles = detectComposeFilesIn(dir)
	nativeLang, nativeEvidence = detectNativeMarkerIn(dir)

	switch {
	case len(composeFiles) > 0 && nativeEvidence != evidenceNone:
		return outcomeHybrid, composeFiles, nativeLang, nativeEvidence
	case len(composeFiles) > 0:
		return outcomeComposeOnly, composeFiles, "", evidenceNone
	case nativeEvidence != evidenceNone:
		return outcomeNativeOnly, nil, nativeLang, nativeEvidence
	default:
		return outcomeNoDiscovery, nil, "", evidenceNone
	}
}

// langEvidence grades how a directory's language was identified. The grade is
// not an implementation detail: it decides what DVA is entitled to *say*. A
// package manifest is a statement that the repository is a project in that
// language; a tool-version pin only says that runtime is installed, which a
// repository may pin for a lint hook in a language it is not written in.
// Calling the latter a "project manifest" asserts something DVA did not
// observe, so the two grades carry different nouns. See phrase.
type langEvidence int

const (
	evidenceNone langEvidence = iota
	evidenceDirectManifest
	evidenceToolPin
)

// phrase renders the noun the user-facing announcements interpolate, so the
// wording cannot drift from the grade that produced it.
func (e langEvidence) phrase(lang string) string {
	if e == evidenceToolPin {
		return lang + " runtime pin"
	}
	return lang + " project manifest"
}

// detectNativeMarkerIn reports a verified language manifest in dir, if any.
// Unlike detectTemplateIn it never falls back to "minimal" — absence of
// evidence must be reported as absence, not silently coerced into a guess.
//
// It accepts two grades of evidence, and callers must not treat them alike.
// detectDirectManifestLangIn is a statement that the repository *is* a project
// in that language; a tool-version pin only says a runtime is available. The
// weaker grade is enough to classify a directory as native-only — whose output
// is comment-only — and detectTemplateIn deliberately does not accept it,
// because a template authors real commands. See detectToolManifestLangIn.
func detectNativeMarkerIn(dir string) (lang string, evidence langEvidence) {
	if lang, ok := detectDirectManifestLangIn(dir); ok {
		return lang, evidenceDirectManifest
	}
	// Weaker, but still verified and self-contained: a tool-version manifest
	// names the language outright. TASK-322 — a workspace root that keeps every
	// package manifest one level down (scripton-dashboard: dashboard-webui/
	// package.json) declares its language only here, so without this the whole
	// repository looked manifest-less.
	if lang, ok := detectToolManifestLangIn(dir); ok {
		return lang, evidenceToolPin
	}
	return "", evidenceNone
}

// detectDirectManifestLangIn reports a language declared by a manifest whose
// mere existence identifies the project: a package manifest, or a Go workspace
// file. This is the only evidence strong enough to pick a template.
func detectDirectManifestLangIn(dir string) (lang string, ok bool) {
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
		// read as "no recognized language manifest". go.work is direct
		// evidence — nothing but a Go workspace has one.
		{"go.work", "go"},
	}
	for _, ind := range indicators {
		if _, err := os.Stat(filepath.Join(dir, ind.file)); err == nil {
			return ind.lang, true
		}
	}
	return "", false
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
// manifest — it says a runtime is available, not that the repository is written
// in it. A repository that pins `python` only for its pre-commit hooks does
// classify as native-only python and gets python-worded output.
//
// The bound is that pin-derived evidence reaches classification and wording
// only, never a generated command. detectTemplateIn refuses it, so a pin can
// never select a template; and a native-only result authors no stack entry at
// all. The worst case is therefore a comment-only dva.yml naming the wrong
// language — which the user edits or deletes — in place of the exit-1 refusal
// that produced nothing. Letting a pin choose a template would break that bound:
// the python template writes `python manage.py`, `python -m pytest` and `pip`
// against a real compose service.
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
	return scaffoldDvaYmlWithPreview(dir, tmpl, false)
}

// scaffoldDvaYmlWithPreview creates a dva.yml unless preview is set. Preview
// uses the same discovery and generation path as creation so `dva init
// --dry-run` shows the exact file a normal invocation would write, while
// leaving every filesystem artifact untouched.
func scaffoldDvaYmlWithPreview(dir, tmpl string, preview bool) (bool, error) {
	target := filepath.Join(dir, config.FileName)
	if _, err := os.Stat(target); err == nil {
		if preview {
			fmt.Printf("[dry-run] %s already exists (no preview generated)\n", target)
		} else {
			fmt.Printf("⏭  dva.yml already exists in %s (skipped)\n", dir)
		}
		return false, nil
	}

	discovered, err := discoverNativeScaffold(dir)
	if err != nil {
		return false, err
	}
	outcome, _, nativeLang, nativeEvidence := classifyDiscovery(dir)

	if outcome == outcomeNoDiscovery && !discovered.hasEvidence() {
		err := fmt.Errorf(`%w in %s; dva.yml was not created
  DVA init also found no recognized language manifest, so it has no verified
  evidence to scaffold from.
  For non-standard or multi-project layouts, inspect the project first:
    am run dva-discover
  If a full rewrite is explicitly intended, run:
    am run dva-improve -p mode=rewrite
  Or create dva.yml manually, then run:
    dva config validate`, errComposeFileNotFound, dir)
		if preview {
			fmt.Printf("[dry-run] no dva.yml preview for %s:\n%s\n", dir, err)
			return false, nil
		}
		return false, err
	}

	if (outcome == outcomeNativeOnly || outcome == outcomeNoDiscovery) && discovered.hasEvidence() {
		content := generateDiscoveredConfig(discovered, nativeLang, nativeEvidence)
		if preview {
			printDvaYmlPreview(target, content)
			return true, nil
		}
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return false, fmt.Errorf("failed to write %s: %w", target, err)
		}
		detectedLanguage := ""
		if nativeLang != "" {
			detectedLanguage = "; " + nativeEvidence.phrase(nativeLang) + " detected"
		}
		fmt.Printf("✅ Created %s (%d declared Makefile native entries, %d workspace subprojects, %d port mappings%s)\n", target, len(discovered.entries), len(discovered.subprojects), len(discovered.ports), detectedLanguage)
		if updated, err := ensureGitignore(dir); err == nil && updated {
			// Named as the rules that were written, not as "ignore .sb/dva/". The
			// second is what DVA used to write and is no longer true of what it
			// writes: the directory is not ignored, its transient contents are, and
			// modules stay committable. A line of output that spells the old rule is
			// the same defect as a fix hint that spells it.
			fmt.Printf("📎 Updated .gitignore with %s\n", defaultIgnoreAdvice())
		}
		return true, nil
	}

	if outcome == outcomeNativeOnly {
		effectiveTmpl := tmpl
		if effectiveTmpl == "" {
			effectiveTmpl = nativeLang
		}
		content := generateNativeOnlyConfigIn(effectiveTmpl, nativeEvidence)
		if preview {
			printDvaYmlPreview(target, content)
			return true, nil
		}
		if err := os.WriteFile(target, []byte(content), 0644); err != nil {
			return false, fmt.Errorf("failed to write %s: %w", target, err)
		}
		fmt.Printf("✅ Created %s (no Compose file; %s detected, no stack entry generated — DVA does not guess native run/build commands)\n", target, nativeEvidence.phrase(nativeLang))
		if updated, err := ensureGitignore(dir); err == nil && updated {
			// Named as the rules that were written, not as "ignore .sb/dva/". The
			// second is what DVA used to write and is no longer true of what it
			// writes: the directory is not ignored, its transient contents are, and
			// modules stay committable. A line of output that spells the old rule is
			// the same defect as a fix hint that spells it.
			fmt.Printf("📎 Updated .gitignore with %s\n", defaultIgnoreAdvice())
		}
		return true, nil
	}

	if outcome == outcomeHybrid {
		if len(discovered.entries) > 0 {
			fmt.Printf("ℹ️  Detected both a Compose file and declared Makefile native targets in %s; generating both runner types\n", dir)
		} else {
			fmt.Printf("ℹ️  Detected both a Compose file and a %s in %s; using the Compose stack (add a native runner manually if you also want one)\n", nativeEvidence.phrase(nativeLang), dir)
		}
	}

	if tmpl == "" {
		tmpl = detectTemplateIn(dir)
	}

	content := generateConfigIn(dir, tmpl)
	if discovered.hasEvidence() {
		content, err = mergeInitDiscovery(content, generateDiscoveredConfig(discovered, nativeLang, nativeEvidence))
		if err != nil {
			return false, err
		}
	}
	if preview {
		printDvaYmlPreview(target, content)
		return true, nil
	}
	if err := os.WriteFile(target, []byte(content), 0644); err != nil {
		return false, fmt.Errorf("failed to write %s: %w", target, err)
	}

	fmt.Printf("✅ Created %s (template: %s)\n", target, tmpl)

	if updated, err := ensureGitignore(dir); err == nil && updated {
		// The template branch of `dva init`, and the last place the old spelling
		// survived. It was missed when the other two were changed because the
		// coupling test cannot reach it: this site interpolated config.DotDirName
		// directly and never called defaultIgnoreAdvice, so no assertion about the
		// advice naming the rules had anything to assert against here. The same
		// reasoning applies as at the other two — a line of output that spells the
		// old rule is the same defect as a fix hint that spells it.
		fmt.Printf("📎 Updated .gitignore with %s\n", defaultIgnoreAdvice())
	}

	return true, nil
}

func printDvaYmlPreview(target, content string) {
	fmt.Printf("[dry-run] would create %s:\n%s", target, content)
	if !strings.HasSuffix(content, "\n") {
		fmt.Println()
	}
}

// generateNativeOnlyConfigIn produces a minimal, self-contained dva.yml for a
// directory with verified language-manifest evidence but no Compose file. It
// deliberately omits `stack:` — DVA does not guess a native run/build command,
// so a stack entry without verified evidence would be an unverified placeholder,
// which the decided TASK-249 contract forbids. The comment tells a human exactly
// what evidence was insufficient and how to add a native runner by hand.
func generateNativeOnlyConfigIn(tmpl string, evidence langEvidence) string {
	var b strings.Builder
	_, _ = fmt.Fprintf(&b, "version: \"%s\"\n\n", config.MinScaffoldVersion)
	b.WriteString("# No Compose file was found, so no `stack:` entry was generated.\n")
	if tmpl != "" {
		_, _ = fmt.Fprintf(&b, "# A %s was detected, but DVA does not guess a native\n", evidence.phrase(tmpl))
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
