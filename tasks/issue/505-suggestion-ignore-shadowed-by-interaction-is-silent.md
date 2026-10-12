---
id: ISSUE-505
title: "suggestion_ignore entry for a target an interaction already covers is neither stale nor shown as ignored"
type: bug
priority: P3
effort: S
status: todo
severity: low
discovered-in: "gizzahub-devbox TASK-1960 independent review (dva.yml suggestion_ignore vs interaction overlap)"
discovered-at: 2026-10-12
created: 2026-10-12
---

## Summary

`dva config validate` reports a `suggestion_ignore` entry as stale only when it matches
no documented Makefile target or package.json script. An entry whose targets are all
already covered by an interaction (same name, subcommand coverage, or a wrapped workflow
target) also suppresses nothing, but it is reported nowhere: not as stale, and not under
`--show-ignored`. Such an entry is dead today and becomes harmful later: if the covering
interaction is removed or renamed, the entry silently hides the suggestion that would
have flagged the target as unwrapped. Consumers who keep long ignore lists cannot see it.

## Reproduction

Installed DVA `/Users/archmagece/go/bin/dva` reports commit `1aa8cd62`; source read at
`395be8c5`. Both validate files are unchanged between the two.

1. In an empty directory create a Makefile with documented targets `foo: ## x` and
   `bar: ## y`, and a `dva.yml` with `suggestion_ignore: ["foo", "bar", "gone"]` and one
   interaction `foo` (`command: "make foo"`, `runner: local`).
2. Run `dva config validate --show-ignored`.

Observed output (exit 0):

```
[warn] stale ignore: suggestion_ignore[2] "gone" matches no Makefile target or package.json script; remove it
[info] ignored: suggestion "bar" — suggestion_ignore: bar
✅ dva.yml is valid (1 suggestion ignored by dva.yml)
```

`foo` appears in neither line. The same holds in a real consumer: gizzahub-devbox
`dva.yml` lists `sigdock-age-assurance-preflight` and `sigdock-provision-client-key` in
`suggestion_ignore` (added 2026-08, commits `462ac652`, `a6ed8c24`) while interactions of
the same names exist; both are documented targets in `.make/sigdock.mk:33,36`, and
`dva config validate --show-ignored` prints nothing for either.

## Cause

`internal/cli/validate_sources.go` skips covered names before consulting the ignore list
(`commandSet` :103, `subcommandCoverage` :106, `wrapped` :111; ignore match :118), so a
covered name never records a suppression. The stale check then judges each pattern
against `universe` (:142–145), every documented target and script before any coverage
filter, so a pattern that matches only covered names counts as matching something.
`staleIgnoreWarnings` is `internal/cli/validate_suppression.go:148`.

## Expected vs Actual

- Expected: a `suggestion_ignore` pattern whose every matched target is already covered
  by an interaction is reported (as stale, or as a distinct "redundant: covered by
  interaction X" warning), so the operator can remove it before it masks a regression.
- Actual: the pattern is silent; only patterns matching no target at all are reported.

## Notes

The fix should keep glob semantics. With documented targets `alpha-one` and `alpha-two`,
`suggestion_ignore: ["alpha-*"]` and an interaction `alpha-one`, the pattern still hides
`alpha-two` (`--show-ignored` lists it) and is not redundant. With interactions for both,
the pattern is silent exactly like `foo` above. Only patterns whose matches are all
covered qualify. Do not use a `docker-*` Makefile example: `shouldIgnoreMakefileTarget`
(`validate_sources.go:445–448`) drops that prefix before the ignore list is consulted. Consumer-side cleanup in gizzahub-devbox is separate and
does not depend on this fix.
