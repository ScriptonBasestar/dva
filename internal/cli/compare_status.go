package cli

import (
	"context"
	"errors"
	"io/fs"

	"github.com/ScriptonBasestar/dva/internal/compare"
	"github.com/ScriptonBasestar/dva/internal/config"
)

// runCompareStatus reads what a comparison started and re-verifies it against the filesystem,
// Git and live processes. It changes nothing: a stale or drifted variant is reported, never
// repaired.
func runCompareStatus(ctx context.Context, c *config.Config, name string) error {
	cmp, err := compareDeclaration(c, name)
	if err != nil {
		return err
	}
	dir := c.FileDir()
	rec, err := compare.ReadRecord(dir, name)
	if errors.Is(err, fs.ErrNotExist) {
		return comparePreflight(nil, "compare %s: not started — no record of an up", name)
	}
	if err != nil {
		return comparePreflight(nil, "compare %s: %v", name, err)
	}

	result := &compareResult{Action: "status", Compare: name, Plan: cmp.Plan, Next: map[string]string{}}
	if len(cmp.Shared) > 0 {
		result.Shared = &compareSharedResult{Entries: cmp.Shared, Outcome: outcomeObserved}
		if obs, err := compareHost.Status(ctx, dir, cmp.Plan, nil, cmp.Shared); err != nil {
			result.Shared.Error = "status unavailable: " + err.Error()
		} else {
			result.Shared.compareObserved = *obs
		}
	}

	stale := map[string]compare.VariantState{}
	if states, err := compare.Check(ctx, dir, rec, cmp.PerVariant); err == nil {
		for _, s := range states {
			stale[s.Name] = s
		}
	}
	for _, v := range rec.Variants {
		row := compareVariantResult{
			Name: v.Name, Mode: v.Mode, Source: v.Source, Commit: v.Commit,
			Path: v.Path, ConfigDir: v.ConfigDir, Ports: v.Ports, Outcome: outcomeObserved,
		}
		if s, ok := stale[v.Name]; ok {
			row.Stale, row.Reasons = s.Stale, s.Reasons
		}
		if now, err := compare.ResolveRef(ctx, dir, v.Source); err != nil {
			row.Drift = &compareDrift{Error: err.Error()}
		} else if now != v.Commit {
			row.Drift = &compareDrift{Commit: now}
		}
		if len(cmp.PerVariant) > 0 && pathExists(v.ConfigDir) {
			if obs, err := compareHost.Status(ctx, v.ConfigDir, cmp.Plan, varsOf(v.Ports), cmp.PerVariant); err != nil {
				row.Error = "status unavailable: " + err.Error()
			} else {
				row.compareObserved = *obs
			}
		}
		result.Variants = append(result.Variants, row)
	}
	result.Next["down"] = downCommand(name, nil)
	return emitCompareResult(result, false)
}
