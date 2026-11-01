package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// EnvInputKind classifies why one declared env file could not contribute values.
//
// These kinds are the JSON wire values as well as the switch keys: adding one is a
// contract change, not an implementation detail. TASK-247 froze the first three;
// EnvInputUnresolvedPath is the v0.4.3 addition, recorded in CHANGELOG.
type EnvInputKind string

const (
	// EnvInputMissingRequired is a declaration marked required whose file is absent.
	EnvInputMissingRequired EnvInputKind = "missing_required"
	// EnvInputInaccessible is a file that exists but cannot be stat'd, opened or read.
	// Required is irrelevant here: an unreadable optional file is still a failure,
	// because absence is a decision the author made and unreadability is not.
	EnvInputInaccessible EnvInputKind = "inaccessible"
	// EnvInputMalformed is a scanner error, or a non-blank non-comment line that is
	// not a dotenv assignment. Only the line number is ever reported.
	EnvInputMalformed EnvInputKind = "malformed"
	// EnvInputUnresolvedPath is a path still holding a ${VAR} reference nothing
	// defines. Required is irrelevant: the declaration names no file at all, so
	// "optional and absent" cannot be concluded, and opening the literal would
	// report the wrong cause.
	EnvInputUnresolvedPath EnvInputKind = "unresolved_path"
)

// EnvInputState is the aggregate verdict over every declaration of one owner.
type EnvInputState string

const (
	// EnvInputComplete means every declaration loaded, or none was declared.
	EnvInputComplete EnvInputState = "complete"
	// EnvInputCompleteWithSkips means the only non-loads were optional missing files.
	// This is a healthy state and produces no failure diagnostic anywhere.
	EnvInputCompleteWithSkips EnvInputState = "complete_with_skips"
	// EnvInputIncomplete means at least one declaration failed. No env-file-derived
	// value from ANY declaration may be applied in this state.
	EnvInputIncomplete EnvInputState = "incomplete"
)

// EnvInputStatus is the per-declaration outcome.
type EnvInputStatus string

const (
	EnvInputLoaded  EnvInputStatus = "loaded"
	EnvInputSkipped EnvInputStatus = "skipped"
	EnvInputFailed  EnvInputStatus = "failed"
)

// EnvInputFailure is the JSON-visible shape of one failed declaration.
//
// File is the path exactly as configured. A relative declaration is never expanded
// to a local absolute path for display, so diagnostics do not leak the checkout
// location; only an author-written absolute path appears absolute.
type EnvInputFailure struct {
	File     string       `json:"file"`
	Required bool         `json:"required"`
	Kind     EnvInputKind `json:"kind"`

	// line carries the offending line for EnvInputMalformed. It is unexported and
	// absent from JSON on purpose: it reaches the user only through Reason.
	line int
	// refs carries the unresolved references for EnvInputUnresolvedPath. They are
	// names written in dva.yml, never env-file content.
	refs []string
}

// Reason renders the stable, content-free explanation for one failure. These
// strings are part of the frozen contract; callers must not build their own.
func (f EnvInputFailure) Reason() string {
	switch f.Kind {
	case EnvInputMissingRequired:
		return "missing required file"
	case EnvInputInaccessible:
		return "cannot read file"
	case EnvInputMalformed:
		if f.line > 0 {
			return fmt.Sprintf("invalid dotenv syntax at line %d", f.line)
		}
		return "invalid dotenv syntax"
	case EnvInputUnresolvedPath:
		return "unresolved variable reference " + strings.Join(f.refs, ", ") + " in path"
	}
	return string(f.Kind)
}

// EnvInputEntry is one declaration and what became of it, in declaration order.
type EnvInputEntry struct {
	File     string
	Required bool
	Status   EnvInputStatus
	Kind     EnvInputKind
	line     int

	refs []string

	// UnresolvedValues counts this entry's keys whose merged value still holds a
	// reference. It is set by ApplyEnvFiles only, and is a count because a value
	// that merely contains `$` must not reach a diagnostic.
	UnresolvedValues int

	// vars holds this entry's parsed assignments while the report is being built.
	// They are merged only if the whole report turns out loadable.
	vars map[string]string
}

// Reason renders this entry's stable explanation, or "" when it did not fail. It
// exists so a diagnostic renderer outside this package can report one declaration
// at a time without reconstructing the failure list and matching on File.
func (e EnvInputEntry) Reason() string {
	if e.Status != EnvInputFailed {
		return ""
	}
	return EnvInputFailure{File: e.File, Required: e.Required, Kind: e.Kind, line: e.line, refs: e.refs}.Reason()
}

