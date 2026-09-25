---
id: ISSUE-032
title: "issue-promote source-branch guard fires before promotion"
type: bug
status: done
priority: P1
effort: S
exec-tier: standard
severity: medium
ownership: upstream
upstream-ref: "ce-agent-kit#10"
discovered-in: "issue-promote apply fails at trailing cleanup on source branch master"
discovered-at: 2026-09-21
created: 2026-09-21
resolution: fixed
resolved-at: 2026-09-25
resolution-summary: "TASK-435 integrated in ce-agent-kit master 05b0b4ae; issue-promote permits named source branches while detached HEAD and destructive-cleanup guards remain."
---

## Summary

Before TASK-435, `ce task issue-promote <issue-path> --json` and
`ce task issue-promote-all --json` reused the destructive-cleanup source-branch
guard **before** calling promotion. They failed before a TASK card could be
written. The failure was not a trailing cleanup error: there is no cleanup call
after `TaskPromote`. `--dry-run` does not call the apply guard. TASK-435 resolved
this at ce-agent-kit master `05b0b4ae`.

## Reproduction

1. On a configured source branch with an eligible issue, run `ce task issue-promote <issue-path> --json`.
2. Observe the shared cleanup guard reject the source branch before `TaskPromote` runs.
3. Confirm no task card is written. The same guard is used by `issue-promote-all`.

## Expected vs Actual

- Expected: issue promotion may create its task from a named source branch; detached HEAD remains refused. Destructive cleanup continues to require a non-source task branch containing the source tip.
- Actual before TASK-435: promotion invoked the cleanup guard and rejected the source branch before creating a task. Current behavior accepts the configured source branch and still rejects detached HEAD.

## 소유권 — 상류다

`ce-agent-kit`의 `issue-promote` 경로가 cleanup 전용 source-branch guard를 공유하는 결함이다. 실제 cleanup 보호는 유지하면서 승격 guard를 분리하는 것이 수정 범위다.

## 후속 (2026-09-24)

`guardIssuePromotionApply`가 `cleanupApplySourceGuard`를 호출하는 것이 원인이다.
이 보드에서 실제 apply를 실행하지 않는다. 작업은
[TASK-435](../../done/435-issue-promote-on-source-branch.md)가 소유한다.
