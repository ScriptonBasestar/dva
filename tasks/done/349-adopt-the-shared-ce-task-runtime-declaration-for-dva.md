---
id: TASK-349
title: "Adopt the shared CE task-runtime declaration for dva"
type: chore
priority: P2
effort: S
exec-tier: standard
created: 2026-09-08
status: done
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

- [x] .ce/task-runtime.yaml exists and ce task run-doctor reports ACTIVE instead of BLOCKED | verify: `ce task run-doctor`
- [x] The declaration is adopted rather than reimplemented: no `git worktree` lifecycle logic is added to the Makefile | verify: `test 0 -eq "$(/usr/bin/grep -c 'git worktree' Makefile)"` (regression-guard)
- [x] Adoption is a decision, not a default | verify: human — 카드 하단 "Design record" 절에 CE 공유 lifecycle 채택 여부와 근거, reclaim 선언 범위(worktree/local branch/remote branch)가 기록되었는지 확인

## Design record

**채택 결정**: CE 공유 lifecycle 을 채택한다. 사용자가 명시적으로 선택했다
(대안: 수동 회수 유지 + wontfix, 카드 전제만 정정 후 groom). 저장소 로컬
Makefile/셸 worktree 로직은 추가하지 않았다 — `grep -c 'git worktree' Makefile`
= 0.

**카드 전제 정정**: 이 카드는 `RECLAIM skipped: reclaim nothing: no declaration`
의 원인을 `.ce/task-runtime.yaml` 부재로 지목했으나 그것은 틀렸다. 두 선언은
서로 다른 파일이 읽는 서로 다른 계약이다.

- `.gz-git.yaml`(저장소 루트) — `gz-git integrate run` 의 reclaim 이 읽는 유일한
  파일이다. 파일 자신의 주석이 "Only this file is read. Missing file means
  reclaim nothing" 이라고 적고 있고, 실제 원인은 이쪽의 부재였다. 형제 저장소
  4곳(ce-agent-kit, ce-workbook, devenv, ce-devenv) 중 어느 `.ce/task-runtime.yaml`
  에도 reclaim scope 가 없는 이유가 이것이다 — 그 키는 거기 살지 않는다.
- `.ce/task-runtime.yaml` — 저장소 **신원** 선언이다. `ce task run-start` /
  `run-status` / `run-finish` 가 이것이 없으면 BLOCKED 로 멈춘다. `run-doctor` 를
  ACTIVE 로 만드는 것은 이 파일이지 reclaim 이 아니다.

원인이 전자였으므로 후자만 추가했다면 `RECLAIM skipped` 는 그대로 떴을 것이다.
두 파일을 모두 추가했다.

**reclaim 선언 범위** — 카드는 worktree / local branch / remote branch 중 어느
것을 선언이 소유할지 "신중히 고르라"고 요구하지만, `.gz-git.yaml` 스키마에는 그
셋을 나누는 표현이 없다. `branch.taskPattern` 에 걸리면 **셋이 함께** 회수된다.
따라서 실제로 고를 수 있는 것은 "어떤 브랜치 이름이 회수 대상인가" 하나뿐이다.
`dev/*/*/*` 를 선언했다 — 형제 저장소 전부와 같은 값이다.

`MatchTaskPattern` 은 glob 이 아니다. 첫 `*` 앞의 텍스트만 남겨 접두사 비교하고
나머지는 버리므로, 이 선언은 `dev/` 로 시작하는 **모든** 브랜치를 깊이와 무관하게
회수한다 — `dev/a/b/c` 와 `dev/scratch` 가 똑같이 걸린다. 즉 `dev/*/*/*` 와
`dev/*` 는 동일한 선언이며, 세 segment 는 브랜치 관례를 눈에 보이게 적어둔 것이지
깊이 검사가 아니다. 보존할 브랜치는 `dev/` 밖에 두어야 한다. `master` 는 `dev/`
로 시작하지 않아 닿지 않고, reclaim 은 integration/default 브랜치를 이 파일을
읽기 전에 이미 거부한다. reclaim 은 통합 중인 브랜치에만 작용하며 쓸어내지 않는다.

