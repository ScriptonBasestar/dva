---
id: ISSUE-002
title: "Progress calculation conflicts between ce task validate and planprogress"
type: bug
status: done
priority: P1
effort: S
exec-tier: standard
severity: medium
discovered-in: "PLAN-006 board currentization"
discovered-at: 2026-09-10
created: 2026-09-10
---

## Summary

The original PLAN-006 measurement had 16 completed children out of 26. The shared
validator required `progress: 61`, while this repository's `go run
./tools/planprogress` required `progress: 62`. No card value satisfied both
checks. TASK-371 changed the local validator to the shared truncation contract;
PLAN-006 has subsequently advanced to 18/26 and persists the same contract value,
`69`.

## Reproduction

This historical conflict is no longer reproducible. The current commands accept
the persisted `progress: 69` for PLAN-006:

1. `ce task validate tasks/plan/006-devbox-dogfood-followup.md`
2. `go run ./tools/planprogress`
3. `make doc-check`

`ce task validate --all` still exits nonzero, but only because TASK-344 and
TASK-371 lack controller-issued review receipts (ISSUE-001); PLAN-006 itself is
valid and is not a progress-calculation failure.

## Expected vs Actual

- Expected: the shared task validator and repository `planprogress` use one documented
  rounding rule and accept the same persisted percentage.
- Resolved: both use integer truncation (`completed * 100 / total`).

## Impact

The progress-contract blocker is removed. TASK-354 can rely on a single
progress convention; its remaining shared-gate blocker is the separate
review-receipt contract in ISSUE-001.

## Recommended Resolution

The shared task-management validator's truncation rule is canonical. TASK-371
changed DVA `planprogress` to apply it, added a 16/26 regression case, and
currentized PLAN-006. TASK-354 records that the progress prerequisite is
complete and that ISSUE-001 is now the remaining blocker.

## Resolution Criteria

- [x] PLAN-006 validates and `make doc-check` accepts the same persisted progress value | verify: `ce task validate "$(/usr/bin/find tasks -name '006-devbox-dogfood-followup.md' -print -quit)" && make doc-check`
- [x] A non-integer progress ratio is covered by the owning validator's regression test | verify: `go test ./tools/planprogress`
- [x] TASK-354 records the resolved shared-gate consequence before it is closed | verify: human — TASK-354 §Summary and PLAN-009 §Parallel lane record that TASK-371 resolved the progress prerequisite and ISSUE-001 remains the blocker

## Resolution Evidence

2026-09-10: `ce task validate tasks/plan/006-devbox-dogfood-followup.md`,
`go test ./tools/planprogress`, `go run ./tools/planprogress`, and `make
doc-check` passed. `tools/planprogress/check_test.go` includes the 16/26 → 61
truncation regression. `ce task validate --all` remains red only for TASK-344
and TASK-371's missing review receipts, tracked separately by ISSUE-001; it no
longer reports a PLAN-006 progress mismatch.