// EnvInputReport is the atomic verdict for one owner's env-file declarations.
//
// It is produced before any value is applied, which is what makes "discard
// everything on any failure" expressible at all: once MergeVars has run there is
// no way to un-merge an earlier file's values.
type EnvInputReport struct {
	State   EnvInputState
	Entries []EnvInputEntry
}

// Incomplete reports whether any declaration failed. This is the single predicate
// every route policy branches on.
func (r *EnvInputReport) Incomplete() bool {
	return r != nil && r.State == EnvInputIncomplete
}

// Failures returns the failed declarations in declaration order.
func (r *EnvInputReport) Failures() []EnvInputFailure {
	if r == nil {
		return nil
	}
	var out []EnvInputFailure
	for _, e := range r.Entries {
		if e.Status == EnvInputFailed {
			out = append(out, EnvInputFailure{File: e.File, Required: e.Required, Kind: e.Kind, line: e.line, refs: e.refs})
		}
	}
	return out
}

// Message renders the human diagnostic shared by every fail-closed route:
//
//	environment inputs are incomplete
//	  - <configured-path>: <stable-reason>
//
// It returns "" when the report is loadable, so callers can treat the empty
// string as "nothing to say".
func (r *EnvInputReport) Message() string {
	failures := r.Failures()
	if len(failures) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("environment inputs are incomplete")
	for _, f := range failures {
		fmt.Fprintf(&b, "\n  - %s: %s", f.File, f.Reason())
	}
	return b.String()
}

// Err returns the fail-closed error for execution, teardown and observation
// routes, or nil when the inputs are loadable.
func (r *EnvInputReport) Err() error {
	if msg := r.Message(); msg != "" {
		return fmt.Errorf("%s", msg)
	}
	return nil
}

// InspectEnvFiles evaluates every declaration without touching any environment.
//
// Every declaration is examined even after one fails, because the diagnostic
// contract promises the author the complete list in declaration order rather
// than a first-error abort that hides the rest.
//
// Paths are expanded against env, so only layers merged before env files are in
// scope: another env file's values are not, and neither is a plan's environment.
// A nil env means the OS environment alone.
func InspectEnvFiles(envFileConfig any, basePath string, env *Environment) *EnvInputReport {
	if env == nil {
		env = NewEnvironment(nil, basePath, basePath)
	}
	report := &EnvInputReport{State: EnvInputComplete}
	skipped := false

	for _, f := range normalizeEnvFileConfig(envFileConfig) {
		entry := EnvInputEntry{File: f.Path, Required: f.Required}

		if refs := unresolvedPathRefs(f.Path, env); len(refs) > 0 {
			entry.Status = EnvInputFailed
			entry.Kind = EnvInputUnresolvedPath
			entry.refs = refs
			report.Entries = append(report.Entries, entry)
			continue
		}

		path := env.Interpolate(f.Path)
		if !filepath.IsAbs(path) {
			path = filepath.Join(basePath, path)
		}

		handle, err := os.Open(path)
		if err != nil {
			if os.IsNotExist(err) && !f.Required {
				entry.Status = EnvInputSkipped
				skipped = true
				report.Entries = append(report.Entries, entry)
				continue
			}
			entry.Status = EnvInputFailed
			if os.IsNotExist(err) {
				entry.Kind = EnvInputMissingRequired
			} else {
				entry.Kind = EnvInputInaccessible
			}
			report.Entries = append(report.Entries, entry)
			continue
		}

		vars, line, err := parseEnvFileStrict(handle)
		_ = handle.Close()
		if err != nil {
			entry.Status = EnvInputFailed
			// A scanner failure mid-file is a read failure, not a syntax failure;
			// only a rejected line carries a line number.
			if line > 0 {
				entry.Kind = EnvInputMalformed
				entry.line = line
			} else {
				entry.Kind = EnvInputInaccessible
			}
			report.Entries = append(report.Entries, entry)
			continue
		}

		entry.Status = EnvInputLoaded
		entry.vars = vars
		report.Entries = append(report.Entries, entry)
	}

	for _, e := range report.Entries {
		if e.Status == EnvInputFailed {
			report.State = EnvInputIncomplete
			return report
		}
	}
	if skipped {
		report.State = EnvInputCompleteWithSkips
	}
	return report
}

// ApplyEnvFiles inspects every declaration and, only if all of them are loadable,
// merges them into env in declaration order and interpolates.
//
// On any failure env is left byte-identical to what the caller passed in. That is
// the whole point: a later file failing must not leave an earlier file's values
// behind for a caller that decides to continue anyway.
func ApplyEnvFiles(envFileConfig any, basePath string, env *Environment) *EnvInputReport {
	report := InspectEnvFiles(envFileConfig, basePath, env)
	if report.Incomplete() {
		return report
	}
	for _, entry := range report.Entries {
		if entry.Status == EnvInputLoaded {
			env.MergeVars(entry.vars)
		}
	}
	interpolateEnvVars(env)
	countUnresolvedValues(report, env)
	return report
}

