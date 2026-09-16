# familybook dva 적용 분석

## 현황
- 파일: `dva.yaml` (288줄, canonical 이름 `dva.yml` 아님), `version: "0.1.26"`
- 섹션: `env_file(files+priority+interpolate)`, `stack`(legacy flat 형식), `modes`(3개), `environments`(dev/test — legacy 필드 포함), `health_checks`(최상위, start 포함), `interaction`, `provision(default_profile)`, `checks`
- `dva validate`: **ERROR — 파싱 실패**. 그룹 A 7개 중 유일하게 현행 구조로 로드조차 안 되는 설정.

## 문제점
- **stack flat 형식 (26–30행)**: `stack.compose`에 `order`/`files`/`project_name`을 직접 선언. validate가 "compose must be declared under runners.compose"로 즉시 거부.
- **`modes:` 섹션 (36–59행)**: 제거된 개념 (docs/42 §11-1 — plans/environments/sites로 분해 대상). `compose_files`/`compose_profiles`까지 mode에 선언.
- **`plans:` 없음**: 실행 표면이 전부 구세대. 문서 주석의 사용법도 구 CLI 기준.
- **`environments.*` legacy 필드**: `test.compose_files` (74–76행) — environment는 vars 중심으로 축소된 개념 (42 §11-1 `environments.*.stack_overrides` 계열 제거).
- **`interaction.clean.replace` (124–129행)**: `clean` built-in이 제거되어 이 hook은 어디에도 걸리지 않음 (다른 프로젝트에서 같은 패턴이 validate ERROR로 잡힘).
- **최상위 `health_checks`의 `start`/`start_hint` (85–108행)**: modes 제거 후 참조 주체가 없어 dead 선언.
- `version: "0.1.26"` — 구버전 명시. 파일명 `dva.yaml`도 rename 경고 대상.

## dva 개선 힌트
- **`dva config migrate`가 `dva.yaml`을 인식하지 못함**: `ERROR: no dva.yml in .` — validate는 dva.yaml을 읽고 rename 경고까지 내는데 migrate는 못 읽음. 가장 마이그레이션이 필요한 설정이 변환기를 못 쓰는 모순. migrate도 dva.yaml fallback(+rename 안내)을 지원해야 함.
- validate가 flat stack 에러에서 rewrite 예시를 보여주는 것은 좋으나, **첫 에러에서 멈춰** modes/clean 등 나머지 legacy 항목을 한 번에 보여주지 못함 — legacy 설정에는 에러 수집 모드가 유용할 것.

## 마이그레이션 난이도
**상** — stack 재작성 + modes 3개의 plans/entries 수동 분해 + environments legacy 필드 정리 + health_checks 재배치 + clean hook 이동 + 파일명 rename까지 전면 개편. migrate 도구도 현재 파일명 문제로 도움을 못 줌.

## 적용 결과 (2026-09-05)

### 변경 항목 요약
dva.yaml rename 전 migrate 실패(TASK-304 증거)를 제외한 전면 수동 전환: flat stack →
`runners.compose`, modes 3개 → plans(`infra`/`full-stack`/`monitoring` + `hybrid`/`test`,
`default_plan: infra`), `environments.test.compose_files` → `stack.test-stack`+`plans.test`,
상위 `health_checks` → 엔트리별 이동(start/start_hint 삭제), `interaction.clean` replace →
`steps:`, env_file map 형식 → 리스트, `checks` tcp → command, provision → `compose_up`.
version 0.1.44, Makefile 제안 194개 → suggestion_ignore 글로브.

### validate 최종 출력
```
[warn] config drift: compose.files is compose.yaml, compose.apps.yaml, compose.monitoring.yaml, compose.test.yaml but detected root compose files are … compose.devcontainer.yaml, compose.gateway.yaml, compose.local-dev.yaml, … compose.sigdock-dev.yaml, compose.sigdock.yaml …
✅ dva.yml is valid
EXIT=0   (warning 1 — 의도적 예외)
```

### 보류/예외 항목
- drift 경고: compose.devcontainer / gateway / local-dev / sigdock-dev / sigdock 5개는 미등록(devcontainer·gateway·sigdock 연동은 Makefile compose-up-* 타깃 영역). 무시 수단 없음(TASK-309).
- `${REDIS_PASSWORD:-changeme}` (interaction.redis-console) — TASK-303, 미수정.
- monitoring plan은 profile 서비스를 이름으로 명시(elk/tracing profile은 제외 — 원본 mode도 monitoring profile만).
- 등록한 native 엔트리(backend `make run`, frontend `make run-web`)와 plan은 실행 미검증.

