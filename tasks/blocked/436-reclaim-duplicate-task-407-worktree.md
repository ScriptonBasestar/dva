---
id: TASK-436
title: "Reclaim the duplicate task-407 worktree"
type: chore
priority: P2
effort: S
exec-tier: standard
needs-human: true
status: blocked
created: 2026-09-24
---

## Summary

[ISSUE-039](../issue/039-discard-guard-blocks-reclaiming-a-worktree-whose-fix-was-already-merged-upstream.md)의
남은 운영 항목이다. 워크트리는 깨끗하지만 local branch는 현재 `master`보다
오래됐고 upstream이 없다. 런타임은 `task-407`을 ACTIVE로 보고하며
`run-abort`만 허용한다. 커밋 내용은 현재 문서와 비슷하지만 카드 상태가 달라
참조·내용을 더 확인하기 전에는 회수하지 않는다.

방향: 사람이 워크트리, 로컬 브랜치, 원격 브랜치를 제거한다. 훅에 에이전트
폐기 경로를 추가하는 일은 워크스테이션 정책(TD-71)이라 이 카드 밖이다.

## Completion Criteria

- [ ] 그 워크트리와 로컬·원격 브랜치가 없다 | verify: human — `git worktree list`에 `claude__mbp__fix__task-407`이 없고, 로컬·원격 브랜치도 없다

## 2026-09-25 확인 결과

- Worktree: `/Users/archmagece/worktrees/misc/dva/claude__mbp__fix__task-407`, clean.
- Branch: `dev/claude/mbp/fix/task-407`, HEAD `4ce30f26`; upstream 미설정, `origin`에 같은 이름의 remote branch 없음.
- 현재 source `master`는 `a9d33977`; `master`가 task HEAD의 조상이 아니어서 stale branch다.
- `ce task run-status task-407 --json`: ACTIVE, `finishReady: false`, 다음 행동은 upstream 설정·push; 허용 작업은 `run-status`와 `run-abort`.
- HEAD commit은 `AGENTS.md`와 TASK-407 카드만 바꾸며 문서 수정은 현재 source에 이미 반영됐지만, branch의 카드 상태 기록은 다르다.

이 상태는 자동 회수로 결정하지 않는다. 사람에게 ref와 커밋 처분 결정을 넘기고, 그 뒤에만 TASK-436을 닫는다.

## Out of scope

- `guard-git-integration.sh`를 고치는 일.
- 이미 `master`에 있는 TASK-407 문서를 다시 쓰는 일.

## Sources

- [ISSUE-039](../issue/039-discard-guard-blocks-reclaiming-a-worktree-whose-fix-was-already-merged-upstream.md)
