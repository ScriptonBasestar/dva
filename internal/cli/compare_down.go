package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/config"
)

// runCompareDown stops variants and reclaims their snapshot checkouts. Per variant it stops the
// per_variant entries first and removes the checkout second, because RemoveSnapshot does not
// look for running processes. The shared entries stop last, and only when no variant of the
// comparison remains. A failure on one variant does not stop the others; whatever is left is
// named in the result with the command to finish it.
func runCompareDown(ctx context.Context, c *config.Config, name string, variantArgs []string) error {
	cmp, err := compareDeclaration(c, name)
	if err != nil {
		return err
	}
	dir := c.FileDir()
	rec, err := compare.ReadRecord(dir, name)
	if errors.Is(err, fs.ErrNotExist) {
		return comparePreflight(nil, "compare %s: nothing to take down — no record of an up", name)
	}
	if err != nil {
		return comparePreflight(nil, "compare %s: %v", name, err)
	}

	targets, err := downTargets(rec, variantArgs)
	if err != nil {
		return comparePreflight(nil, "compare %s: %v", name, err)
	}
	result := &compareResult{Action: "down", Compare: name, Plan: cmp.Plan, DryRun: dryRun, Next: map[string]string{}}
	for _, v := range targets {
		result.Variants = append(result.Variants, compareVariantResult{
			Name: v.Name, Mode: v.Mode, Source: v.Source, Commit: v.Commit,
			Path: v.Path, ConfigDir: v.ConfigDir, Ports: v.Ports, Outcome: outcomePlanned,
		})
	}
	takesShared := len(targets) == len(rec.Variants) && len(cmp.Shared) > 0
	if takesShared {
		result.Shared = &compareSharedResult{Entries: cmp.Shared, Outcome: outcomePlanned}
	}
	if dryRun {
		return emitCompareResult(result, false)
	}

	var failures []string
	for _, v := range targets {
		row := result.variant(v.Name)
		if err := downVariant(ctx, dir, cmp, rec, v); err != nil {
			row.Outcome, row.Error = outcomeFailed, err.Error()
			failures = append(failures, fmt.Sprintf("variant %s: %v", v.Name, err))
			continue
		}
		row.Outcome = outcomeStopped
		rec.Variants = removeVariant(rec.Variants, v.Name)
		if err := compare.WriteRecord(dir, rec); err != nil {
			row.Outcome, row.Error = outcomeFailed, err.Error()
			failures = append(failures, fmt.Sprintf("variant %s: %v", v.Name, err))
		}
	}

	if result.Shared != nil {
		if len(failures) > 0 {
			result.Shared.Outcome = outcomeNotAttempted
		} else if err := compareHost.Down(ctx, dir, cmp.Plan, nil, cmp.Shared); err != nil {
			result.Shared.Outcome, result.Shared.Error = outcomeFailed, err.Error()
			failures = append(failures, fmt.Sprintf("shared entries %s: %v", strings.Join(cmp.Shared, ","), err))
		} else {
			result.Shared.Outcome = outcomeStopped
		}
	}
	if len(failures) == 0 && len(rec.Variants) == 0 {
		if err := removeRecord(dir, name); err != nil {
			failures = append(failures, err.Error())
		}
	}

	if len(failures) > 0 {
		left := leftVariants(result)
		result.Next = map[string]string{"retry": downCommand(name, left), "status": "dva compare status " + shellWord(name)}
		_ = emitCompareResult(result, true)
		return compareFailure(codeCompareDown, result, "compare %s: down did not finish:\n  %s", name, strings.Join(failures, "\n  "))
	}
	return emitCompareResult(result, false)
}

// downTargets picks the variants to take down: those named, or all of the record. A name the
// record does not hold is an error, not a no-op.
func downTargets(rec *compare.Record, names []string) ([]compare.Variant, error) {
	if len(names) == 0 {
		return append([]compare.Variant(nil), rec.Variants...), nil
	}
	seen := map[string]bool{}
	var out []compare.Variant
	for _, n := range names {
		v := rec.Variant(n)
		switch {
		case v == nil:
			return nil, fmt.Errorf("variant %q is not in the record", n)
		case seen[n]:
			return nil, fmt.Errorf("variant %q is given twice", n)
		}
		seen[n] = true
		out = append(out, *v)
	}
	return out, nil
}

// downVariant stops v's per_variant entries (skipped when the checkout is already gone, since
// nothing is left there to stop) and then removes the snapshot. A dirty snapshot is refused by
// RemoveSnapshot; the entries are already stopped by then, and the error says the checkout stays.
func downVariant(ctx context.Context, dir string, cmp *config.CompareConfig, rec *compare.Record, v compare.Variant) error {
	unverified := false
	if len(cmp.PerVariant) > 0 && pathExists(v.ConfigDir) {
		err := compareHost.Down(ctx, v.ConfigDir, cmp.Plan, varsOf(v.Ports), cmp.PerVariant)
		switch {
		case errors.Is(err, errVariantConfigUnreadable):
			unverified = true
		case err != nil:
			return fmt.Errorf("stop entries: %w", err)
		}
	}
	if v.Mode == compare.ModeWorktree {
		return nil // a work worktree is the provider's; only its processes were dva's
	}
	if !pathExists(v.Path) {
		return nil // already gone; only the record is left to drop
	}
	if err := compare.RemoveSnapshot(ctx, dir, rec, v.Name); err != nil {
		if unverified {
			return fmt.Errorf("dva.yml in the checkout cannot be loaded, so its entries were not stopped by dva and may still run; checkout kept: %w", err)
		}
		return fmt.Errorf("entries stopped, checkout kept: %w", err)
	}
	return nil
}

func removeVariant(vs []compare.Variant, name string) []compare.Variant {
	out := vs[:0:0]
	for _, v := range vs {
		if v.Name != name {
			out = append(out, v)
		}
	}
	return out
}

// removeRecord deletes the record of a comparison that has no variant left.
func removeRecord(dir, name string) error {
	path, err := compare.RecordPath(dir, name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove record %s: %w", path, err)
	}
	// The per-comparison directory of checkouts is empty now. os.Remove refuses a directory
	// that still holds anything, so a stray file keeps it and nothing is deleted by accident.
	if snap, err := compare.SnapshotPath(dir, name, "x"); err == nil {
		_ = os.Remove(filepath.Dir(snap))
	}
	return nil
}

func leftVariants(result *compareResult) []string {
	var left []string
	for _, v := range result.Variants {
		if v.Outcome != outcomeStopped {
			left = append(left, v.Name)
		}
	}
	return left
}
