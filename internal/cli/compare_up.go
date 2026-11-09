package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/config"
)

// upVariant is one variant of an `up`, after its source was resolved and its ports assigned.
type upVariant struct {
	spec   variantSpec
	commit string
	ports  map[string]int
	// skip is set when the record shows this variant running from the same source and commit.
	skip bool
}

// runCompareUp starts the variants of a comparison: resolve and show everything first, fail
// before creating anything if a port is taken, then start the shared entries once and each
// variant in turn. It never rolls back; see compareFailure's result for the way back.
func runCompareUp(ctx context.Context, c *config.Config, name string, args []string, wait bool) error {
	cmp, err := compareDeclaration(c, name)
	if err != nil {
		return err
	}
	specs, err := parseVariantSpecs(name, cmp, args)
	if err != nil {
		return comparePreflight(nil, "compare %s: %v", name, err)
	}
	dir := c.FileDir()
	rec, recorded, err := loadRecordOrNew(dir, name)
	if err != nil {
		return comparePreflight(nil, "compare %s: %v", name, err)
	}

	variants, result, problems := resolveUp(ctx, dir, name, cmp, rec, specs)
	result.DryRun = dryRun
	if len(problems) == 0 {
		problems = checkPorts(variants)
	}
	if len(problems) > 0 {
		return comparePreflight(result, "compare %s: nothing was started:\n  %s", name, strings.Join(problems, "\n  "))
	}
	if dryRun {
		return emitCompareResult(result, false)
	}

	if !recorded {
		// Write the record before anything starts, so a failure in the shared entries still
		// leaves a comparison `dva compare down` can find and finish.
		if err := compare.WriteRecord(dir, rec); err != nil {
			return comparePreflight(result, "compare %s: %v", name, err)
		}
	}
	return startUp(ctx, dir, name, cmp, rec, variants, result, wait)
}

// loadRecordOrNew reads the record of name. A comparison never started gets an empty record;
// a record that exists but cannot be read is an error, never replaced.
func loadRecordOrNew(dir, name string) (rec *compare.Record, recorded bool, err error) {
	rec, err = compare.ReadRecord(dir, name)
	switch {
	case err == nil:
		return rec, true, nil
	case errors.Is(err, fs.ErrNotExist):
		return &compare.Record{Name: name}, false, nil
	}
	return nil, false, err
}

// resolveUp pins each source to a commit, decides which variants need starting, and assigns
// ports. It creates nothing. problems lists every reason the request cannot run, so one report
// names them all.
func resolveUp(ctx context.Context, dir, name string, cmp *config.CompareConfig, rec *compare.Record, specs []variantSpec) ([]*upVariant, *compareResult, []string) {
	result := &compareResult{Action: "up", Compare: name, Plan: cmp.Plan, Next: map[string]string{}}
	states := staleStates(ctx, dir, rec, cmp)

	var problems, fresh []string
	var variants []*upVariant
	for _, s := range specs {
		commit, err := compare.ResolveRef(ctx, dir, s.Source)
		if err != nil {
			problems = append(problems, fmt.Sprintf("variant %s: %v", s.Name, err))
			continue
		}
		v := &upVariant{spec: s, commit: commit}
		if prev := rec.Variant(s.Name); prev != nil {
			if prev.Mode != compare.ModeSnapshot || prev.Source != s.Source || prev.Commit != commit {
				problems = append(problems, fmt.Sprintf("variant %s is already started from %s at %s, not %s at %s; run `%s` first",
					s.Name, prev.Source, shortCommit(prev.Commit), s.Source, shortCommit(commit), downCommand(name, []string{s.Name})))
				continue
			}
			v.ports = prev.Ports
			v.skip = !states[s.Name] && variantRunning(ctx, cmp, prev)
		} else {
			fresh = append(fresh, s.Name)
		}
		variants = append(variants, v)
	}

	assigned, err := compare.AssignPorts(cmp, rec.Variants, fresh)
	if err != nil {
		problems = append(problems, err.Error())
	}
	for _, v := range variants {
		if p, ok := assigned[v.spec.Name]; ok {
			v.ports = p
		}
		path, _ := compare.SnapshotPath(dir, name, v.spec.Name)
		row := compareVariantResult{
			Name: v.spec.Name, Mode: compare.ModeSnapshot, Source: v.spec.Source,
			Commit: v.commit, Path: path, Ports: v.ports, Outcome: outcomePlanned,
		}
		if v.skip {
			row.Outcome = outcomeSkipped
		}
		result.Variants = append(result.Variants, row)
	}
	if len(cmp.Shared) > 0 {
		result.Shared = &compareSharedResult{Entries: cmp.Shared, Outcome: outcomePlanned}
	}
	return variants, result, problems
}