// countUnresolvedValues records, per entry, the keys still holding a reference after
// the repair pass. The runtime keeps those values verbatim (see interpolateWith), so
// this is diagnostic only. A key the OS defines is skipped: its value is not the
// declaration's, and MergeVars never looked at that.
func countUnresolvedValues(report *EnvInputReport, env *Environment) {
	for i := range report.Entries {
		entry := &report.Entries[i]
		for key := range entry.vars {
			if _, inOS := os.LookupEnv(key); inOS {
				continue
			}
			if len(findVarRefs(env.Vars[key])) > 0 {
				entry.UnresolvedValues++
			}
		}
	}
}

// unresolvedRefs returns the references in value that env cannot satisfy. A defined
// name expands to a placeholder rather than its value, so a value that happens to
// contain `$` is not mistaken for a reference. Emptiness is preserved because `:-`
// branches on it: `${A:-${B}}` needs B only when A is empty.
func unresolvedRefs(value string, env *Environment) []string {
	return findVarRefs(checkExpand(value, env))
}

// unresolvedPathRefs is unresolvedRefs for an env_file path. Syntax the expander does
// not implement (`${A:?x}`) survives expansion and would be opened literally, and a
// path expanding to nothing would open the config directory, so both count as
// unresolved. The real expansion is checked too, because a variable's value can carry
// the same `${` that the placeholder expansion hides.
func unresolvedPathRefs(path string, env *Environment) []string {
	expanded := checkExpand(path, env)
	for i := strings.Index(expanded, "${"); i >= 0 && i < len(expanded); {
		if _, end, ok := parseBracedRef(expanded, i); ok {
			next := strings.Index(expanded[end:], "${")
			if next < 0 {
				break
			}
			i = end + next
			continue
		}
		return append(findVarRefs(expanded[:i]), leftoverRef(expanded[i:]))
	}
	if refs := findVarRefs(expanded); len(refs) > 0 {
		return refs
	}
	actual := env.Interpolate(path)
	if ref := leftoverRef(actual); ref != "" {
		return []string{ref}
	}
	if strings.Contains(path, "$") {
		if clean := filepath.Clean(actual); clean == "." || clean == "/" {
			return []string{path}
		}
		if ref := emptyRootPrefix(path, env); ref != "" {
			return []string{ref}
		}
	}
	return nil
}

// emptyRootPrefix returns the leading references, or those inside a default taken by one,
// that expand to nothing right before a `/` and would silently turn `${E}/.env` into a
// root-based path. A `/` the author wrote into a default, as in `${E}${U:-/x}`, is deliberate
// and stays allowed. It returns "" when the path has no such prefix.
func emptyRootPrefix(path string, env *Environment) string {
	rest := path
	for strings.HasPrefix(rest, "$") {
		end := 1 + len(scanVarName(rest[1:]))
		if strings.HasPrefix(rest, "${") {
			ref, braced, ok := parseBracedRef(rest, 0)
			if !ok {
				return ""
			}
			if v, set := env.lookup(ref.name); ref.takesDefault(v, set) {
				if inner := emptyRootPrefix(ref.def, env); inner != "" {
					return inner
				}
			}
			end = braced
		}
		if end == 1 || env.Interpolate(rest[:end]) != "" {
			return ""
		}
		rest = rest[end:]
	}
	if strings.HasPrefix(rest, "/") {
		return path[:len(path)-len(rest)]
	}
	return ""
}

// leftoverRef returns the first `${...}` fragment in value, or "" when there is none.
func leftoverRef(value string) string {
	start := strings.Index(value, "${")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(value); i++ {
		switch value[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return value[start : i+1]
			}
		}
	}
	return value[start:]
}

func checkExpand(value string, env *Environment) string {
	return interpolateWith(value, func(name string) (string, bool) {
		v, ok := env.lookup(name)
		if !ok {
			return "", false
		}
		if v == "" {
			return "", true
		}
		return "x", true
	})
}

// EnvFileScope builds the layers env_file paths may reference, without applying any
// env file, so a check that does not load env files sees the runtime's answer.
func (c *Config) EnvFileScope() *Environment {
	env := NewEnvironment(c.Vars, c.FileDir(), c.FileDir())
	env.MergeVars(c.Environment)
	return env
}
