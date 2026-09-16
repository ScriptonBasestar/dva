---
id: TASK-376
title: "Prepare a rehearsed dogfood run harness for the live verification round"
type: chore
priority: P2
effort: M
exec-tier: standard
status: done
created: 2026-09-13
source: "2026-09-13 사람 작업 분류 — TASK-328·348이 미뤄지는 이유가 결정 부재가 아니라 착수 비용임이 드러났다"
blocks: [TASK-328, TASK-348]
quality-review: conditional
quality-reviewed-at: 2026-09-14
quality-review-evidence: "독립 리뷰 review-376(core:code-reviewer, 저자 아님). Critical 0 / High 2 / Medium 3 / Low 5. High 2건(스텝 루프의 stdin 삼킴, bare word가 --execute 대상을 덮어씀)과 Medium 2건(purge 미리보기가 docker 오류를 자원 목록처럼 출력, 기동 실패 후 파괴적 스텝 강행), Low 5건을 같은 브랜치에서 전부 수정했다. 세 번째 Medium(카드 Evidence staleness)은 인자 없는 출력 블록을 현재 출력으로 재생성하고 primeno1 항목을 TASK-379 재조준 사실로 대체해 닫았다. --execute 경로는 이 워크스테이션이 컨테이너 105개가 도는 살아있는 개발 환경이라 여전히 정적 리뷰로만 닫았다 — conditional의 근거가 이것이다"
quality-review-receipt: tasks/done/evidence/TASK-376/done-review-9515457848da06f480f0b08878eed6c35f3f88811bc3c01b62c55af7f16d9d78.json
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

- [x] 하네스 스크립트가 저장소에 있고 인자 없이 실행하면 파괴적 명령을 실행하지 않는다 | verify: human — 스크립트를 인자 없이 돌린 출력에 계획만 있고 `up`/`down`/`build`가 실행되지 않았음이 확인된다
- [x] `down --purge`가 지울 volume·network를 실행 전에 나열하는 단계가 있다 | verify: human — 미리보기 출력이 이 카드 `## Evidence`에 첨부된다
- [x] 하네스가 `docs/dogfood/*.md`에 붙일 `실기동` 절 형식을 정의하고 그 형식으로 출력한다 | verify: human — 카드 `## Evidence`의 형식 블록이 `emit_report`가 내는 것과 같은지 확인한다
      (2026-09-13 기준 변경: 원래 문구는 "기존 `실기동` 절 형식과 일치한다"였으나 그런 절은
      `docs/dogfood/` 어디에도 없다 — 4건의 `실기동` 문자열은 전부 산문 언급이다. 맞출 대상이
      없으므로 하네스가 형식을 정의하는 쪽이 맞고, "손대지 않고 붙여넣어진다"는 확인은
      `--execute` 출력이 있어야만 가능해 [[TASK-328]]·[[TASK-348]]로 옮겼다.)
- [x] TASK-348의 pre-TASK-315 대조군 바이너리 커밋이 스크립트에 고정돼 있다 | verify: human — 스크립트가 지명한 커밋과 그 선택 근거가 이 카드에 적혀 있다
- [x] 기존 게이트 통과 | verify: `make doc-check` (regression-guard)

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

2026-09-14 재확인(리뷰 지적 반영 후): 인자 없음 / `--list` / `--list familybook` /
`--plan familybook` / `--plan task348` / `--preview` / `--preview flow-taskchain` 전부 exit 0,
docker 상태(컨테이너 105 · 볼륨 791 · 네트워크 31)는 실행 전후 동일했다. 대상을 두 번
넘기면(`--list familybook flow-taskchain`) exit 1로 거부된다.

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

