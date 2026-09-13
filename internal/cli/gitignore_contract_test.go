package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestEveryGitignoreNoticeNamesTheRules reads source text, which no other test here does, and
// the reason is that the defect it guards lives in the gap the behavioural tests leave.
//
// TestAdviceSpellsExactlyWhatDvaWrites pins the advice to the rules, but it can only reach a
// message that calls defaultIgnoreAdvice. A print site that interpolates config.DotDirName
// itself is invisible to it — there is nothing for an assertion about the advice to attach to.
// That is exactly how the third `dva init` print site kept saying "Updated .gitignore to ignore
// .sb/dva/" for a commit after the other two stopped: every test that could have noticed was
// written in terms of the function that site does not call.
//
// So the assertion is the one a reader would make by eye: every line that announces the write
// names it by the rules that were written. Reading the file means this test fails on a new site
// added in the old spelling, which a test driving runInit would not, since that site is reached
// only through template selection and disk state.
//
// One consequence for anyone mutation-testing this package: the read is os.ReadFile at run time,
// so this is the only test here that sees the file on disk rather than what the compiler saw. A
// mutation injected through `go test -overlay` is invisible to it and the package reports ok —
// measured, not supposed. Every other test in this package is indifferent to which technique put
// the mutation there; this one has to be a real edit or it confirms nothing.
func TestEveryGitignoreNoticeNamesTheRules(t *testing.T) {
	const source = "init_scaffold.go"
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("ReadFile %s: %v", source, err)
	}

	notice := regexp.MustCompile(`(?m)^.*📎 Updated \.gitignore.*$`)
	sites := notice.FindAllString(string(data), -1)
	if len(sites) == 0 {
		t.Fatalf("no .gitignore notice found in %s; has the message moved?", source)
	}

	for _, site := range sites {
		if !strings.Contains(site, "defaultIgnoreAdvice()") {
			t.Errorf("notice does not name the rules it wrote: %s", strings.TrimSpace(site))
		}
		if strings.Contains(site, "config.DotDirName") {
			t.Errorf("notice spells the directory instead of the rules: %s", strings.TrimSpace(site))
		}
	}
}

// TestReaderNeverClaimsIgnoredWhereGitWouldCommit is the property the hand-written tables cannot
// state. Every other case here asserts one expectation a person worked out and checked once; this
// one asks git.
//
// Only one direction is a failure. isDvaIgnored deliberately under-reads — unhandled globs,
// non-root anchoring and .git/info/exclude all make it answer "not ignored" for a path git does
// ignore, which costs a warning. The opposite answer costs a pid file in the commit, and it is
// the answer a well-meaning extension to the reader produces: reading one more exclusion spelling
// without reading what re-includes it. That is how the contents form shipped broken.
//
// The safe direction is counted rather than shrugged at, and the count is logged rather than
// asserted, because it is a standing invitation to make things worse. Roughly a fifth of these
// files are cases where git ignores everything and the reader still warns — and every one of
// them closes the same way, by teaching the reader one more exclusion spelling. Each such
// spelling needs its own answer to "which paths would a negation have to reach", which is the
// step that was skipped before. Anyone driving that number down should expect this test to be
// the thing that stops them, and should treat it stopping them as the test working.
func TestReaderNeverClaimsIgnoredWhereGitWouldCommit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	exclusions := []string{
		"",
		".sb/\n",
		".sb/dva/\n",
		"/.sb/dva/\n",
		".sb/*\n",
		".sb/dva/*\n",
		"/.sb/dva/*\n",
		".sb/dva/**\n",
		"**/.sb/\n",
		".sb/d*\n",
		".sb/dva/pids/\n",
		"*.pid\n",
		"logs/\n",
	}
	negations := []string{
		"",
		"!.sb/dva/*.yml\n",
		"!.sb/dva\n",
		"!.sb/dva/*\n",
		"!.sb/dva/p*\n",
		"!.sb/dva/provisioned-*\n",
		"!.sb/dva/pids/probe.pid\n",
		"!.sb/dva/logs/\n",
		"!.sb/dva/logs/*.log\n",
		"!*.log\n",
		"!*\n",
		"!probe\n",
		"!**/probe.pid\n",
		"!.sb\n",
		"!.sb/\n",
	}
	// Order matters as much as content: last-matching-pattern-wins means the same two lines
	// swapped are two different files, and the reader that scanned negations regardless of
	// position disagreed with git on the second form.
	orders := []func(ex, neg string) string{
		func(ex, neg string) string { return ex + neg },
		func(ex, neg string) string { return neg + ex },
		func(ex, neg string) string { return neg + ex + "!" + ".sb/dva/*.yml\n" },
	}

	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q"}, {"config", "user.email", "p@example.com"}, {"config", "user.name", "p"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	gitignorePath := filepath.Join(repo, ".gitignore")
	probes := dvaTransientProbes()

	checked, underRead := 0, 0
	for _, ex := range exclusions {
		for _, neg := range negations {
			for _, order := range orders {
				content := order(ex, neg)
				if err := os.WriteFile(gitignorePath, []byte(content), 0o644); err != nil {
					t.Fatalf("WriteFile .gitignore: %v", err)
				}
				checked++

				// One git call per file rather than one per probe: check-ignore reads
				// paths on stdin and echoes back only the ignored ones.
				cmd := exec.Command("git", "check-ignore", "--no-index", "--stdin")
				cmd.Dir = repo
				cmd.Stdin = strings.NewReader(strings.Join(probes, "\n") + "\n")
				out, err := cmd.Output()
				if err != nil && cmd.ProcessState.ExitCode() != 1 {
					t.Fatalf("git check-ignore on %q: %v", content, err)
				}
				ignored := map[string]bool{}
				for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
					if line != "" {
						ignored[line] = true
					}
				}

				var addable []string
				for _, p := range probes {
					if !ignored[p] {
						addable = append(addable, p)
					}
				}
				sort.Strings(addable)

				if isDvaIgnored(content) && len(addable) > 0 {
					t.Errorf("reader says ignored but git leaves %s addable, for .gitignore %q",
						strings.Join(addable, ", "), content)
				}
				if !isDvaIgnored(content) && len(addable) == 0 {
					underRead++
				}
			}
		}
	}
	t.Logf("checked %d .gitignore files against git; %d under-read in the safe direction", checked, underRead)
}
