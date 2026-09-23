---
id: ISSUE-037
title: "branch-integrate refuses to push a master-local housekeeping commit"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: low
ownership: local
discovered-in: "2026-09-23 TASK-410 독립 리뷰 — review-run이 카드를 done/으로 옮기며 master에 직접 커밋"
discovered-at: 2026-09-23
created: 2026-09-23
---

## Summary

`task.quality-review` 스킬의 `ce task review-run` 절차는 카드를 `review/` → `done/`로
옮기는 커밋을 (기존 관례대로) 태스크 브랜치 없이 로컬 `master`에 직접 만든다. 그
결과 로컬 `master`가 `origin/master`보다 1커밋 앞서지만, 그 커밋을 push할 방법이
없다:

- `git push origin master` → 훅이 차단: "대상 브랜치 push는 설치본 PATH 래퍼
  branch-integrate를 단일 bare 명령으로 수행하라."
- `branch-integrate` (단일 bare, 같은 체크아웃) → 실패: "implicit source branch is
  the integration target: master; run branch-integrate from a task-branch worktree"

`branch-integrate`는 태스크 브랜치 워크트리에서 target으로 병합·push하는 도구라
source==target(=master 그 자체)인 경우를 다루지 않는다. [[ISSUE-034]]와 증상이
닮았지만 다른 경로다 — 034는 "새 태스크 브랜치의 첫 발행", 이것은 "이미 master에
있는 로컬 전용 커밋의 push"다.

이것은 워크스테이션 launcher/hook의 빈틈이지 DVA 제품 결함이 아니다. DVA todo로
승격하지 않는다.

## Reproduction

1. `master`에서 카드 상태 변경만 담은 커밋을 만든다 (예: `ce task review-run` →
   `ce task move ... done` → `git commit`).
2. `git push origin master` 실행 → 정책 차단.
3. 같은 체크아웃에서 `branch-integrate` 실행 → "run branch-integrate from a
   task-branch worktree"로 실패.

측정: 2026-09-23, DVA 저장소, 로컬 `master` `eb230bc7` vs `origin/master`
`a2e75dd3` (정확히 1커밋 차이, fast-forward 가능).

## Expected vs Actual

**Expected:** a task-board housekeeping commit made directly on the primary
checkout's `master` (this repo's established convention for `ce task move`/
review-stamp commits) can reach `origin/master` through some approved,
single-bare-command path — either a direct push allowance for this commit
shape, or a `branch-integrate` mode that handles "checkout already holds the
target branch" instead of only "task-branch worktree merging into target."

**Actual:** both approved paths refuse it. `git push origin master` is
blocked by the push guard unconditionally for the target branch. `branch-
integrate` refuses with "implicit source branch is the integration target:
master; run branch-integrate from a task-branch worktree" because it assumes
its caller is always a task-branch worktree, never the target checkout
itself. No launcher covers "local target-branch checkout has commits target
doesn't" — the commit is stuck local-only until a human runs a raw command.

## Workaround

에이전트가 우회할 수 있는 승인된 경로가 없다. 사용자가 직접 처리해야 한다.

**2026-09-23 갱신 — 단순 push에서 merge 필요로 악화됐다.** TASK-407 리뷰가
`branch-integrate`로 정상 통합되면서 `origin/master`가 `a2e75dd3` 이후 별도로
전진했다(`fc7c41ef` → `e5ed747b` → `5870da00`). 로컬 `master`는 여전히
`a2e75dd3` 위에 `eb230bc7`(TASK-410 리뷰 커밋) 하나만 얹은 상태라 이제 단순
fast-forward push가 아니라 두 갈래를 합쳐야 한다. 같은 훅이 `git merge
origin/master`도 막는다: "대상 브랜치의 raw merge는 허용하지 않는다." —
`branch-integrate`는 이 체크아웃(=target 그 자체)에서 여전히 소스=타겟이라
거부한다.

사용자가 직접 실행할 명령 (둘 중 하나, 충돌 없음 — 겹치는 파일 없음):
```
! git -C /Users/archmagece/mywork/scripton/dva merge origin/master
! git -C /Users/archmagece/mywork/scripton/dva push origin master
```
또는 `eb230bc7`가 반영하는 TASK-410 review-stamp를 다시 태스크 브랜치에서
만들어 `branch-integrate`로 정식 통합하는 방법도 있으나, 이미 만들어진 로컬
커밋을 폐기해야 하므로 위 2줄이 더 간단하다.

## Downstream impact — TASK-411's review is blocked on this

`ce task done-finalize tasks/done/410-repository-tooling-hardcodes-the-legacy-archive-spelling-and-cannot-see-archive.md --dry-run` refuses outright:
"configured source tip `master` does not match locally recorded upstream
`origin/master`" — it requires local `master` synced with `origin/master`
before it will even dry-run. TASK-410's done card carries `blocks:
[TASK-411]`, and `ce task validate`/`gate` require a `quality-review-receipt`
(host-issued, presumably by `done-finalize`) on any done card that blocks
another. So until this card's merge+push is done, `done-finalize` can't run,
the receipt can't be issued, and TASK-411's independent review stays
`blocked/` (see `tasks/blocked/411-fold-the-legacy-archive-spelling-into-archive.md`'s
`## Review Attempts`). Resolving the two-line workaround above is the
critical-path unblock for TASK-411, not just board hygiene.

## 소유권 — 이 저장소다

원인은 워크스테이션 설치본(`~/.claude/hooks/scripts/guard-git-integration.sh`,
`branch-integrate` launcher)에 있지만, `ownership:`이 가리는 축은 "누가
고치는가"가 아니라 "이 저장소가 취할 행동이 있는가"다. 이 카드는 DVA
저장소의 기존 관례(태스크보드 하우스키핑 커밋을 브랜치 없이 로컬 `master`에
직접 만드는 것)가 워크스테이션 도구의 가정(항상 태스크 브랜치 워크트리에서
호출된다)과 부딪히는 지점을 기록한다 — DVA 쪽이 관례를 계속 쓸지, 이런
커밋도 태스크 브랜치를 거치게 바꿀지 판단해야 하므로 로컬로 분류한다.
launcher/hook 자체의 재설계는 [[Out of scope]]에 남긴다.

## Out of scope

- launcher/hook 자체의 재설계 — 워크스테이션 소유
- [[ISSUE-034]]의 재현/종결 판단 — 별개 카드

## Sources

- `~/.claude/hooks/scripts/guard-git-integration.sh` — push 차단 메시지 출처
- `branch-integrate --help` — "Run a bare integrate from a task-branch worktree,
  not from the target checkout."
- [[ISSUE-034]] — 닮은 증상, 다른 경로
