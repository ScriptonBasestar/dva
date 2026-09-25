---
id: ISSUE-043
title: "run-finish cleanup failure leaves recovery outside Worktrunk inventory"
type: bug
status: done
priority: P2
effort: S
severity: medium
ownership: upstream
upstream-ref: "ce-agent-kit#2"
discovered-in: "TASK-438 disposable cleanup fixture"
discovered-at: 2026-09-25
created: 2026-09-25
resolution: fixed
resolved-at: 2026-09-25T01:54:37Z
resolution-summary: "Resolved as fixed by TASK-439."
---

## Summary

When `gz-git integrate run` successfully integrates and pushes a task branch but
returns exit 3 because filesystem cleanup is denied, CE preserves the
`run-finish` failure receipt. The task directory can still exist on disk, while
Worktrunk no longer lists it. A subsequent `ce task run-status` returns
`BLOCKED: Worktrunk inventory does not match execution metadata` and offers
only status/abort, leaving no documented safe action to reconcile the integrated
execution and its orphaned path/refs.

## Reproduction

1. In a disposable local bare-repo fixture, start a task runtime and create a
   clean task commit with local `origin` refs.
2. Make the task worktree read-only and run `ce task run-finish` so the provider
   integrates successfully but `gz-git integrate run` exits 3 while deleting
   the worktree path.
3. Inspect the `run-finish` receipt and run `ce task run-status` again.
4. Compare the runtime identity, filesystem path, Worktrunk registration, local
   and remote refs, and source/task commit SHAs before and after.

The exact fixture outputs are in
[`cleanup-finish.json`](../../done/evidence/TASK-438/cleanup-finish.json),
[`cleanup-status-after.json`](../../done/evidence/TASK-438/cleanup-status-after.json),
and the surrounding TASK-438 evidence files.

## Expected vs Actual

- Expected: CE exposes a supported, auditable recovery action for an execution
  whose provider integrated the task commit but only partially reclaimed its
  worktree. Recovery verifies the source contains the task commit and does not
  delete an unverified path or ref.
- Actual: the filesystem path and branch refs remain, but the worktree is absent
  from Worktrunk inventory. `run-status` blocks on metadata mismatch and offers
  no finish/reconcile action. The failure itself is correctly classified as
  incomplete recovery, not stale-base; this issue is about the remaining
  recovery path.

## Resolution Criteria

- [x] A supported recovery operation reconciles the recorded execution after a
  partial provider reclaim without repeating integration or deleting unverified
  user data | verify: human — upstream recovery tests and [TASK-439 evidence](../../done/evidence/TASK-439/README.md) cover the recorded failure mode
- [x] `run-status` and `run-list` advertise `run-recover` only for the exact partial-cleanup receipt; normal integration failures and stale-base receipts remain ineligible | verify: human — eligible/ineligible status/list assertions in `ce-agent-kit@7eaf596a`
- [x] Both supported `integrate run` forms (`--no-fetch` when configured) recover idempotently and preserve residual refs/paths; source-not-pushed, source-containment, and execution-identity mismatches fail closed | verify: human — tests, independent review, and exact-tree full CI at `ce-agent-kit@7eaf596a`

## Recommendation

Add a supported `run-recover` action that checks the recorded execution identity
and verifies the integrated source contains the task commit. When those checks
pass, write a terminal recovery receipt that names the leftover path and refs,
leaves them untouched for explicit later review, and does not repeat integration.
Keep incomplete cleanup distinct from stale-base; refuse recovery when identity
or source containment does not match.

## Related

- [ISSUE-042](042-preserve-incomplete-recovery-verdicts.md)
- [TASK-438](../../done/438-name-both-commits-in-stale-base-finish-verdict.md)
- [TASK-439](../../done/439-reconcile-run-finish-after-partial-cleanup-failure.md)

## 소유권 — 상류

`ce task run-finish`와 `run-status`의 recovery contract는 ce-agent-kit#2가
소유한다. DVA는 격리 fixture와 발견 증거를 보존한다.
