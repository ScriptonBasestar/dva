package cli

import (
	"fmt"
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

		// One glob form is read, and only this one: the contents-exclusion that
		// defaultIgnoreRules writes. Excluding a directory's contents puts the transients
		// out of reach exactly as excluding the directory does, so this predicate answers
		// both alike — at the ancestor as well as at the dot directory itself.
		//
		// It is read because DVA writes it. Without these cases ensureGitignore would
		// consult this reader, fail to see the block it had just written, and append the
		// same two lines on every run of `dva init` on a machine where git cannot be asked.
		{"contents glob at the dot dir", ".sb/dva/*\n", true},
		{"contents glob at an ancestor", ".sb/*\n", true},
		{"contents glob root-anchored", "/.sb/dva/*\n", true},
		{"the block DVA writes", ".sb/dva/*\n!.sb/dva/*.yml\n", true},

		// Reading the contents form means the negations beside it have to be read too, and
		// these are the cases that say why. Each was measured against real git over
		// dvaTransientProbes():
		//
		//	.sb/*      + !.sb/dva     → git ignores 0 of 4
		//	.sb/dva/*  + !.sb/dva/p*  → git ignores 2 of 4 (pid and marker committable)
		//	.sb/       + !.sb/d*      → git ignores 4 of 4
		//
		// The last one is the contrast that matters. An unread negation under a *directory*
		// exclusion is inert — git does not descend, so it cannot change the answer, which
		// is why this reader was safe ignoring globs before it read any. Under a *contents*
		// exclusion the directory stays live and the same unread negation decides real
		// paths. Answering "ignored" there is not a missing warning, it is a pid file in
		// the commit. So a negation that could reach a probe forfeits the contents form.
		{"negation re-including the dot dir from an ancestor", ".sb/*\n!.sb/dva\n", false},
		{"negation narrower than the contents exclusion", ".sb/dva/*\n!.sb/dva/p*\n", false},
		{"negation naming a marker class", ".sb/dva/*\n!.sb/dva/provisioned-*\n", false},
		{"contents glob negated after being excluded", ".sb/dva/*\n!.sb/dva/*\n", false},

		// A negation before the exclusion has already lost to it — last-matching-pattern-
		// wins is the whole rule — so it is not scanned and does not forfeit. This case
		// read false while the forfeit scanned the entire file, on the argument that one
		// extra warning on a file nobody writes was cheaper than ordering logic. The cost
		// turned out not to be a warning: a `.gitignore` opening with `!*` and carrying
		// DVA's own block below it is ignored by git and was read as unignored here, so
		// ensureGitignore appended the block again on every run.
		{"a negation before the exclusion has already lost", "!.sb/dva/*\n.sb/dva/*\n", true},
		{"a file-wide negation above DVA's own block", "!*\n.sb/dva/*\n!.sb/dva/*.yml\n", true},

		// The same rule at the directory spelling. `.sb/` is not the end of the story when a
		// later negation names `.sb` itself at a level git still reaches: measured, `!*`
		// leaves all four probes addable, while `!.sb/d*` — naming only something inside the
		// excluded directory — leaves all four ignored.
		{"file-wide negation after a directory exclusion", ".sb/\n!*\n", false},
		{"file-wide negation after the dot dir exclusion", ".sb/dva/\n!*\n", false},

		// Directory exclusion outranks a negation on its contents: git does not descend
		// into an excluded directory, so the negation never applies. These two assert
		// git's behaviour, which was measured; they do not pin the order the two branches
		// are consulted in, and an earlier version of this comment claimed they did.
		// Swapping the branches was tried and the package still passes — they are
		// independent, each decisive on its own. What makes the contents branch decline
		// here is lastMatchExcludes, not the order.
		{"directory exclusion beats a contents negation", ".sb/dva/\n!.sb/dva/*\n", true},
		{"directory exclusion beats a negation on the directory's children", ".sb/\n!.sb/d*\n", true},

		// Still not interpreted — pinned so the remaining limits stay a decision on record
		// rather than an accident. Each makes DVA warn about a path git does ignore, which
		// is the harmless direction.
		{"other globs are not interpreted", ".sb/d*\n", false},
		{"double-star anchoring is not interpreted", "**/.sb/\n", false},
		{"double-star in a negation forfeits rather than being guessed at", ".sb/dva/*\n!**/probe.pid\n", false},
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
//
// The block DVA writes belongs in the quiet half, and it did not always: `.sb/dva/*` +
// `!.sb/dva/*.yml` stood here as the example of a rule the reader could not parse, back when
// DVA wrote `.sb/dva/` instead. Once DVA writes the glob — it must, or its own modules become
// unaddable — a reader blind to it would warn about the rule DVA had just written, and
// ensureGitignore would append that rule again on every run. So the case moved sides, and
// something the reader genuinely cannot parse took its place to keep the other half honest.
// TestEnsureGitignoreIsIdempotentWithoutGit is the test the forfeit rule needs, and the reason
// it exists is that the forfeit rule is where this can go wrong without anyone noticing.
//
// ensureGitignore consults isDvaIgnored when git cannot answer. If the reader declines to
// recognize the block DVA just wrote, DVA appends it again — every run, forever — and nothing
// else in the package fails. The first version of the forfeit did exactly that for a `.gitignore`
// containing `!*.log`, which is an ordinary thing to have beside a `logs/` rule: it matched the
// log probe's basename, even though `.sb/dva/logs` is excluded and git never descends to it.
//
// The unrelated-negation cases are the point. Each is a negation that has nothing to do with
// DVA, and a reader that forfeits on them writes three copies of its block after three runs.
func TestEnsureGitignoreIsIdempotentWithoutGit(t *testing.T) {
	for _, pre := range []string{
		"",
		"node_modules/\ndist/\n",
		"*.log\n!important.log\n",
		"build/\n!build/keep/\n",
		"logs/\n!*.log\n",
		"*.pid\n!probe\n",
		"!*\n",
		"secrets/\n!secrets/example.yml\n",
	} {
		t.Run(fmt.Sprintf("%q", pre), func(t *testing.T) {
			dir := t.TempDir()
			gitignorePath := filepath.Join(dir, ".gitignore")
			if pre != "" {
				if err := os.WriteFile(gitignorePath, []byte(pre), 0o644); err != nil {
					t.Fatalf("WriteFile .gitignore: %v", err)
				}
			}

			old := gitCheckIgnore
			gitCheckIgnore = func(string, []string) (map[string]string, bool) { return nil, false }
			defer func() { gitCheckIgnore = old }()

			var wrote []bool
			for range 3 {
				w, err := ensureGitignore(dir)
				if err != nil {
					t.Fatalf("ensureGitignore: %v", err)
				}
				wrote = append(wrote, w)
			}

			after, err := os.ReadFile(gitignorePath)
			if err != nil {
				t.Fatalf("ReadFile .gitignore: %v", err)
			}
			if blocks := strings.Count(string(after), defaultIgnoreSection); blocks != 1 {
				t.Errorf("wrote the block %d times over three runs (wrote=%v); .gitignore is now %q", blocks, wrote, after)
			}
			if wrote[1] || wrote[2] {
				t.Errorf("wrote = %v, want only the first run to write", wrote)
			}
		})
	}
}

func TestGitignoreFallsBackWhenGitCannotAnswer(t *testing.T) {
	for _, tt := range []struct {
		name      string
		gitignore string
		wantWarn  bool
	}{
		{"literal reader still recognizes its own spelling", ".sb/dva/\n", false},
		{"and the block DVA writes", ".sb/dva/*\n!.sb/dva/*.yml\n", false},
		{"and still warns on what it cannot read", "**/.sb/\n", true},
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

// TestDvaTransientProbesCoverEveryWriter pins the list to the writers in both directions.
//
// A dropped probe is the direction that goes unnoticed: the verdict is unanimous over these
// paths, so losing one fails nothing — it widens the set of configurations that pass.
//
// An added or altered probe is the direction this test used to miss entirely, because it asked
// only whether each expected path was present. Both are now errors, and the second matters as
// much as the first here: a probe is a claim that some writer produces paths of that shape, and
// defaultIgnoreRules rests on that claim being exhaustive — `!.sb/dva/*.yml` is safe only
// because no writer puts a `.yml` at that level. A probe nobody writes makes the list say
// something about DVA that is not true, and a rule derived from it inherits the untruth.
//
// Each expectation is rebuilt from the constant its writer uses rather than spelled out, so a
// renamed directory moves both sides together and this test keeps testing the same thing.
func TestDvaTransientProbesCoverEveryWriter(t *testing.T) {
	probes := dvaTransientProbes()

	writers := []struct {
		what string
		path string
	}{
		{"pid files", path.Join(config.DotDirName, config.PidsDirName, "probe.pid")},
		{"log files", path.Join(config.DotDirName, config.LogsDirName, "probe.log")},
		{"git source clones", path.Join(config.DotDirName, config.SourcesDirName, "probe")},
		{"provision markers", path.Join(config.DotDirName, provisionMarkerName("probe"))},
	}

	expected := make(map[string]string, len(writers))
	for _, want := range writers {
		expected[want.path] = want.what
		if !slices.Contains(probes, want.path) {
			t.Errorf("no probe for %s: %q missing from %q", want.what, want.path, probes)
		}
	}

	// The other direction. A probe here that no writer above accounts for is either a real
	// writer this test has not learned about — in which case the entry belongs in `writers`
	// with the constant it comes from — or a path nothing produces, which leaves the checker
	// asking git about a shape that cannot occur.
	seen := make(map[string]bool, len(probes))
	for _, probe := range probes {
		if _, ok := expected[probe]; !ok {
			t.Errorf("probe %q matches no known writer; add the writer to this test or drop "+
				"the probe", probe)
		}
		// Duplicates would not change the unanimous verdict, so nothing else would catch
		// one; it is still a list that has stopped meaning one entry per class.
		if seen[probe] {
			t.Errorf("probe %q listed twice in %q", probe, probes)
		}
		seen[probe] = true
	}

	if len(probes) != len(writers) {
		t.Errorf("dvaTransientProbes returned %d probes for %d writers: %q",
			len(probes), len(writers), probes)
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

// TestEnsureGitignoreKeepsModulesAddable is the acceptance test for what DVA writes, and it
// asks git rather than inspecting the text, because the defect it covers was invisible in the
// text. The old rule, `.sb/dva/`, reads as an obviously correct way to discard a scratch
// directory. What it actually does is exclude the directory, and git does not descend into an
// excluded directory — so `.sb/dva/gates.yml`, which is hand-authored configuration DVA reads
// and expects in the commit, could not be added without `-f`. DVA's own advice disabled DVA's
// modules feature, and `dva doctor --fix` applied it automatically.
//
// The failure had a shape that kept it quiet. A repository that committed its first module
// before running the fix kept it, because git does not apply .gitignore to tracked files; only
// the *next* module was refused, in a `git add` that reported nothing and exited 0. So the
// two halves are asserted separately below: transients ignored is the rule's purpose, modules
// addable is the thing it must not cost.
func TestEnsureGitignoreKeepsModulesAddable(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)

	if _, err := ensureGitignore(dir); err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}

	// check-ignore needs nothing on disk, but `git add` does, so the module is written for
	// real — the refusal this test is about is one `git add` reports, not one a pattern match
	// predicts.
	modulePath := filepath.Join(dir, config.DotDirName, "gates.yml")
	if err := os.MkdirAll(filepath.Dir(modulePath), 0o755); err != nil {
		t.Fatalf("MkdirAll module dir: %v", err)
	}
	if err := os.WriteFile(modulePath, []byte("checks: {}\n"), 0o644); err != nil {
		t.Fatalf("WriteFile module: %v", err)
	}

	sources, decided := gitCheckIgnore(dir, dvaTransientProbes())
	if !decided {
		t.Fatalf("git could not answer in a repository this test created")
	}
	for _, probe := range dvaTransientProbes() {
		if _, ok := sources[probe]; !ok {
			gitignore, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
			t.Errorf("%q is not ignored under the rule DVA wrote:\n%s", probe, gitignore)
		}
	}

	// The half the old rule failed. `git add` on an ignored path without `-f` fails, and that
	// is the exact command a person adding a module runs.
	modulePathInRepo := path.Join(config.DotDirName, "gates.yml")
	if ignored, _ := gitCheckIgnore(dir, []string{modulePathInRepo}); len(ignored) != 0 {
		gitignore, _ := os.ReadFile(filepath.Join(dir, ".gitignore"))
		t.Errorf("%q is ignored, so `git add` refuses it without -f; DVA reads it as "+
			"configuration. Rule written:\n%s", modulePathInRepo, gitignore)
	}

	add := exec.Command("git", "add", modulePathInRepo)
	add.Dir = dir
	if out, err := add.CombinedOutput(); err != nil {
		t.Errorf("git add %q: %v: %s", modulePathInRepo, err, out)
	}

	// Writing is not enough; the writer must also recognize what it wrote. Without the
	// contents-glob form in the literal reader this second call appends the block again,
	// and does so on every subsequent `dva init` in any tree where git cannot be asked.
	before, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile .gitignore: %v", err)
	}
	wrote, err := ensureGitignore(dir)
	if err != nil {
		t.Fatalf("ensureGitignore (second call): %v", err)
	}
	after, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile .gitignore: %v", err)
	}
	if wrote || string(after) != string(before) {
		t.Errorf("second ensureGitignore wrote = %v and left %q, want no change from %q",
			wrote, after, before)
	}
	// The same question with git taken away, which is the path that actually regressed:
	// dvaRepoAlreadyDeclares consults git first and would mask a reader that cannot see the
	// rule. isDvaIgnored is that reader, asked directly.
	if !isDvaIgnored(string(after)) {
		t.Errorf("the literal reader does not recognize the block DVA wrote, so a tree "+
			"without git re-appends it every run:\n%s", after)
	}
}

// TestGitignoreStatusNamesTheRightRemedy pins doctor's wording to what is on disk.
//
// Asking git had to happen before reading the file — git applies rules the file does not carry,
// so reading first calls a correctly-configured tree broken. But moving the question up left the
// "no .gitignore here" branch behind the read, unreachable in any real repository, and doctor
// began telling first-run users to add a line to a file that does not exist. That is the
// `dva init` → `dva doctor` path, so it is the first thing a new user sees.
// TestAdviceSpellsExactlyWhatDvaWrites pins the coupling this whole change exists to establish.
//
// The defect being closed is not that one spelling was wrong. It is that two spellings existed:
// what `dva init` wrote and what the messages told a person to write were separate strings, free
// to disagree, and they did. defaultIgnoreAdvice reads defaultIgnoreRules for that reason — but
// "reads it" is a property of the current source text, and nothing failed when that property was
// removed. Restating the old literal here compiled, passed the package, and put DVA back to
// recommending a rule that blocks its own modules while writing one that does not.
//
// So the assertion is on the relationship, not on either wording: whatever the rules are, the
// advice names all of them and invents nothing.
func TestAdviceSpellsExactlyWhatDvaWrites(t *testing.T) {
	rules := defaultIgnoreRules()
	advice := defaultIgnoreAdvice()

	if len(rules) == 0 {
		t.Fatalf("defaultIgnoreRules() is empty; there is no rule to advise")
	}
	for _, rule := range rules {
		if !strings.Contains(advice, "'"+rule+"'") {
			t.Errorf("advice %q does not name the rule %q that DVA writes", advice, rule)
		}
	}

	// And nothing else. Quoted runs in the advice are what a reader copies into .gitignore,
	// so an extra one is a recommendation DVA does not follow — the same disagreement in the
	// other direction.
	var quoted []string
	for i, part := range strings.Split(advice, "'") {
		if i%2 == 1 {
			quoted = append(quoted, part)
		}
	}
	if !slices.Equal(quoted, rules) {
		t.Errorf("advice names %q, want exactly the rules %q", quoted, rules)
	}

	// The advice is only worth pinning because it reaches people. Every user-facing string
	// that tells someone what to put in .gitignore has to be built from it rather than
	// spelling a rule of its own.
	for _, site := range []struct{ what, text string }{
		{"doctor's hint when .gitignore is missing", checkGitignoreStatus(t.TempDir()).FixHint},
	} {
		if !strings.Contains(site.text, advice) {
			t.Errorf("%s reads %q, want it to contain the advice %q", site.what, site.text, advice)
		}
	}
}

// TestDoctorReportsTheRuleThatBlocksModules covers the population this change would otherwise
// miss entirely: a repository that already ran `dva init` under the old advice and carries
// `.sb/dva/` today. Changing what DVA writes does nothing for it — ensureGitignore sees the
// transients ignored and correctly declines to append — so without a second question the
// repository is left with `git add` refusing its modules while doctor prints a pass.
//
// The assertions are paired on purpose. That the transients really are ignored is what makes
// the case hard: any check that only asks the original question has already answered "healthy"
// by the time this one is reached. And the git commands are real rather than faked, because the
// claim under test is about what git does with a spelling, which a stub would only restate.
func TestDoctorReportsTheRuleThatBlocksModules(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(defaultIgnoreSection+"\n"+config.DotDirName+"/\n"), 0o644); err != nil {
		t.Fatalf("WriteFile .gitignore: %v", err)
	}

	if ignored, decided := dvaTransientsIgnored(dir); !decided || !ignored {
		t.Fatalf("the premise of this test is that the old rule does ignore transients: ignored=%v decided=%v", ignored, decided)
	}

	// The user-visible symptom, from git rather than from a predicate of ours.
	if err := os.MkdirAll(filepath.Join(dir, config.DotDirName), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	module := filepath.Join(config.DotDirName, "gates.yml")
	if err := os.WriteFile(filepath.Join(dir, module), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile module: %v", err)
	}
	add := exec.Command("git", "add", module)
	add.Dir = dir
	if out, err := add.CombinedOutput(); err == nil {
		t.Fatalf("premise failed: git add %s succeeded under the old rule: %s", module, out)
	}

	r := checkGitignoreStatus(dir)
	if r.Passed {
		t.Fatalf("doctor passed a repository whose modules git refuses to add")
	}
	if !strings.Contains(r.Finding, "modules") {
		t.Errorf("Finding = %q, want it to name modules as what is blocked", r.Finding)
	}
	if !strings.Contains(r.FixHint, defaultIgnoreAdvice()) {
		t.Errorf("FixHint = %q, want it to name the rule to write instead (%s)", r.FixHint, defaultIgnoreAdvice())
	}
	// Not fixable, and that is the finding rather than an omission. Appending the correct
	// rules below `.sb/dva/` changes nothing — git does not descend into an excluded
	// directory — so an automatic fix here would report success, leave the finding standing,
	// and append the same block again on the next run.
	if r.Fixable || r.fixFunc != nil {
		t.Errorf("the row must not claim to be auto-fixable: Fixable = %v, fixFunc set = %v", r.Fixable, r.fixFunc != nil)
	}
	if wrote, err := ensureGitignore(dir); err != nil || wrote {
		t.Errorf("ensureGitignore(dir) = %v, %v; it must not append under an exclusion that makes the append inert", wrote, err)
	}

	// And the rule this change writes is the one that answers both questions.
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(defaultIgnoreSection+"\n"+strings.Join(defaultIgnoreRules(), "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile .gitignore: %v", err)
	}
	if fixed := checkGitignoreStatus(dir); !fixed.Passed {
		t.Errorf("doctor still fails after the recommended edit: %q", fixed.Finding)
	}
}

// TestTransientPathspecsAndProbesNameTheSameClasses is the coupling test between the two lists,
// and it is here because they will drift otherwise: a new writer under the dot directory gets a
// probe, because the ignore check fails loudly without one, and gets no pathspec, because the
// tracked check stays quiet either way. Quiet is the failure mode worth a test.
//
// Exactly one, in both directions. A probe matched by two pathspecs means the classes overlap
// and a finding would name the same state twice; a pathspec matched by no probe means it is
// asking about something no writer produces, which is how a list goes stale without failing.
func TestTransientPathspecsAndProbesNameTheSameClasses(t *testing.T) {
	specs := dvaTransientPathspecs()
	hits := make(map[string]int, len(specs))

	for _, probe := range dvaTransientProbes() {
		var matched []string
		for _, spec := range specs {
			glob, err := path.Match(spec, probe)
			if err != nil {
				t.Fatalf("pathspec %q is not a valid pattern: %v", spec, err)
			}
			if glob || strings.HasPrefix(probe, spec+"/") {
				matched = append(matched, spec)
				hits[spec]++
			}
		}
		if len(matched) != 1 {
			t.Errorf("probe %q is covered by %d pathspecs (%v), want exactly 1", probe, len(matched), matched)
		}
	}

	for _, spec := range specs {
		if hits[spec] == 0 {
			t.Errorf("pathspec %q covers no probe, so nothing keeps it pointed at a real writer", spec)
		}
	}
}

// TestDoctorReportsTransientStateAlreadyCommitted uses a real repository because the claim is
// about git: that a tracked file goes on being committed no matter how correct .gitignore is.
// A stub restating that would prove only that the stub was written to agree.
//
// The premise is the whole point. The rules in force are the ones DVA writes, and the ignore row
// passes — so this repository is healthy by every other measure in the package while a pid file
// rides along in every commit.
func TestDoctorReportsTransientStateAlreadyCommitted(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(defaultIgnoreSection+"\n"+strings.Join(defaultIgnoreRules(), "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile .gitignore: %v", err)
	}
	pid := filepath.Join(config.DotDirName, config.PidsDirName, "web.pid")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, pid)), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, pid), []byte("4242\n"), 0o644); err != nil {
		t.Fatalf("WriteFile pid: %v", err)
	}
	// -f is how it gets there in life too: someone committed it before the rule existed, or
	// forced it past the rule once.
	add := exec.Command("git", "add", "-f", pid)
	add.Dir = dir
	if out, err := add.CombinedOutput(); err != nil {
		t.Fatalf("git add -f %s: %v: %s", pid, err, out)
	}

	if ignored, decided := dvaTransientsIgnored(dir); !decided || !ignored {
		t.Fatalf("premise failed: the rules here must be the correct ones (ignored=%v decided=%v)", ignored, decided)
	}
	if r := checkGitignoreStatus(dir); !r.Passed {
		t.Fatalf("premise failed: the ignore row must pass, or this repository is not the hard case: %q", r.Finding)
	}

	r, answered := checkTrackedTransients(dir)
	if !answered {
		t.Fatalf("the row was omitted inside a real repository with git available")
	}
	if r.Passed {
		t.Fatalf("doctor passed a repository that commits a pid file on every change")
	}
	if !strings.Contains(r.Finding, path.Join(config.DotDirName, config.PidsDirName)) {
		t.Errorf("Finding = %q, want it to name the class that is tracked", r.Finding)
	}
	if !strings.Contains(r.FixHint, "git rm") {
		t.Errorf("FixHint = %q, want it to name the command that untracks", r.FixHint)
	}
	// Untracking is a commit someone has to mean to make, so the row must not offer to do it.
	if r.Fixable || r.fixFunc != nil {
		t.Errorf("the row must not claim to be auto-fixable: Fixable = %v, fixFunc set = %v", r.Fixable, r.fixFunc != nil)
	}

	// And it clears by the means the hint names, with the file still on disk.
	rm := exec.Command("git", "rm", "-r", "--cached", "--quiet", "--", filepath.Join(config.DotDirName, config.PidsDirName))
	rm.Dir = dir
	if out, err := rm.CombinedOutput(); err != nil {
		t.Fatalf("git rm --cached: %v: %s", err, out)
	}
	if cleared, _ := checkTrackedTransients(dir); !cleared.Passed {
		t.Errorf("the row survived the repair it recommends: %q", cleared.Finding)
	}
	if _, err := os.Stat(filepath.Join(dir, pid)); err != nil {
		t.Errorf("the repair removed the file from disk, which the hint promises it does not: %v", err)
	}
}

