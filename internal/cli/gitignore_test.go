package cli

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// TestIsDvaIgnored pins what "already ignored" means. This file did not exist before
// TASK-065, which is how the check shipped recognizing only a literal ".sb/dva" line: with
// no test encoding the intent, an ancestor rule like ".sb/" read as unignored and every
// command warned about a path git was already excluding.
func TestIsDvaIgnored(t *testing.T) {
	for _, tt := range []struct {
		name    string
		content string
		want    bool
	}{
		// The rule that was already recognized.
		{"exact path", ".sb/dva", true},
		{"exact path with trailing slash", ".sb/dva/", true},

		// The regression TASK-065 fixes: git excludes the whole subtree from an ancestor.
		{"ancestor directory", ".sb/", true},
		{"ancestor bare", ".sb", true},
		{"ancestor root-anchored", "/.sb/", true},
		{"exact path root-anchored", "/.sb/dva/", true},

		// Real .gitignore files are long and the rule is rarely first or last.
		{"ancestor among many rules", "node_modules/\n*.log\n.sb/\ndist/\n", true},
		{"indented line still counts", "  .sb/dva/  \n", true},

		// Must keep warning: nothing here excludes the directory.
		{"empty", "", false},
		{"unrelated rules only", "node_modules/\n*.log\ndist/\n", false},
		{"commented out", "# .sb/dva/\n", false},
		{"sibling directory", ".sbx/\n", false},
		{"shorter prefix that is not an ancestor", ".s/\n", false},

		// A rule *below* the dot dir ignores only that child, so the markers DVA writes
		// elsewhere in .sb/dva are still committable and the warning is correct.
		{"descendant only", ".sb/dva/cache/\n", false},

		// Negations, verified against real git with the paths on disk. Before these cases
		// the check returned on its first covering match and read every one of them as
		// ignored — suppressing the warning on the two that git does not ignore.
		{"ancestor negated after being excluded", ".sb/\n!.sb/\n", false},
		{"exact path negated after being excluded", ".sb/dva/\n!.sb/dva/\n", false},
		{"negation in another spelling still counts", "/.sb\n!.sb/\n", false},
		{"negation before the exclusion loses", "!.sb/\n.sb/\n", true},
		{"descendant negation cannot re-include", ".sb/\n!.sb/dva/\n", true},
		{"negation alone excludes nothing", "!.sb/\n", false},
		{"unrelated negation is not a match", ".sb/\n!dist/\n", true},

		// Not interpreted on purpose — documented as out of scope in TASK-065. Pinned so
		// the limitation is a decision on record rather than an accident.
		{"glob is not interpreted", ".sb/*\n", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := isDvaIgnored(tt.content); got != tt.want {
				t.Errorf("isDvaIgnored(%q) = %v, want %v", tt.content, got, tt.want)
			}
		})
	}
}

// TestIgnoreRulesCoveringIsDerivedFromDotDirName guards against the set being hand-written
// for today's ".sb/dva". If DotDirName ever gains or loses a segment, the rules must follow
// it — a stale set would silently warn (or stay silent) about the wrong path.
func TestIgnoreRulesCoveringIsDerivedFromDotDirName(t *testing.T) {
	rules := ignoreRulesCovering(config.DotDirName)
	if !rules[config.DotDirName] {
		t.Errorf("rules do not cover DotDirName %q itself: %v", config.DotDirName, rules)
	}

	// Every segment boundary must be represented, so a subtree rule at any depth is honored.
	for _, want := range []string{".sb", ".sb/", ".sb/dva", ".sb/dva/"} {
		if !rules[want] {
			t.Errorf("rules missing %q (DotDirName = %q): %v", want, config.DotDirName, rules)
		}
	}

	// A single-segment dir must not produce an empty-string rule, which would match every
	// blank line in a .gitignore and report any file as ignored.
	for _, dir := range []string{"tmp", "/tmp", ""} {
		if ignoreRulesCovering(dir)[""] {
			t.Errorf("ignoreRulesCovering(%q) contains the empty rule, which matches blank lines", dir)
		}
	}
}

