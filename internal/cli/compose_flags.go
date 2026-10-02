package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/config"
)

// parseDvaFlags extracts --mode/-M, --env/-E, --tags/-T, and --exclude-tags from args.
// It also consumes root persistent --dry-run/--debug/--json because callers set
// DisableFlagParsing and cobra therefore never parses them for them. --debug/--json
// are also pre-parsed in PersistentPreRun for logger.Init; stripping them here
// prevents them from being treated as entry/service names.
func parseDvaFlags(args []string) (mode, env string, includeTags, excludeTags []string, filtered []string, err error) {
	// --dry-run is handled in the switch below, not by a consumeDryRunFlag pre-pass. The
	// pre-pass was a second walk that did the same thing, and after TASK-145 it would have
	// been a walk with its own idea of where the `--` terminator is.
	end := dvaFlagEnd(args)

	// A malformed boolean value (`--debug=notabool`) is rejected here rather than passed down
	// in filtered. It used to fall through "for the caller's own rejectUnknownFlags to name",
	// which most call sites do not have; `dva build` instead appended it to docker's argv. No
	// caller can take over this job, because a passthrough command must forward the flags it
	// does not recognise — this is the last code that knows `--debug` is DVA's. TASK-172.
	//
	// The first bad flag wins: reporting one is what the user has to fix first. That is what
	// the `if err == nil` guards below are for, and it is pinned by
	// TestParseDvaFlags_FirstBadFlagIsReported — a review deleted both guards, making the
	// last flag win, and nothing failed until that test existed. The loop then runs to the
	// end rather than returning early because a closure has no return to take — not because
	// anything reads the values it keeps filling in. Every caller checks err before touching
	// any other return value, so those values are never observed.
	//
	// Both sentences carried stale call-site counts until TASK-213. Neither was ever reread,
	// which is TASK-208's subject exactly. They are stated as properties now; the counts are one
	// command away and belong in a commit message rather than in a comment that outlives it:
	//   grep -n 'parseDvaFlags(args)' internal/cli/*.go | grep -v _test  # callers: 6 today
	//   grep -n 'rejectUnknownFlags(' internal/cli/*.go | grep -v _test | grep -v 'func '  # call sites: 2, upCmd + restartCmd
	//
	// The second command carried `# 2` for one commit while returning 3, because it also
	// matched the func declaration in selectors.go. A command written to retire a stale
	// count is worth running once before it ships.
	takeBool := func(name, value string, hasValue bool, target *bool) {
		if v, ok := flagBoolValue(value, hasValue); ok {
			*target = v
			return
		}
		if err == nil {
			err = fmt.Errorf("invalid boolean value %q for %s", value, name)
		}
	}

	// A value-taking flag with nothing to take is an error, reported here for the same
	// reason takeBool reports a malformed boolean: only this code knows a value was
	// required. flagValue stays a bool — what it cannot supply is the flag's name, and it
	// has no err to set. Note that its own doc comment used to justify the silence by
	// saying the helper is also used where taking the next token is optional; it is not.
	// Before this change flagValue had exactly four callers and they were the four cases
	// below, so the helper's neutrality was prospective rather than a current constraint.
	// It now has exactly one — this closure — which is the point of the change: the four
	// cases no longer touch it. Past tense on purpose. TASK-208 exists because five
	// comments went on quoting a call-site count a refactor had already changed, and the
	// first draft of this one said "grep says flagValue has exactly four callers" in the
	// present tense, in the very commit that made it one. TASK-211.
	//
	// Until then every case ignored ok=false, and a recognised flag is never appended to
	// filtered, so the token vanished: `dva restart --mode` ran the whole stack and
	// reported success — the widest possible result for someone who typed a narrowing
	// flag. Same for --env, --tag and --exclude-tag.
	//
	// "Nothing to take" includes a flag sitting just before the terminator, because
	// dvaFlagEnd puts end at the first `--`: in `--mode --` the flag is last as far as
	// flagValue is concerned even though a token follows. Both spellings have to error or
	// the fix closes one door and leaves its twin open — that pair is what TASK-207's
	// review was looking at when it filed this.
	// An empty value is refused here too, and for the same harm rather than for symmetry.
	// `--mode=` reaches flagValue's hasValue branch, which reports ok=true because a value
	// was supplied — so before TASK-213 mode was set to "", which is precisely what no
	// --mode at all leaves behind, applyDefaultMode then filled in the default, and the
	// narrowing flag produced the widest possible run at rc=0. `--tag=` is the mirror:
	// includeTags becomes [""], which matches nothing, so the run is empty and still
	// reported as success. Both spellings of emptiness are covered — `--mode=` through the
	// hasValue branch and `--mode ""` through the next-token branch — because a fix aimed
	// at the `=` spelling alone leaves its twin open, the same way TASK-211 had to take
	// `--mode --` along with `--mode`.
	//
	// Distinct wording on purpose: the user did supply a value, so "requires a value" would
	// describe a different mistake than the one they made. It also keeps the two branches
	// separable by test.
	//
	// Returning ok=false without consuming n means the empty next token in `--mode ""` is
	// The rejection paths return the token count they consumed, and the cases below advance
	// by it whether or not the value was accepted. That is not symmetry either: a rejected
	// `--mode ""` used to leave i where it was, so the loop re-read the empty token and
	// appended it to filtered — the value of a recognised flag reaching a passthrough
	// command's argv. It was unreachable, because err is set and every non-test caller of
	// parseDvaFlags (upCmd, teardownCommon, downCmd, stopCmd, restartCmd, buildCmd) checks it
	// on the line immediately after the call. "Unreachable" was the first draft of this
	// comment and a review was right to call it a latent invariant violation rather than a
	// property: it holds because of what six callers happen to do next, and buildCmd is one
	// reorder away from forwarding that "" to docker. flagValue returns consumed=0 on the
	// nothing-to-take branch, so advancing unconditionally is a no-op there and the fix costs
	// one line per case.
	takeValue := func(name, value string, hasValue bool, i int) (string, int, bool) {
		if !hasValue && i+1 < end && isRecognizedDVAFlagToken(args[i+1]) {
			if err == nil {
				err = fmt.Errorf("%s requires a value, got the flag %s", name, args[i+1])
			}
			return "", 1, false
		}
		v, n, ok := flagValue(args, i, end, value, hasValue)
		if !ok {
			if err == nil {
				err = fmt.Errorf("%s requires a value", name)
			}
			return v, n, ok
		}
		if v == "" {
			if err == nil {
				err = fmt.Errorf("%s requires a non-empty value", name)
			}
			return "", n, false
		}
		if strings.TrimSpace(v) == "" {
			if err == nil {
				err = fmt.Errorf("%s requires a non-blank value, got %q", name, v)
			}
			return "", n, false
		}
		return v, n, ok
	}

	// takeList is takeValue plus the rule the comma-separated flags need, and it exists
	// because the check above sits on the wrong side of the split to see the whole defect.
	// `--exclude-tag=,` is one character, so it passes as non-empty, and strings.Split turns
	// it into ["", ""] — two tags nothing carries. For --exclude-tag, matching nothing means
	// excluding nothing, so that spelling bounced the entire stack at rc=0: the same harm
	// TASK-211 and the check above were filed against, reached one character past the
	// spelling they refuse. `--tag=,` is the mirror and fails the other way, running nothing
	// and still exiting 0. Both were measured on the unfixed build, and the first draft of
	// TASK-213 examined only the second and called the family harmless.
	//
	// TrimSpace rather than == "" because a blank element is the same non-value written
	// differently — for the ~25 runes unicode.IsSpace calls space, which is the boundary
	// this draws and not the boundary "invisible" would draw. `--tag=<U+200B>` is a
	// zero-width space, is not IsSpace, and passes as a one-element list; it then matches
	// no declared tag, which lands it in TASK-214 with every other tag no entry carries.
	// Widening this check to "unprintable" would be a second, different rule about what a
	// tag may be made of, and it belongs with the code that validates tag names.
	//
	// Nothing is trimmed off the values that survive: an element with surrounding spaces is
	// a tag that will not match, which is an unknown-tag complaint rather than this one.
	// Rewriting the user's input on the way through would hide that from them.
	takeList := func(name, value string, hasValue bool, i int) ([]string, int, bool) {
		v, n, ok := takeValue(name, value, hasValue, i)
		if !ok {
			return nil, n, false
		}
		parts := strings.Split(v, ",")
		for _, p := range parts {
			if strings.TrimSpace(p) == "" {
				if err == nil {
					err = fmt.Errorf("%s requires non-empty tags, got %q", name, v)
				}
				return nil, n, false
			}
		}
		return parts, n, true
	}

	for i := 0; i < end; i++ {
		a := args[i]
		name, value, hasValue := splitFlagToken(a)
		switch name {
		case "--mode", "-M":
			v, n, ok := takeValue(name, value, hasValue, i)
			i += n
			if ok {
				mode = v
			}
		case "--env", "-E":
			v, n, ok := takeValue(name, value, hasValue, i)
			i += n
			if ok {
				env = v
			}
		case "--tag", "--tags", "-T":
			v, n, ok := takeList(name, value, hasValue, i)
			i += n
			if ok {
				includeTags = append(includeTags, v...)
			}
		case "--exclude-tag", "--exclude-tags":
			v, n, ok := takeList(name, value, hasValue, i)
			i += n
			if ok {
				excludeTags = append(excludeTags, v...)
			}
		// Callers set DisableFlagParsing, so cobra never parses the root
		// persistent --dry-run. Without this it falls through to filtered and
		// is read as an entry/service name, leaving dryRun false: `dva up
		// --dry-run` then executes for real.
		case "--dry-run":
			takeBool(name, value, hasValue, &dryRun)
		case "--debug":
			takeBool(name, value, hasValue, &debug)
		case "--json":
			takeBool(name, value, hasValue, &jsonOutput)
		default:
			filtered = append(filtered, a)
		}
	}
	// The terminator itself is kept, unlike in consumeRootPersistentFlags. This output is
	// still inside DVA, and the callers that reject unknown flags have always rejected a
	// stray `--` — dropping it here would newly accept it.
	filtered = append(filtered, args[end:]...)
	return
}

