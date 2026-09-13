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
// That is the worse half of the hazard — an ignore rule stops the next commit, while a file
// already in the index keeps being committed regardless of what .gitignore says — and it is not
// answerable here, because it is a different question: `git ls-files` against the real directory
// rather than check-ignore against stand-ins. trackedTransients asks it, and doctor reports the
// two as separate rows because they have separate repairs. Nothing below needs to account for
// tracked files; a repository can be wrong in both ways at once and hear about both.
func dvaTransientsIgnored(configDir string) (ignored bool, decided bool) {
	return dvaTransientsCovered(configDir, false)
}

// dvaTransientPathspecs names the same four classes dvaTransientProbes stands in for, spelled as
// pathspecs git can match against the index instead of as probe paths.
//
// The two lists exist for two different questions and cannot be one list. A probe is a path that
// need not exist, because check-ignore matches a pattern against a name; a pathspec has to name
// what is really there, because ls-files reports what the index holds. `probe.pid` answers the
// first and is useless for the second — no writer produces that name, which is exactly why no
// probe is ever tracked and why check-ignore's own tracked-path behaviour never fires here.
//
// So the classes are named one level up: the directory that holds the pid files rather than a
// pid file, and a glob for the markers, which are files at this level and have no directory to
// stand for them. Both spellings were measured against `git ls-files --cached --error-unmatch`:
// the directory form exits 0 when anything inside it is tracked, the glob form exits 0 when any
// marker is, and both exit non-zero when nothing matches.
//
// Every element is built from the constant its writer uses, for the reason dvaTransientProbes
// gives: a literal keeps answering after the thing it names has moved.
func dvaTransientPathspecs() []string {
	return []string{
		path.Join(config.DotDirName, config.PidsDirName),
		path.Join(config.DotDirName, config.LogsDirName),
		path.Join(config.DotDirName, config.SourcesDirName),
		path.Join(config.DotDirName, provisionMarkerName("*")),
	}
}

// trackedTransients reports which classes of transient state this repository has already
// committed, and whether git was in a position to say.
//
// This is the question dvaTransientsIgnored structurally cannot ask. An ignore rule governs what
// git will pick up next; it says nothing about what is already in the index, and git does not
// apply .gitignore to a tracked path. A repository that committed `.sb/dva/pids/web.pid` once
// and then wrote the perfect rule keeps committing every change to that file forever, while
// every other check in this file reports it healthy.
//
// It goes through bridgeGit rather than running git here so the rows can be driven from a stub:
// the interesting cases are combinations of tracked classes, and building a real repository per
// combination would make the test slow enough that the combinations get dropped.
//
// Undecided is not a failure. Outside a repository there is no index to ask about, and without
// git the question has no arbiter — in both cases doctor omits the row rather than printing a
// pass it did not earn.
func trackedTransients(configDir string) (tracked []string, decided bool) {
	if !bridgeGit.InsideRepo(configDir) || !bridgeGit.Available() {
		return nil, false
	}
	for _, spec := range dvaTransientPathspecs() {
		if bridgeGit.Tracked(configDir, spec) {
			tracked = append(tracked, spec)
		}
	}
	return tracked, true
}

// checkTrackedTransients is doctor's row for that question, reported separately from the ignore
// row because the two have separate repairs and a repository can need both.
//
// Not fixable, for the same reason blockedModules is not: the repair changes something a person
// owns. `git rm --cached` rewrites the index and stages a deletion, which is a commit someone
// has to mean to make — and on a shared branch it is a commit other people have to merge. The
// row names the command and stops there.
//
// The finding names classes rather than paths. Which files are tracked is what `git ls-files`
// is for; what a person needs from a diagnostic is which kind of state escaped and one command
// that clears it.
func checkTrackedTransients(configDir string) (DoctorResult, bool) {
	tracked, decided := trackedTransients(configDir)
	if !decided {
		return DoctorResult{}, false
	}

	r := DoctorResult{Name: fmt.Sprintf("no transient state under %s/ is tracked in git", config.DotDirName)}
	if len(tracked) == 0 {
		r.Passed = true
		return r, true
	}

	quoted := make([]string, 0, len(tracked))
	for _, spec := range tracked {
		quoted = append(quoted, "'"+spec+"'")
	}
	r.Passed = false
	r.Finding = fmt.Sprintf("already committed, so .gitignore cannot stop it: %s", strings.Join(tracked, ", "))
	r.Fixable = false
	r.FixHint = fmt.Sprintf("Run: git rm -r --cached -- %s (the files stay on disk; commit the removal)", strings.Join(quoted, " "))
	return r, true
}