// staleStates maps each recorded variant to whether compare.Check found it stale. A record
// that cannot be checked marks nothing stale: the start that follows reports the real cause.
func staleStates(ctx context.Context, dir string, rec *compare.Record, cmp *config.CompareConfig) map[string]bool {
	stale := map[string]bool{}
	if len(rec.Variants) == 0 {
		return stale
	}
	states, err := compare.Check(ctx, dir, rec, cmp.PerVariant)
	if err != nil {
		return stale
	}
	for _, s := range states {
		stale[s.Name] = s.Stale
	}
	return stale
}

// checkPorts reports every assigned port that is taken or that the caller's environment would
// override. dva moves nothing: a port it cannot use is a failure, not a different port.
func checkPorts(variants []*upVariant) []string {
	var problems []string
	for _, v := range variants {
		if v.skip {
			continue
		}
		keys := make([]string, 0, len(v.ports))
		for k := range v.ports {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			port := v.ports[k]
			if ev, ok := os.LookupEnv(k); ok && ev != strconv.Itoa(port) {
				problems = append(problems, fmt.Sprintf("variant %s: environment sets %s=%s, which would override the port %d dva assigned; unset it", v.spec.Name, k, ev, port))
			}
			busy, err := compare.PortBusy(port)
			switch {
			case err != nil:
				problems = append(problems, fmt.Sprintf("variant %s: %v", v.spec.Name, err))
			case busy:
				problems = append(problems, fmt.Sprintf("variant %s: port %d (%s) is already in use", v.spec.Name, port, k))
			}
		}
	}
	return problems
}

// startUp starts the shared entries and then each variant, stopping at the first failure.
// Everything it started stays up; the returned error carries the result and the commands to
// continue or undo.
func startUp(ctx context.Context, dir, name string, cmp *config.CompareConfig, rec *compare.Record, variants []*upVariant, result *compareResult, wait bool) error {
	var failure string
	pending := 0
	for _, v := range variants {
		if !v.skip {
			pending++
		}
	}

	if result.Shared != nil && pending > 0 {
		obs, err := compareHost.Up(ctx, dir, cmp.Plan, nil, cmp.Shared, wait)
		if err != nil {
			result.Shared.Outcome, result.Shared.Error = outcomeFailed, err.Error()
			failure = fmt.Sprintf("shared entries %s failed to start: %v", strings.Join(cmp.Shared, ","), err)
		} else {
			result.Shared.Outcome = outcomeStarted
			result.Shared.compareObserved = *obs
		}
	} else if result.Shared != nil {
		result.Shared.Outcome = outcomeSkipped
	}

	for _, v := range variants {
		row := result.variant(v.spec.Name)
		switch {
		case v.skip:
			observeSkipped(ctx, cmp, rec, row)
		case failure != "":
			row.Outcome = outcomeNotAttempted
		default:
			if err := startVariant(ctx, dir, name, cmp, rec, v, row, wait); err != nil {
				row.Outcome, row.Error = outcomeFailed, err.Error()
				failure = fmt.Sprintf("variant %s failed to start: %v", v.spec.Name, err)
			}
		}
	}

	result.Next = upNext(name, rec, variants, result)
	if failure != "" {
		_ = emitCompareResult(result, true)
		return compareFailure(codeComparePartial, result, "compare %s: %s; what already started keeps running — see the next commands", name, failure)
	}
	return emitCompareResult(result, false)
}