// consumeDryRunFlag strips --dry-run and reports whether it was set. Callers that also need
// --mode/--env/--tags use parseDvaFlags instead; this one exists for the paths that must
// leave every other flag where it is.
//
// It reports "not found" for an explicit `--dry-run=false` while still consuming the token.
// Every caller uses the result as `if found { dryRun = true }`, so absent and explicitly
// false already mean the same thing to them.
func consumeDryRunFlag(args []string) ([]string, bool) {
	end := dvaFlagEnd(args)
	filtered := make([]string, 0, len(args))
	found := false
	for i := range end {
		a := args[i]
		if name, value, hasValue := splitFlagToken(a); name == "--dry-run" {
			if v, ok := flagBoolValue(value, hasValue); ok {
				found = v
				continue
			}
		}
		filtered = append(filtered, a)
	}
	// Terminator kept: this feeds back into DVA (wrapWithHooks hands its result to the
	// built-in's own RunE, which parses flags again), so a later consumer still needs it.
	filtered = append(filtered, args[end:]...)
	return filtered, found
}

// applyEnv resolves and applies environment configuration from --env flag.
func applyEnv(e *config.Environment, c *config.Config, envName string) error {
	if envName == "" {
		return nil
	}
	ec, ok := c.Environments[envName]
	if !ok {
		available := make([]string, 0, len(c.Environments))
		for k := range c.Environments {
			available = append(available, k)
		}
		if len(available) == 0 {
			return fmt.Errorf("env '%s' not found. No environments defined in dva.yml under 'environments:'", envName)
		}
		return fmt.Errorf("env '%s' not found. Available: %s", envName, strings.Join(available, ", "))
	}
	fmt.Fprintf(os.Stderr, "[env: %s] %s\n", envName, ec.Description)
	if len(ec.Stack) > 0 {
		fmt.Fprintf(os.Stderr, "[env: %s] stack: %s\n", envName, strings.Join(ec.Stack, ", "))
	}
	if len(ec.Environment) > 0 {
		e.MergeVars(ec.Environment)
	}
	return nil
}

