---
id: ISSUE-506
title: "TestApplyEnv_MergesVars fails on any machine that exports DB_HOST — the fixture is not isolated from the process environment"
type: bug
priority: P3
effort: S
status: todo
severity: low
discovered-in: "ISSUE-505 gate run (triaging the only failure in go test ./internal/cli/)"
discovered-at: 2026-10-13
created: 2026-10-13
---

## Summary

`internal/cli/compose_flags_test.go` `TestApplyEnv_MergesVars` builds a `stg`
profile whose `environment:` sets `DB_HOST: stg-db`, applies it, and asserts
`e.Vars["DB_HOST"] == "stg-db"`. `MergeVars` deliberately resolves each key
through `os.LookupEnv` first (`internal/config/environment.go:99`; pinned by
`TestMergeVarsOSEnvPriority`, `TestMergeVarsOSEnvStillWinsOverBatch`,
`TestMergeVarsOSEnvShadowsSelfReferentialDeclaration`), so an ambient
`DB_HOST` in the shell wins over the profile value. On this workstation the
shell exports `DB_HOST=localhost` and the test fails everywhere the package is
run from that shell:

```
compose_flags_test.go:258: DB_HOST = "localhost", want "stg-db"
```

The product behavior is intended; the defect is the test. A profile-merge
assertion that reads the process environment is machine-dependent: it passes
on clean environments and fails on any machine exporting one of the fixture's
keys, which poisons unrelated gate runs (it surfaced as the sole red test in
the ISSUE-505 verification).

## Reproduction

```sh
cd "$(git rev-parse --show-toplevel)"
DB_HOST=localhost go test ./internal/cli/ -run TestApplyEnv_MergesVars -count=1
# FAIL: DB_HOST = "localhost", want "stg-db"
env -u DB_HOST go test ./internal/cli/ -count=1
# ok — whole package passes
```

Reproduced on both a task worktree and the primary checkout at `07944cb1`
(baseline, before ISSUE-505) — pre-existing, not introduced by any recent
change.

## Expected vs Actual

- Expected: the test pins profile-var merging, not the OS-env-wins precedence
  (the latter already has dedicated tests in `internal/config`), so it holds
  regardless of what the invoking shell exports.
- Actual: with `DB_HOST` present in the environment, `e.Vars["DB_HOST"]` is
  `"localhost"` and the assertion fails; the test is machine-dependent and
  poisons unrelated gate runs as the sole red test.

## Notes

Scope is the test fixture only — no production change is proposed, and the
OS-env-wins semantics stay as pinned. `t.Setenv("DB_HOST", "")` alone is NOT a
fix: an empty-but-present variable still satisfies `os.LookupEnv` and shadows
`stg-db`; the fixture needs the variable genuinely absent (for example
`t.Setenv` followed by `os.Unsetenv`, which keeps the restore), for every key
the fixture's profile declares. Any other test that asserts merged var values
without isolation has the same shape and should be swept in the same pass.
