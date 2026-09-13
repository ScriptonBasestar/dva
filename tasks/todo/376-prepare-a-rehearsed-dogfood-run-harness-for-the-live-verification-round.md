---
id: TASK-376
title: "Prepare a rehearsed dogfood run harness for the live verification round"
type: chore
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-13
source: "2026-09-13 사람 작업 분류 — TASK-328·348이 미뤄지는 이유가 결정 부재가 아니라 착수 비용임이 드러났다"
blocks: [TASK-328, TASK-348]
---

## Summary

TASK-328과 TASK-348은 사람이 결정할 것이 남아서가 아니라 **착수 비용 때문에** 미뤄지고
있다. 두 카드 모두 "실제로 돌려보고 출력을 붙여라"인데, 돌리려면 먼저 대상 3개 저장소의
plan·entry 구성을 파악하고, 명령 순서를 조립하고, exit code를 받아적고, 리포트 형식에
맞춰 옮겨야 한다. 그 준비가 실행보다 오래 걸린다.

이 카드는 준비를 전부 미리 해 둔다. 사람에게 남는 일은 **스크립트를 실행하고 결과를
눈으로 확인하는 것**뿐이다. 실행 자체는 사람이 한다 — 이 워크스테이션은 컨테이너 105개가
도는 살아있는 개발 환경이고 `down --purge`는 named volume과 network를 지운다. 에이전트가
방아쇠를 당기지 않는 이유는 deny 규칙이 아니라 되돌릴 수 없는 실제 상태다.

## Scope

대상은 TASK-328이 지명한 3개다: `~/mydevbox/primeno1-devbox`(native 6종, gate 체인 +
`exec`), `~/mydevbox/familybook-devbox`, `~/mydevbox/flow-taskchain-devbox`(composition
plan). TASK-348의 profile-gated 실제 이미지 빌드도 같은 하네스가 다룬다 — 대상과 실행
형태가 같고, 리포트도 같은 절에 붙는다.

## Design

**1. purge 미리보기가 첫 단계다.** `down --purge` 실행 전에 그 명령이 지울 volume과
network를 먼저 나열한다. 105개 컨테이너가 도는 환경에서 이것 없이 실행을 권하는 것은
무책임하다. 미리보기 결과가 예상 밖이면 사람이 거기서 멈출 수 있어야 한다.

**2. 하네스는 파괴적 명령을 스스로 실행하지 않는다.** 기본 동작은 계획을 출력하는 것이고,
실행은 명시적 opt-in 플래그로만 일어난다. 이것은 편의가 아니라 이 카드의 안전 계약이다 —
`dva` 자신의 `--dry-run` 기본값 설계와 같은 이유다.

**3. 출력이 곧 리포트다.** 각 단계의 명령·exit code·요약을 `docs/dogfood/*.md`의 `실기동`
절에 **그대로 붙일 수 있는 형식**으로 낸다. 사람이 형식을 맞추느라 손대야 하면 준비가
덜 된 것이다.

**4. TASK-348의 대조군을 포함한다.** profile-gated 서비스가 현재 바이너리로는 빌드되고
pre-TASK-315 바이너리로는 빌드되지 않음을 같은 실행에서 보여야 한다. 어느 커밋을 대조군
바이너리로 쓸지는 이 카드가 결정해 스크립트에 고정한다.

산출물 위치는 `tools/` 아래 저장소 소유 스크립트다. `tmp/`는 ignore 대상이라 재현 가능한
절차가 되지 못한다.

## Completion Criteria

- [ ] 하네스 스크립트가 저장소에 있고 인자 없이 실행하면 파괴적 명령을 실행하지 않는다 | verify: human — 스크립트를 인자 없이 돌린 출력에 계획만 있고 `up`/`down`/`build`가 실행되지 않았음이 확인된다
- [ ] `down --purge`가 지울 volume·network를 실행 전에 나열하는 단계가 있다 | verify: human — 미리보기 출력이 이 카드 `## Evidence`에 첨부된다
- [ ] 하네스 출력이 `docs/dogfood/*.md`의 `실기동` 절 형식과 일치한다 | verify: human — 출력 한 덩어리를 리포트에 손대지 않고 붙일 수 있음이 확인된다
- [ ] TASK-348의 pre-TASK-315 대조군 바이너리 커밋이 스크립트에 고정돼 있다 | verify: human — 스크립트가 지명한 커밋과 그 선택 근거가 이 카드에 적혀 있다
- [ ] 기존 게이트 통과 | verify: `make doc-check` (regression-guard)

