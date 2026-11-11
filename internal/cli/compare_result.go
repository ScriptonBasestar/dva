package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/lifecycle"
	"github.com/ScriptonBasestar/dva/internal/output"
)

// Per-variant outcomes. A person reads them in the table and an agent branches on them in
// `--json`, so the set is closed: every variant of every compare action ends in exactly one.
const (
	outcomePlanned      = "planned"       // --dry-run: resolved, nothing created or started
	outcomeStarted      = "started"       // up started it in this call
	outcomeSkipped      = "skipped"       // up found it already running (a variant: from the same source and commit)
	outcomeFailed       = "failed"        // up or down tried it and it did not finish
	outcomeNotAttempted = "not_attempted" // an earlier failure stopped the run before this variant
	outcomeStopped      = "stopped"       // down stopped it and reclaimed its checkout
	outcomeObserved     = "observed"      // status read it
	outcomeKept         = "kept"          // down left the shared entries running: another comparison uses them
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
// checkout. Kept, on a down, maps each shared entry left running to the other comparisons
// whose records still use it.
type compareSharedResult struct {
	Entries []string            `json:"entries"`
	Outcome string              `json:"outcome"`
	Error   string              `json:"error,omitempty"`
	Kept    map[string][]string `json:"kept,omitempty"`
	compareObserved
}

// fanOutSharedEndpoints replaces each endpoint of the shared result whose URL depends on a
// variant's vars with one copy per variant, interpolated with that variant's vars and named by
// it. The shared entries run without the variants' vars, so such an endpoint (one that names a
// per_variant port) has no single value: reported once, it would show an unsubstituted or
// default URL. Endpoints that read the same for every variant stay as they are.
func fanOutSharedEndpoints(ctx context.Context, dir, plan string, shared *compareSharedResult, variants []compare.Variant) error {
	if shared == nil || len(shared.Endpoints) == 0 || len(variants) == 0 {
		return nil
	}
	per := make([]map[string]string, len(variants)) // variant i → endpoint name → URL
	for i := range variants {
		eps, err := compareHost.Endpoints(ctx, dir, plan, variantVars(&variants[i]))
		if err != nil {
			return fmt.Errorf("variant %s: %w", variants[i].Name, err)
		}
		per[i] = make(map[string]string, len(eps))
		for _, e := range eps {
			per[i][e.Name] = e.URL
		}
	}
	varies := map[string]bool{}
	for _, e := range shared.Endpoints {
		for i := range variants {
			if u, ok := per[i][e.Name]; ok && u != e.URL {
				varies[e.Name] = true
			}
		}
	}
	if len(varies) == 0 {
		return nil
	}
	out := make([]planEndpointOutput, 0, len(shared.Endpoints))
	for _, e := range shared.Endpoints {
		if !varies[e.Name] {
			out = append(out, e)
			continue
		}
		for i, v := range variants {
			copied := e
			copied.Variant = v.Name
			if u, ok := per[i][e.Name]; ok {
				copied.URL = u
			}
			out = append(out, copied)
		}
	}
	shared.Endpoints = out
	return nil
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
		kept := make([]string, 0, len(r.Shared.Kept))
		for e := range r.Shared.Kept {
			kept = append(kept, e)
		}
		sort.Strings(kept)
		for _, e := range kept {
			_, _ = fmt.Fprintf(w, "  shared %s kept: in use by comparison %s\n", e, strings.Join(r.Shared.Kept[e], ","))
		}
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "VARIANT\tSOURCE\tCOMMIT\tPORTS\tOUTCOME\tREADY\tPID\tLOG\tURL")
	for _, v := range r.Variants {
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n", v.Name, v.Source, shortCommit(v.Commit),
			portsText(v.Ports), outcomeText(v), readinessText(v.Status), pidsText(v.Pids),
			dashIfEmpty(strings.Join(v.LogPaths, ",")), firstURL(v.Endpoints))
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

// readinessText summarizes whether the variant is ready to use. It never claims "ready" unless
// every service in the status is running and healthy (or running with no health check), and
// every health check passes: a starting, unhealthy, stopped or exited service, or a failing
// check, is "not ready". "error" means an entry could not be queried; "-" means nothing was
// observed (no status, or no services and no health checks to judge).
func readinessText(st *lifecycle.AggregatedStatus) string {
	if st == nil {
		return "-"
	}
	seen, bad, broken := false, false, false
	for _, e := range st.Entries {
		if e.Error != "" {
			broken = true
		}
		for _, h := range e.Health {
			seen = true
			bad = bad || !h.Ready
		}
		for _, s := range e.Services {
			seen = true
			if s.State != "running" || (s.Health != "" && s.Health != "healthy") {
				bad = true
			}
		}
	}
	switch {
	case bad:
		return "not ready"
	case broken:
		return "error"
	case seen:
		return "ready"
	}
	return "-"
}

func pidsText(pids map[string]int) string {
	keys := make([]string, 0, len(pids))
	for k := range pids {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", k, pids[k]))
	}
	return dashIfEmpty(strings.Join(parts, ","))
}

func dashIfEmpty(s string) string {
	if s == "" {
		return "-"
	}
	return s
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
