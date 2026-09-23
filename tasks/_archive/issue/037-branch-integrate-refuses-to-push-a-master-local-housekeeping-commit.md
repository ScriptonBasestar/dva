---
id: ISSUE-037
title: "branch-integrate refuses to push a master-local housekeeping commit"
type: bug
status: done
priority: P2
effort: S
exec-tier: standard
severity: low
ownership: local
discovered-in: "2026-09-23 TASK-410 독립 리뷰 — review-run이 카드를 done/으로 옮기며 master에 직접 커밋"
discovered-at: 2026-09-23
created: 2026-09-23
resolution: fixed
resolved-at: 2026-09-23T04:01:44Z
resolution-summary: "Resolved as fixed by TASK-416."
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

**2026-09-23 재갱신 — divergence가 8/8로 더 벌어짐, merge 경로는 여전히
유효할 것으로 보임(파일 하나가 겹치지만 라인은 안 겹침).** 이 세션의
하우스키핑 커밋(ISSUE-034 wontfix 확정, BACKLOG-009 정리, 이 카드 자체,
ISSUE-039)이 로컬 쪽에 4개 더 쌓여 로컬이 `eb230bc7`(TASK-410) 기준 8개 앞,
`origin/master`도 TASK-407·413·414 통합으로 8개 앞이 됐다
(`git rev-list --left-right --count origin/master...master` → `8  8`, 분기점
`a2e75dd3`). `git diff a2e75dd3 master --name-only`와
`git diff a2e75dd3 origin/master --name-only`로 양쪽이 건드린 파일을 실측
대조한 결과, 겹치는 파일이 하나 있다 — BACKLOG-009 카드: 로컬은
문서 끝에 새 절을 추가했고 원격은 frontmatter에 `status: backlog` 한 줄을
끼워 넣었다. 서로 다른 라인 범위라 3-way merge가 충돌 없이 자동 병합될
가능성이 높지만, 실제 merge를 실행하기 전까지는 git이 그렇게 판단할지
확정할 수 없다 — 나머지 겹치지 않는 파일들과 달리 이 파일만은 사람이 병합
직후 diff를 눈으로 확인하는 편이 안전하다.

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

## 2026-09-23 3차 갱신 — 승인된 우회로가 없음을 실측으로 확정

`--help` 텍스트 재검토에 그치지 않고 프라이머리 체크아웃(현재 `master`, 로컬
전용 커밋 보유)에서 다음 네 가지를 실제로 실행해 봤다:

```
$ branch-integrate --target origin/master
exit 1: implicit source branch is the integration target: master; run
branch-integrate from a task-branch worktree
$ branch-integrate --direct-to-default --target origin/master
exit 1: implicit source branch is the integration target: master; run
branch-integrate from a task-branch worktree
$ gz-git integrate run --target origin/master
차단(정책, 훅): Claude의 직접 gz-git integrate 호출은 검사한 바이너리와 실제
바이너리를 결속할 수 없다 — launcher(branch-integrate)만 단일 명령으로 허용.
```

`--target`과 `--direct-to-default`는 `--help`가 적은 대로 "통합 브랜치를 못
찾을 때"를 위한 플래그였고, 이 저장소는 `.gz-git.yaml`에
`integrationBranch: master`를 이미 선언해 그 조건에 안 걸린다 — 그래서 둘 다
같은 "source==target" 거부로 떨어진다. launcher를 우회해 `gz-git` 바이너리를
직접 부르는 것도 별개 훅이 정책으로 막는다. `.gz-git.yaml`에는 이 상황(체크아웃
자체가 이미 target)을 위한 필드가 없다(readiness/branch 두 섹션만 허용, 위
파일 주석 참조). **결론: 이 저장소·이 훅 조합에서 에이전트가 접근 가능한
우회 경로는 없다 — 추측이 아니라 네 가지 실행 결과로 확정.**

## 2026-09-23 4차 갱신 — todo/의 4장 중 3장이 이미 origin에서 처리 완료

이 divergence 때문에 로컬 `tasks/todo/`가 신뢰 불가 상태다. 직접 대조:

| 카드 | 로컬 `tasks/todo/` | `origin/master` |
|---|---|---|
| TASK-407 | `todo/` (재개 카드) | `done/` (완료, `quality-review: pass`) — [[ISSUE-039]] |
| TASK-413 | `todo/` | `done/` |
| TASK-414 | `todo/` | `done/` |
| TASK-370 | `todo/` | (동일, 다음 `release-notes/v*.md`까지 의도적 보류) |

즉 로컬에서 보이는 `todo/` 4장 중 3장(407·413·414)은 이미 upstream에서 끝난
스테일 스냅샷이고, 나머지 1장(370)은 원래부터 보류 대상이다. **로컬 master가
`origin/master`와 합쳐지기 전까지 이 저장소에는 새로 시작해도 안전한 todo
항목이 실질적으로 0장이다** — `run-start`를 아무 카드에 대해서든 돌리면
[[ISSUE-039]]와 같은 중복 작업을 다시 만들 위험이 있다. 이 사실 자체가 왜
ISSUE-037의 두 줄 merge/push가 board hygiene이 아니라 critical path인지의
세 번째 증거다(TASK-411 리뷰 차단, 중복 작업 위험에 이어).

## Downstream impact — `run-start` silently branches from the stale side

2026-09-23 관측: `ce task run-start`는 `origin/<source>`가 아니라 로컬
`master`에서 새 태스크 브랜치를 딴다. 이 카드가 기록하는 divergence가 열려
있는 동안 새로 시작하는 모든 태스크는 이미 `origin/master`에 반영된 커밋을
못 보고 시작하며, 경고도 없다. 실제로 이 세션에서 TASK-407의 재개된 카드를
다시 고치는 워크트리를 열었더니, 정확히 같은 수정이 다른 세션에서 이미
`origin/master`에 병합·완료돼 있었다 — 로컬이 그 사실을 볼 수 없어서였다.
그 중복 작업의 폐기 시도가 [[ISSUE-039]]를 낳았다. 즉 이 카드가 열려 있는
기간이 길어질수록 TASK-411 지연뿐 아니라 이런 중복·낭비 작업의 위험도
누적된다.

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

## 2026-09-23 해결 확인

검토 시 로컬 master, origin/master 및 원격 refs/heads/master가
`1d7544b5d8bca73d55a30669e1b1268afe9f811b`로 일치했다. 이후 TASK-415도
정식 ce task run-finish로 c33d946b에 통합·push·회수됐다. 분기 차단은 해소됐고
TASK-410의 별도 finalize 제한은 ISSUE-024가 소유한다. 과거 재현은 아래 기록으로 보존한다.
