# mydevbox 2차 점검 — 규칙별 발견

[현황](README.md)의 발견 요약을 규칙별로 푼다. 원자료는 [data/audit-2026-10-10.jsonl](data/audit-2026-10-10.jsonl)이다.
규칙 번호 R은 [shared-guardrails.md](../../agent-mesh-flows/shared/library/shared-guardrails.md),
A~D는 [devbox-apply.md](../../agent-mesh-flows/shared/library/devbox-apply.md) 절이다.

각 항목의 "성격"은 고쳤을 때 무엇이 바뀌는지다.

- 선언: `dva up`/`dva run`의 동작이 바뀌지 않는다. 메타데이터, 억제, 저장소 위생.
- 동작: 명령의 결과나 기본값이 바뀐다.
- 구조: 새 파일이나 다른 저장소의 변경이 필요하다.

## F-1 스키마 헤더 없음 (R15) — 선언

1행에 `# yaml-language-server: $schema=...`가 없다. 편집기 검증이 꺼진 상태다.

airouter, careerarchive, familybook, funbricks-notifire, funbricks-postkit, reviewrary, server-farm, sigdock-pass, task-manager

## F-2 `.sb/dva/`가 `.gitignore`에 없음 (doctor) — 선언

DVA 런타임 상태 디렉터리가 추적될 수 있다. doctor가 fail로 낸다.

reviewrary, sadawiki, scripton-dns-bridge(보드 TASK-024), scripton-signalhub, sigdock-pass

## F-3 compose 최상위 `name:` 없음 (R18) — 선언

scripton-signalhub `compose.yaml`에 `name:`이 없다. `dva.yml`의 `project_name: signalhub`와 맞지 않아
validate `compose_name` 경고와 doctor `Compose project name alignment` fail이 함께 난다.
DVA는 `-p signalhub`를 넘기므로 `name: signalhub`를 넣어도 DVA 경로의 컨테이너 이름은 바뀌지 않는다.
바뀌는 것은 DVA 없이 `docker compose`를 직접 부를 때의 기본 프로젝트 이름뿐이다.

## F-4 stack에 없는 compose 파일 (validate `config_drift`) — 선언

stack에 넣거나 `drift_ignore`로 의도를 남겨야 한다. `drift_ignore`는 이 경고만 억제한다
(`internal/cli/validate_drift.go`). `dva up`은 바뀌지 않는다.

| 제품 | 미분류 파일 |
| --- | --- |
| familybook | `compose.apps.yaml`, `compose.devcontainer.yaml`, `compose.gateway.yaml`, `compose.local-dev.yaml`, `compose.sigdock-dev.yaml`, `compose.sigdock.yaml`, `compose.test.yaml` |
| flow-knowchain | `compose.debug.override.yaml`, `compose.production.override.yaml`, `compose.production.yaml`, `compose.test.yaml` |
| flow-pipechain | `deploy/local/compose.app.yaml`, `compose.https.yaml`, `compose.infra.yaml` |
| flow-taskchain | `deploy/local/compose.e2e.yaml` |
| gorisa | `compose/compose.rustfs.yaml`, `gorisa-rails/compose/compose.dev.yaml` |
| hek | `compose/compose.test.yml` |
| primeno1 | `env/docker-compose/docker-compose.task44.verify.yml`, `docker-compose.task45.verify.yml` |
| scripton-nd-stack | `deploy/local/compose.disposable-project.yaml`, `compose.persistent-artifacts.yaml`, `compose.schema-isolation.yaml` |
| sigdock-idp | `compose.e2e.delivery-tls.yaml`, `compose.e2e.yaml`, `compose.multi-region-postgres.yaml` |

파일마다 stack 편입과 ignore 중 하나를 고르려면 그 파일을 읽어야 한다. production·검증·e2e 파일은
로컬 개발 plan이 아니므로(R41) ignore 쪽이 기본이다.

## F-5 `default_plan`이 full-stack (R42) — 동작

