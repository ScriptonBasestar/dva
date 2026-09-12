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
  3곳(ce-agent-kit, ce-workbook, devenv) 중 어느 `.ce/task-runtime.yaml`
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
깊이 검사가 아니다. 보존할 브랜치는 `dev/` 밖에 두어야 한다.

`taskPattern` 은 마지막 방어선이지 첫 번째가 아니다 — 이 점은 처음에 안전한
방향으로 **부정확하게** 적었고(review-349 INFO 1) 소스를 확인해 정정한다.
실제 순서는 셋이다.

1. **로드 시점** — `rejectLiteralProtected`(`pkg/config/task_pattern.go:204-215`)가
   `*` 로 시작하는 패턴을 `"matches every name"` 으로 거절하고, 보호된 브랜치
   이름과 같은 패턴도 거절한다. 위험한 `taskPattern` 은 적용되는 게 아니라
   **로드에 실패**한다.
2. **`repository.IsProtected(opts.Branch)`**(`pkg/integrate/reclaim.go:44-47`)가
   가장 먼저 돈다 — target/default/integration 비교보다도, 패턴 매칭보다도 앞이다.
3. **integration/default/target 비교**(`reclaim.go:48-53`). `opts.Integration` 은
   `integrationBranch` 키에서만 채워진다.

패턴 매칭은 이 셋을 모두 통과한 뒤에야 돈다. `master` 는 `dev/` 로 시작하지 않아
닿지 않는 것에 더해 위 1~3 에서 이미 여러 번 걸러진다. reclaim 은 통합 중인
브랜치(`opts.Branch`) 하나에만 작용하며 브랜치 목록을 받지 않으므로 쓸어내지
않는다. 이 문단을 늘린 이유는 원래 문장이 `taskPattern` 을 "삭제를 막는 유일한
것" 처럼 읽히게 해서다 — 안전한 방향의 오류지만 과신을 부른다.

**`integrationBranch: [master]`** 도 함께 선언했다. 없으면 resolver 가
`origin/HEAD` 로 폴백하는데(`pkg/integrate/resolve.go:38` 의 주석이 이를
"remote-HEAD heuristic" 이라 부른다), 그것은 새로 clone 하거나 remote 를 다시
가리키면 없을 수 있는 로컬 상태다. 그때 `integrate run` 은 target 선택을 거부하고
매번 `--target` 을 요구한다.

편의 문제만은 아니다. 위 3번 가드의 `opts.Integration` 은 **이 키에서만** 채워지므로,
선언하면 reclaim 이 건드릴 수 있는 범위가 오히려 **좁아진다**. `taskPattern` 만 넣고
`integrationBranch` 를 빼는 쪽이 덜 안전한 반쪽 채택이었다.

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

### 독립 리뷰(review-349) 대응

**Finding 1 — `.gitignore` 3줄 형태 누수 (수정)**: 위 "정정" 문단 참조. 앵커된
4줄 형태로 f9f4c9f 에 반영했다.

**INFO 1 — 카드가 자기 안전 마진을 축소해 적음 (수정)**: `taskPattern` 을 첫 번째
가드처럼 읽히게 적었으나 실제로는 세 번째다. 소스를 직접 확인해(`reclaim.go:44-53`,
`task_pattern.go:204-215`) 위 문단을 늘렸다. 안전한 방향의 오류였지만, 미래의
독자가 `taskPattern` 을 "삭제를 막는 유일한 것" 으로 과신할 수 있는 형태였다.

**INFO 2 — 형제 저장소 개수 (수정)**: `4곳(ce-agent-kit, ce-workbook, devenv,
ce-devenv)` 이라고 적었으나 `~/mywork/ce/ce-devenv` 와 `~/mywork/foundation/devenv`
는 둘 다 `repository-id: devenv` 를 선언하는 **같은 논리 저장소의 두 체크아웃**이다.
체크아웃 수로는 4지만 저장소 수로는 3이라 `3곳` 으로 통일했다(뒤 문단이 이미
`3곳` 이었다). 표본 결론은 그대로다 — 어느 쪽도 `.ce/task-runtime.yaml` 에
reclaim scope 를 두지 않는다.

**(d) `integrationBranch` 는 범위 확대가 아니라는 리뷰어 판단**에 근거 하나를
추가로 받아 본문에 반영했다 — 이 키가 `opts.Integration` 의 유일한 출처이고,
그 값이 reclaim 의 거부 조건에 들어가므로 선언이 범위를 **좁힌다**. 이건 내가
카드에 쓰지 못했던 근거다.

**(e) 과장 감사**: 리뷰어가 `[x]` 로 표시된 세 기준을 모두 증거와 대조해 확인했고
근거 없는 표시는 없었다. 발견된 과장 한 건은 Finding 1 의 "ignore 를 좁혀" —
실제로는 넓혀 놓고 좁혔다고 적은 것이었고, 위에서 원본 형태·누수 증거·교체본을
모두 남기는 방식으로 정정했다.

**최종 verdict: `pass`** (review-349, 3800834 재검증 후, 조건 없음).

리뷰어가 세 기준을 각각 직접 재현했다 — `ce task run-doctor` → ACTIVE,
`grep -c 'git worktree' Makefile` → 0, `git ls-files --error-unmatch` 로 두 선언
파일이 모두 추적됨. Finding 1 은 `git status --porcelain -uall | grep -c '\.ce/'`
→ 0 으로 닫혔다. 게이트 다섯 개(`build`/`lint`/`test`/`doc-check`/`check-generate`)를
리뷰어가 직접 돌려 전부 exit 0.

**남은 증거 공백(기록, 비차단)**: 설치본 `/Users/archmagece/go/bin/gz-git` 이 이
선언으로 실제 reclaim 을 수행하는지는 리뷰어가 *관측*하지 못했다 — 직접
`gz-git integrate status` 호출이 `guard-git-integration.sh` 정책 훅에 막히고,
리뷰어는 우회하지 않았다(옳은 판단이다). 리뷰어의 소스 근거는 vendored
`ce-devenv/deps/gzh-cli-gitforge/` 사본에서 나왔고 설치 바이너리의 빌드와
일치하는지는 확인되지 않았다. 이 공백을 메우는 것은 이 세션이 관측한
`RECLAIMED … worktree local-branch remote-branch` 로그다 — b777f2c 이후 네 번의
integration(f9f4c9f, b8b0b1a, 3800834, 690c2ad)에서 매번 나왔다. 소스 독해와
실행 관측이 서로 다른 쪽에서 같은 결론에 도달한 형태이고, 어느 한쪽만으로는
닫히지 않았을 것이다.