2026-09-14에 재생성했다. 아래는 현재 `tools/dogfoodrun/dogfood-run.sh`가 인자 없이
내는 출력 그대로이며, 저장소 절대 경로만 `<REPO_ROOT>`로 치환했다 — worktree 경로를
박아 두면 통합과 동시에 다시 낡기 때문이다. `[파괴적]`으로 표시된 줄은 계획일 뿐
실행되지 않았다.

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
    아래 스텝은 TASK-328 첫 기준의 대상이다 — plan `dev`를 먼저 돌고 `external-db`를
    이어 돈다(TASK-379로 재조준했다).
    
    **선행 조건: 체크아웃이 origin/master여야 한다.** plan `dev`는 0caeaf9에서 들어왔다.
    작성 시점의 로컬 체크아웃 b432a01에는 native 엔트리가 0건이고 plan `dev`도 없어
    `dva up dev`가 unknown plan으로 죽는다. 실기동 전에 primeno1-devbox를 origin/master로
    올려라 — `dva ls` 출력에 `dev`가 보이는지로 확인한다.
    
    **선행 조건: 회차는 order 10의 sigdock 게이트에서 먼저 죽는다.** 엔트리 체인은
    sigdock-local-runtime(script) -> compose -> api/frontend(native) -> gateway(native)이고,
    첫 관문인 scripts/sigdock-local-up.sh는 fail-closed다. 이 게이트가 요구하는 것:
      - SIGDOCK_CLIENTS_FILE — dva.yml에도 .env에도 .env.example에도 없다.
        env/templates/.env.template와 docs/LOCAL_EXECUTION_GUIDE.md에만 있으므로 회차에서
        따로 넣어야 한다.
      - sigdock-idp compose 프로젝트의 컨테이너·네트워크가 0건일 것. 하나라도 남아 있으면
        "refusing to mutate resources this invocation does not own"으로 즉시 실패한다.
        (2026-09-13 이 워크스테이션 실측: 컨테이너 1건(sigdock-idp-postgres-1, exited),
        네트워크 1건(sigdock-idp_default) — 지금 돌리면 여기서 끝난다.)
      - 포트 11300에 리스너 없음, TMPDIR 아래 ownership marker 없음.
      - $SIGDOCK_DEVBOX_DIR(기본 ../sigdock-idp-devbox)가 cd 가능한 디렉터리일 것(:258),
        그 안의 dva.yml이 파일로 존재할 것(:400). 별개 검사인데 실패 메시지가 같다
        (adjacent SigDock devbox not found; 뒤쪽만 경로를 덧붙인다).
      - scripts/sigdock-local-contract.sh가 실행 가능(-x).
      - dva/docker/curl/lsof 네 바이너리가 PATH에 있을 것.
      - sigdock.localhost가 loopback 주소로만 해석될 것
        (require_loopback_provider_host, 내부적으로 python3을 쓴다).
      - SIGDOCK_IDP_ISSUER_PROFILE=fapi2 — 이것은 dva.yml 최상위 vars 블록에 이미 있다.
      검사 순서: devbox 디렉터리(:258) -> 바이너리 -> fapi2 -> loopback -> devbox dva.yml
      -> contract 실행 비트 -> SIGDOCK_CLIENTS_FILE -> ownership marker -> 컨테이너 ->
      네트워크 -> 포트 리스너. 위반 중인 둘은 7번과 9/10번이므로 앞 여섯 관문을 통과한
      뒤에 실패한다.
    
    게이트를 넘긴 뒤에야 두 번째 표면이 나온다: api는 PRIMENO1_ENGINE_DIR(기본
    primeno1-engine-kt)에서 Gradle bootRun을, frontend는 primeno1-frontend에서 npm run dev를,
    gateway는 scripts/sigdock-local-contract.sh와 scripts/verify-sigdock-gateway-tls.sh를
    통과해야 한다. Gradle 캐시·npm 설치·로컬 TLS 자재(GATEWAY_LOCAL_TLS_CA_FILE)가 여기서
    필요하다. api/gateway의 health check ready_timeout이 180초라 회차가 길다.
    
    plan `full`은 더 이상 돌지 않는다. `dev`의 compose 엔트리가 `full`과 같은 엔트리(같은
    compose 파일, 같은 프로젝트 `primeno1`)라 별도 회차가 새로 재는 것이 없다.
    `external-db`는 남긴다 — script 게이트 체인
    (external-db-contract -> compose-external-db -> api-external-db/stream-external-db)은
    `dev`가 지나지 않는 경로이고, 재조준 이전 회차와 비교할 기준선이기도 하다.
    
    native 엔트리 6종 중 5종을 덮는다: `dev`가 api/frontend/gateway를, `external-db`가
    api-external-db/stream-external-db를 돈다. `stream`은 어느 쪽도 돌지 않는다 — plan
    `dev-stream`(= `dev` + stream)에만 있다. 6종 전부가 필요하면 `up dev`를 `up dev-stream`
    으로 바꿔라. 게이트도 compose도 같고 Gradle bootRun 하나가 더 붙을 뿐이라 회차가
    늘지는 않는다. 여기서는 재조준 범위를 `dev`로 잡았다(TASK-379).
    
    `dev` 회차는 이 하네스가 스스로 정한 증거 기준 하나를 만족하지 못한다. 위 헤더는
    "검증 대상은 항상 이 저장소가 빌드한 바이너리"라고 선언하지만, sigdock 게이트 안의
    adjacent_dva()는 PATH의 `dva`를 부른다(이 워크스테이션에서는
    /Users/archmagece/go/bin/dva, version 0.2.0 commit b18f7831). plan `full`은 순수
    compose라 이 경로가 없었고 `dev`로 옮기며 새로 생겼다. 인접 SigDock 기동에만 쓰이므로
    리포트에서 "전부 이 저장소 바이너리로 쟀다"고 쓰지 마라.
    
    purge 미리보기가 `dev` 회차 전체를 덮지도 않는다. 아래 compose 프로젝트 목록은
    `dva down --purge`가 지우는 범위 그대로지만, `down dev --purge`는
    `sigdock-local-up.sh --down`도 부르고 그것은 `sigdock-idp` 프로젝트를 건드린다. 그쪽은
    자기 invocation이 만든 자원만 지우므로(down_owned + ownership marker) 데이터 손실
    위험은 아니다. 상태를 보려면 따로 조회하라 —
    `docker ps -a --filter label=com.docker.compose.project=sigdock-idp`. 위 fail-closed
    선행 조건을 확인하는 명령과 같은 것이다.
    
    compose 프로젝트 `primeno1`은 이 워크스테이션에서 실제로 쓰이는 개발 환경일 수 있다.
    purge 미리보기를 반드시 먼저 읽어라.
  단계:
    [읽기]   validate
             <REPO_ROOT>/bin/dva validate
    [읽기]   plan list — dev가 보여야 한다 (없으면 체크아웃이 낡았다)
             <REPO_ROOT>/bin/dva ls
    [기동]   up dev (sigdock 게이트 → compose → native api/frontend → gateway)
             <REPO_ROOT>/bin/dva up dev
    [읽기]   status (native 엔트리 포함)
             <REPO_ROOT>/bin/dva status
    [파괴적] down dev --purge
             <REPO_ROOT>/bin/dva down dev --purge --force
    [기동]   up external-db (script 게이트 → compose → native)
             <REPO_ROOT>/bin/dva up external-db
    [읽기]   status (gate chain)
             <REPO_ROOT>/bin/dva status
    [파괴적] down external-db --purge
             <REPO_ROOT>/bin/dva down external-db --purge --force

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
             <REPO_ROOT>/bin/dva validate
    [읽기]   plan list
             <REPO_ROOT>/bin/dva ls
    [기동]   up hybrid (composition)
             <REPO_ROOT>/bin/dva up hybrid
    [읽기]   status
             <REPO_ROOT>/bin/dva status
    [파괴적] down hybrid --purge (scoped to infra)
             <REPO_ROOT>/bin/dva down hybrid --purge --project infra --force

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
             <REPO_ROOT>/bin/dva validate
    [읽기]   plan list
             <REPO_ROOT>/bin/dva ls
    [기동]   up local-dev (composition)
             <REPO_ROOT>/bin/dva up local-dev
    [읽기]   status
             <REPO_ROOT>/bin/dva status
    [파괴적] down local-dev --purge (scoped to local-infra)
             <REPO_ROOT>/bin/dva down local-dev --purge --project local-infra --force

