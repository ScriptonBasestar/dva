---
id: ISSUE-003
title: "TASK-362 historic flow-style panic is no longer reproducible"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: low
discovered-in: "TASK-362 guard-removal overlay"
discovered-at: 2026-09-10
created: 2026-09-10
---

## Summary

TASK-362 says that adding a banner to a flow-style root fixture must reproduce a
panic when the same-line key guard is removed. That was true for the historic
revision cited by the card, but it is no longer true on the current source. With
the guard present, the flow-style root is safely rejected; with a disposable
overlay that removes the guard, both fixtures fail their unchanged/report
assertion without panicking.

## Reproduction

1. Run the current `TestMigrateSectionOrderBailsOnUnrepresentableShapes` fixture.
2. Use a disposable overlay to remove the same-line key guard.
3. Run the focused test with the original fixture and with
   `# banner\n{stack: b, version: a}\n`.

Both variants fail the report assertion but do not panic on the current source.
Temporary overlay artifacts were deleted.

## Expected vs Actual

- Expected by TASK-362: only the banner fixture panics after the guard is removed.
- Actual: neither fixture panics; with the guard removed, both fail the current
  unchanged/report assertion instead.

## Impact

The card's human completion criterion cannot be satisfied truthfully. Retaining
it would pressure a runner to fabricate historical evidence or add a regression
that protects no current failure mode.

## Recommended Resolution

Keep the banner fixture because it exercises the documented unrepresentable
shape, but revise TASK-362 to assert the current observable contract: the shape
returns the original input with a `Blocked` report and does not panic. Record
this issue as the historical-panic disposition, remove the obsolete overlay-panic
criterion, and retain the focused passing test.

## Resolution Criteria

- [ ] TASK-362 no longer claims that the current guard-removal overlay panics | verify: human — TASK-362 links this issue and describes the current observable contract
- [ ] The flow-style banner fixture is covered by a focused current-source test | verify: human — TASK-362 links the passing focused test evidence
