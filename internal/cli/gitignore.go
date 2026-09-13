package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const (
	defaultIgnoreSection = "# ignore ScriptonBasestar tmp files"
)

// gitCheckIgnoreTimeout bounds the one git call on the hot path. The work is matching four
// constant paths against the ignore rules, which is microseconds; the budget is this large
// because the thing being waited on is not the work. A cold index, a core.fsmonitor hook that
// has to start a daemon, or a repository on a network filesystem can all put seconds in front of
// a trivial query, and cutting the answer off there would trade a rare hang for a routine wrong
// answer. Two seconds is past every one of those and far short of the point where a person
// decides dva is broken.
//
// A var rather than a const so the test for this can shrink it. Testing the deadline means
// letting it expire, and the alternative is a test that spends the real budget twice over —
// once for the deadline, once for WaitDelay — every run.
var gitCheckIgnoreTimeout = 2 * time.Second

// defaultIgnoreRules is the block DVA writes, and it is two lines rather than one because one
// line cannot express what DVA needs. `.sb/dva/` — what this wrote until now — excludes the
// directory, and git does not descend into an excluded directory, so no later negation can
// reach back inside it. That is not a subtlety about spelling; it means the rule DVA itself
// recommended made DVA's own modules feature unusable. `.sb/dva/<name>.yml` is hand-authored
// configuration that belongs in the commit, and under the old rule `git add` refused it
// without `-f`. A repository that had already committed one kept it (git does not apply
// .gitignore to tracked files) and then silently could not add the second.
//
// Excluding the contents instead leaves the directory itself un-excluded, which is the only
// arrangement in which the negation on the next line has anything to attach to. `*` does not
// cross `/`, so `!.sb/dva/*.yml` re-includes modules at the one level DVA puts them and nothing
// deeper — a `.yml` inside a cloned source stays ignored.
//
// Nothing DVA writes at runtime is re-included by that negation: the four classes are
// `pids/*.pid`, `logs/*.log`, `sources/<entry>` and `provisioned-<profile>`, none of them a
// `.yml` at this level. dvaTransientProbes is the list, and the test over it is what keeps that
// claim true as writers come and go.
func defaultIgnoreRules() []string {
	dir := config.DotDirName
	return []string{dir + "/*", "!" + dir + "/*.yml"}
}

// defaultIgnoreAdvice spells the same block for the messages that tell a person what to write.
// It reads defaultIgnoreRules rather than restating it: the advice and the automatic fix
// disagreeing is the failure this whole change is about, and two independent spellings of one
// rule is how that disagreement gets reintroduced.
func defaultIgnoreAdvice() string {
	rules := defaultIgnoreRules()
	quoted := make([]string, 0, len(rules))
	for _, rule := range rules {
		quoted = append(quoted, "'"+rule+"'")
	}
	return strings.Join(quoted, " + ")
}

// ensureGitignore ensures that the .gitignore file contains the necessary entries for DVA.
// Returns true if it wrote them.
//
// It used to return true when the rule was "updated or already present", and all three callers
// print "📎 Updated .gitignore" on true — so `dva init` claimed an edit it had not made in every
// repository that already had the rule. Asking git widened the set of repositories that reach
// that branch, which is what turned a narrow oddity into the ordinary case. "Already present"
// belongs to whoever wants to ask for it; nobody does, and no caller is worse off for the
// answer changing, since doctor's fix discards it.
func ensureGitignore(configDir string) (bool, error) {
	gitignorePath := filepath.Join(configDir, ".gitignore")
	ignoreRules := strings.Join(defaultIgnoreRules(), "\n")

	// Read first, ask second, write last. The order matters: the two branches below used to
	// ask different questions — the missing-file branch wrote without consulting git at all
	// while the existing-file branch consulted it and declined — so whether DVA appended a
	// rule your repository already had depended on whether some unrelated line put a
	// .gitignore beside it. One question, asked once, for both.
	data, readErr := os.ReadFile(gitignorePath)
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, fmt.Errorf("failed to read .gitignore: %w", readErr)
	}
	content := string(data)

	if dvaRepoAlreadyDeclares(configDir, content) {
		return false, nil
	}

	if os.IsNotExist(readErr) {
		// No .gitignore, creating a new one
		created := fmt.Sprintf("%s\n%s\n", defaultIgnoreSection, ignoreRules)
		if err := os.WriteFile(gitignorePath, []byte(created), 0644); err != nil {
			return false, fmt.Errorf("failed to create .gitignore: %w", err)
		}
		return true, nil
	}

	// Not ignored, append to the end
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return false, fmt.Errorf("failed to open .gitignore for appending: %w", err)
	}
	defer func() { _ = f.Close() }()

	if !strings.HasSuffix(content, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return false, err
		}
	}

	ignoreBlock := fmt.Sprintf("\n%s\n%s\n", defaultIgnoreSection, ignoreRules)
	if _, err := f.WriteString(ignoreBlock); err != nil {
		return false, fmt.Errorf("failed to append to .gitignore: %w", err)
	}

	return true, nil
}

