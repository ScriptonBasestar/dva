---
id: TASK-436
title: "Reclaim the duplicate task-407 worktree"
type: chore
priority: P2
effort: S
exec-tier: standard
needs-human: true
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-28
quality-review-evidence: "Independent review session /root/review_task443_board_refresh verified CE discard receipt, absent worktree and local/remote branch, source revision, tracked evidence, and prior user authorization; final review found only README/PLAN-011 status wording stale, corrected in this commit."
---

## Summary

[ISSUE-039](../issue/039-discard-guard-blocks-reclaiming-a-worktree-whose-fix-was-already-merged-upstream.md)의
남은 운영 항목이다. stale branch의 내용이 source에 이미 반영된 것을 확인하고
사용자가 회수를 승인했다. CE `run-discard`가 정확한 실행 identity와 clean
worktree, remote 부재를 확인해 2026-09-28에 통합 없이 회수했다.

## Completion Criteria

- [x] 그 워크트리와 로컬·원격 브랜치가 없다 | verify: human — [회수 영수증과 독립 Git 조회](../../done/evidence/TASK-436/discard-verification-20260928.json)에 worktree 미등록, local show-ref exit 1, origin ls-remote exit 2를 기록

## 2026-09-25 확인 결과

- Worktree: `/Users/archmagece/worktrees/misc/dva/claude__mbp__fix__task-407`, clean after restoring the original branch state.
- Branch: `dev/claude/mbp/fix/task-407`, HEAD `4ce30f26`; upstream 미설정, `origin`에 같은 이름의 remote branch 없음.
- Configured source `master` and `origin/master` are `97b1a97d`. `ce task run-doctor` reported ACTIVE/ready before the rebase.
- Rebase onto `origin/master` conflicted in `AGENTS.md` and TASK-407 rename/add/delete history. Per repository policy, no side was selected; `git rebase --abort` restored HEAD `4ce30f26` and the clean worktree.
- The unique direct link from the DUP-ID instruction to `docs/407-correction-procedure.md` is preserved in the active DVA task worktree. The stale branch's card edit conflicts with the newer source card and was not integrated.
- `ce task run-status task-407 --json` reports ACTIVE, `finishReady: false`, and only `run-status`/`run-abort`. `run-abort` records a receipt but does not remove the worktree or branch.

당시에는 지원되는 통합 없는 회수 경로가 없어 보존했다. 아래 2026-09-28
결과는 이후에 추가된 shared lifecycle 명령으로 회수한 기록이다.

## Out of scope

- `guard-git-integration.sh`를 고치는 일.
- 이미 `master`에 있는 TASK-407 문서를 다시 쓰는 일.

## Sources

- [ISSUE-039](../issue/039-discard-guard-blocks-reclaiming-a-worktree-whose-fix-was-already-merged-upstream.md)

## 2026-09-27 재측정

ce task run-status task-407 --json은 상태 ACTIVE, finishReady false, upstream 미설정으로
반환했고 허용 동작은 run-status와 run-abort뿐이었다. worktree는
/Users/archmagece/worktrees/misc/dva/claude__mbp__fix__task-407에 clean 상태로
남아 있고 branch dev/claude/mbp/fix/task-407의 HEAD는 4ce30f26. 2026-09-27 현재
source master보다 53 commit 뒤지고 8개 고유 commit이 있어 통합 상태가 같지 않다. 이후 `git cherry`
재검토에서는 7개가 patch-equivalent하고 마지막 AGENTS.md 수정의 의미도 source
문서에 반영됐음을 확인했다. 회수 승인과 콘텐츠 판단은 완료됐다. 재개 조건은
shared lifecycle의 검증 가능한 discard/reclaim 경로다. 그 전에는 기존 worktree를
그대로 보존한다.

## 2026-09-28 회수 결과

CE `0d4b8b16`의 `run-discard`가 사용자 승인된 중복 실행을 처리했다.
`--take-over-from claude/mbp`로 원래 owner를 명시하고 실제 수행자
`codex/mbp`를 영수증에 기록했다. Worktrunk 제거 뒤 worktree·local branch·
origin branch 부재를 검증해 terminal `ABORTED` receipt를 추가했다.
[추적 증거](../../done/evidence/TASK-436/discard-verification-20260928.json)는
source-built CE revision과 별도 Git 조회 종료 코드도 보존한다.
