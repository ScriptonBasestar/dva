# primeno1-devbox 마이그레이션 이력 (2026-09-05)

[primeno1.md](primeno1.md)에서 분리했다. 그 문서가 `tools/doccheck`의 10 KiB 상한에
닿았고(2026-09-13 기준 잔여 여유 212바이트), 상한에는 per-file 예외가 없다 —
`tools/doccheck/policy.go`가 "a document that cannot meet the limits is split, not
exempted"라고 못박는다. 그래서 회차별 적용 이력을 이쪽으로 옮기고 본 문서에는 현황과
현재 상태만 남겼다.

여기 있는 기록은 전부 2026-09-05 회차의 것이며 편집 없이 그대로 옮겼다.

## 적용 결과 (2026-09-05)

### 변경 항목 체크리스트
- [x] `dva config migrate` 미리보기: "nothing to convert". modes 6개 각각 "description → plans.<m>.description, stack → plans.<m>.entries[].name" 대응표만 출력(1:1 매핑인데 자동 변환 없음). `stack.*.order`(-10 두 곳), clean replace, 최상위 `environment`, `dva up -M` 언급 없음.
- [x] `interaction.clean` replace(validate 하드 에러) → `steps:` (`docker compose -f … down -v`, DESTRUCTIVE 설명).
- [x] modes 6개 + `default_mode` → plans: `minimal`, `full`(default_plan), `observability`, `tracing`, `sigdock-local`(sigdock-local-runtime order 10 → compose order 20 depends_on), `external-db`(external-db-contract order 10 → compose-external-db order 20 depends_on). `stack.*.order: -10` 제거 후 plan entry order로 이관(게이트 선행 유지).
- [x] **entry 조합 시도 → 되돌림**: overlay 엔트리를 base와 plan에서 조합하면 TASK-288 경고(validate_warnings.go:893)로 거부되어 원래 형태(엔트리당 전체 -f 목록)로 복귀. 엔트리 이름을 `observability`/`tracing`으로 정리, 미사용 앵커 제거.
- [x] `dev-up`: `dva up -M full` → `dva up full`. sigdock-browser-e2e 주석의 `-M sigdock-local`도 갱신.
- [x] 최상위 `environment:` → `vars:` (loadEnv 순서 vars < environment < env_file이라 interaction에 보이는 값 동일). `environments.dev.environment: {PRIMENO1_ENV: development}`로 환경 축 부여.
- [x] `env_file` map 형식 → 리스트 형식. `logs` replace 제거(`dva logs full`). provision `docker compose up` → `compose_up: [postgres, redis, zookeeper, broker, schema-registry]` (PrimaryComposeEntry = order 없을 때 이름순 최소 = `compose`이므로 base 파일 사용).
- [x] version 0.1.44. Makefile 제안 `validate*`/`version` suggestion_ignore 추가.

### validate 최종 출력
```
[warn] semantic: env/docker-compose/docker-compose.yml: missing top-level 'name: primeno1'
✅ dva.yml is valid
EXIT=0   (warning 1 — 의도적 예외)
```

### 보류/예외 항목
- compose 파일 `name:` 누락 경고 — 다른 파일 수정 금지 → 예외.
- `scripts/dev-run.sh`, `scripts/seed-data.sh`, `scripts/sigdock-local-up.sh`가 여전히 `dva up -M <mode>`를 호출(다른 파일 → 미수정, **plans 전환 후 깨짐** — 사용자 후속 필요).
- `${POSTGRES_DB:-primeno1_dev}` ×2, `${TOPIC:-test}` — TASK-303, 미수정.
- `docker-compose.task44.verify.yml` / `task45.verify.yml` 미등록(검증 픽스처). env/docker-compose/ 하위라 drift 감지 대상이 아니어서 경고 없음.

