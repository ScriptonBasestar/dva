---
id: ISSUE-042
title: "Preserve incomplete-recovery verdicts in stale-base run-finish"
type: bug
status: done
priority: P2
effort: S
severity: medium
ownership: upstream
upstream-ref: "ce-agent-kit#2"
discovered-in: "TASK-438 independent review of ce-agent-kit commit c3e54bda"
discovered-at: 2026-09-25
created: 2026-09-25
resolution: fixed
resolved-at: 2026-09-25T01:10:07Z
resolution-summary: "Resolved as fixed by TASK-438."
---

## Summary

TASK-438 commit `c3e54bda` adds both commit SHAs to stale-base run-finish
verdicts, but independent review found that the new stale classifier can replace
an incomplete-recovery verdict after provider cleanup failure. The same change
also checks an old rendered-reason prefix that no longer matches the new text,
so stale evidence handling is not preserved. Operators can receive the wrong
recovery instruction even though the full CI run passes.

## Reproduction

1. Make the provider's `Check` report a stale base and confirm the verdict names both commits.
2. Make provider `Run` exit 3 after integration succeeds but recovery cleanup fails; the incomplete-recovery verdict must remain distinct.
3. Include a post-operation evidence error in the stale-base fixture; the stale verdict must not append it as a new recovery failure.
4. In both fixtures, assert that `git rebase` is never invoked.

## Expected vs Actual

- Expected: stale-base classification applies only to the provider check failure, incomplete recovery remains generic and distinct, and control flow does not depend on parsing human-readable reason text.
- Actual: the common stale check can overwrite a run-exit-3 cleanup failure, while the stale suppression prefix no longer matches the SHA-bearing reason.

## Resolution Criteria

- [x] Check-failure stale verdict and run-exit-3 incomplete-recovery verdict remain distinct, and stale evidence handling is preserved | verify: `ce-agent-kit@7046479e` tests both reasons/actions and no rebase invocation; independent review and full CI passed
- [x] Disposable stale and incomplete-recovery fixtures record both SHA values, pre/post execution identity, and the exact cleanup result | verify: human — `tasks/done/evidence/TASK-438/README.md` and saved run-finish/run-status JSON capture both cases; the cleanup fixture's separate Worktrunk inventory mismatch is tracked by ISSUE-043

## Recommendation

Gate stale classification on the provider check outcome, carry stale state as an
explicit value instead of parsing the reason string, and keep recovery-cleanup
failures separate. Close the evidence criterion through TASK-438 after both
fixtures pass.

## Related

- [ISSUE-008](008-run-finish-does-not-name-a-stale-base-as-the-reason-it-blocked.md)
- [TASK-438](../../done/438-name-both-commits-in-stale-base-finish-verdict.md)

## 소유권 — 상류

`ce task run-finish`의 provider 분류와 verdict는 ce-agent-kit#2가 소유한다.
DVA는 재현 fixture와 task acceptance evidence를 관리한다.
