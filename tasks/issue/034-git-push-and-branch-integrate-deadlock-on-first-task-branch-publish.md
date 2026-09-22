---
id: ISSUE-034
title: "git push and branch-integrate deadlock on first task-branch publish"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: medium
ownership: local
discovered-in: "2026-09-22 wave — TASK-405/407/408 review branch ready but unpublished"
discovered-at: 2026-09-22
created: 2026-09-22
---

## Summary

A new task branch cannot reach origin. `git push` is refused until
`branch-integrate` runs as a single bare command. `branch-integrate` then
fails because the branch has no upstream. Neither step can complete, so
verified work stays local.

This is a workstation launcher/hook deadlock, not a DVA product defect.
Do not promote it to a DVA todo.

## Reproduction

1. On a clean task worktree with no upstream, run `git push -u origin HEAD`.
2. Observe: `차단(정책): 대상 브랜치 push는 설치본 PATH 래퍼 branch-integrate를 단일 bare 명령으로 수행하라.`
3. In that same worktree, run the single bare command `branch-integrate`.
4. Observe: `FAIL  push — no upstream — git push -u origin <branch>` and `NOT READY`.

Measured 2026-09-22 on `dev/opencode/mbp/chore/review-405-407-408` at `9d71a40`
against `origin/master` `7a61a43`. Other integrate checks passed: freshness,
merge-tree, working-tree, `ce task gate` READY.

## Expected vs Actual

- Expected: the declared launcher can publish a task branch that has never
  had an upstream, or `git push -u origin HEAD` is allowed for that first
  publish.
- Actual: push requires integrate, integrate requires upstream.

## 소유권 — 이 저장소다

결함은 워크스테이션 push 훅과 `branch-integrate` launcher에 있다. 이 카드는
에이전트가 `git push`를 재시도하지 않게 이 보드에 남긴 관측이다. DVA 코드로
고치지 않고, todo로 승격하지 않는다. 상류 보고 번호가 생기면 `ownership:
upstream`과 `upstream-ref:`로 옮긴다.

## Related

- `.gz-git.yaml` — `integrationBranch: master`, readiness runner
  `.gz-git/readiness/check`
- ISSUE-032 — `issue-promote apply` also refuses work on source branch
  `master` (upstream `ce-agent-kit#10`)
