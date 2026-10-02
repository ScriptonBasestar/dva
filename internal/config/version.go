package config

import (
	"fmt"
	"regexp"
	"strconv"
)

// MinScaffoldVersion is the compatibility floor `dva init` writes into a new
// dva.yml: the oldest DVA that understands the config init emits (the multi-runner
// stack model, 2e25daf). It is deliberately not Version. `version:` states what a
// config requires of its reader, not which binary produced it — scaffolding the
// running version would make every new config refuse to load on any older DVA,
// ratcheting the floor upward on each release. Raise this only when init starts
// emitting something an older DVA cannot parse.
//
// Unlike Version it is a const: no build may inject a different floor.
const MinScaffoldVersion = "0.1.44"

var (
	// Version is the current DVA version (bump manually for releases).
	Version = "0.3.0"
	// Commit is the git commit hash, injected at build time via ldflags.
	Commit = "dev"
	// BuildDate is the build timestamp, injected at build time via ldflags.
	BuildDate = "unknown"
)

// checkConfigVersion refuses configs that declare a minimum version newer than
// the running DVA binary. Empty version is allowed (no gate) — see version.go.
func checkConfigVersion(cfg *Config) error {
	if cfg == nil || cfg.Version == "" {
		return nil
	}
	ok, err := isVersionCompatible(cfg.Version)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("your dva version is `%s`, but config requires minimum version `%s`. Please upgrade dva", Version, cfg.Version)
	}
	return nil
}

// isVersionCompatible reports whether the running DVA satisfies required. It returns
// an error when either version is unreadable rather than treating it as 0.0.0.
func isVersionCompatible(required string) (bool, error) {
	req, err := parseVersion(required)
	if err != nil {
		return false, fmt.Errorf("%w. Omit `version:` entirely for no compatibility gate", err)
	}
	cur, err := parseVersion(Version)
	if err != nil {
		// Version is a var set by ldflags, so an unreadable one is a build defect
		// rather than a config defect. Say which of the two is at fault, and do not
		// suggest editing `version:` — no edit to the config can fix this one.
		return false, fmt.Errorf("this dva binary reports an unreadable version: %w. Reinstall dva or rebuild it with `make build`", err)
	}
	for i := range 3 {
		if cur[i] < req[i] {
			return false, nil
		}
		if cur[i] > req[i] {
			return true, nil
		}
	}
	return true, nil
}

// versionPattern is the accepted shape of a `version:` value. The optional patch
// segment is required for backward compatibility: an unquoted `version: 0.1` is a
// YAML number that yaml.v3 coerces into the Go string "0.1", so rejecting two
// segments would break configs that load today.
//
// schema.json's `version` property carries the same rule as a `pattern`, because the
// schema runs only under `dva validate` (Config.Validate has one call site) while this
// gate runs on every Load. TestVersionPatternMatchesSchema fails if the two diverge.
var versionPattern = regexp.MustCompile(`^v?(\d+)\.(\d+)(?:\.(\d+))?$`)

// parseVersion splits a version into its three segments.
//
// It reports an error instead of returning a zero value, because the previous
// implementation discarded fmt.Sscanf's error: anything unparseable stayed [0,0,0],
// isVersionCompatible then asked whether the running DVA was at least 0.0.0, and the
// answer was always yes. That did not merely tolerate junk — it defeated the gate with
// a typo. `O.2.0` (letter O) was written to require 0.2.0 and instead required nothing,
// silently, on every command.
func parseVersion(v string) ([3]int, error) {
	var parts [3]int
	m := versionPattern.FindStringSubmatch(v)
	if m == nil {
		return parts, malformedVersionError(v)
	}
	for i, segment := range m[1:] {
		if segment == "" {
			continue // the patch group is optional and defaults to 0
		}
		n, err := strconv.Atoi(segment)
		if err != nil {
			// The pattern guarantees digits, so the only way here is overflow.
			return [3]int{}, malformedVersionError(v)
		}
		parts[i] = n
	}
	return parts, nil
}

// malformedVersionError names the offending value and the shape expected, since the
// whole point of the check is that a misspelled version is otherwise invisible.
//
// It carries no remedy on purpose. parseVersion runs on two different values — the
// config's `version:` and the binary's own Version — and the remedy differs: one is
// fixed by editing the file, the other only by rebuilding. isVersionCompatible appends
// the right one at each call site.
func malformedVersionError(v string) error {
	return fmt.Errorf("version %q is not a version: expected MAJOR.MINOR.PATCH or MAJOR.MINOR, optionally v-prefixed (e.g. %q)", v, MinScaffoldVersion)
}