// dvaTransientProbes lists one representative path per class of file DVA writes under its dot
// directory, relative to the config directory. These four are the whole set: pid files, log
// files, clones of git sources, and provision markers. Nothing else writes there — the modules
// DVA reads are authored by hand and belong in the commit.
//
// Every element is built from the constant its writer uses, so renaming a directory moves the
// probe with it. Spelled as literals they would keep passing while asking about a path nothing
// produces any more, which is the failure mode that leaves a checker green and useless.
//
// The names are probes, not predictions: check-ignore matches a pattern against a pathname and
// does not require the file to exist, so "probe" stands in for whichever pid, log, source entry
// or profile happens to be there.
//
// That substitution is exact for every pattern but one. A pattern ending in `/` matches only a
// directory, and with nothing on disk git has no way to know the path names one — so a rule
// spelled `.sb/dva/sources/*/`, which ignores every real clone, does not match the sources
// probe, and the unanimous verdict below turns that into a warning. It is the safe direction and
// it is still a false one. Note which half of the path decides: `.sb/dva/pids/` matches
// `pids/probe.pid` because `pids` is an ancestor, and an ancestor's directory-ness git can read
// off the path itself. Only a trailing-slash rule aimed at the probe's own last component needs
// the thing to exist, which was measured — creating `.sb/dva/sources/probe` as a directory makes
// the same pattern match.
//
// The separator is path.Join, not filepath.Join: these strings are handed to git and compared
// against what git echoes back, and git speaks forward slashes on every platform.
func dvaTransientProbes() []string {
	return []string{
		path.Join(config.DotDirName, config.PidsDirName, "probe.pid"),
		path.Join(config.DotDirName, config.LogsDirName, "probe.log"),
		path.Join(config.DotDirName, config.SourcesDirName, "probe"),
		path.Join(config.DotDirName, provisionMarkerName("probe")),
	}
}

