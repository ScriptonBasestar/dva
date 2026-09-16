# primeno1-devbox dva 적용 분석

## 적용 전 상태 요약 (2026-09-05 분석, 이후 전부 해소)

`version: "0.1.0"`의 구세대 config로 8개 중 유일하게 `dva validate`가 실패했다
(`interaction.clean.replace` — 제거된 built-in에 붙은 replace 훅). `modes:` 6개 +
`default_mode`가 plans의 책임을 대신했고, `stack.*.order`·최상위 `environment:`·
mode당 `-f` 목록 복제가 남아 있었다. `dva config migrate`는 "nothing to convert" +
modes 전부 `Left for you` 보고뿐이라 수동 분해 비용이 컸다(개선 힌트는 dva 쪽 기록으로
전달됨). 마이그레이션 난이도 **상** 판정. 변경 체크리스트·validate 출력·보류 항목 원문은
[primeno1-migration-log.md](primeno1-migration-log.md)에 있다.

## 적용 이력 (2026-09-05)

회차별 적용 기록 — 변경 체크리스트, `validate` 출력, 보류/예외 항목, CLI 잔재 정리,
TASK-303/305/306/308 재검증, docs/57 §4 재점검과 native 엔트리 6종 도입 — 은
[primeno1-migration-log.md](primeno1-migration-log.md)에 있다.

## 2026-09-05의 검증 한계는 해소됐다 (2026-09-13)

당시의 두 blocker는 둘 다 없다: TASK-312(드라이런이 health 대기에 걸림)는
`Orchestrator.Up`의 entry-level wait가 `opts.DryRun`을 보지 않던 것을 고쳐 닫혔고,
TASK-311은 archive에 있다. 하네스 재조준(TASK-379)으로 primeno1 스텝은 plan `dev` 먼저,
`external-db`를 이어 돈다 — `full`은 `dev`와 같은 compose 엔트리를 쓰므로 별도 회차가
재는 것이 없고, native 6종 중 `stream`만 `dev-stream`에 있다.

실기동 전제는 TASK-328 카드와 하네스 `target_notes()`가 소유한다(체크아웃이
`origin/master`일 것, sigdock 게이트의 선행 조건과 위반 정리 절차). 게이트 다음 표면은
Gradle bootRun · `npm run dev` · 로컬 TLS 검증이고 `api`/`gateway`의 `ready_timeout`은
180초라 회차가 길다.
## 실기동 (2026-09-16 14:59:13, dva version 0.2.0)

- 대상: `/Users/archmagece/mydevbox/primeno1-devbox/dva.yml`
- 하네스: `tools/dogfoodrun/dogfood-run.sh --execute primeno1`
- compose 프로젝트: primeno1 primeno1-external-db
- 전체 출력: `tmp/dogfood-run/primeno1-20260916-145913.log`

- 기동 스텝이 실패한 뒤 파괴적 스텝 `down dev --purge`에 도달했다.
- 기동 스텝이 실패한 뒤 파괴적 스텝 `down external-db --purge`에 도달했다.
| 명령 | exit | 마지막 출력 줄 |
|------|------|----------------|
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva validate` | 0 | ✅ dva.yml is valid (6 suggestions ignored by dva.yml) |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva ls` | 0 |   tracing        # Full infra + observability + Jaeger (OTLP) |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva up dev` | 1 | ERROR: entry "sigdock-local-runtime" up failed: script up: exit status 1 |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva status` | 0 |   [compose] compose |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva down dev --purge --force` | 0 |   $ ./scripts/sigdock-local-up.sh --down |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva up external-db` | 1 | ERROR: entry "external-db-contract" up failed: script up: exit status 1 |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva status` | 0 |   [compose] compose |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva down external-db --purge --force` | 0 |   $ true |

### 선행 확인

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

### purge 미리보기 (파괴적 단계 실행 전)

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
```