R42는 `default_plan`을 로컬·비파괴 plan으로 두고 full-stack을 기본값으로 쓰지 말라고 한다.
바꾸면 인자 없는 `dva up`이 띄우는 범위가 줄어든다.

| 제품 | 현재 | 후보 |
| --- | --- | --- |
| funbricks-elemhant | `full-stack` (infra + Go gateway + Cloudflare worker) | `local-infra` (RustFS + sigdock-idp) |

primeno1 `default_plan: full`은 이름만 비슷하다. `full`은 "Full infra (PostgreSQL + Redis + Kafka +
Schema Registry)"로 앱 없이 로컬 인프라만 띄운다. R42가 막는 full-stack이 아니라 위반이 아니다.

careerarchive는 반대로 `default_plan`이 없어 validate가 `semantic` 경고를 낸다. plan이 `design`, `verify`,
`verify-workspace`뿐이라 안전한 기본값을 증명할 수 없다. R42는 이때 `default_plan`을 생략하라고 하므로
결함이 아니다.

## F-6 provision·interaction이 plan lifecycle을 실행 (R10, R36, R37) — 동작

provision은 준비 작업만 하고 기동은 plan이 맡는다(R36). 아래는 provision 단계가 `dva up/down/build`를 부른다.

| 제품 | 위치 | 호출 |
| --- | --- | --- |
| scripton-gitrump | provision `default`/`full`/`reset` | `dva up infra`, `dva up dev-full`, `dva down dev-full --volumes` |
| scripton-db-orchestrator | provision `full` | `dva build full-stack`, `dva up full-stack` |

primeno1의 `dva down full`은 provision이 아니라 interaction `clean`의 step이다. R37(interaction은 lifecycle
래퍼가 아니다) 대상이고, 지우면 `dva run clean` 표면이 사라지므로 제안만 한다.

고치면 provision이 더는 서비스를 띄우거나 내리지 않는다. 사용자는 provision 뒤에 `dva up <plan>`을 따로 부른다.
`reset`처럼 볼륨까지 지우는 프로필은 파괴적이라 문서와 함께 바꿔야 한다.

## F-7 provision이 `dva run`/`dva provision`을 호출 (R10) — 동작

| 제품 | 호출 |
| --- | --- |
| dripter | 루트 provision `default`/`full`이 `dva provision backend/default`, `frontend/default` 등 자식 프로필을 부른다 |
| matdosa | provision `default`가 `dva run install-api`, `install-web` |
| primeno1 | provision이 `dva run env-check` |

R10의 근거는 순환이다. 이 호출들은 순환이 아니고, DVA에는 자식 provision 프로필이나 interaction을
provision 단계에서 재사용하는 수단이 없다(`internal/config/provision.go`의 `ProvisionItem`에 참조 필드 없음).
명령을 인라인으로 옮기면 같은 명령이 두 곳에 생긴다. 그래서 제품 결함이 아니라 D-3으로 본다.

## F-8 (없음)

수집기의 정적 검사는 flow-pipechain `gen-secrets`를 echo 래퍼(R24)로 잡았다. 이 명령은 `openssl rand`로
값을 만들어 출력하는 실제 생성기라 R24 대상이 아니다. 오탐으로 뺀다.

## F-9 `command: make …` 래핑 (A.2) — 구조

A.2는 Make를 감싸는 interaction을 이관으로 보지 않는다. 구현을 `dva.yml`로 옮기고 Make 쪽을 한 줄 alias로
남기는 것(`legacy_surface=alias`)이 정본 방향이다. 기본 적용 모드는 `propose`다.