// gitCheckIgnore reports, for each ignored path, the ignore file whose rule matched, and
// whether git answered at all. A path missing from the map is not ignored.
//
// `--stdin` settles every path in one process and reports them individually, which is what
// makes the verdict readable per path: `--quiet` collapses the same question to "was any one
// of them ignored", and the configuration worth warning about — contents excluded, one class
// re-included — is exactly the one that answer cannot distinguish from a correct setup.
//
// `-v` is here for the source, not for the diagnostics. Two callers ask this function different
// questions — "will this working tree commit transient state" and "does the repository itself
// say not to" — and only the second one cares that a rule in the user's core.excludesFile is
// not a rule the repository ships. Without the source those two questions have one answer, and
// giving them one answer is the defect ensureGitignore had.
//
// `-z` rather than parsing "source:line:pattern\tpath": that spelling is ambiguous, because a
// gitignore pattern may itself contain a colon and so may a directory name. With -z the four
// fields arrive NUL-separated and need no parsing at all. It changes the INPUT separator too,
// which is why the probes are joined with NUL below.
//
// Exit 1 means "none of these are ignored". That is an answer, not a failure, and the two have
// to be told apart: a fake or broken `.git` (a worktree pointer to a gitdir that has moved,
// among others) exits 128, and folding that into "not ignored" turns an unanswerable question
// into a warning about a repository that may well be configured correctly. One case does slip
// through that reading: git exits 1 for an unknown subcommand too, so a git predating
// check-ignore reports "nothing is ignored" rather than "cannot answer". It warns, which is the
// safe direction, and separating the two would cost a version probe on every invocation.
//
// The deadline is not a nicety. This runs on the hot path — one fork per command, ahead of the
// output the user asked for — and without a bound a git that never returns takes every dva
// command with it. Measured with a check-ignore that sleeps: `dva ls` went from 457ms to
// hanging until it was killed from outside. Timing out is not a failure mode this function has
// to invent an answer for, because `decided` already says "git could not answer" and every
// caller falls back to reading .gitignore literally. WaitDelay bounds the second half of the
// same hazard: killing the process does not by itself unblock the goroutine copying the probes
// into its stdin.
var gitCheckIgnore = func(dir string, paths []string) (sources map[string]string, decided bool) {
	ctx, cancel := context.WithTimeout(context.Background(), gitCheckIgnoreTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "git", "check-ignore", "-vz", "--stdin")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\x00") + "\x00")
	cmd.WaitDelay = gitCheckIgnoreTimeout
	var out bytes.Buffer
	cmd.Stdout = &out
	// Discarded: git's complaints here are about the repository, not about DVA, and this
	// check runs in front of another command's output.
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		// Checked before the exit code, not after. A killed process reports a code of its
		// own, and reading that code as git's verdict is exactly the confusion the 128 case
		// above exists to avoid.
		if ctx.Err() != nil {
			return nil, false
		}
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return nil, false
		}
	}

	// Records of four fields — source, line number, pattern, pathname — each NUL-terminated,
	// so a split leaves one trailing empty element. Stepping by four and requiring a whole
	// record present means a truncated tail is dropped rather than read as a short record.
	//
	// A record is not by itself a verdict. Without -v git prints only the ignored paths; with
	// -v it prints every path some pattern matched, and a path whose last match is a negation
	// is reported here while not being ignored — `.sb/dva/*` + `!.sb/dva/provisioned-*` prints
	// the marker with pattern `!.sb/dva/provisioned-*`. Reading the record as the answer turns
	// exactly that configuration, the half-correct one this check exists to catch, into a pass.
	fields := strings.Split(out.String(), "\x00")
	sources = make(map[string]string, len(paths))
	for i := 0; i+3 < len(fields); i += 4 {
		if strings.HasPrefix(fields[i+2], "!") {
			continue
		}
		sources[fields[i+3]] = fields[i]
	}
	return sources, true
}

// gitignoreSourceIsShared reports whether a rule from this source travels with the repository.
//
// Only rules in a committed .gitignore do. The user's core.excludesFile is that one machine's
// preference, and .git/info/exclude is that one clone's; a collaborator cloning the repository
// gets neither. So a check that decides whether the repository needs the rule WRITTEN has to
// discount both, or it will read one developer's setup as the project's and leave the project
// with nothing.
//
// git names an in-repository ignore file relative to the directory it ran in (".gitignore",
// "../.gitignore") and names core.excludesFile as configured, which is an absolute path in
// every ordinary setup. That is what "relative" tests for here, and it is the part that
// matters when the excludes file is ~/.gitignore — a common choice whose base name alone would
// otherwise pass for the repository's own.
func gitignoreSourceIsShared(source string) bool {
	if source == "" || filepath.IsAbs(source) {
		return false
	}
	if path.Base(source) != ".gitignore" {
		return false
	}
	// .git/info/exclude is relative and inside the tree, so only its location rules it out.
	return !slices.Contains(strings.Split(filepath.ToSlash(source), "/"), ".git")
}

