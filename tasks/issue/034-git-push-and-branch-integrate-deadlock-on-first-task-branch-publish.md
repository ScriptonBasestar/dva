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
narrowed-at: 2026-09-22
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

## 2026-09-22 반대 측정 — 태스크 워크트리에서는 재현되지 않는다

독립 리뷰 세션(TASK-406·409)이 같은 날 첫 발행을 성공시켰다. upstream이 없는 새
태스크 브랜치에서 `git push -u origin HEAD`가 훅에 막히지 않고 그대로 통과했다:

```
$ cd ~/worktrees/misc/dva/claude__mbp__docs__tasks-done-409-…
$ git push -u origin HEAD
 * [new branch]  HEAD -> dev/claude/mbp/docs/tasks-done-409-…
```

훅 소스가 그 차이를 설명한다. push 거부는
`~/.claude/hooks/lib/guard-git-branch.sh:228`에 있고 조건이
`on_trunk || current == integrationName`이다 — **태스크 브랜치는 둘 다 아니라서
이 거부에 애초에 닿지 않는다.** 즉 "push가 integrate를 요구한다"는 이 카드의
전제는 태스크 브랜치 자체의 성질이 아니다.

그러면 위 Reproduction의 2단계는 무엇이었나. 같은 세션이 **다른** 훅에는 실제로
막혔다 — `guard-git-integration.sh`의 "판정 대상 디렉토리를 확정하지 못했다:
cd가 명령문 맨 앞이 아니다". 그 훅은 명령문 맨 앞의 리터럴 `cd`만 대상
디렉토리로 인정하고, 뽑지 못하면 자기 cwd(= master에 고정된 프라이머리
체크아웃)로 떨어진다. 프라이머리는 `on_trunk`가 참이므로 **워크트리를 겨냥한
push가 master의 브랜치 상태로 재어져 228행 거부에 걸린다.** 훅 주석 자신이 그
오판을 금지 사유로 적는다.

**그래서 이 카드는 닫지 않고 좁힌다.** 원래 세션의 명령문 모양을 지금 볼 수
없어서 위 가설을 확정할 수 없다. 확정된 것은 둘이다 — (1) 맨 앞에 리터럴 `cd`를
둔 태스크 워크트리 push는 통과한다, (2) 228행은 트렁크/통합 브랜치에서만 발화한다.
다음 관측자가 할 일은 `cd`를 맨 앞에 두고 원래 절차를 다시 밟아, 교착이
남는지 확인하는 것이다. 남지 않으면 결함은 "교착"이 아니라 **"거부 메시지가
오판의 원인을 가리키지 않는다"**로 다시 쓰여야 한다.

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
