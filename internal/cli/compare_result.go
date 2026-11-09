package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ScriptonBasestar/dva/internal/lifecycle"
	"github.com/ScriptonBasestar/dva/internal/output"
)

// Per-variant outcomes. A person reads them in the table and an agent branches on them in
// `--json`, so the set is closed: every variant of every compare action ends in exactly one.
const (
	outcomePlanned      = "planned"       // --dry-run: resolved, nothing created or started
	outcomeStarted      = "started"       // up started it in this call
	outcomeSkipped      = "skipped"       // up found it already running from the same source and commit
	outcomeFailed       = "failed"        // up or down tried it and it did not finish
	outcomeNotAttempted = "not_attempted" // an earlier failure stopped the run before this variant
	outcomeStopped      = "stopped"       // down stopped it and reclaimed its checkout
	outcomeObserved     = "observed"      // status read it
)

// Error codes of the compare commands. They ride on the existing failure envelope's optional
// `error.code` key; a consumer that ignores the key sees the same envelope as for any failure.
//   - compare_preflight_failed: nothing was created or started.
//   - compare_partial_failure: up started something (the shared entries or a variant) and
//     stopped before finishing. What it started is still running.
//   - compare_down_failed: down left at least one variant, or the shared entries, in place.
const (
	codeComparePreflight = "compare_preflight_failed"
	codeComparePartial   = "compare_partial_failure"
	codeCompareDown      = "compare_down_failed"
)

// compareObserved is what one start or status call saw: the entries' status and the plan's
// endpoints, in the shapes `dva up <plan> --json` already uses.
type compareObserved struct {
	Status    *lifecycle.AggregatedStatus `json:"status,omitempty"`
	Endpoints []planEndpointOutput        `json:"endpoints,omitempty"`
}

// compareSharedResult is the shared entries' part of a result: started once from the calling
// checkout.
type compareSharedResult struct {
	Entries []string `json:"entries"`
	Outcome string   `json:"outcome"`
	Error   string   `json:"error,omitempty"`
	compareObserved
}

// compareDrift reports that a variant's source now names another commit than the one the
// variant was pinned to. The variant keeps running the pinned commit; only a `down` and a new
// `up` re-pins it.
type compareDrift struct {
	Commit string `json:"commit,omitempty"`
	Error  string `json:"error,omitempty"`
}

// compareVariantResult is one variant's row of a result.
type compareVariantResult struct {
	Name      string         `json:"name"`
	Mode      string         `json:"mode,omitempty"`
	Source    string         `json:"source,omitempty"`
	Commit    string         `json:"commit,omitempty"`
	Path      string         `json:"path,omitempty"`
	ConfigDir string         `json:"config_dir,omitempty"`
	Ports     map[string]int `json:"ports,omitempty"`
	Outcome   string         `json:"outcome"`
	Error     string         `json:"error,omitempty"`
	Stale     bool           `json:"stale,omitempty"`
	Reasons   []string       `json:"stale_reasons,omitempty"`
	Drift     *compareDrift  `json:"drift,omitempty"`
	Pids      map[string]int `json:"pids,omitempty"`
	LogPaths  []string       `json:"logs,omitempty"`
	compareObserved
}

// compareResult is the whole document of an up, status or down call. `next` holds the exact
// commands to run after it; on a failure they are the way back.
type compareResult struct {
	Action   string                 `json:"action"`
	Compare  string                 `json:"compare"`
	Plan     string                 `json:"plan"`
	DryRun   bool                   `json:"dry_run"`
	Shared   *compareSharedResult   `json:"shared,omitempty"`
	Variants []compareVariantResult `json:"variants"`
	Next     map[string]string      `json:"next,omitempty"`
}

// variant returns the row named name, or nil.
func (r *compareResult) variant(name string) *compareVariantResult {
	for i := range r.Variants {
		if r.Variants[i].Name == name {
			return &r.Variants[i]
		}
	}
	return nil
}

// compareError is a failed compare call. It carries the whole result so the failure envelope
// can hold the per-variant outcomes and the commands to run next, not just a message.
type compareError struct {
	code    string
	message string
	result  *compareResult
}

func (e *compareError) Error() string { return e.message }

// Code is the frozen machine code for `error.code`.
func (e *compareError) Code() string { return e.code }

// FailureResult is the document the failure envelope carries under `error.result`.
func (e *compareError) FailureResult() any { return e.result }

func compareFailure(code string, result *compareResult, format string, args ...any) error {
	return &compareError{code: code, message: fmt.Sprintf(format, args...), result: result}
}

// emitCompareResult prints result for the active output mode. It is the only writer of a
// compare document: on success it is the command's output, and on failure the same table goes
// to stderr so the failure envelope stays the single JSON document on stdout.
func emitCompareResult(result *compareResult, failed bool) error {
	if jsonOutput {
		if failed {
			return nil // the failure envelope carries the result
		}
		return output.PrintJSON(result)
	}
	var w io.Writer = os.Stdout
	if failed {
		w = os.Stderr
	}
	printCompareTable(w, result)
	return nil
}

func printCompareTable(w io.Writer, r *compareResult) {
	title := r.Action
	if r.DryRun {
		title += " (dry-run)"
	}
	_, _ = fmt.Fprintf(w, "[compare %s] %s plan=%s\n", r.Compare, title, r.Plan)
	if r.Shared != nil {
		_, _ = fmt.Fprintf(w, "shared %s: %s", strings.Join(r.Shared.Entries, ","), r.Shared.Outcome)
		if r.Shared.Error != "" {
			_, _ = fmt.Fprintf(w, " — %s", r.Shared.Error)
		}
		_, _ = fmt.Fprintln(w)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "VARIANT\tSOURCE\tCOMMIT\tPORTS\tOUTCOME\tURL")
	for _, v := range r.Variants {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\n", v.Name, v.Source, shortCommit(v.Commit),
			portsText(v.Ports), outcomeText(v), firstURL(v.Endpoints))
	}
	_ = tw.Flush()
	for _, v := range r.Variants {
		if v.Error != "" {
			_, _ = fmt.Fprintf(w, "  %s: %s\n", v.Name, v.Error)
		}
		for _, reason := range v.Reasons {
			_, _ = fmt.Fprintf(w, "  %s stale: %s\n", v.Name, reason)
		}
		if v.Drift != nil {
			_, _ = fmt.Fprintf(w, "  %s drift: %s\n", v.Name, driftText(v.Drift))
		}
	}
	keys := make([]string, 0, len(r.Next))
	for k := range r.Next {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "next %s: %s\n", k, r.Next[k])
	}
}

func outcomeText(v compareVariantResult) string {
	switch {
	case v.Stale:
		return v.Outcome + " (stale)"
	case v.Drift != nil:
		return v.Outcome + " (drift)"
	}
	return v.Outcome
}

func driftText(d *compareDrift) string {
	if d.Error != "" {
		return "source cannot be resolved: " + d.Error
	}
	return "source now names " + shortCommit(d.Commit)
}

func shortCommit(c string) string {
	if len(c) > 12 {
		return c[:12]
	}
	return c
}

func portsText(ports map[string]int) string {
	keys := make([]string, 0, len(ports))
	for k := range ports {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, ports[k]))
	}
	return strings.Join(parts, ",")
}

func firstURL(endpoints []planEndpointOutput) string {
	for _, e := range endpoints {
		if e.URL != "" {
			return e.URL
		}
	}
	return ""
}