// trackedProbeGit answers per pathspec, which fakeGit cannot: its `tracked` is one bool for
// every target, and the cases worth testing here are which classes are tracked, not whether any
// are. It records what it was asked so the test can state that the row asks about every class
// rather than stopping at the first hit.
type trackedProbeGit struct {
	inside, available bool
	tracked           map[string]bool
	asked             []string
}

func (g *trackedProbeGit) InsideRepo(string) bool { return g.inside }
func (g *trackedProbeGit) Available() bool        { return g.available }
func (g *trackedProbeGit) Tracked(_, target string) bool {
	g.asked = append(g.asked, target)
	return g.tracked[target]
}
func (g *trackedProbeGit) Ignored(string, string) bool { return false }

// TestTrackedTransientsRowOmittedWhenGitCannotAnswer covers the two states in which there is no
// index to ask about. Both used to be a judgement call between "pass" and "fail", and neither is
// either: doctor omits the row, because a pass here would be a claim nobody verified and a fail
// would send someone looking for files that are not committed.
func TestTrackedTransientsRowOmittedWhenGitCannotAnswer(t *testing.T) {
	for _, tt := range []struct {
		name              string
		inside, available bool
	}{
		{name: "outside a repository", available: true},
		{name: "inside a repository without git", inside: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			restore := bridgeGit
			bridgeGit = &trackedProbeGit{inside: tt.inside, available: tt.available}
			defer func() { bridgeGit = restore }()

			if r, answered := checkTrackedTransients(t.TempDir()); answered {
				t.Errorf("the row was reported anyway: Passed = %v, Finding = %q", r.Passed, r.Finding)
			}
		})
	}
}