// dvaModuleProbe is a stand-in for a module file, the same way dvaTransientProbes stands in for
// the files DVA writes. Modules are `.sb/dva/<name>.yml` at exactly one level, and no rule that
// distinguishes one module name from another is a rule anyone writes, so one name answers for
// all of them.
func dvaModuleProbe() string {
	return path.Join(config.DotDirName, "probe.yml")
}

// dvaModulesBlocked reports whether the ignore rules in force also swallow DVA's modules, and
// whether git was able to say.
//
// This is a second question, and it exists because the first one cannot detect the defect this
// change is about. "Are the transients ignored?" is answered yes by `.sb/dva/` — the rule DVA
// itself wrote until now — and also yes by the two lines it writes instead, so every predicate
// that asks only that leaves a repository configured the old way looking perfectly healthy while
// `git add .sb/dva/gates.yml` is refused. Fixing what DVA writes fixes new repositories only;
// every repository that already ran `dva init` keeps the broken rule and hears nothing about it.
//
// Nothing here rewrites that rule. Replacing a line a person owns in a file a person owns is a
// different act from appending one DVA is missing, and `dva doctor --fix` cannot do it in any
// case: appending the correct two lines under `.sb/dva/` changes nothing, because git does not
// descend into an excluded directory and the appended contents rules never apply. An automatic
// fix that leaves the finding standing would append the same block on every run. So this reports,
// names the edit, and stops.
func dvaModulesBlocked(configDir string) (blocked bool, known bool) {
	sources, decided := gitCheckIgnore(configDir, []string{dvaModuleProbe()})
	if !decided {
		return false, false
	}
	_, ignored := sources[dvaModuleProbe()]
	return ignored, true
}

// dvaModulesUsedHere reports whether this repository authors modules at all, which is what
// decides whether the blocked-modules finding is worth anyone's attention.
//
// The finding is correct wherever the old rule is in force, and correct is not the same as
// useful. Every repository that ever ran `dva init` under `.sb/dva/` carries it, including the
// ones that use no modules, and it cannot be cleared by any means DVA offers: the repair is a
// hand edit to a line a person owns, and `--fix` declines it for the reasons dvaModulesBlocked
// gives. A row that is permanently red and not clearable is a row people learn to skip, and the
// cost of that is not this finding — it is the next real one this row carries.
//
// So the question is asked of the disk, not of the config: any `.sb/dva/*.yml` is a module,
// because nothing else writes a `.yml` there (see dvaTransientProbes), and a module file on disk
// is the evidence that the feature is in use whether or not `modules:` names it yet. Reading the
// loaded config instead would miss a module authored before it is declared — the moment the
// person is most likely to run into the refused `git add` and want this row to explain it.
//
// A repository that adds its first module later starts hearing about it then, which is the
// point: the row fires where the feature is used and is silent where it is not.
func dvaModulesUsedHere(configDir string) bool {
	matches, err := filepath.Glob(filepath.Join(configDir, config.DotDirName, "*.yml"))
	return err == nil && len(matches) > 0
}

