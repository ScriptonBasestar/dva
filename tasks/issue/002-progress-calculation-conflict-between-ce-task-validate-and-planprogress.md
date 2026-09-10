---
id: ISSUE-002
title: "Progress calculation conflicts between ce task validate and planprogress"
type: bug
status: todo
priority: P1
effort: S
exec-tier: standard
severity: medium
discovered-in: "PLAN-006 board currentization"
discovered-at: 2026-09-10
created: 2026-09-10
---

## Summary

PLAN-006 has 16 completed children out of 26. The shared `ce task validate --all`
requires `progress: 61`, while this repository's `go run ./tools/planprogress`
requires `progress: 62`. No card value satisfies both gates, so a card-only
currentization cannot make the board and the repository documentation gate green
at the same time.

## Reproduction

1. Keep `tasks/plan/006-devbox-dogfood-followup.md` at `progress: 62` and run
   `make doc-check`; `planprogress` passes.
2. Run `ce task validate --all`; PLAN-006 fails with `Declared progress is 62,
   want 61 from children`.
3. Change only that value to `61`; `ce task validate --all` passes, but
   `make doc-check` fails because `planprogress` reports `progress=61, want 62`.

## Expected vs Actual

- Expected: the shared task validator and repository `planprogress` use one documented
  rounding rule and accept the same persisted percentage.
- Actual: the shared validator truncates this ratio to 61, while `planprogress`
  rounds it to 62.

## Impact

`ce task gate` cannot become ready while this repository preserves its current
`planprogress` contract. TASK-354 cannot truthfully connect the shared gate until
the two validators share one progress convention or one explicitly delegates to
the other.

## Recommended Resolution

Treat the shared task-management validator's truncation rule as the canonical
contract and change DVA `planprogress` to apply the same rule, with a regression
test for 16/26 and an audit of every persisted plan value. The shared engine
already applies integer truncation; changing it would alter the contract for
every repository that uses the task runtime, while DVA owns its local gate.

## Resolution Criteria

- [ ] `ce task validate --all` and `make doc-check` accept the same progress value for PLAN-006 | verify: human — linked upstream fix or an agreed DVA validator change records both command outputs
- [ ] A non-integer progress ratio is covered by the owning validator's regression test | verify: human — linked test covers 16 completed children out of 26
- [ ] TASK-354 records the resolved shared-gate consequence before it is closed | verify: human — TASK-354 decision record links this issue and the resolution
