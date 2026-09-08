---
id: TASK-349
title: "Adopt the shared CE task-runtime declaration for dva"
type: chore
priority: P2
effort: S
exec-tier: standard
created: 2026-09-08
status: todo
needs-human: true
---

## Summary

`ce task run-doctor` is BLOCKED in this repository — there is no `.ce/task-runtime.yaml`, so
the CE lifecycle commands (`run-start`, `run-status`, `run-finish`) are unavailable and every
task runs on plain git worktrees instead.

The visible cost is at the end of each task. `branch-integrate` integrates correctly but always
finishes with `RECLAIM skipped: reclaim nothing: no declaration`, so the worktree, the local
branch, the remote branch and the empty parent directory are all reclaimed by hand, every time.
That is four manual steps after every integration, each of which is silently skippable — and a
skipped one leaves a stale branch that the next `cross-merge` check has to reason about.

The correct response is adoption, not local reimplementation: repository-local Makefile or shell
logic for worktree creation, integration and cleanup is explicitly the thing to avoid, because
it forks a shared contract per repository. Until adoption, the existing declared workflow stands
and should be followed as-is.

Needs a human: adopting a shared lifecycle contract is a repository-governance decision, and the
reclaim scope (which of worktree / local branch / remote branch the declaration owns) should be
chosen deliberately rather than copied.

## Completion Criteria

- [ ] .ce/task-runtime.yaml exists and ce task run-doctor reports ACTIVE instead of BLOCKED | verify: `ce task run-doctor`
- [ ] The declaration is adopted rather than reimplemented: no `git worktree` lifecycle logic is added to the Makefile | verify: `test 0 -eq "$(/usr/bin/grep -c 'git worktree' Makefile)"`
- [ ] Adoption is a decision, not a default | verify: human — 카드 하단 "Design record" 절에 CE 공유 lifecycle 채택 여부와 근거, reclaim 선언 범위(worktree/local branch/remote branch)가 기록되었는지 확인
