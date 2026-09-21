---
id: ISSUE-032
title: "issue-promote apply fails at trailing cleanup on source branch master"
type: bug
status: todo
priority: P1
effort: S
exec-tier: standard
severity: medium
ownership: upstream
upstream-ref: "ce-agent-kit#10"
discovered-in: "issue-promote apply fails at trailing cleanup on source branch master"
discovered-at: 2026-09-21
created: 2026-09-21
---

## Summary

`ce task issue-promote <issue-path> --json` (apply) fails after candidate selection at the transaction's trailing `cleanup --apply` step with `cleanup --apply refuses configured source branch "master"`, so no TASK card is created and the board is unchanged. This blocks the entire issue→task pipeline from the trunk checkout. Observed 2026-09-21 promoting ISSUE-004 (eligible, unambiguous, skill-accepted); `--dry-run` returns agent-handoff normally.

## Reproduction

1. From the master checkout run `ce task issue-promote tasks/issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md --json`.
2. Observe `Error: issue-promote: cleanup --apply refuses configured source branch "master"`.
3. Confirm `tasks/todo/` unchanged and `git status` shows no new card.

## Expected vs Actual

- Expected: promotion commits the card creation, then cleanup either runs or degrades to a warning without rolling back the created card.
- Actual: cleanup --apply requires a non-source task branch containing that source, so the whole transaction fails on master with zero mutation.

## 소유권 — 상류다

`ce-agent-kit#10` 관련 상류 도구(`ce task issue-promote`)의 결함이다. master 체크아웃에서 issue-promote 실행 시 cleanup --apply가 소스 브랜치를 거부하여 전체 트랜잭션이 실패하는 문제다.