// TestGitignoreWarningNeedsSomethingCommittable pins the gate TASK-080 added. loadConfig calls
// this on every command that reads a config, so before the gate `dva ls` in a fresh clone printed
// two lines of hygiene advice above its own answer — about a directory that did not exist, and
// that no read-only command creates (ls, show, validate, status and manifest were each measured
// leaving the tree untouched).
//
// Every case flips exactly one condition away from the warning case, so a gate that stopped
// consulting any single condition fails here instead of passing on the strength of the others.
// The name has to contain "Gitignore": the task's acceptance criterion runs `-run Gitignore`,
// which matches neither TestIsDvaIgnored nor TestIgnoreRulesCoveringIsDerivedFromDotDirName.
func TestGitignoreWarningNeedsSomethingCommittable(t *testing.T) {
	for _, tt := range []struct {
		name      string
		git       bool
		markers   bool
		gitignore string
		json      bool
		wantWarn  bool
	}{
		{name: "markers on disk and nothing ignores them", git: true, markers: true, wantWarn: true},
		{name: "no markers written yet", git: true, markers: false, wantWarn: false},
		{name: "markers but exactly ignored", git: true, markers: true, gitignore: ".sb/dva/\n", wantWarn: false},
		{name: "markers but an ancestor is ignored", git: true, markers: true, gitignore: ".sb/\n", wantWarn: false},
		{name: "markers but not a git working tree", git: false, markers: true, wantWarn: false},
		{name: "markers but json output has no schema for it", git: true, markers: true, json: true, wantWarn: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.git {
				if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
					t.Fatalf("MkdirAll .git: %v", err)
				}
			}
			if tt.markers {
				if err := os.MkdirAll(filepath.Join(dir, config.DotDirName), 0o755); err != nil {
					t.Fatalf("MkdirAll %s: %v", config.DotDirName, err)
				}
			}
			if tt.gitignore != "" {
				if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tt.gitignore), 0o644); err != nil {
					t.Fatalf("WriteFile .gitignore: %v", err)
				}
			}

			oldJSON := jsonOutput
			jsonOutput = tt.json
			defer func() { jsonOutput = oldJSON }()

			out := captureOutput(t, func() { checkGitignoreForWarning(dir) })

			if warned := strings.Contains(out, "is not in your .gitignore"); warned != tt.wantWarn {
				t.Errorf("warned = %v, want %v; output was %q", warned, tt.wantWarn, out)
			}
			// A warning that does not name the remedy is the noise this task is about. Asserted
			// only where one is expected, since elsewhere the absence is what passes.
			if tt.wantWarn && !strings.Contains(out, "dva doctor --fix") {
				t.Errorf("the warning must name the command that fixes it: %q", out)
			}
		})
	}
}

// initGitRepo makes dir a real repository, which the cases below need because they are about
// what git decides rather than about what a line of .gitignore looks like. The other tests in
// this file create a bare `.git` directory instead; that is enough for the existence gate and
// deliberately not enough for check-ignore, which is the difference the fallback test relies on.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	// The developer's own core.excludesFile is an input to check-ignore, so without this a
	// machine whose global excludes list `.sb/` would report every case here as ignored and
	// the negative cases — the ones that prove the check still warns — would pass vacuously.
	// git reads these two variables in place of the global and system config files.
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)

	cmd := exec.Command("git", "init", "--quiet")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}
}