// dvaTransientsIgnored asks git whether every class of transient state is ignored, and reports
// whether git was in a position to answer at all.
//
// Asking git rather than reading .gitignore ourselves is the point. The question — would this
// working tree commit DVA's transient state? — has exactly one correct arbiter, and it applies
// rules no hand-rolled reader here covers: globs, `**` anchoring, directory re-inclusion,
// .git/info/exclude, the user's core.excludesFile, and nested .gitignore files down the tree.
// isDvaIgnored's own comment lists the first three as known gaps; this closes all of them at
// once, and closes them by construction rather than by adding cases.
//
// The verdict has to be unanimous. One class of marker left committable is the whole hazard,
// so every probe must come back ignored before this reports that it is.
//
// What this does NOT see is a transient that is already tracked. check-ignore does report a
// tracked path as not ignored, but that behaviour cannot reach here: every probe is a synthetic
// constant — `probe.pid`, `probe.log`, `probe`, `provisioned-probe` — and no writer produces
// those names, so no probe is ever a tracked path. A repository that already committed
// `.sb/dva/pids/web.pid` and then wrote a correct rule passes every check in this file.
//
// That is a gap, not a decision, and it is worth stating because the gap is the worse half of
// the hazard: an ignore rule stops the next commit, while a file already in the index keeps
// being committed regardless of what .gitignore says. Closing it means asking a different
// question — `git ls-files` against the real directory rather than check-ignore against
// stand-ins — and gitProbe.Tracked in config_env_git.go is the seam that would answer it.
func dvaTransientsIgnored(configDir string) (ignored bool, decided bool) {
	return dvaTransientsCovered(configDir, false)
}

// dvaRepoDeclaresIgnore asks the narrower question: does the repository itself carry the rules,
// so that a fresh clone has them too?
//
// This is the question the writer has to ask, and asking the wider one instead is a regression
// with no symptom on the machine that has it. A developer whose global excludes file lists
// .sb/ sees every check pass and every command stay quiet, while the .gitignore they commit
// says nothing about DVA and every collaborator without that file commits pid files and logs.
func dvaRepoDeclaresIgnore(configDir string) (declared bool, decided bool) {
	return dvaTransientsCovered(configDir, true)
}

func dvaTransientsCovered(configDir string, sharedOnly bool) (covered bool, decided bool) {
	// Both guards avoid spawning git where it cannot help, and the second is the one that
	// matters: with git missing, the literal reader is all there is, and it must be reached
	// through "undecided" rather than through a wrong "not ignored".
	if !bridgeGit.InsideRepo(configDir) || !bridgeGit.Available() {
		return false, false
	}

	probes := dvaTransientProbes()
	sources, decided := gitCheckIgnore(configDir, probes)
	if !decided {
		return false, false
	}
	for _, probe := range probes {
		source, ok := sources[probe]
		if !ok {
			return false, true
		}
		if sharedOnly && !gitignoreSourceIsShared(source) {
			return false, true
		}
	}
	return true, true
}

// dvaStateIsIgnored settles the question for every caller: git when it can answer, the literal
// .gitignore reader when it cannot.
//
// The fallback runs where git is absent — the check still has something useful to say about a
// plainly-spelled rule, and staying silent there would trade a warning for nothing. It is the
// weaker reader, so it keeps its bias: anything it cannot parse comes out unignored and warns.
func dvaStateIsIgnored(configDir, gitignoreContent string) bool {
	if ignored, decided := dvaTransientsIgnored(configDir); decided {
		return ignored
	}
	return isDvaIgnored(gitignoreContent)
}

// dvaRepoAlreadyDeclares answers dvaStateIsIgnored's question narrowed to what a clone would
// get, for the one caller that writes.
//
// The fallback is the same literal reader, and it needs no narrowing: it only ever read the
// repository's own .gitignore, so it was already answering the narrow question. That is worth
// saying because it is the reason this pair can share a fallback — the wide and narrow readings
// differ only in what git can see beyond the file, and the fallback cannot see beyond it.
func dvaRepoAlreadyDeclares(configDir, gitignoreContent string) bool {
	if declared, decided := dvaRepoDeclaresIgnore(configDir); decided {
		return declared
	}
	return isDvaIgnored(gitignoreContent)
}

