---
id: ISSUE-025
title: "The worktree registry keeps a broken entry that every doctor run reports"
type: bug
status: todo
priority: P2
severity: low
ownership: local
created: 2026-09-15
discovered-at: 2026-09-15
discovered-in: "TASK-395 작업 세션 (2026-09-15)"
resolution: fixed
resolved-at: 2026-09-17T02:06:41Z
resolution-summary: "Resolved as fixed."
---

## Summary

이 저장소의 worktree 등록부(`.git/worktrees/source/`)에 **깨진 항목 하나**가 남아
있다. 매 `ce task run-doctor`가 경고 한 줄을 낸다:

```
warning: Worktrunk listed 1 item(s) with no branch (a detached worktree lists
this way); they were skipped and no task lookup is affected
```

원인은 2026-07-16 dogfood(`tmp/dogfood-dva/.../source`)가 만든 detached worktree다.
dva 저장소가 `dev-virtual-auto`에서 개명되면서 worktree 내부 `.git` 파일의 주 저장소
경로가 무효가 됐다 — 체크아웃 디렉터리는 디스크에 남아 있으므로 주 저장소는 이 항목을
"유효"로 보지만, 그 체크아웃은 이미 git 저장소로서 죽어 있다.

## Evidence

2026-09-15 실측:

```
$ git worktree list
.../dva                                       7d54663 [master]
.../mydevbox/matdosa-devbox/tmp/dogfood-dva/20260716-.../source  a701c4f (detached HEAD)

$ git -C <그 경로> status
fatal: not a git repository: /Users/archmagece/mywork/scripton/dev-virtual-auto/.git/worktrees/source

$ git worktree remove --force <그 경로>
fatal: validation failed, cannot remove working tree: '.../source/.git' is not a .git file, error code 7
```

`git worktree prune`도 지우지 않는다 — 디스크 디렉터리가 살아 있으면 prune의 대상이
아니고, remove는 검증 단계에서 같이 죽는다. 표준 인터페이스로는 못 지운다.

## Reproduction

1. 이 저장소에서 `ce task run-doctor`를 돌린다.
2. doctor는 ACTIVE로 뜨지만 detached worktree 경고 한 줄이 같이 나온다.
3. 경고가 가리키는 체크아웃에서 `git status`를 돌리면 "not a git repository"로
   죽는다 — 이미 git 저장소로서 죽어 있다.
4. `git worktree remove --force`와 `git worktree prune`을 차례로 시도한다.

둘 다 그 항목 앞에서 실패하는 것이 재현이다.

## Expected vs Actual

| | |
|---|---|
| 기대 | 깨진 worktree는 prune이나 remove로 정리된다 |
| 실제 | 둘 다 그 항목 앞에서 실패하고, 남은 방법은 등록부 디렉터리 직접 제거뿐이다 |

## Impact

낮다 — ce는 그 항목을 스킵하고 작업 조회에 영향이 없다고 스스로 밝힌다. 비용은
매 doctor 실행의 경고 한 줄과, 경고에 익숙해지는 것(다음 진짜 경고를 흘려듣는 것)이다.

## 소유권 — 이 저장소다

등록부는 이 저장소의 `.git/` 안에 있다. 2026-09-15 세션에서 등록부 내용을 목록
확인했고(표준 worktree 메타데이터 7파일, 2026-07-16 타임스탬프), 제거는
`rm -rf .git/worktrees/source`로 한정된다 — 그 자리에서 권한 거부가 있었으므로
실행은 사람 승인 아래에서만. 디스크의 체크아웃 디렉터리는 matdosa-devbox tmp
소유이므로 건드리지 않는다.

## Resolution Criteria

- [x] 등록부에서 그 항목이 사라진다 | verify: `git worktree list | /usr/bin/grep -c dogfood-dva` 출력이 0
- [x] doctor가 더는 detached 경고를 내지 않는다 | verify: `ce task run-doctor 2>&1 | /usr/bin/grep -c "no branch"` 출력이 0
