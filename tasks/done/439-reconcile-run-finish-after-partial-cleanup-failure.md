---
id: TASK-439
title: "Reconcile run-finish after partial cleanup failure"
type: bug
priority: P2
effort: M
exec-tier: strong
created: 2026-09-25
status: done
depends-on: [TASK-438]
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: >-
  Independent done-review PASS by /root/review_433_434: checked all five
  completion criteria against integrated CE commit 7eaf596a, full CI run
  3b919373 (unchanged attestation), fail-closed/no-mutation/list/idempotence
  tests, and the DONE run-finish receipt with all push and cleanup flags true.
---

## Summary

Add a supported `run-recover` action for an execution whose provider integrated
and pushed the task commit but only partially reclaimed its worktree. The action
must verify execution identity and source containment, then close the runtime
receipt while preserving the remaining path and refs for explicit later review.
It must not repeat integration or delete unverified data.

## Completion Criteria

- [x] Recovery is available only after runtime identity matches and the integrated source contains the recorded task HEAD; mismatches remain blocked | verify: human — upstream regression tests reject source-containment and execution-identity mismatches
- [x] The recovery action records source identity and outstanding path/ref state without replaying integration or deleting unverified data | verify: human — upstream recovery tests assert receipt evidence and no provider/cleanup mutation
- [x] `run-status` and `run-list` offer `run-recover` only for the proven partial-cleanup receipt; normal failures and stale-base outcomes do not offer it | verify: human — eligible/ineligible status and list assertions
- [x] Exact `integrate run` and `integrate run --no-fetch` exit-3 receipts are recoverable, and a second recovery is idempotent | verify: human — no-fetch and repeated-call regression tests
- [x] Recovery refuses when source does not contain task HEAD or is not pushed, and refuses a mismatched execution identity | verify: human — disposable test fixtures cover each fail-closed outcome

## Evidence

`ce-agent-kit` commit `7eaf596a8ce656603201e3967b374620982eb7fb` is integrated
and pushed to `master`/`origin/master`. Full CI run
`3b919373aca7539c22154c11d1279e01` passed (profile `full`, 5m43s; input
attestation unchanged at `d4045e249efa4da572940778588e027f833ceae38f5a27a93a9fef0937207573`).
The focused `taskruntime` tests and `validate-filesize` also passed. Test cases
cover eligible status/list actions, stale/ineligible actions, no-fetch, repeat
recovery, source containment/push, identity mismatch, residual preservation,
and the absence of integration or cleanup mutation.

`ce task run-finish task-439-reconcile-run-finish-after-partial-cleanup-failure`
recorded `DONE` with `sourcePushed`, `worktreeRemoved`, `localBranchRemoved`,
and `remoteBranchRemoved` all true. `sourceHead` and `taskHead` are both
`7eaf596a8ce656603201e3967b374620982eb7fb`; the previous source tip was
`71b1d1699313414013062b0d2bc9c44af13fadb4`. Independent code review: PASS.