// isDvaIgnored checks if the config directory is already ignored in the gitignore content.
//
// This is the fallback for working trees where git cannot be asked; dvaStateIsIgnored prefers
// git. What follows describes how far this reader gets on its own.
//
// Matching the full path literally is not enough. DotDirName is a two-segment path
// (".sb/dva"), and git excludes an entire subtree when any ancestor directory is listed, so
// a .gitignore saying ".sb/" already ignores ".sb/dva". The literal check called that
// unignored and told the user to add a rule they did not need — on the better-written
// config, since ".sb/" covers DVA's whole dot directory rather than one child.
//
// Negations have to be read, and they do not behave the same way at every depth. Verified
// against git with the paths on disk:
//
//	.sb/ + !.sb/dva/   → still ignored. A path cannot be re-included once a parent
//	                     directory is excluded, so negating a descendant loses.
//	.sb/ + !.sb/        → NOT ignored. The exclusion and the negation name the same
//	                     directory, no ancestor of it is excluded, so ordinary
//	                     last-matching-pattern-wins applies and the negation wins.
//
// So each ancestor is resolved on its own, outermost first, and the first one that comes out
// excluded is decisive. Reading the file for a single match and stopping — as this did
// before — reports "already ignored" for the second case above and suppresses the warning
// while git leaves the markers committable.
//
// One glob form is read, and only because DVA writes it. defaultIgnoreRules spells the rule
// `.sb/dva/*`, since excluding the directory itself would make DVA's modules unaddable; a reader
// that could not see that spelling would report DVA's own freshly-written block as missing, and
// ensureGitignore — which consults this when git cannot be asked — would append the same block
// on every run. Directory-contents exclusion answers this predicate the same way directory
// exclusion does: the transients live under the directory either way.
//
// This still stops short of implementing gitignore semantics: other globs, non-root anchoring
// ("**/.sb/"), and .git/info/exclude stay uninterpreted. Those gaps make DVA warn about a path
// that is in fact ignored, which is the harmless direction — an extra warning, never a silently
// committed marker.
//
// The one glob read here does have a hole in the other direction, and it predates this: a
// negation narrower than the exclusion, `!.sb/dva/p*` say, would re-include provision markers
// while nothing below names that pattern, so this would report them ignored when git does not.
// It is the same hole `.sb/` + `!.sb/d*` already had — an uninterpreted negation is invisible
// whether or not the exclusion beside it is a glob. Both are closed on the git-backed path,
// which is every working tree that has git; this reader runs only where it does not.
func isDvaIgnored(content string) bool {
	lines := strings.Split(content, "\n")
	for _, prefix := range ancestorsAndSelf(config.DotDirName) {
		// The path's own spellings are asked first, and the order is load-bearing. Given
		// `.sb/dva/` and `!.sb/dva/*` together, git excludes the directory and never
		// descends, so the negation on its contents never applies; reading the contents
		// form first would let that negation win an argument git does not give it.
		if lastMatchExcludes(lines, pathSpellings(prefix)) {
			return true
		}
		if lastMatchExcludes(lines, contentsSpellings(prefix)) {
			return true
		}
	}
	return false
}

// lastMatchExcludes applies gitignore's last-matching-pattern-wins rule to one path: it
// reports whether the last line naming that path excludes it rather than negates it, and
// false when no line names it at all.
func lastMatchExcludes(lines []string, forms map[string]bool) bool {
	excluded := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		pattern := strings.TrimPrefix(line, "!")
		if forms[pattern] {
			excluded = line == pattern
		}
	}
	return excluded
}

// pathSpellings returns the .gitignore lines that name path, in each spelling git treats
// alike: bare, trailing slash, and root-anchored with a leading slash.
func pathSpellings(path string) map[string]bool {
	forms := make(map[string]bool, 4)
	if path == "" {
		return forms
	}
	for _, form := range []string{path, path + "/", "/" + path, "/" + path + "/"} {
		forms[form] = true
	}
	return forms
}