### 발견된 dva 개선점
- **overlay 조합 불가 (설계)**: base+overlay 엔트리 조합은 TASK-288 경고라 overlay마다 base 서비스/태그를 재선언해야 한다(구조적 중복). 개선 후보는 TASK-307의 엔트리 `extends:`/PlanEntry `overlays:`.
- **PrimaryComposeEntry 암묵 선택**: order 제거 후 primary compose 엔트리는 이름순(internal/config/lifecycle_helpers.go:164). compose 엔트리가 여러 개인 config에서 provision compose_up / `dva db`(service 지정 interaction)가 어느 파일 세트로 실행되는지 문서화·명시 수단(`primary: true` 등) 필요.
- **migrate가 modes.*.stack → plans 자동 변환 안 함**(1:1 매핑), `dva up -M` 잔재(interaction command·스크립트) 탐지 없음. TASK-306.
- **drift 감지가 서브디렉터리 compose 파일을 보지 않음**(env/docker-compose/*.verify.yml) — sigdock-pass의 `compose-*.yaml` 미감지와 같은 계열.

## CLI 잔재 정리 (2026-09-05)
실행 파일:
- scripts/dev-run.sh:37, :41, :42 `dva up -M $MODE` → `dva up $MODE` (MODE 값 full/minimal/external-db = plan 이름 그대로)
- scripts/seed-data.sh:96 `dva up -M minimal/full` → `dva up minimal/full`
- scripts/sigdock-local-contract.sh:65 `dva down -M sigdock-local` → `dva down sigdock-local`
- scripts/sigdock-local-up.sh:430 `adjacent_dva up -M infra` → `adjacent_dva up infra` (sigdock-idp-devbox의 plan `infra`)
문서:
- CLAUDE.md:34-37 (AGENTS.md 심링크) `dva up -M X` → `dva up X`; :39 `dva stack down -v` → `dva down full --volumes`; :41 `dva logs` → `dva logs full`; :50 `dva clean` → `dva run clean`; :172 `dva dev` → `dva frontend-dev`; "(via dva mode)" → plan
- README.md:34-37, :70 `dva up -M X` → `dva up X`
- docs/LOCAL_EXECUTION_GUIDE.md:101-113, 172, 191, 223, 245, 273, 336-347, 382-385, 417 `-M` 제거; :118/389/458 `dva logs` → `dva logs full`; :121-123 `dva stack down -v` → `dva down full --volumes`, 흐름 문구 갱신; :122/414 `dva clean` → `dva run clean`
- docs/CICD_GUIDE.md:73, docs/MULTI_REPO_STRUCTURE.md:134 `-M` 제거
- 보류 0. 산문 속 "mode"(SigDock IDP local mode, external DB mode 절 제목)는 CLI 표면이 아니라 유지.

## TASK-303 반영 후 재검증 (2026-09-05, dva d7636a3)

- `${POSTGRES_DB:-primeno1_dev}` ×2, `${TOPIC:-test}`: 유지 확정.

## 2차 재검증 (2026-09-05, dva ecae43d: TASK-305/306/308 반영 빌드)

- (TASK-308 semantic warning으로 발견) `interaction.dev-up.command`의 `dva up -M full` 잔재 → `dva up full`. 이전 CLI 잔재 정리에서 dva.yml 내부 문자열은 놓쳤음.
- (결정 반영) bare `dva down` → `dva down full` (CLAUDE.md, docs/LOCAL_EXECUTION_GUIDE.md ×3), sigdock-idp 호출은 `dva down infra`.

## docs/57 §4 재점검 (2026-09-05, TASK-310 가이드 기준)

- §4-1 해당(미적용, 소유자 결정): 앱 프로세스 4종이 stack 엔트리 없이 interaction으로만 기동된다 — `api-run`, `api-run.gateway`,
  `api-run.external-db`, `api-run.stream.external-db`, `frontend-dev`. `dva status`가 앱을 못 보고 `dva down`이 앱을 남긴다.
- 미적용 이유: 각 기동이 SigDock contract gate(`scripts/sigdock-local-contract.sh`), TLS 검증 wrapper(`--exec`), external-db credential wrapper와
  결합돼 있고 health 경로가 확인되지 않아 실기동(TASK-311/312 이후) 없이 옮기면 검증 불가.
- 권장안: native 엔트리 4종 + plan `dev`, 문서 참조 8곳 `dva up dev`로 치환 (아래 적용).

### 권장안 적용 (2026-09-05, 소유자 수용)

- native 엔트리 6종 추가: `api`/`gateway`/`stream`/`frontend`(plan `dev`, `dev-stream`) + `api-external-db`/`stream-external-db`(plan `external-db`).
  `run:`은 gate 스크립트 체인을 그대로 두고 마지막을 `exec`로 넘겨 Gradle/Vite 프로세스가 dva의 추적 대상이 되게 했다(§3 devbox 소유 gate 스크립트 허용).
- interaction `api-run*`/`frontend-dev`/`dev-up` 삭제, 문서·스크립트 11곳을 `dva up dev`/`dev-stream`/`external-db`로 치환. `dva validate` warning 0.
- 검증 한계(2026-09-05 시점): `dva --dry-run up dev`가 health 대기에 걸려 멈췄다(kill 필요). 실기동 검증은 TASK-311/312 이후로 미뤘다.
