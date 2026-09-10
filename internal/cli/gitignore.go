package cli

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

const (
	defaultIgnoreSection = "# ignore ScriptonBasestar tmp files"
)

func defaultIgnorePath() string {
	return config.DotDirName + "/"
}

// ensureGitignore ensures that the .gitignore file contains the necessary entries for DVA.
// Returns true if it was updated or already present.
func ensureGitignore(configDir string) (bool, error) {
	gitignorePath := filepath.Join(configDir, ".gitignore")
	ignorePath := defaultIgnorePath()

	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		// No .gitignore, creating a new one
		content := fmt.Sprintf("%s\n%s\n", defaultIgnoreSection, ignorePath)
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			return false, fmt.Errorf("failed to create .gitignore: %w", err)
		}
		return true, nil
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return false, fmt.Errorf("failed to read .gitignore: %w", err)
	}

	content := string(data)
	if dvaStateIsIgnored(configDir, content) {
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

	ignoreBlock := fmt.Sprintf("\n%s\n%s\n", defaultIgnoreSection, ignorePath)
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

// gitCheckIgnore reports which of paths git considers ignored, and whether git answered at all.
//
// `--stdin` settles all of them in one process and names the ignored ones on stdout, which is
// what makes the verdict readable per path: `--quiet` collapses the same question to "was any
// one of them ignored", and the configuration worth warning about — contents excluded, one class
// re-included — is exactly the one that answer cannot distinguish from a correct setup.
//
// Exit 1 means "none of these are ignored". That is an answer, not a failure, and the two have
// to be told apart: a fake or broken `.git` (a worktree pointer to a gitdir that has moved,
// among others) exits 128, and folding that into "not ignored" turns an unanswerable question
// into a warning about a repository that may well be configured correctly.
var gitCheckIgnore = func(dir string, paths []string) (ignored map[string]bool, decided bool) {
	cmd := exec.Command("git", "check-ignore", "--stdin")
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(strings.Join(paths, "\n") + "\n")
	var out bytes.Buffer
	cmd.Stdout = &out
	// Discarded: git's complaints here are about the repository, not about DVA, and this
	// check runs in front of another command's output.
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) || exit.ExitCode() != 1 {
			return nil, false
		}
	}

	ignored = make(map[string]bool, len(paths))
	for line := range strings.SplitSeq(out.String(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			ignored[line] = true
		}
	}
	return ignored, true
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
// check-ignore deliberately reports a tracked path as not ignored. That is the right answer
// here too: a transient that is already tracked is already being committed, which is the
// hazard this check exists to name.
func dvaTransientsIgnored(configDir string) (ignored bool, decided bool) {
	// Both guards avoid spawning git where it cannot help, and the second is the one that
	// matters: with git missing, the literal reader is all there is, and it must be reached
	// through "undecided" rather than through a wrong "not ignored".
	if !bridgeGit.InsideRepo(configDir) || !bridgeGit.Available() {
		return false, false
	}

	probes := dvaTransientProbes()
	ignoredPaths, decided := gitCheckIgnore(configDir, probes)
	if !decided {
		return false, false
	}
	for _, probe := range probes {
		if !ignoredPaths[probe] {
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
// This still stops short of implementing gitignore semantics: globs (".sb/*"), non-root
// anchoring ("**/.sb/"), and .git/info/exclude stay uninterpreted. Every one of those gaps
// makes DVA warn about a path that is in fact ignored, which is the harmless direction — an
// extra warning, never a silently committed marker.
func isDvaIgnored(content string) bool {
	lines := strings.Split(content, "\n")
	for _, prefix := range ancestorsAndSelf(config.DotDirName) {
		if lastMatchExcludes(lines, pathSpellings(prefix)) {
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
//     would. What creates the directory is the pid, log, module-cache and source-cache writers
//     under internal/lifecycle and internal/config, and the invocation after one of those is
//     where the warning has something to point at.
//   - Already ignored: isDvaIgnored settles it, ancestors and negations included.
//
// `dva doctor` deliberately keeps the unconditional form (checkGitignoreStatus, doctor.go:181),
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
	fmt.Fprintf(os.Stderr, "         Run 'dva doctor --fix' to auto-fix or add '%s/' to .gitignore manually.\n\n", config.DotDirName)
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
	notIgnored := fmt.Sprintf("%s/ is NOT ignored in .gitignore", config.DotDirName)
	addTheRule := fmt.Sprintf("Add '%s/' to .gitignore to avoid committing transient state", config.DotDirName)

	// git is asked before the file is read, because it applies rules this function cannot
	// see — .git/info/exclude, core.excludesFile, nested .gitignore files — so a tree with no
	// .gitignore at all may already be ignoring every marker correctly. Reading the file
	// first reports that tree as failing and offers to fix what is not broken.
	if ignored, decided := dvaTransientsIgnored(configDir); decided {
		if ignored {
			r.Passed = true
			return r
		}
		return failGitignore(r, configDir, notIgnored, addTheRule)
	}

	gitignorePath := filepath.Join(configDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return failGitignore(r, configDir,
				fmt.Sprintf("no .gitignore here, so %s/ is not ignored", config.DotDirName),
				fmt.Sprintf("Create .gitignore and add '%s/' to avoid committing transient state", config.DotDirName))
		}
		r.Passed = false
		r.Finding = fmt.Sprintf(".gitignore could not be read, so %s/ cannot be confirmed ignored: %v", config.DotDirName, err)
		return r
	}

	if isDvaIgnored(string(data)) {
		r.Passed = true
		return r
	}
	return failGitignore(r, configDir, notIgnored, addTheRule)
}
