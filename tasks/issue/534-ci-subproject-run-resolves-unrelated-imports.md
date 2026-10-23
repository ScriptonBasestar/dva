---
id: ISSUE-534
title: "dva ci --subproject fails when an unrelated subproject's import cannot be resolved"
type: bug
priority: P2
effort: M
exec-tier: standard
status: todo
severity: medium
ownership: local
discovered-in: "singlepageweb-devbox TASK-270 predeploy profile check (2026-10-23)"
discovered-at: 2026-10-23
created: 2026-10-23
---

## Summary

`dva ci <profile> --subproject <name>` runs one root CI profile scoped to one declared
subproject, yet config loading resolves the imports of every subproject first. When any other
subproject's `dva.yml` is missing or invalid, the command fails before the profile starts, even
though that subproject is not used by the run.

## 소유권 — 이 저장소다

`internal/config/load.go:112-116` calls `resolveSubprojectImports` for every config load.
`internal/config/subproject.go:95-115` loads all subprojects that declare `import:`, with no
selection. `internal/cli/ci.go:90` checks only that `--subproject` names a declared subproject,
so name validation does not need the imported configs at all.

## Reproduction

1. In a root `dva.yml`, declare two subprojects that both import an interaction
   (`import: { interactions: [{ name: hello }] }`), `a` at `ws/a` and `b` at `ws/b`. Give
   `ws/a/dva.yml` a local `hello` interaction and leave `ws/b` without a `dva.yml`.
2. Declare a root CI profile `p` with `timeout: "1m"` and one step `echo "$DVA_CI_SUBPROJECT"`.
3. Run `dva ci p --subproject a --dry-run` (dva 0.4.0, commit `22d49732`).
4. Observed: exit 1, `ERROR: resolving subprojects: loading subproject "b"
   (.../ws/b/dva.yml): open .../ws/b/dva.yml: no such file or directory`, and the profile does
   not run. Copying `ws/a/dva.yml` to `ws/b/` makes the same command exit 0 with a dry-run plan
   for subproject `a`.

Seen in singlepageweb-devbox: `dva ci predeploy --subproject singlepageweb-calculator --dry-run`
fails on `singlepageweb-draw`, whose prepared workspace is absent.

## Expected vs Actual

- Expected: a run scoped to subproject `a` needs only the root config and, if the profile uses
  it, `a`'s imports. A broken or absent unrelated subproject does not block the run. A command
  that does need every import (for example `dva config validate`) still reports it.
- Actual: every config load fails on the first subproject import it cannot resolve.

## 영향

A devbox with many children cannot run one child's pre-deploy check unless every child workspace
is prepared and current. singlepageweb-devbox TASK-270 criterion 3 is blocked on this.

## 범위

Decide in the fix whether import resolution becomes lazy (resolved when a command consumes it)
or is filtered by the command's selected subprojects. Keep the full resolution for validation
commands. A root-only CI profile run without `--subproject` should also not need subproject
imports.