// TestTrackedTransientsAsksAboutEveryClass pins that the verdict is over the whole set. Stopping
// at the first tracked class would report one and leave the others unmentioned, and a person who
// untracks what the row names would then see it fire again with something new — the shape of
// finding that teaches people the tool is unreliable.
func TestTrackedTransientsAsksAboutEveryClass(t *testing.T) {
	specs := dvaTransientPathspecs()
	stub := &trackedProbeGit{
		inside:    true,
		available: true,
		tracked:   map[string]bool{specs[0]: true, specs[len(specs)-1]: true},
	}
	restore := bridgeGit
	bridgeGit = stub
	defer func() { bridgeGit = restore }()

	r, answered := checkTrackedTransients(t.TempDir())
	if !answered || r.Passed {
		t.Fatalf("answered = %v, Passed = %v; want a reported failure", answered, r.Passed)
	}
	if !slices.Equal(stub.asked, specs) {
		t.Errorf("asked about %v, want every class in order: %v", stub.asked, specs)
	}
	for _, spec := range []string{specs[0], specs[len(specs)-1]} {
		if !strings.Contains(r.Finding, spec) {
			t.Errorf("Finding = %q, want it to name %q", r.Finding, spec)
		}
	}
	if strings.Contains(r.Finding, specs[1]) {
		t.Errorf("Finding = %q, want it to leave out the class that is not tracked (%q)", r.Finding, specs[1])
	}
}