// TestGitignoreCheckAsksGit pins the rules the literal reader cannot parse and git can.
//
// Every case here was measured against git first. The pair `.sb/dva/*` + `!.sb/dva/*.yml` is
// the one that sent this task: it is how a repository keeps DVA's modules tracked while
// discarding its transient state, and the literal reader saw no line naming `.sb/dva` in any
// of its spellings, so it warned on a correctly-configured tree. Adding glob cases to that
// reader would have closed this one spelling and left `**` anchoring and .git/info/exclude —
// gaps its own comment names — open. Asking git closes the class.
//
// The negative cases are the point of the exercise. A checker that stops warning is only an
// improvement if it still warns when a marker really is committable, so each one re-includes
// exactly one class of transient and must be caught.
func TestGitignoreCheckAsksGit(t *testing.T) {
	for _, tt := range []struct {
		name      string
		gitignore string
		exclude   string
		wantWarn  bool
	}{
		{
			// The pair this task exists for.
			name:      "contents excluded, modules re-included",
			gitignore: ".sb/dva/*\n!.sb/dva/*.yml\n",
			wantWarn:  false,
		},
		{
			name:      "contents excluded outright",
			gitignore: ".sb/dva/*\n",
			wantWarn:  false,
		},
		{
			// Non-root anchoring, the second gap isDvaIgnored's comment names.
			name:      "matched anywhere in the tree",
			gitignore: "**/.sb/dva/\n",
			wantWarn:  false,
		},
		{
			// The third gap: a rule git honours that never appears in .gitignore at all.
			name:     "ignored through .git/info/exclude",
			exclude:  ".sb/dva/\n",
			wantWarn: false,
		},
		{
			// Re-includes the provision markers. `*` does not match `/`, so this negation
			// applies at depth 1 where the markers live and they become committable —
			// while pids, logs and sources stay excluded. Precisely the half-correct
			// configuration a unanimous verdict exists to catch.
			name:      "provision markers re-included",
			gitignore: ".sb/dva/*\n!.sb/dva/provisioned-*\n",
			wantWarn:  true,
		},
		{
			name:      "unrelated rules only",
			gitignore: "node_modules/\n*.log\n",
			wantWarn:  true,
		},
		{
			name:     "no rules at all",
			wantWarn: true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			initGitRepo(t, dir)
			if err := os.MkdirAll(filepath.Join(dir, config.DotDirName), 0o755); err != nil {
				t.Fatalf("MkdirAll %s: %v", config.DotDirName, err)
			}
			if tt.gitignore != "" {
				if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tt.gitignore), 0o644); err != nil {
					t.Fatalf("WriteFile .gitignore: %v", err)
				}
			}
			if tt.exclude != "" {
				excludePath := filepath.Join(dir, ".git", "info", "exclude")
				if err := os.MkdirAll(filepath.Dir(excludePath), 0o755); err != nil {
					t.Fatalf("MkdirAll info: %v", err)
				}
				if err := os.WriteFile(excludePath, []byte(tt.exclude), 0o644); err != nil {
					t.Fatalf("WriteFile exclude: %v", err)
				}
			}

			out := captureOutput(t, func() { checkGitignoreForWarning(dir) })

			if warned := strings.Contains(out, "is not in your .gitignore"); warned != tt.wantWarn {
				t.Errorf("warned = %v, want %v; output was %q", warned, tt.wantWarn, out)
			}
		})
	}
}

// TestGitignoreFallsBackWhenGitCannotAnswer pins the distinction between "not ignored" and
// "unanswerable". check-ignore exits 1 for the first and 128 for the second — a `.git` that
// points at a gitdir which has moved, among others — and folding them together would warn
// about repositories whose configuration was never read.
//
// The fallback is the literal reader, so a plainly-spelled rule still suppresses the warning
// where git is missing entirely.
func TestGitignoreFallsBackWhenGitCannotAnswer(t *testing.T) {
	for _, tt := range []struct {
		name      string
		gitignore string
		wantWarn  bool
	}{
		{"literal reader still recognizes its own spelling", ".sb/dva/\n", false},
		{"and still warns on what it cannot read", ".sb/dva/*\n!.sb/dva/*.yml\n", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
				t.Fatalf("MkdirAll .git: %v", err)
			}
			if err := os.MkdirAll(filepath.Join(dir, config.DotDirName), 0o755); err != nil {
				t.Fatalf("MkdirAll %s: %v", config.DotDirName, err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tt.gitignore), 0o644); err != nil {
				t.Fatalf("WriteFile .gitignore: %v", err)
			}

			old := gitCheckIgnore
			gitCheckIgnore = func(string, []string) (map[string]string, bool) { return nil, false }
			defer func() { gitCheckIgnore = old }()

			out := captureOutput(t, func() { checkGitignoreForWarning(dir) })

			if warned := strings.Contains(out, "is not in your .gitignore"); warned != tt.wantWarn {
				t.Errorf("warned = %v, want %v; output was %q", warned, tt.wantWarn, out)
			}
		})
	}
}