| 제품 | 래핑 interaction |
| --- | --- |
| careerarchive | `check`, `api-check`, `pdf-determinism-check`, `contract-drift-check`, `contract-ledger-sync` |
| cwrapper | `health`, `env`, `sigdock`, `ws`, `k8s` |
| dripter | `check` |
| flow-agent-mesh | `dev-build`, `install`, `test`, `test-cover`, `lint`, `fmt`, `vet`, `check`, `mod-tidy` |
| flow-knowchain | `check`과 `db`/`e2e` 하위 명령 19개 |
| flow-observechain | `make-validate`, `test`, `codegen`, `e2e-ci`, `e2e-headed` |
| flow-pipechain | `check` |
| flow-task-automator | `verify`, `test`, `fmt`, `vet` |
| flow-taskchain | `db`, `check-docs`, `ui-module`, `sigdock-idp`, `prepare`, `deploy-validate`, `services` |
| funbricks-postkit | `test`, `repo-status`, `check`, `check-archive-drift` |
| gizza-plane | `check`, `validate-all`, `test`, `sync-web` |
| gizzahub | `fmt`, `lint`, `vet`, `quality-check`, `task-list`, `task-next`, `task-validate`, `sigdock-provision-client-key`, `sigdock-age-assurance-preflight` |
| scripton-db-orchestrator | `quickstart`, `docker-build`, `setup`, `test`, `lint`, `fmt`, `check`, `sdk`, `verify`, `env`, `ci-check` |
| scripton-dns-bridge | `docker-push`, `verify-completion` |
| scripton-signalhub | `setup`, `prepare` |
| server-farm | `test`, `test-integration`, `lint`, `vet` |
| sigdock-idp | `build-docker`, `build-native`, `clean-build`, `test`, `helm-guardrails`, `helm-kind-smoke`, `lint`, `fmt`, `vet` |

`check` 계열은 대부분 gz-git integration gate가 `make check`로 직접 부르는 타깃이다. 구현을 옮기면
gate가 부르는 쪽이 alias가 되므로 gate 동작까지 함께 확인해야 한다.

## F-10 `.gz-git.yaml` 자식 미연결 (B) — 구조

`access: read-only`와 `legacy/` 경로는 B.6에 따라 뺐다. 남은 자식은 모두 로컬에 있고 Makefile이나
언어 manifest가 있다.

| 제품 | 자식 `dva.yml` 없음 | 루트 선언만 있음(B.2 위반) |
| --- | --- | --- |
| airouter | `airouter-macos-app` | — |
| careerarchive | `careerarchive-server-go`, `careerarchive-web-react`, `careerarchive-e2e-playwright`, `careerarchive-design` | — |
| flow-agent-mesh | `agent-mesh`, `agent-mesh-flows`, `agent-mesh-runner` | — |
| flow-knowchain | `flow-knowchain-client` | `ai`, `backend`, `frontend`, `admin` (자식 `dva.yml` 없음) |
| flow-task-automator | `flow-task-automator` (보드 TASK-001) | — |
| flow-taskchain | `flow-taskchain-cli` (보드 TASK-169) | — |
| gzh-cli | 자식 12곳, 루트 `dva.yml`도 없음 (보드 TASK-288) | — |
| hek | `hek-engine-fiber`, `hek-web-next` (보드 TASK-003) | — |
| scripton-db-orchestrator | — | — (정정: `db-orchestrator-rs`는 오탐, 아래 참고) |
| scripton-gitrump | `gitrump-cli` | `gitrump-ce` |
| sigdock-idp | `sigdock-idp-sdk-py` | — |
| sigdock-pass | `sigdock-pass-server`, `-ts`, `-vscode`, `-client`, `-extension` | — |
| task-manager | `taskchain-task-manager` | — |

B.2는 자식 `dva.yml`이 없는 subproject 선언을 금한다. airouter 루트 주석처럼 "import 없는 선언은
초기화되지 않은 자식을 가리켜도 된다"는 관례가 있지만, 그것은 자식이 아직 클론되지 않은 워크트리의
경우이고 위 표의 자식은 체크아웃이 있는데 `dva.yml`이 없다.

정정(3단계): `db-orchestrator-rs`는 B.2 위반이 아니다. 원격 master에는 이미 유효한 `dva.yml`(114줄)이 있다.
로컬 `~/mydevbox/scripton-db-orchestrator-devbox/db-orchestrator-rs/`는 `.git`이 없는 잔여 디렉터리
(`.ce`, `crates`, `target` 등 5.4G)라 측정이 그 파일을 못 봤다. 남은 B.2 위반은 다섯 곳이다.