## task348
  작업 디렉토리 : <REPO_ROOT>/tools/dogfoodrun/fixtures/task348-profile-build
  설정 파일     : <REPO_ROOT>/tools/dogfoodrun/fixtures/task348-profile-build/dva.yml
  리포트        : tasks/todo/348-confirm-plan-profiles-reach-a-real-docker-build-not-just-argv.md
  compose 프로젝트: dva-dogfood-task348
  선행 확인:
    대조군은 이 저장소가 스스로 만든다: git archive로 대조 커밋 트리를 tmp/에 풀고
    그 안에서 make build를 돌린다. 저장소 체크아웃·브랜치·worktree는 건드리지 않는다.
    대조군 바이너리는 config version 0.1.48을 보고하고, 픽스처는 0.1.44로 선언해
    두 바이너리 모두 로드할 수 있게 맞춰 두었다.
  단계:
    [읽기]   current validate
             <REPO_ROOT>/bin/dva validate
    [읽기]   control validate — schema는 profiles를 거부한다 (exit 1 기대)
             <REPO_ROOT>/tmp/dogfood-run/control-275c8c98/bin/dva validate
    [읽기]   argv: current, gated (--profile rust 있어야 한다)
             <REPO_ROOT>/bin/dva --dry-run build gated
    [읽기]   argv: control, gated (--profile 없어야 한다)
             <REPO_ROOT>/tmp/dogfood-run/control-275c8c98/bin/dva --dry-run build gated
    [읽기]   출발점: 이미지가 없어야 한다 (exit 1 기대)
             docker image inspect dva-dogfood-task348-gated:latest
    [기동]   대조군 빌드: control, gated
             <REPO_ROOT>/tmp/dogfood-run/control-275c8c98/bin/dva build gated
    [읽기]   대조군 이후에도 이미지 없음 (exit 1 기대)
             docker image inspect dva-dogfood-task348-gated:latest
    [기동]   프로필 없는 플랜: current, legacy
             <REPO_ROOT>/bin/dva build legacy
    [읽기]   legacy 이후에도 이미지 없음 (exit 1 기대)
             docker image inspect dva-dogfood-task348-gated:latest
    [기동]   실제 빌드: current, gated
             <REPO_ROOT>/bin/dva build gated
    [읽기]   이미지 생성 확인 (exit 0 기대)
             docker image inspect --format '{{.Id}} {{.Created}}' dva-dogfood-task348-gated:latest
    [읽기]   빌드 결과물 확인
             docker run --rm dva-dogfood-task348-gated:latest cat /task348-marker
    [파괴적] 픽스처 teardown (프로젝트·볼륨·local 이미지 제거)
             <REPO_ROOT>/bin/dva down gated --purge --force
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
================ 아래 블록을 <리포트 경로> 에 그대로 붙인다 ================