// TestGitCheckIgnoreGivesUpOnAStalledGit pins the deadline, which is the one property of
// gitCheckIgnore that the gitCheckIgnore swap used everywhere else in this file cannot reach:
// every other test replaces the function, and the timeout lives inside it.
//
// The hazard is not hypothetical. Before the deadline, a git that never returned took the whole
// command with it — `dva ls` against a check-ignore that sleeps ran until it was killed from
// outside, having printed nothing. The fork is on the hot path, ahead of the output the user
// asked for, so "one slow git" and "dva is hung" were the same observation.
//
// Timing out has to land as undecided rather than as a verdict. `decided == false` is what sends
// every caller back to reading .gitignore literally; returning "nothing is ignored" instead
// would warn about repositories that are configured correctly, which is the failure this whole
// file is built to avoid.
func TestGitCheckIgnoreGivesUpOnAStalledGit(t *testing.T) {
	// A git that answers nothing, ever.
	shimDir := t.TempDir()
	shim := "#!/bin/sh\nif [ \"$1\" = check-ignore ]; then sleep 600; fi\nexit 1\n"
	if err := os.WriteFile(filepath.Join(shimDir, "git"), []byte(shim), 0o755); err != nil {
		t.Fatalf("WriteFile git shim: %v", err)
	}
	t.Setenv("PATH", shimDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Shrunk so the test spends milliseconds proving what production spends seconds enforcing.
	// Both the deadline and WaitDelay read this, so the wait is bounded at twice it.
	old := gitCheckIgnoreTimeout
	gitCheckIgnoreTimeout = 150 * time.Millisecond
	defer func() { gitCheckIgnoreTimeout = old }()

	start := time.Now()
	sources, decided := gitCheckIgnore(t.TempDir(), dvaTransientProbes())
	elapsed := time.Since(start)

	if decided {
		t.Errorf("decided = true, want false: a git that never answered was read as a verdict")
	}
	if sources != nil {
		t.Errorf("sources = %v, want nil", sources)
	}

	// Both bounds are load-bearing, and the lower one is the less obvious of the two.
	//
	// Every assertion above is also satisfied by a run where the shim was never reached: a real
	// git in a temp directory that is not a repository exits 128, which is likewise nil and
	// undecided, and it does so in about 15ms. So the test would keep passing if the PATH
	// injection above ever stopped taking effect, having stopped testing the deadline entirely.
	// Waiting at least the full budget is what distinguishes "gave up on a stalled git" from
	// "asked a healthy git a question it answered immediately".
	if elapsed < gitCheckIgnoreTimeout {
		t.Errorf("returned after %v, before the %v deadline could expire: the stalled git was "+
			"never reached, so this asserted nothing about the timeout", elapsed, gitCheckIgnoreTimeout)
	}
	// Generous against a loaded CI box while still failing outright if the bound is gone,
	// since the shim sleeps for ten minutes.
	if limit := 30 * time.Second; elapsed > limit {
		t.Errorf("took %v, want under %v: the deadline did not bound the call", elapsed, limit)
	}
}

// TestDvaTransientProbesCoverEveryWriter guards the list against silently shrinking. The
// verdict is unanimous over these four paths, so a probe dropped here does not fail anything —
// it widens the set of configurations that pass, which is the direction that goes unnoticed.
//
// Each expectation is rebuilt from the constant its writer uses rather than spelled out, so a
// renamed directory moves both sides together and this test keeps testing the same thing.
func TestDvaTransientProbesCoverEveryWriter(t *testing.T) {
	probes := dvaTransientProbes()

	for _, want := range []struct {
		what string
		path string
	}{
		{"pid files", path.Join(config.DotDirName, config.PidsDirName, "probe.pid")},
		{"log files", path.Join(config.DotDirName, config.LogsDirName, "probe.log")},
		{"git source clones", path.Join(config.DotDirName, config.SourcesDirName, "probe")},
		{"provision markers", path.Join(config.DotDirName, provisionMarkerName("probe"))},
	} {
		if !slices.Contains(probes, want.path) {
			t.Errorf("no probe for %s: %q missing from %q", want.what, want.path, probes)
		}
	}

	// Forward slashes even on Windows: these go to git, which uses them everywhere.
	for _, probe := range probes {
		if strings.Contains(probe, `\`) {
			t.Errorf("probe %q must be spelled with forward slashes for git", probe)
		}
	}
}

// TestGitignoreSourceIsShared pins which ignore files travel with a clone.
//
// The spellings are not invented here; each was read off `git check-ignore -v` on a real
// repository. That matters most for the excludes-file cases: git prints core.excludesFile as
// configured, and the usual configuration is an absolute path — which is the only thing
// separating "~/.gitignore, one developer's preference" from ".gitignore, the project's".
func TestGitignoreSourceIsShared(t *testing.T) {
	for _, tt := range []struct {
		name   string
		source string
		want   bool
	}{
		{"the repository's own file", ".gitignore", true},
		{"a nested one further down the tree", "web/.gitignore", true},
		{"one above the config dir", "../.gitignore", true},

		{"this clone only", ".git/info/exclude", false},
		{"this clone, spelled from a subdirectory", "../.git/info/exclude", false},
		{"this machine only", "/Users/someone/.gitignore", false},
		{"this machine, XDG spelling", "/Users/someone/.config/git/ignore", false},

		// check-ignore leaves the source empty for a path no rule matched; the map lookup
		// already rejects those, and a nil string must not read as the repository's file.
		{"no source at all", "", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := gitignoreSourceIsShared(tt.source); got != tt.want {
				t.Errorf("gitignoreSourceIsShared(%q) = %v, want %v", tt.source, got, tt.want)
			}
		})
	}
}

// TestEnsureGitignoreWritesWhatACloneWouldLack is the regression the shared/non-shared split
// exists for.
//
// Once the check started asking git, the writer inherited the wide answer — and the wide answer
// includes rules that no collaborator receives. On the machine with `.sb/` in its global
// excludes file, `dva doctor --fix` reported the row fixed and wrote nothing; the committed
// .gitignore stayed silent about DVA and every other clone committed pid files and logs. The
// symptom is invisible precisely where the fix is run, which is why it needs a test rather than
// a look.
//
// .git/info/exclude stands in for core.excludesFile here: it is per-clone rather than
// per-machine, so it is equally not shared, and it needs no global git configuration to set up.
func TestEnsureGitignoreWritesWhatACloneWouldLack(t *testing.T) {
	for _, tt := range []struct {
		name         string
		gitignore    string // "" means the file does not exist
		exclude      string
		wantAppended bool // .gitignore must have gained the rule it did not carry
	}{
		{
			// The defect, in its two shapes: with and without a .gitignore to append to.
			name:         "ignored only per-clone, no .gitignore yet",
			exclude:      ".sb/\n",
			wantAppended: true,
		},
		{
			name:         "ignored only per-clone, .gitignore exists",
			gitignore:    "node_modules/\n",
			exclude:      ".sb/\n",
			wantAppended: true,
		},
		{
			// The repository already says it, in a spelling only git can read. Writing here
			// would be the duplicate-rule noise that asking git was meant to stop, and it is
			// why the writer asks git at all rather than reading the file itself.
			name:      "repository declares it with a glob",
			gitignore: ".sb/dva/*\n!.sb/dva/*.yml\n",
		},
		{
			// Both the repository's file and the per-clone one cover it. The shared source
			// is enough on its own, so nothing is appended.
			name:      "declared in both places",
			gitignore: ".sb/\n",
			exclude:   ".sb/\n",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			initGitRepo(t, dir)
			gitignorePath := filepath.Join(dir, ".gitignore")
			if tt.gitignore != "" {
				if err := os.WriteFile(gitignorePath, []byte(tt.gitignore), 0o644); err != nil {
					t.Fatalf("WriteFile .gitignore: %v", err)
				}
			}
			if tt.exclude != "" {
				excludePath := filepath.Join(dir, ".git", "info", "exclude")
				if err := os.MkdirAll(filepath.Dir(excludePath), 0o755); err != nil {
					t.Fatalf("MkdirAll info: %v", err)
				}
				if err := os.WriteFile(excludePath, []byte(tt.exclude), 0o644); err != nil {
					t.Fatalf("WriteFile exclude: %v", err)
				}
			}

			wrote, err := ensureGitignore(dir)
			if err != nil {
				t.Fatalf("ensureGitignore: %v", err)
			}
			// All three callers turn this bool straight into "📎 Updated .gitignore", so a
			// true here that no write backs up is a message about an edit that never
			// happened — the one thing a user reading init's output cannot check.
			if wrote != tt.wantAppended {
				t.Errorf("ensureGitignore reported wrote = %v, want %v", wrote, tt.wantAppended)
			}

			after, err := os.ReadFile(gitignorePath)
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("ReadFile .gitignore: %v", err)
			}
			// Whether the file changed, not whether the rule is in it: two of these cases
			// start out already declaring it, so "the rule is there" is satisfied by doing
			// nothing and would not tell a working writer from a silent one.
			if appended := string(after) != tt.gitignore; appended != tt.wantAppended {
				t.Errorf("appended = %v, want %v; %q became %q", appended, tt.wantAppended, tt.gitignore, after)
			}
			if tt.wantAppended {
				// A clone gets this file and nothing else, so the literal reader — which
				// also sees nothing else — is the right judge of what that clone would know.
				if !isDvaIgnored(string(after)) {
					t.Errorf("something was written but a clone still would not ignore the state: %q", after)
				}
				// Appending must not cost the rules that were already there.
				if tt.gitignore != "" && !strings.Contains(string(after), strings.TrimSpace(tt.gitignore)) {
					t.Errorf("the existing rules were lost: %q became %q", tt.gitignore, after)
				}
			}
		})
	}
}

// TestGitignoreStatusNamesTheRightRemedy pins doctor's wording to what is on disk.
//
// Asking git had to happen before reading the file — git applies rules the file does not carry,
// so reading first calls a correctly-configured tree broken. But moving the question up left the
// "no .gitignore here" branch behind the read, unreachable in any real repository, and doctor
// began telling first-run users to add a line to a file that does not exist. That is the
// `dva init` → `dva doctor` path, so it is the first thing a new user sees.
func TestGitignoreStatusNamesTheRightRemedy(t *testing.T) {
	for _, tt := range []struct {
		name        string
		gitignore   string // "" means the file does not exist
		wantPass    bool
		wantFinding string
	}{
		{name: "no .gitignore at all", wantFinding: "no .gitignore here"},
		{name: "a .gitignore that says nothing about it", gitignore: "node_modules/\n", wantFinding: "is NOT ignored"},
		{name: "ignored through a glob git understands", gitignore: ".sb/dva/*\n", wantPass: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			initGitRepo(t, dir)
			if tt.gitignore != "" {
				if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(tt.gitignore), 0o644); err != nil {
					t.Fatalf("WriteFile .gitignore: %v", err)
				}
			}

			r := checkGitignoreStatus(dir)

			if r.Passed != tt.wantPass {
				t.Fatalf("Passed = %v, want %v (finding %q)", r.Passed, tt.wantPass, r.Finding)
			}
			if tt.wantPass {
				return
			}
			if !strings.Contains(r.Finding, tt.wantFinding) {
				t.Errorf("Finding = %q, want it to contain %q", r.Finding, tt.wantFinding)
			}
			// The hint is the half that was wrong: it named a file to edit without saying
			// to create it. Both halves have to agree about what exists.
			wantHint := "Add '"
			if tt.gitignore == "" {
				wantHint = "Create .gitignore"
			}
			if !strings.Contains(r.FixHint, wantHint) {
				t.Errorf("FixHint = %q, want it to contain %q", r.FixHint, wantHint)
			}
			if !r.Fixable || r.fixFunc == nil {
				t.Errorf("the row must stay fixable: Fixable = %v, fixFunc set = %v", r.Fixable, r.fixFunc != nil)
			}
		})
	}
}