// startVariant gives v its snapshot checkout, records it, and starts the per_variant entries.
// The record is written before the entries start, so a variant whose start fails is still
// something `down` can reclaim.
func startVariant(ctx context.Context, dir, name string, cmp *config.CompareConfig, rec *compare.Record, v *upVariant, row *compareVariantResult, wait bool) error {
	snap, _, err := compare.EnsureSnapshot(ctx, dir, name, v.spec.Name, v.spec.Source, rec.Variant(v.spec.Name))
	if err != nil {
		return err
	}
	snap.Ports = v.ports
	row.Path, row.ConfigDir = snap.Path, snap.ConfigDir
	replaceVariant(rec, snap)
	if err := compare.WriteRecord(dir, rec); err != nil {
		return err
	}
	if len(cmp.PerVariant) == 0 {
		row.Outcome = outcomeStarted
		return nil
	}
	obs, err := compareHost.Up(ctx, snap.ConfigDir, cmp.Plan, varsOf(v.ports), cmp.PerVariant, wait)
	if err != nil {
		return err
	}
	row.Outcome = outcomeStarted
	row.compareObserved = *obs
	return nil
}

// observeSkipped fills the row of a variant that was already running with what status sees
// now. A failed read keeps the outcome and says so on the row.
func observeSkipped(ctx context.Context, cmp *config.CompareConfig, rec *compare.Record, row *compareVariantResult) {
	v := rec.Variant(row.Name)
	if v == nil || len(cmp.PerVariant) == 0 {
		return
	}
	row.Path, row.ConfigDir = v.Path, v.ConfigDir
	obs, err := compareHost.Status(ctx, v.ConfigDir, cmp.Plan, varsOf(v.Ports), cmp.PerVariant)
	if err != nil {
		row.Error = "status unavailable: " + err.Error()
		return
	}
	row.compareObserved = *obs
}

func replaceVariant(rec *compare.Record, v compare.Variant) {
	for i := range rec.Variants {
		if rec.Variants[i].Name == v.Name {
			rec.Variants[i] = v
			return
		}
	}
	rec.Variants = append(rec.Variants, v)
}

// upNext returns the commands to run after an up. After a failure they are the way out: a
// down for exactly what this call started or left half-started, and the same up to retry it.
func upNext(name string, rec *compare.Record, variants []*upVariant, result *compareResult) map[string]string {
	failed := false
	var touched, retry []string
	retry = []string{"dva", "compare", "up", name}
	for _, v := range variants {
		retry = append(retry, shellWord(v.spec.Name+"="+v.spec.Source))
		row := result.variant(v.spec.Name)
		if row.Outcome == outcomeFailed || row.Outcome == outcomeNotAttempted {
			failed = true
		}
		if (row.Outcome == outcomeStarted || row.Outcome == outcomeFailed) && rec.Variant(v.spec.Name) != nil {
			touched = append(touched, v.spec.Name)
		}
	}
	if result.Shared != nil && result.Shared.Outcome == outcomeFailed {
		failed = true
	}
	if !failed {
		return map[string]string{"status": "dva compare status " + shellWord(name), "down": downCommand(name, nil)}
	}
	return map[string]string{"retry": strings.Join(retry, " "), "down": downCommand(name, touched)}
}

// variantRunning reports whether every per_variant entry of a recorded variant is up. A record
// alone proves only that dva began the variant: an `up` that failed after writing it leaves a
// consistent, non-stale record with nothing running, and that variant must be started again,
// not skipped. A status that cannot be read counts as not running; Up is idempotent, so
// starting an entry that was in fact up costs nothing.
func variantRunning(ctx context.Context, cmp *config.CompareConfig, v *compare.Variant) bool {
	if len(cmp.PerVariant) == 0 {
		return true // nothing runs per variant; the checkout is the whole variant
	}
	obs, err := compareHost.Status(ctx, v.ConfigDir, cmp.Plan, varsOf(v.Ports), cmp.PerVariant)
	if err != nil || obs.Status == nil {
		return false
	}
	for _, name := range cmp.PerVariant {
		up := false
		for _, e := range obs.Status.Entries {
			if e.Name != name || e.Error != "" {
				continue
			}
			for _, s := range e.Services {
				up = up || s.State == "running"
			}
		}
		if !up {
			return false
		}
	}
	return true
}