### 발견된 dva 개선점
- **migrate가 dva.yaml을 못 읽음**: `ERROR: no dva.yml in .` — validate는 fallback+rename 경고를 내므로 migrate도 동일 fallback 필요. TASK-304.
- **validate 조기 중단이 스키마 에러까지 가림**: flat stack 에러를 고치자 `checks[].type: tcp` 스키마 에러가 새로 등장, 그걸 고치자 경고 195개 등장. 3단계 반복 필요. TASK-305.
- **migrate `--profile <모드명>` 오기** 재현(full-stack, monitoring).
- **migrate가 `environments.*.compose_files`, `env_file` map 형식의 잉여 키(priority/interpolate), 상위 health_checks start/start_hint를 전혀 언급하지 않음**.

## TASK-303 반영 후 재검증 (2026-09-05, dva d7636a3)

- `${REDIS_PASSWORD:-changeme}`: 그대로 유지, 정상.

## docs/57 §4 재점검 (2026-09-05, TASK-310 가이드 기준)

- §4-3 해당(미적용, 소유자 결정): `backend`(dir familybook-engine-fiber, `make run`)·`frontend`(dir familybook-client-flutter, `make run-web`)가 자식 Makefile 타겟 이름을 루트가 기억한다.
  자식이 타겟을 바꾸면 루트는 validate를 통과한 채 실행에서 깨진다. 해법은 자식 저장소에 `dva.yml`을 두고 §2 `subprojects` import로 전환하는 것인데
  devbox 밖(자식 저장소) 변경이라 이 dogfood 범위에서는 기록만 한다. 루트 Makefile 타겟을 가리키는 경우(flow-knowchain, gorisa, postkit)는 루트가 소유자이므로 해당 없음.

### 권장안 적용 (2026-09-05, 소유자 수용)

- 자식 저장소에 `dva.yml` 추가: familybook-engine-fiber(`backend`, `make run`, http health), familybook-client-flutter(`frontend`, `make run-web`).
  각각 `plans.dev` + `default_plan: dev`.
- 루트: `backend`/`frontend` 엔트리 삭제, `subprojects.backend`/`frontend`(`exclude_tags: [infra]`, `import.plans: [dev]`) 추가.
  `hybrid`는 composition plan으로 전환 — `infra`(0) → `backend/dev`(1) → `frontend/dev`(2). 루트 `dva.yaml`→`dva.yml` 개명 포함.
- 통합 결과: 루트 7f1e6e2, engine-fiber 3d30df87, client-flutter b792a5b9 모두 develop에 통합. `dva.yaml`→`dva.yml` 개명은
  readiness contract(`.gz-git/readiness/check` 필수 파일 목록)가 `dva.yaml`을 요구해 되돌렸고, 계약 수정 브랜치
  `dev/claude/mst/chore/readiness-dva-yml`(3538cf2)은 사람 통합 대기. client-flutter는 devbox `.gz-git.yaml`에 workspace 항목과
  빈 `integration: {}`를 추가해야 `--controller-config`로 통합 가능했다(prepareProfile 미지정 시 준비 단계 생략).
- 발견: composition plan에 `environment:`/`site:`를 두면 validate ERROR라 삭제했다(§2 문서에 명시 필요, TASK-323).
- `dva --dry-run up hybrid`는 wave 순서를 올바르게 출력하고 블록되지 않음(TASK-312는 entries가 있는 plan에서만 재현).

## 실기동 (2026-09-16 14:52:40, dva version 0.2.0)

- 대상: `/Users/archmagece/mydevbox/familybook-devbox/dva.yaml`
- 하네스: `tools/dogfoodrun/dogfood-run.sh --execute familybook`
- compose 프로젝트: familybook-devbox
- 전체 출력: `tmp/dogfood-run/familybook-20260916-145240.log`


| 명령 | exit | 마지막 출력 줄 |
|------|------|----------------|
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva validate` | 0 | ✅ dva.yml is valid |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva ls` | 0 |   monitoring  # monitoring |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva up dev` | 0 | outcome: up |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva status` | 0 |   redis      running   healthy |
| `/Users/archmagece/worktrees/misc/dva/claude__mbp__test__task-328/bin/dva down dev --purge --project infra --force` | 0 | outcome: down |

### 선행 확인

설정 파일 이름이 아직 `dva.yaml`이다 (TASK-329로 개명 대기). composition plan은
`--purge`에 `--project <child>`가 필수라 teardown이 infra 하위로 스코프된다
(internal/cli/composition_flags.go). backend/dev·frontend/dev는 자식 저장소의
native plan이라 purge 대상이 아니다.

### purge 미리보기 (파괴적 단계 실행 전)

```text
## purge 미리보기 — familybook
`dva down ... --purge`는 아래 프로젝트를 `docker compose down --remove-orphans --volumes --rmi local`로 지운다.

### compose project: familybook-devbox
  containers:
    familybook-postgres  [running]  postgres:16-alpine
    familybook-redis  [running]  redis:7-alpine
  volumes:
    familybook-postgres-data
    familybook-redis-data
  networks:
    familybook-dev-network
  images: (없음)
  networks (이름만 비슷함 — purge 대상 아님): (없음)
  volumes (이름만 비슷함 — purge 대상 아님): (없음)
```