## 실기동 (<YYYY-MM-DD HH:MM:SS>, <dva version 첫 줄>)

- 대상: `<devbox>/dva.yml`
- 하네스: `tools/dogfoodrun/dogfood-run.sh --execute <TARGET>`
- compose 프로젝트: <프로젝트 목록>
- 전체 출력: `tmp/dogfood-run/<TARGET>-<ts>.log`

<경고가 있으면 여기 — 예: 기동 실패 후 파괴적 스텝에 도달했다>
| 명령 | exit | 마지막 출력 줄 |
|------|------|----------------|
| ... | ... | ... |

### 선행 확인

...

### purge 미리보기 (파괴적 단계 실행 전)

```text
...
```

================================ 블록 끝 ================================

````

실제로 채워진 블록은 `--execute`를 돌려야 나오므로 이 카드에는 없다. 실기동 회차에서
붙여넣기 그대로 되는지 확인해야 할 마지막 항목이다.

### 검증하지 못한 것

- `--execute` 경로 전체(`up`/`down --purge`/`build`)는 이 워크스테이션이 컨테이너 105개가
  도는 살아있는 개발 환경이라 실행하지 않았다. 이 카드의 안전 계약상 실행은 사람의
  명시적 opt-in에서만 일어난다.
- **primeno1의 native 엔트리는 착지했고, 하네스는 그에 맞춰 재조준됐다(TASK-379).**
  이 카드를 처음 쓸 때는 `primeno1-devbox` master(`b432a01`)에 native stack 엔트리가 0건이라
  "native 엔트리 검증은 이 하네스로 닫을 수 없다"고 적었다. 그 서술은 더 이상 맞지 않는다 —
  `afb13c7`("chore(tasks): record that primeno1 native entries actually landed")이 착지를
  기록했고, 하네스는 plan `full` 대신 `dev`를 돌도록 바뀌어 native 엔트리 6종 중 5종을
  덮는다(`dev`가 api/frontend/gateway, `external-db`가 api-external-db/stream-external-db).
  남은 하나는 `stream`이며 plan `dev-stream`에만 있다 — 필요하면 `up dev`를 `up dev-stream`
  으로 바꾼다. 대신 **새로운 선행 조건이 생겼다**: 체크아웃이 `origin/master`여야 하고
  (plan `dev`는 `0caeaf9`에서 들어왔다), 회차는 order 10의 fail-closed sigdock 게이트를
  먼저 통과해야 한다. 그 게이트가 요구하는 항목과 현재 이 워크스테이션이 위반 중인 둘은
  하네스의 `--plan primeno1` 선행 확인에 그대로 나열돼 있다.

- `familybook-devbox`의 설정 파일은 아직 `dva.yaml`이다 (TASK-329 개명 대기). 하네스는
  현재 이름을 그대로 읽는다.
