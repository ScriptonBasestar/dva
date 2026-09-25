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
남은 운영 항목이다. 현재 master로 stale branch를 rebase했을 때 AGENTS.md와 TASK-407
카드 이력에서 충돌했다. 유용한 복구 절차 직접 링크는 active branch에 보존했지만,
shared lifecycle은 conflict 상태의 통합 없는 회수 경로를 제공하지 않는다.

사용자가 회수를 승인했지만 `run-abort`는 worktree/ref를 제거하지 않고 저장소 정책상
직접 제거도 사용할 수 없어, supported lifecycle recovery route가 생길 때까지 blocked다.

## Completion Criteria

- [ ] 그 워크트리와 로컬·원격 브랜치가 없다 | verify: human — `git worktree list`에 `claude__mbp__fix__task-407`이 없고, 로컬·원격 브랜치도 없다

## 2026-09-25 확인 결과

- Worktree: `/Users/archmagece/worktrees/misc/dva/claude__mbp__fix__task-407`, clean after restoring the original branch state.
- Branch: `dev/claude/mbp/fix/task-407`, HEAD `4ce30f26`; upstream 미설정, `origin`에 같은 이름의 remote branch 없음.
- Configured source `master` and `origin/master` are `97b1a97d`. `ce task run-doctor` reported ACTIVE/ready before the rebase.
- Rebase onto `origin/master` conflicted in `AGENTS.md` and TASK-407 rename/add/delete history. Per repository policy, no side was selected; `git rebase --abort` restored HEAD `4ce30f26` and the clean worktree.
- The unique direct link from the DUP-ID instruction to `docs/407-correction-procedure.md` is preserved in the active DVA task worktree. The stale branch's card edit conflicts with the newer source card and was not integrated.
- `ce task run-status task-407 --json` reports ACTIVE, `finishReady: false`, and only `run-status`/`run-abort`. `run-abort` records a receipt but does not remove the worktree or branch.

Blocked until the shared lifecycle provides an approved cleanup/recovery route. User authorization is recorded; repository policy does not permit bypassing the lifecycle with direct worktree/ref deletion.

## Out of scope

- `guard-git-integration.sh`를 고치는 일.
- 이미 `master`에 있는 TASK-407 문서를 다시 쓰는 일.

## Sources

- [ISSUE-039](../issue/039-discard-guard-blocks-reclaiming-a-worktree-whose-fix-was-already-merged-upstream.md)