// resolvedMode holds the result of resolving a --mode flag against config modes.
type resolvedMode struct {
	Mode *config.ModeConfig
}

// applyDefaultMode returns the effective mode and whether it was applied from default_mode.
// It does NOT print any output — callers decide how to log.
func applyDefaultMode(c *config.Config, mode string) (string, bool) {
	if mode != "" || c.DefaultMode == "" {
		return mode, false
	}
	if _, ok := c.Modes[c.DefaultMode]; ok {
		return c.DefaultMode, true
	}
	fmt.Fprintf(os.Stderr, "[warn] default_mode '%s' not found in modes — starting all services\n", c.DefaultMode)
	return mode, false
}

// resolveMode looks up a mode name in config modes and returns resolved settings.
func resolveMode(c *config.Config, mode string) (*resolvedMode, error) {
	if mode == "" {
		return &resolvedMode{}, nil
	}

	m, ok := c.Modes[mode]
	if !ok {
		available := make([]string, 0, len(c.Modes))
		for k := range c.Modes {
			available = append(available, k)
		}
		if len(available) == 0 {
			return nil, fmt.Errorf("mode '%s' not found. No modes defined in dva.yml under 'modes:'", mode)
		}
		return nil, fmt.Errorf("mode '%s' not found. Available: %s", mode, strings.Join(available, ", "))
	}

	return &resolvedMode{
		Mode: &m,
	}, nil
}

// suggestProvision checks if a provision profile has been run before
// (via marker file in .sb/dva/) and prints a suggestion if not.
func suggestProvision(c *config.Config, provisionProfile string) {
	if provisionMarkerExists(c.FileDir(), provisionProfile) {
		return // already provisioned
	}

	if _, ok := c.Provision.Profiles[provisionProfile]; !ok {
		return
	}

	fmt.Fprintf(os.Stderr, "\n[hint] Provision profile '%s' has not been run yet.\n", provisionProfile)
	fmt.Fprintf(os.Stderr, "       Run: dva provision %s\n\n", provisionProfile)
}