// blockedModules is deliberately not routed through failGitignore: that attaches ensureGitignore
// as the repair, and ensureGitignore is correct to do nothing here. See dvaModulesBlocked.
func blockedModules(r DoctorResult) DoctorResult {
	r.Passed = false
	r.Finding = fmt.Sprintf("transient state is ignored, but so are modules: %s is not addable, so DVA's modules feature is disabled here", dvaModuleProbe())
	r.Fixable = false
	r.FixHint = fmt.Sprintf("Replace the rule excluding %s/ itself with %s, which ignores the same transient state and leaves modules committable", config.DotDirName, defaultIgnoreAdvice())
	return r
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
// That harmless direction is also an invitation, and it is worth naming because the invitation
// is what shipped the bug. Every one of those gaps closes the same way — by reading one more
// exclusion spelling — and reading the contents form without reading what re-includes it is
// exactly the extension that turned an extra warning into a committed pid file. So an exclusion
// spelling is not readable on its own. It is readable only together with the set of paths a
// negation would have to name to undo it: the directory itself for `.sb/dva/`, its immediate
// children for `.sb/dva/*`. Adding a spelling means deriving that set and handing it to
// negationReaches. A spelling added without one is not an approximation, it is a regression in
// the single direction this reader may never get wrong.
//
// Negations have to be read, and negationReaches is where that happens. Measured, `.sb/` +
// `!.sb/d*` has git ignoring all four probes while `.sb/*` + `!.sb/dva` has git ignoring none of
// them. Answering both "ignored" — as reading the exclusions without reading the negations does
// — is not a missing warning, it is a silently committed pid file.
//
// What separates those two is not the exclusion spelling but which paths the negation names.
// An exclusion is honoured only when no negation below it names a path git would still reach:
// the directory itself for `.sb/dva/`, its immediate children for `.sb/dva/*`. Excluding a
// directory does buy something the contents form does not — a negation aimed strictly inside it
// is inert, because git does not descend — but that protection stops at the directory's own
// name, and `!*` names it.
func isDvaIgnored(content string) bool {
	lines := strings.Split(content, "\n")
	probes := dvaTransientProbes()
	for _, prefix := range ancestorsAndSelf(config.DotDirName) {
		// The two questions are independent — either answering yes is decisive, and
		// swapping them changes nothing, which was verified by swapping them and running
		// the package. What keeps `.sb/dva/` plus `!.sb/dva/*` answering "ignored" is not
		// the order but lastMatchExcludes: `!.sb/dva/*` is a contents spelling and the last
		// line naming it negates, so the contents branch declines on its own. Said the
		// other way, pathSpellings and contentsSpellings share no member, so no line can be
		// read by both branches and no line can be decided twice.
		//
		// What differs between them is only which paths a negation would have to reach to
		// undo the exclusion, because that is what each spelling excludes. `.sb/dva/`
		// excludes the directory, so the one path is `.sb/dva` itself. `.sb/dva/*` excludes
		// its immediate children, so the paths are those children. Measured: `.sb/` with
		// `!*` leaves all four probes addable — `*` names `.sb` at a level git still reaches
		// — while `.sb/` with `!.sb/d*` ignores all four, since that negation names only
		// something inside the excluded directory and git never descends there.
		if at, excludes := lastMatchExcludes(lines, pathSpellings(prefix)); excludes &&
			!negationReaches(lines, at, []string{prefix}) {
			return true
		}
		if at, excludes := lastMatchExcludes(lines, contentsSpellings(prefix)); excludes &&
			!negationReaches(lines, at, childrenOf(prefix, probes)) {
			return true
		}
	}
	return false
}

// negationReaches reports whether any negation line after index at could re-include one of
// candidates, in which case the exclusion that sits at at must not be read as "ignored".
//
// It answers "could", not "does". The question git settles exactly is not worth reimplementing
// here, and every uncertainty inside the scanned region resolves toward forfeiting: an
// unparseable pattern, a `**` this does not interpret. Forfeiting makes DVA warn about a path
// that may well be ignored, which is the direction the rest of this reader already errs in.
// Getting it wrong the other way commits state.
//
// Two things bound what is scanned, and both were learned by being wrong in the wide direction.
// Neither costs an extra warning; both cost idempotency, because ensureGitignore consults this
// reader when git cannot be asked, so a reader that cannot see the block DVA just wrote appends
// it again — three copies after three runs of `dva init`.
//
// Position is the first. Last-matching-pattern-wins is the entire rule, so a negation earlier in
// the file than the exclusion has already lost to it and cannot re-include anything. A
// `.gitignore` opening with `!*` and carrying DVA's block below it has git ignoring all four
// probes; scanning the whole file made this reader say otherwise.
//
// Depth is the second. `prefix/*` excludes prefix's immediate children — the `pids` directory,
// not just the pid file inside it — so git stops at that level and a negation aimed deeper can
// re-include nothing. An earlier version tested every ancestor of every probe, which made an
// ordinary `!*.log` beside a `logs/` rule forfeit: it matches the log probe's basename, but
// `.sb/dva/logs` is excluded and git never reaches it.
//
// path.Match is the matcher for the anchored case: `*` does not cross `/` in either, which is
// what makes `!.sb/dva/*.yml` — the negation DVA itself writes — provably match no child, and so
// the block DVA writes reads back as ignored.
func negationReaches(lines []string, at int, candidates []string) bool {
	if len(candidates) == 0 {
		return false
	}
	for i, line := range lines {
		if i <= at {
			continue
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "!") {
			continue
		}
		pattern := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(line, "!"), "/"), "/")
		if pattern == "" {
			continue
		}
		// `**` is git's cross-directory wildcard and path.Match has no equivalent, so it
		// reads as two ordinary `*` and quietly matches less than git would. Unread.
		if strings.Contains(pattern, "**") {
			return true
		}
		anchored := strings.Contains(pattern, "/")
		for _, candidate := range candidates {
			if !anchored {
				// A pattern with no slash is matched by git against a path component,
				// not against the path.
				candidate = path.Base(candidate)
			}
			matched, err := path.Match(pattern, candidate)
			if err != nil || matched {
				return true
			}
		}
	}
	return false
}