// TestTheModulesRowFiresOnlyWhereModulesExist pins the gate by changing one thing. The
// repository is the same one TestDoctorReportsTheRuleThatBlocksModules builds — old rule in
// force, git genuinely refusing a module — and the only edit between the two assertions is
// that a module file appears on disk. Anything else that could flip the row, a different
// .gitignore or a different git state, is held constant, so a pass here is about the gate
// rather than about the rule.
//
// The first half is the population the gate is for: every repository that ran `dva init` under
// the old advice carries the blocked rule whether or not it uses modules, and telling someone
// to hand-edit a file to clear a finding that costs them nothing is how a row stops being read.
// The second half is why the gate is not simply "drop the row": where the feature is in use the
// finding explains a `git add` that is already failing.
func TestTheModulesRowFiresOnlyWhereModulesExist(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(defaultIgnoreSection+"\n"+config.DotDirName+"/\n"), 0o644); err != nil {
		t.Fatalf("WriteFile .gitignore: %v", err)
	}
	// A dot directory holding transient state and no module, which is what a repository
	// that never used the feature looks like.
	if err := os.MkdirAll(filepath.Join(dir, config.DotDirName, config.PidsDirName), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// The premise: git does block modules here, so the row is suppressed by the gate and not
	// by the question coming back negative.
	if blocked, known := dvaModulesBlocked(dir); !known || !blocked {
		t.Fatalf("premise failed: modules are not blocked here (blocked=%v known=%v), so this test proves nothing", blocked, known)
	}

	if r := checkGitignoreStatus(dir); !r.Passed {
		t.Fatalf("a repository with no modules got the blocked-modules row: %q", r.Finding)
	}

	if err := os.WriteFile(filepath.Join(dir, config.DotDirName, "gates.yml"), []byte("x: 1\n"), 0o644); err != nil {
		t.Fatalf("WriteFile module: %v", err)
	}

	r := checkGitignoreStatus(dir)
	if r.Passed {
		t.Fatalf("the row stayed silent in a repository that does author modules")
	}
	if !strings.Contains(r.Finding, "modules") {
		t.Errorf("Finding = %q, want it to name modules as what is blocked", r.Finding)
	}
}

func TestGitignoreStatusNamesTheRightRemedy(t *testing.T) {
	for _, tt := range []struct {
		name        string
		gitignore   string // "" means the file does not exist
		wantPass    bool
		wantFinding string
	}{
		{name: "no .gitignore at all", wantFinding: "no .gitignore here"},
		{name: "a .gitignore that says nothing about it", gitignore: "node_modules/\n", wantFinding: "is NOT ignored"},
		// Both lines, not one. `.sb/dva/*` on its own ignores the transients and the
		// modules alike, so it is no longer a passing configuration — the row it produces
		// is the modules finding, which TestDoctorReportsTheRuleThatBlocksModules covers.
		// The case moved here rather than being deleted because what it was for, a rule
		// git understands and this package's literal reader need not, is still worth
		// pinning; the block DVA writes is that rule.
		{name: "ignored through the rules DVA writes", gitignore: ".sb/dva/*\n!.sb/dva/*.yml\n", wantPass: true},
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