// contentsSpellings returns the .gitignore lines that exclude everything inside path rather
// than path itself: ".sb/dva" → ".sb/dva/*" and "/.sb/dva/*".
//
// Kept apart from pathSpellings because the two name different things and only happen to
// answer this predicate alike. Folding them together would also fold their negations together,
// and `!.sb/dva/` and `!.sb/dva/*` are not interchangeable — the first negates a directory, the
// second its contents, and git resolves them at different points.
func contentsSpellings(path string) map[string]bool {
	forms := make(map[string]bool, 2)
	if path == "" {
		return forms
	}
	for _, form := range []string{path + "/*", "/" + path + "/*"} {
		forms[form] = true
	}
	return forms
}

// ancestorsAndSelf lists dir's path prefixes outermost first: ".sb/dva" → [".sb", ".sb/dva"].
func ancestorsAndSelf(dir string) []string {
	var prefixes []string
	parts := strings.Split(dir, "/")
	for i := range parts {
		if prefix := strings.Join(parts[:i+1], "/"); prefix != "" {
			prefixes = append(prefixes, prefix)
		}
	}
	return prefixes
}

// ignoreRulesCovering returns every .gitignore line that names dir or one of its ancestors,
// in every spelling. It is the union of what isDvaIgnored evaluates; isDvaIgnored keeps the
// prefixes separate because a negation resolves differently at each depth.
func ignoreRulesCovering(dir string) map[string]bool {
	rules := make(map[string]bool)
	for _, prefix := range ancestorsAndSelf(dir) {
		for form := range pathSpellings(prefix) {
			rules[form] = true
		}
	}
	return rules
}

// checkGitignoreForWarning warns when this working tree would commit DVA's transient markers.
// loadConfig calls it, so it sits in front of every command that reads a config; what keeps it
// from preempting their output is that all three conditions below have to hold at once.
//
//   - No `.git`: nothing here is tracked, so nothing can be committed.
//   - No `.sb/dva` on disk: the directory that would be committed does not exist. Read-only
//     commands cannot create it — `ls`, `show`, `validate`, `status` and `manifest` were each
//     measured leaving the working tree untouched — so on a fresh clone the warning named a
//     hazard that had not happened and, for a reader who only ever inspects the config, never
//     would. What creates the directory is the four writers dvaTransientProbes enumerates —
//     pid files and log files under internal/lifecycle, source clones under internal/config,
//     provision markers here in internal/cli — and the invocation after one of those is where
//     the warning has something to point at. There is no module cache and no source cache; an
//     earlier spelling of this list named both, and a reader who went looking for them found
//     neither the caches nor the real writers.
//   - Already ignored: dvaStateIsIgnored settles it, ancestors and negations included. Naming
//     isDvaIgnored here, as this used to, understates it — that one reads the file literally
//     and its own comment lists globs and re-inclusion as gaps. Ancestors and negations are
//     covered only on the git-backed path dvaStateIsIgnored prefers.
//
// `dva doctor` deliberately keeps the unconditional form (checkGitignoreStatus, called from
// runDoctorChecks in doctor.go; it lives in this file, which the pointer this replaces still
// placed in doctor.go at a line number that had already moved),
// which does not gate on existence: "is my setup right?" is the question doctor is asked, and it
// is asked before anything has run. This path answers the narrower one — something committable is
// on disk right now — and that is the only version worth putting ahead of another command's
// answer.
// gitignoreWarningSuppressedFor reports whether the banner should stay quiet because the
// command about to run reports this finding itself.
//
// Only `dva doctor` does. Its whole job is to enumerate findings of exactly this class, and
// it emits the gitignore state as its own [FAIL] row — so the banner made it the one finding
// doctor reported twice, once on stderr before the report and once inside it. Two reports of
// one problem read as two problems, which is the opposite of what a diagnostic owes its
// reader. Every other command gets the banner, because for them it is the only warning there
// is (TASK-139).
func gitignoreWarningSuppressedFor(cmdName string) bool {
	return cmdName == "doctor"
}

