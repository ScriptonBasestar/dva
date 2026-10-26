---
id: ISSUE-534
title: "dva ci --subproject fails when an unrelated subproject's import cannot be resolved"
type: bug
priority: P2
effort: M
exec-tier: standard
status: done
severity: medium
ownership: local
discovered-in: "singlepageweb-devbox TASK-270 predeploy profile check (2026-10-23)"
discovered-at: 2026-10-23
created: 2026-10-23
resolution: fixed
resolved-at: 2026-10-26T06:46:15Z
resolution-summary: "Resolved as fixed by TASK-533."
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

- Expected: root CI, with or without `--subproject a`, needs the root config and the declared
  name only; it does not load child configs because CI does not consume their plans or
  interactions. `--project a` needs the selected child's config, but not unrelated children.
  Default config loading and commands that validate or consume all imports still report broken
  imported children.
- Actual: every config load fails on the first subproject import it cannot resolve.

## 영향

A devbox with many children cannot run one child's pre-deploy check unless every child workspace
is prepared and current. singlepageweb-devbox TASK-270 criterion 3 is blocked on this.

## 범위

Use an explicit CI-only load option to skip imported-child resolution for root CI. Keep default
`config.Load` strict. For `--project`, load only the selected child and preserve errors for an
unknown selection or missing selected child config. A root-only CI profile without
`--subproject` also does not need child configs. Implementation and regression criteria are
tracked by TASK-533.