## F-11 같은 compose project name을 부모와 자식이 다르게 소유 (R28) — 동작

scripton-nd-stack 루트(`deploy/local/compose.yaml`)와 자식 `ndea-rs`(`compose/compose.infra.yml` +
`compose.ndea.yml`)가 모두 `project_name: nd-stack-dev`다. 둘 다 `postgres`, `redis`, `rustfs`를 정의한다.
한쪽에서 `dva down`하면 다른 쪽 컨테이너가 orphan으로 잡히고, 같은 서비스 이름이 서로 다른 정의로
덮어쓰인다. doctor가 fail로 낸다. 자식 project name을 바꾸면 기존 로컬 볼륨 이름도 바뀐다.

## F-12 구 스키마 (R1, R2) — 구조

1차 남은 것 1번 그대로다. 각 보드 카드가 추적한다.

- hek: `modes`, `default_mode`, `stack.compose.order`, plan 없음 (보드 TASK-001)
- reviewrary: `stack.*.order`, plan 없음 (보드 TASK-001)

## F-13 환경 전용 doctor 실패 — 설정 아님

다음은 로컬 파일·볼륨·네트워크가 없어 실패한다. 설정으로 숨기지 않는다.

| 제품 | 실패 |
| --- | --- |
| careerarchive | `Verification-environment listeners stay on loopback` |
| familybook | `KMP checkout` (`familybook-client-kmp` 미클론) |
| flow-agent-mesh, sadawiki, scripton-gitrump | `.env file exists` |
| scripton-nd-stack | `Local environment file exists` |
| flow-knowchain, funbricks-notifire, hek, primeno1, reviewrary, sigdock-pass | `Compose config resolves` (변수 미설정) |
| funbricks-elemhant | `Compose config resolves`, `elemhant-net` 네트워크, `elemhant-rustfs-data-v2` 볼륨 |
| hek | `Compose env file exists` (`compose/.env`) |
| primeno1 | `POSTGRES_PASSWORD`, `REDIS_PASSWORD` 비어 있음 |

`Encrypted env source declared`(notifire, hek, reviewrary, nd-stack, signalhub)는 1차가 각 보드에 카드로 넘겼다.

## DVA 쪽 발견

### D-1 `dva status`의 종료 코드가 plan 모양에 따라 다르다

기본 plan이 단일 stack 엔트리면 서비스가 하나도 없어도 `dva status`는 exit 0이다(예: gorisa `local-infra`,
`not found`). 기본 plan이 subproject plan을 묶은 composition이면 "composition "local-dev" is not fully up"으로
exit 1이다(flow-taskchain, funbricks-postkit, `internal/lifecycle/composition_orchestrator.go`).
조회 명령의 종료 코드가 plan 모양에 따라 달라 스크립트가 `dva status`를 상태 조회로 쓸 수 없다.

### D-2 `dva --json ls`의 출력 모양이 plan 유무로 바뀐다

plan이 있으면 `{"interaction_commands": {...}, "plans": {...}}`, 없으면 interaction 맵 자체를 낸다
(`internal/cli/list.go` `printJSON`/`printYAML`). `--json`은 "LLM-optimized"로 안내되는데 소비자가 두 모양을
모두 처리해야 한다. 이번 수집기도 이 차이로 plan 없는 제품의 interaction을 0으로 셌다.

### D-3 provision 단계에서 자식 프로필·interaction을 재사용할 수단이 없다

F-7의 원인이다. R10은 `run: dva …`를 일괄 금지하지만 `dva provision <child>/<profile>`과
`dva run <interaction>`은 순환이 아니다. 규칙을 "lifecycle 동사 금지"로 좁히거나, provision 항목이
interaction이나 자식 프로필을 참조하는 필드를 갖는 것 중 하나가 필요하다.