func checkGitignoreForWarning(configDir string) {
	// --json's audience is a program and this warning has no schema, so it is noise there whatever
	// the streams do. Gating here rather than at the call site keeps the decision next to the
	// message it silences; doctor's structured result already carries the same finding for
	// consumers that want it.
	if jsonOutput {
		return
	}
	if gitignoreWarningSuppressedFor(runningCommand) {
		return
	}

	gitignorePath := filepath.Join(configDir, ".gitignore")

	// If .gitignore doesn't exist, we might not be in a git repo or user doesn't care.
	// But if we have a .git directory, we should probably warn.
	if _, err := os.Stat(filepath.Join(configDir, ".git")); os.IsNotExist(err) {
		return
	}

	if _, err := os.Stat(filepath.Join(configDir, config.DotDirName)); os.IsNotExist(err) {
		return
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		// Not ignored since we can't read it or it doesn't exist
		if !os.IsNotExist(err) {
			return
		}
	}

	if dvaStateIsIgnored(configDir, string(data)) {
		return
	}

	fmt.Fprintf(os.Stderr, "⚠️  [warn] %s/ is not in your .gitignore. Transient markers might be committed.\n", config.DotDirName)
	fmt.Fprintf(os.Stderr, "         Run 'dva doctor --fix' to auto-fix or add %s to .gitignore manually.\n\n", defaultIgnoreAdvice())
}

// failGitignore marks the row failed with the finding and remedy its branch found, and attaches
// the fix all of them share. Extracted because the branches number three and the closure is the
// part a reader has to confirm is the same in each.
func failGitignore(r DoctorResult, configDir, finding, fixHint string) DoctorResult {
	r.Passed = false
	r.Finding = finding
	r.Fixable = true
	r.FixHint = fixHint
	r.fixFunc = func() error {
		_, err := ensureGitignore(configDir)
		return err
	}
	return r
}

// checkGitignoreStatus is doctor's row for the same question checkGitignoreForWarning answers
// on stderr. It lives here, beside that warning and the resolver they share, because it did not:
// while it sat in doctor.go the two could be changed apart, and were — one of them reporting a
// correctly-configured tree as fine while the other called it broken is the state this move
// makes hard to reach again.
func checkGitignoreStatus(configDir string) DoctorResult {
	r := DoctorResult{Name: fmt.Sprintf("%s/ is ignored in .gitignore", config.DotDirName)}
	gitignorePath := filepath.Join(configDir, ".gitignore")

	// Which of the two failure wordings is right depends on whether the file exists, and
	// nothing else — so the choice is made here, once, for every branch that fails. Folding
	// it into the read below is what made "no .gitignore here" unreachable as soon as git was
	// asked first: doctor then told people with no .gitignore to add a line to it.
	fail := func() DoctorResult {
		if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
			return failGitignore(r, configDir,
				fmt.Sprintf("no .gitignore here, so %s/ is not ignored", config.DotDirName),
				fmt.Sprintf("Create .gitignore and add %s to avoid committing transient state", defaultIgnoreAdvice()))
		}
		return failGitignore(r, configDir,
			fmt.Sprintf("%s/ is NOT ignored in .gitignore", config.DotDirName),
			fmt.Sprintf("Add %s to .gitignore to avoid committing transient state", defaultIgnoreAdvice()))
	}

	// git is asked before the file is read, because it applies rules this function cannot
	// see — .git/info/exclude, core.excludesFile, nested .gitignore files — so a tree with no
	// .gitignore at all may already be ignoring every marker correctly. Reading the file
	// first reports that tree as failing and offers to fix what is not broken.
	//
	// The wide reading is the right one for a diagnostic: doctor reports on the machine it is
	// run on, and on this machine those rules do apply. ensureGitignore asks the narrow one,
	// because what it decides is what to leave in the repository for everyone else.
	if ignored, decided := dvaTransientsIgnored(configDir); decided {
		if ignored {
			r.Passed = true
			return r
		}
		return fail()
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return fail()
		}
		r.Passed = false
		r.Finding = fmt.Sprintf(".gitignore could not be read, so %s/ cannot be confirmed ignored: %v", config.DotDirName, err)
		return r
	}

	if isDvaIgnored(string(data)) {
		r.Passed = true
		return r
	}
	return fail()
}
