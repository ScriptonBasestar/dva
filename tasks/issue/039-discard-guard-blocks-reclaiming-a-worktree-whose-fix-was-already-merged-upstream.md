---
id: ISSUE-039
title: "Discard guard blocks reclaiming a worktree whose fix already landed upstream via a parallel session"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: low
ownership: local
discovered-in: "2026-09-23 TASK-407 rework — reopened card's fix duplicated already-merged upstream work"
discovered-at: 2026-09-23
created: 2026-09-23
---

## Summary

이 세션이 `tasks/todo/407-card-ids-are-allocated-by-scanning-the-board-so-parallel-worktrees-collide.md`
(재개된 카드, `reopen-reason`이 criterion 1 미충족을 지적)를 보고 `ce task run-start TASK-407 --type fix`로 새 워크트리
(`~/worktrees/misc/dva/claude__mbp__fix__task-407`, 브랜치
`dev/claude/mbp/fix/task-407`)를 열어 `AGENTS.md`의 "Parallel-safe card-ID
allocation" 절을 다시 썼다(커밋 `4ce30f26`).

`origin/master`로 rebase를 시도하자 `AGENTS.md`와 카드 파일 양쪽에서 충돌이
났다 — 원인은 **같은 수정이 이미 다른 세션에서 독립적으로 이루어져
`origin/master`에 통합·리뷰·완료돼 있었다**는 것이다
(`fc7c41ef` "docs(agents): stop claiming max+1 prevents card-ID collisions" →
`e5ed747b` 독립 리뷰 pass → `5870da00` 상태 동기화; 카드는 이미 `done/`이고
`quality-review: pass`다). 로컬 `master`가 [[ISSUE-037]]이 기록한 divergence
때문에 그 세 커밋을 아직 못 받아서, 이 세션이 읽은 `todo/`쪽 카드가 낡은
스냅샷이었다.

두 쪽 수정 내용은 사실상 동일하다 — 문장까지 거의 같다. 통합하면 안 되는
중복 작업이라 `git rebase --abort`로 되돌리고, 이 워크트리/브랜치를 폐기
(reclaim)하려 했으나 `guard-git-integration.sh`가 차단했다:

```
차단(정책): 워크트리 .../claude__mbp__fix__task-407 의 브랜치
dev/claude/mbp/fix/task-407 은(는) refs/remotes/origin/master 에 아직
착지하지 않았다 — SHA 도달성도 내용(patch-id)도 아니다. 회수는 통합이
성공한 뒤 같은 단계에서만 한다 ... 통합하지 않고 버리는 것은 사람이
결정할 일이고, 이 게이트에는 그 결정을 받아들일 채널이 없다(TD-71).
사용자에게 확인받고, 폐기는 사용자가 직접 실행하게 하라.
```

훅 자신이 명시한 대로, 에이전트가 우회할 수 있는 승인된 경로가 없다.

## Reproduction

1. 로컬 `master`가 `origin/master`보다 뒤처진 상태([[ISSUE-037]])에서, 이미
   `origin/master`에 병합된 수정과 우연히 같은 내용을 별도 태스크 브랜치에서
   다시 만든다.
2. `git rebase origin/master` → 같은 파일에 대한 rename/content 충돌.
3. 통합이 무의미함을 확인하고 `git rebase --abort` 후
   `git worktree remove <path>` 또는 `git branch -d <branch>` 시도.
4. `guard-git-integration.sh`가 "브랜치가 아직 `origin/master`에 착지하지
   않았다"며 폐기를 차단 — 통합 성공 시에만 회수를 허용하는 정책이, "통합할
   필요가 없어진(이미 상류에 있는) 브랜치"를 구분하지 못한다.

측정: 2026-09-23, `dev/claude/mbp/fix/task-407` `4ce30f26`, `origin/master`
`a76b2c3d`(이후 `5870da00`까지 더 전진).

## Expected vs Actual

**Expected:** 브랜치의 내용이 이미 `origin/master`에 (다른 경로로) 반영돼
있어 통합이 무의미하다고 에이전트가 판단한 경우, 그 판단을 사람에게 넘기는
승인된 단일 경로가 있다 — 예를 들어 회수 전에 사람이 확인하는 절차 안내,
혹은 명시적 "통합 없이 폐기" 플래그.

**Actual:** 훅은 "통합 성공"만을 회수의 유일한 선행 조건으로 인정한다.
"이미 상류에 중복 반영됨"은 훅이 구분할 수 없는 별개의 사례라서, 사람이
직접 `git worktree remove`/`git branch -d`/`git push origin --delete`를
실행해야 한다 — 훅 메시지 자신이 그렇게 요구한다.

## Workaround

사용자가 직접 아래를 실행해 폐기한다 (겹치는 파일 없음, 브랜치 내용은
`origin/master`의 `fc7c41ef`와 동등하므로 버려도 손실 없음):

```
! git worktree remove /Users/archmagece/worktrees/misc/dva/claude__mbp__fix__task-407
! git branch -D dev/claude/mbp/fix/task-407
! git push origin --delete dev/claude/mbp/fix/task-407
```

## 소유권 — 이 저장소다

원인은 워크스테이션 훅(`~/.claude/hooks/scripts/guard-git-integration.sh`,
TD-71이 언급하는 "통합하지 않고 버리는 결정을 받아들일 채널 없음")에 있지만,
이 카드는 DVA 저장소에서 그 빈틈에 부딪힌 관측을 남긴다. 훅 자체의 재설계는
워크스테이션 소유이므로 todo로 승격하지 않는다.

## Related

- [[ISSUE-037]] — 이 사례를 유발한 로컬/원격 divergence
- TASK-407 — `origin/master` 기준으로는 이미 `done/`에 있는 완료된 원본 작업
  (커밋 `fc7c41ef`/`e5ed747b`/`5870da00`; 로컬 `master`는 아직 미반영)

## 후속 (2026-09-24)

워크트리 `claude__mbp__fix__task-407`은 2026-09-24에도 남아 있다. 사람 폐기는
[TASK-436](../blocked/436-reclaim-duplicate-task-407-worktree.md)가 소유한다.
훅 재설계는 그 카드 밖이다.

## 2026-09-25 재확인

현재 TASK-436 작업에서도 같은 ref를 확인했다. worktree는 clean, branch HEAD는
`4ce30f26`, source `master`는 `a9d33977`이며 `master`는 해당 HEAD의 조상이 아니다.
branch upstream은 설정되지 않았고 `git ls-remote --heads origin
dev/claude/mbp/fix/task-407`는 결과가 없다. `ce task run-status task-407 --json`은
ACTIVE·`finishReady: false`·`run-status/run-abort`만 허용한다고 보고한다. 유일한
커밋은 AGENTS.md와 TASK-407 문서만 바꾸며 주요 설명은 source에 이미 반영됐지만,
카드 상태 기록은 달라졌다. 그러므로 자동 abort/branch deletion은 실행하지 않고
사람의 ref 처분 결정을 기다린다.
