---
id: TASK-436
title: "Reclaim the duplicate task-407 worktree"
type: chore
priority: P2
effort: S
exec-tier: standard
needs-human: true
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-039](../issue/039-discard-guard-blocks-reclaiming-a-worktree-whose-fix-was-already-merged-upstream.md)의
남은 운영 항목이다. `~/worktrees/misc/dva/claude__mbp__fix__task-407`과
브랜치 `dev/claude/mbp/fix/task-407`이 아직 있다. 같은 문서 수정은 이미
`master`에 들어갔다. `guard-git-integration.sh`는 그 브랜치가
`origin/master`에 커밋으로 닿지 않았다며 에이전트의 폐기를 막는다.
훅은 그 결정을 사람이 직접 실행하라고 적는다.

방향: 사람이 워크트리, 로컬 브랜치, 원격 브랜치를 제거한다. 훅에 에이전트
폐기 경로를 추가하는 일은 워크스테이션 정책(TD-71)이라 이 카드 밖이다.

## Completion Criteria

- [ ] 그 워크트리와 로컬·원격 브랜치가 없다 | verify: human — `git worktree list`에 `claude__mbp__fix__task-407`이 없고, 로컬·원격 브랜치도 없다

## Out of scope

- `guard-git-integration.sh`를 고치는 일.
- 이미 `master`에 있는 TASK-407 문서를 다시 쓰는 일.

## Sources

- [ISSUE-039](../issue/039-discard-guard-blocks-reclaiming-a-worktree-whose-fix-was-already-merged-upstream.md)