## Evidence

작성 시점: 2026-09-13. 파괴적 명령(`up`/`down`/`build`/`docker rm`/`volume rm`/`network rm`)은
하나도 실행하지 않았다. 아래 근거는 전부 계획 출력, 읽기 전용 docker 조회, `--dry-run`,
그리고 저장소 게이트로만 얻은 것이다.

### 산출물

- `tools/dogfoodrun/dogfood-run.sh` — 하네스 (실행 권한 있음)
- `tools/dogfoodrun/fixtures/task348-profile-build/dva.yml`
- `tools/dogfoodrun/fixtures/task348-profile-build/compose.yaml`
- `tools/dogfoodrun/fixtures/task348-profile-build/Dockerfile.gated`

### 실행한 명령과 exit code

| 명령 | exit | 비고 |
|------|------|------|
| `make build` | 0 | `bin/dva` 0.2.0 |
| `shellcheck tools/dogfoodrun/dogfood-run.sh` | 0 | 경고 없음 |
| `./tools/dogfoodrun/dogfood-run.sh` (인자 없음) | 0 | 계획만 출력, 실행 없음 |
| `./tools/dogfoodrun/dogfood-run.sh --list` | 0 | 대상 4종 |
| `./tools/dogfoodrun/dogfood-run.sh --plan familybook` | 0 | |
| `./tools/dogfoodrun/dogfood-run.sh --plan task348` | 0 | |
| `./tools/dogfoodrun/dogfood-run.sh --preview` | 0 | 읽기 전용 docker 조회 |
| `./tools/dogfoodrun/dogfood-run.sh --preview flow-taskchain` | 0 | |
| `git archive 275c8c98 \| tar -x -C tmp/dogfood-run/control-275c8c98` | 0 | git 상태 변경 없음 |
| `make -C tmp/dogfood-run/control-275c8c98 build COMMIT=275c8c98` | 0 | 대조군 `dva` 0.1.48 |
| 현재 `dva validate` (픽스처) | 0 | 경고 0 |
| 대조군 `dva validate` (픽스처) | 1 | `plans.gated.entries.0: Additional property profiles is not allowed` |
| 현재 `dva --dry-run build gated` | 0 | argv에 `--profile rust` **있음** |
| 대조군 `dva --dry-run build gated` | 0 | argv에 `--profile` **없음** |
| `make doc-check` | 0 | doccheck·cilabels·flowcheck·planprogress·yamlcheck 전부 OK |

argv 비교 원문:

```text
현재  : docker compose -f .../compose.yaml --project-name dva-dogfood-task348 --profile rust build
대조군: docker compose -f .../compose.yaml --project-name dva-dogfood-task348 build
```

### TASK-348 대조군 커밋: `275c8c98`