**`integrationBranch: [master]`** 도 함께 선언했다. 없으면 resolver 가
`origin/HEAD` 로 폴백하는데, 그것은 새로 clone 하거나 remote 를 다시 가리키면
없을 수 있는 로컬 상태다. 그때 `integrate run` 은 target 선택을 거부하고 매번
`--target` 을 요구한다.

**검증**: `ce task run-doctor` → `ACTIVE: task runtime dependencies are ready`.
이 카드 자신의 통합에서 reclaim 이 실제로 동작하는지가 마지막 증거다.

**선언이 추적되지 않는 문제** — 파일을 쓰고 나서야 `.ce/` 가 무시되고 있다는 것을
발견했다. 저장소 `.gitignore` 의 `# Claude/AI` 블록에 `.claude/`·`.serena/`·
`.cursor/` 와 나란히 있었고, `~/.gitignore_global` 에도 같은 항목이 있다. 즉
어시스턴트 스크래치로 묶여 무시된 것이지 CE 계약을 의도적으로 거부한 것이 아니다 —
형제 저장소 3곳은 모두 자기 `.ce/task-runtime.yaml` 을 **추적**한다.

추적되지 않으면 채택이 이 작업트리 안에서만 성립한다. 새 clone 이나 다른 장비는
그대로 BLOCKED 다. 그래서 ignore 를 좁혀 이 파일 하나만 추적한다. 세 줄이 필요했다:

```
!.ce/
.ce/*
!.ce/task-runtime.yaml
```

`!.ce/task-runtime.yaml` 한 줄로는 되지 않는다. **무시된 디렉토리 안의 파일은
negation 으로 되살릴 수 없다** — git 이 디렉토리 자체를 제외하면 안을 열어보지
않으므로 안쪽 규칙에 닿지 못한다. 그래서 디렉토리를 먼저 되살리고, 디렉토리가
아니라 **내용물**을 다시 무시한 뒤, 마지막에 파일 하나만 예외로 꺼낸다.
`~/.gitignore_global` 은 프로젝트 밖이라 건드리지 않았다.

**정정 — 처음 쓴 3줄 형태는 새는 형태였다 (review-349 Finding 1)**. 최초 커밋
b777f2c 에서 나는 이것을 세 줄로 썼다:

```
!.ce/
.ce/*
!.ce/task-runtime.yaml
```

이유 설명은 맞았지만 구현이 틀렸다. 선행 `/` 가 없어 **앵커되지 않았고**, 맨 앞의
`.ce/` 를 없앤 탓에 루트뿐 아니라 **모든 깊이**의 `.ce/` 가 함께 열렸다. 리뷰어가
스크래치 저장소로 실증했고, 나도 이 저장소에서 같은 탐침으로 재현했다 —
`sub/.ce/cache.json` 과 `deep/er/.ce/cache.json` 을 두면 3줄 형태에서
`?? sub/` · `?? deep/` 가 뜬다. 런타임 캐시가 추적 후보로 새는 것이다.

형제 저장소(ce-workbook, ce-devenv)가 쓰는 앵커된 4줄 형태로 고쳤다:

```
.ce/
!/.ce/
/.ce/*
!/.ce/task-runtime.yaml
```

앞의 두 줄을 하나로 합칠 수 없다는 것이 핵심이다. `.ce/` 는 어느 깊이든 무시하고,
`!/.ce/` 는 선행 `/` 로 **루트의 것 하나만** 되살린다. 같은 탐침에서 이 형태는
아무것도 새지 않는다. ce-devenv 의 주석은 그 트리에만 `.ce/` 가 17곳 있다고
적고 있다 — 앵커 누락은 작은 실수가 아니다.
