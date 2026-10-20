---
id: ISSUE-519
title: "Real-sops version check and PATH fallback accept two edge inputs"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
severity: low
ownership: local
discovered-in: "TASK-516 independent review 2 (Grok 4.7)"
discovered-at: 2026-10-20
created: 2026-10-20
---

## Summary

TASK-516's independent review passed the change and named two minor edges in
`internal/integration/config_env_helpers_test.go`. Neither affects the official
sops 3.13.1/3.13.3 builds or the default mise shim directory, so TASK-516 was
integrated; this card keeps them from being forgotten.

## Reproduction

1. `sopsVersionRE` (line 128) takes the first `major.minor.patch` and ignores a
   suffix: `sops 3.13.1-rc1` passes the 3.13.1 floor, while the CI verify step
   (`.github/workflows/ci.yml:251`) rejects the same string. A wrapper printing
   another `sops N.N.N` before the real line is also taken as the version.
2. `lookPathOutsideShims` (line 111) skips empty PATH entries and mise `shims`
   directories, but returns a relative path for a relative entry such as `bin`;
   `toolPATH` then symlinks that relative target, which resolves against the temp
   directory. A non-mise `shims` directory (for example `~/.asdf/shims`) is still
   searched.

## Expected vs Actual

- Expected: a prerelease below or at the floor is treated as below it (or
  rejected like CI), only the first output line is parsed, and relative PATH
  entries are skipped or made absolute.
- Actual: as above.

Evidence: `tasks/done/evidence/TASK-516/review2-pass.txt`.