// childrenOf returns the immediate children of prefix that lie on the path to one of paths:
// ".sb" → [".sb/dva"], ".sb/dva" → [".sb/dva/pids", ".sb/dva/logs", …]. A path not under prefix
// contributes nothing, and prefix itself is not a child of prefix.
func childrenOf(prefix string, paths []string) []string {
	var children []string
	for _, p := range paths {
		rest, ok := strings.CutPrefix(p, prefix+"/")
		if !ok || rest == "" {
			continue
		}
		child := prefix + "/" + strings.SplitN(rest, "/", 2)[0]
		if !slices.Contains(children, child) {
			children = append(children, child)
		}
	}
	return children
}

// lastMatchExcludes applies gitignore's last-matching-pattern-wins rule to one path: it
// reports whether the last line naming that path excludes it rather than negates it, and
// false when no line names it at all.
//
// It also reports where that line is, because "last" is the whole rule and the negation
// scan needs the same yardstick: a negation earlier in the file than the exclusion being
// relied on has already lost to it, and forfeiting on one costs idempotency for nothing.
// at is -1 when no line names the path.
func lastMatchExcludes(lines []string, forms map[string]bool) (at int, excludes bool) {
	at = -1
	for i, line := range lines {
		line = strings.TrimSpace(line)
		pattern := strings.TrimPrefix(line, "!")
		if forms[pattern] {
			at = i
			excludes = line == pattern
		}
	}
	return at, excludes
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
			// The glob is asked first because it is the cheaper question and the one that
			// decides whether the other is worth asking: dvaModulesBlocked spawns git, and
			// in a repository with no modules its answer changes nothing. See
			// dvaModulesUsedHere for why the row is gated rather than always reported.
			if dvaModulesUsedHere(configDir) {
				if blocked, known := dvaModulesBlocked(configDir); known && blocked {
					return blockedModules(r)
				}
			}
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