`275c8c98` = `d79ceaeb^`. `d79ceaeb` ("feat(plans): select compose profiles from a plan
entry")가 `PlanEntry.Profiles`를 도입한 커밋이다.

PLAN-006은 TASK-315의 착지점으로 `5f2d85d3`을 적었지만, `5f2d85d3`(문서)과 그 부모
`e2fe2551`(build/logs forwarding 수정)은 **둘 다 이미 기능을 갖고 있다**. 기능의 어떤
조각도 없는 마지막 트리는 `d79ceaeb^ = 275c8c98`이고, 대조군은 그것이어야 한다.
`5f2d85d3^`을 골랐다면 profiles가 이미 동작하는 바이너리를 "pre-TASK-315"라고 부르는
셈이라 대조 자체가 성립하지 않는다.

이 근거는 스크립트 `CONTROL_COMMIT` 선언 바로 위 주석에도 같은 내용으로 박아 두었다.

픽스처는 `version: "0.1.44"`로 선언한다. 대조군 바이너리가 0.1.48이라 `0.2.0`으로 두면
최소 버전 검사에서 먼저 걸려 정작 보고 싶은 스키마 거부·argv 차이를 볼 수 없다.

### 인자 없이 실행한 출력 (파괴적 명령 0건)

```text
dogfood-run.sh — 계획만 출력했다. 아무것도 실행하지 않았다.
실행하려면: dogfood-run.sh --execute <TARGET>
purge 미리보기만 보려면: dogfood-run.sh --preview <TARGET>

TASK-348 대조군 커밋: 275c8c98 (d79ceaeb^ — PlanEntry.Profiles 도입 직전)

## primeno1
  작업 디렉토리 : /Users/archmagece/mydevbox/primeno1-devbox
  설정 파일     : /Users/archmagece/mydevbox/primeno1-devbox/dva.yml
  리포트        : docs/dogfood/primeno1.md
  compose 프로젝트: primeno1 primeno1-external-db
  선행 확인:
    TASK-328은 "native 엔트리 6종(gate 체인 + exec)"을 말하지만, 현재 primeno1-devbox
    master(b432a01)의 dva.yml에는 native stack 엔트리가 하나도 없다. docs/dogfood/primeno1.md의
    "권장안 적용" 절이 기록한 native 6종 + plan `dev`는 devbox 저장소에 반영되지 않았고,
    gate 체인 + `exec` 핸드오프는 여전히 interaction api-run/api-run.gateway 안에 있다.
    따라서 이 하네스가 도는 gate 체인은 plan `external-db`의 script 엔트리
    (external-db-contract → compose-external-db)다. native 엔트리 검증은 devbox 설정이
    먼저 바뀌어야 가능하다.
    compose 프로젝트 `primeno1`은 이 워크스테이션에서 실제로 쓰이는 개발 환경일 수 있다.
    purge 미리보기를 반드시 먼저 읽어라.
  단계:
    [읽기]   validate
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva validate
    [읽기]   plan list
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva ls
    [기동]   up full
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva up full
    [읽기]   status
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva status
    [파괴적] down full --purge
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva down full --purge --force
    [기동]   up external-db (script gate → compose)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva up external-db
    [읽기]   status (gate chain)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva status
    [파괴적] down external-db --purge
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva down external-db --purge --force

## familybook
  작업 디렉토리 : /Users/archmagece/mydevbox/familybook-devbox
  설정 파일     : /Users/archmagece/mydevbox/familybook-devbox/dva.yaml
  리포트        : docs/dogfood/familybook.md
  compose 프로젝트: familybook-devbox
  선행 확인:
    설정 파일 이름이 아직 `dva.yaml`이다 (TASK-329로 개명 대기). composition plan은
    `--purge`에 `--project <child>`가 필수라 teardown이 infra 하위로 스코프된다
    (internal/cli/composition_flags.go). backend/dev·frontend/dev는 자식 저장소의
    native plan이라 purge 대상이 아니다.
  단계:
    [읽기]   validate
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva validate
    [읽기]   plan list
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva ls
    [기동]   up hybrid (composition)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva up hybrid
    [읽기]   status
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva status
    [파괴적] down hybrid --purge (scoped to infra)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva down hybrid --purge --project infra --force

## flow-taskchain
  작업 디렉토리 : /Users/archmagece/mydevbox/flow-taskchain-devbox
  설정 파일     : /Users/archmagece/mydevbox/flow-taskchain-devbox/dva.yml
  리포트        : docs/dogfood/flow-taskchain.md
  compose 프로젝트: taskchain
  선행 확인:
    composition plan이라 teardown은 `--project local-infra`로 스코프된다. engine/mcp/portal은
    자식 저장소의 native plan이며 purge 대상이 아니다.
  단계:
    [읽기]   validate
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva validate
    [읽기]   plan list
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva ls
    [기동]   up local-dev (composition)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva up local-dev
    [읽기]   status
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva status
    [파괴적] down local-dev --purge (scoped to local-infra)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva down local-dev --purge --project local-infra --force

## task348
  작업 디렉토리 : /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/tools/dogfoodrun/fixtures/task348-profile-build
  설정 파일     : /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/tools/dogfoodrun/fixtures/task348-profile-build/dva.yml
  리포트        : tasks/todo/348-confirm-plan-profiles-reach-a-real-docker-build-not-just-argv.md
  compose 프로젝트: dva-dogfood-task348
  선행 확인:
    대조군은 이 저장소가 스스로 만든다: git archive로 대조 커밋 트리를 tmp/에 풀고
    그 안에서 make build를 돌린다. 저장소 체크아웃·브랜치·worktree는 건드리지 않는다.
    대조군 바이너리는 config version 0.1.48을 보고하고, 픽스처는 0.1.44로 선언해
    두 바이너리 모두 로드할 수 있게 맞춰 두었다.
  단계:
    [읽기]   current validate
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva validate
    [읽기]   control validate — schema는 profiles를 거부한다 (exit 1 기대)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/tmp/dogfood-run/control-275c8c98/bin/dva validate
    [읽기]   argv: current, gated (--profile rust 있어야 한다)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva --dry-run build gated
    [읽기]   argv: control, gated (--profile 없어야 한다)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/tmp/dogfood-run/control-275c8c98/bin/dva --dry-run build gated
    [읽기]   출발점: 이미지가 없어야 한다 (exit 1 기대)
             docker image inspect dva-dogfood-task348-gated:latest
    [기동]   대조군 빌드: control, gated
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/tmp/dogfood-run/control-275c8c98/bin/dva build gated
    [읽기]   대조군 이후에도 이미지 없음 (exit 1 기대)
             docker image inspect dva-dogfood-task348-gated:latest
    [기동]   프로필 없는 플랜: current, legacy
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva build legacy
    [읽기]   legacy 이후에도 이미지 없음 (exit 1 기대)
             docker image inspect dva-dogfood-task348-gated:latest
    [기동]   실제 빌드: current, gated
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva build gated
    [읽기]   이미지 생성 확인 (exit 0 기대)
             docker image inspect --format '{{.Id}} {{.Created}}' dva-dogfood-task348-gated:latest
    [읽기]   빌드 결과물 확인
             docker run --rm dva-dogfood-task348-gated:latest cat /task348-marker
    [파괴적] 픽스처 teardown (프로젝트·볼륨·local 이미지 제거)
             /Users/archmagece/worktrees/misc/dva/claude__mbp__chore__dogfood-run-harness/bin/dva down gated --purge --force
    [읽기]   teardown 후 이미지 없음 (exit 1 기대)
             docker image inspect dva-dogfood-task348-gated:latest

주의: 이 워크스테이션은 컨테이너가 다수 도는 살아있는 개발 환경이다.
[파괴적] 단계는 named volume·network·local 이미지를 지운다. --execute 전에
반드시 --preview 출력을 눈으로 확인하라.
```

### purge 미리보기 (읽기 전용)

`docker ps/volume ls/network ls`의 `com.docker.compose.project` 라벨 필터만 쓴다. 두 번째
"이름만 비슷함" 목록은 라벨이 일치하는 항목을 빼고 낸다 — 같은 이름이 두 목록에 다 오르면
미리보기가 스스로를 반박하기 때문이다.

```text
## purge 미리보기 — primeno1
`dva down ... --purge`는 아래 프로젝트를 `docker compose down --remove-orphans --volumes --rmi local`로 지운다.

### compose project: primeno1
  containers: (없음)
  volumes: (없음)
  networks: (없음)
  images: (없음)
  networks (이름만 비슷함 — purge 대상 아님): (없음)
  volumes (이름만 비슷함 — purge 대상 아님): (없음)

### compose project: primeno1-external-db
  containers: (없음)
  volumes: (없음)
  networks: (없음)
  images: (없음)
  networks (이름만 비슷함 — purge 대상 아님): (없음)
  volumes (이름만 비슷함 — purge 대상 아님): (없음)

## purge 미리보기 — familybook
`dva down ... --purge`는 아래 프로젝트를 `docker compose down --remove-orphans --volumes --rmi local`로 지운다.

### compose project: familybook-devbox
  containers: (없음)
  volumes: (없음)
  networks: (없음)
  images: (없음)
  networks (이름만 비슷함 — purge 대상 아님): (없음)
  volumes (이름만 비슷함 — purge 대상 아님): (없음)

## purge 미리보기 — flow-taskchain
`dva down ... --purge`는 아래 프로젝트를 `docker compose down --remove-orphans --volumes --rmi local`로 지운다.

### compose project: taskchain
  containers: (없음)
  volumes:
    taskchain_postgres-data
    taskchain_redis-data
  networks: (없음)
  images: (없음)
  networks (이름만 비슷함 — purge 대상 아님): (없음)
  volumes (이름만 비슷함 — purge 대상 아님):
    taskchain-qa024_postgres-data
    taskchain-qa024_redis-data

## purge 미리보기 — task348
`dva down ... --purge`는 아래 프로젝트를 `docker compose down --remove-orphans --volumes --rmi local`로 지운다.

### compose project: dva-dogfood-task348
  containers: (없음)
  volumes: (없음)
  networks: (없음)
  images: (없음)
  networks (이름만 비슷함 — purge 대상 아님): (없음)
  volumes (이름만 비슷함 — purge 대상 아님): (없음)
```

### 리포트 블록 형식

`--execute <TARGET>`이 마지막에 내는 블록의 형식은 다음과 같다 (`emit_report`).
`docs/dogfood/*.md`에는 아직 `실기동` 절이 없어서 이 하네스가 그 형식을 정의한다.

````text
## 실기동 (<타임스탬프>, dva version X.Y.Z)

- 대상: `<devbox>/dva.yml`
- 하네스: `tools/dogfoodrun/dogfood-run.sh --execute <TARGET>`
- compose 프로젝트: <프로젝트 목록>
- 전체 출력: `tmp/dogfood-run/<TARGET>-<ts>.log`

| 명령 | exit | 마지막 출력 줄 |
|------|------|----------------|
| ... | ... | ... |

### 선행 확인
...

### purge 미리보기 (파괴적 단계 실행 전)
```text
...
```
````

실제로 채워진 블록은 `--execute`를 돌려야 나오므로 이 카드에는 없다. 실기동 회차에서
붙여넣기 그대로 되는지 확인해야 할 마지막 항목이다.

### 검증하지 못한 것

- `--execute` 경로 전체(`up`/`down --purge`/`build`)는 이 워크스테이션이 컨테이너 105개가
  도는 살아있는 개발 환경이라 실행하지 않았다. 이 카드의 안전 계약상 실행은 사람의
  명시적 opt-in에서만 일어난다.
- **primeno1의 native 엔트리는 존재하지 않는다.** TASK-328은 "native 엔트리 6종(gate 체인
  + exec)"을 전제하지만 `primeno1-devbox` master(`b432a01`)의 `dva.yml`에는 native stack
  엔트리가 하나도 없다. `docs/dogfood/primeno1.md`의 "권장안 적용" 절이 기록한 native 6종과
  plan `dev`는 devbox 저장소에 반영되지 않았고, gate 체인과 `exec` 핸드오프는 여전히
  interaction `api-run`/`api-run.gateway` 안에 있다. 그래서 하네스는 plan `external-db`의
  script 엔트리 gate 체인을 대신 돈다. native 엔트리 검증은 devbox 설정이 먼저 바뀌어야
  가능하다 — TASK-328의 해당 항목은 이 하네스로 닫을 수 없다.
- `familybook-devbox`의 설정 파일은 아직 `dva.yaml`이다 (TASK-329 개명 대기). 하네스는
  현재 이름을 그대로 읽는다.
