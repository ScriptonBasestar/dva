---
id: TASK-319
title: "native entries: optional dir, post-build steps, explicit primary compose entry"
type: feature
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{dripter,scripton-db-orchestrator,primeno1,funbricks-postkit}.md"
status: todo
needs-human: true
---

# Task 319: native 엔트리 표현력

## Summary

이 카드는 dripter, scripton-db-orchestrator, primeno1, funbricks-postkit 네 프로젝트를
dva로 마이그레이션하는 과정에서 native 엔트리 선언의 표현력이 부족해 겪은 문제들을 모은
것이다. optional dir 체크, post-build step, 다중 compose 엔트리 중 primary 지정, native
`env:` 필드 문서화 부재가 각각 다른 프로젝트에서 마이그레이션을 가로막거나 우회를
강요했다. 아래 항목은 각각 설계 결정이 필요한 별개 이슈이며 아직 채택 여부가 확정되지
않았다.

1. **optional/dir-exists 조건 없음** — 미체크아웃 subproject dir이면 plan 전체 실패 (dripter). `optional: true` 제안.
2. **post-build step 없음** — `make build-api`는 build/api 복사까지 하지만 `dva build`는 cargo만 실행 (db-orchestrator).
3. **PrimaryComposeEntry 암묵 선택** — order 제거 후 이름순 최소 엔트리(lifecycle_helpers.go:164)가 provision compose_up과
   service 지정 interaction의 compose 파일 세트를 결정. 다중 compose 엔트리에 `primary: true` 같은 명시 수단 필요 (primeno1).
4. native runner `env:` 필드는 존재하나 문서화 부족으로 Makefile 경유가 선택됨 (postkit) — TASK-323 문서 범위와 연계.

## Completion Criteria

- [x] 각 항목 설계 결정 기록 | verify: human — 카드 하단에 "Design record" 절이 추가되어 1)optional/dir-exists, 2)post-build step, 3)PrimaryComposeEntry 명시 수단, 4)native env: 문서화 4개 항목 각각에 대해 채택/보류 결정과 근거가 기록되었는지 확인
- [x] 구현 + 테스트 | verify: go test ./internal/lifecycle ./internal/config ./internal/cli -run "TestResolvePlanSkipsOptionalEntryWithMissingDir|TestResolvePlanKeepsOptionalEntryWhenDirExists|TestResolvePlanKeepsNonOptionalEntryWithMissingDir|TestResolvePlanSkipsOptionalProcessEntryWithMissingDir|TestResolvePlanKeepsOptionalEntryWithoutDir|TestPrimaryComposeEntryExplicitBeatsOrder|TestPrimaryComposeEntryFallsBackToOrder|TestPrimaryComposeEntryMultiplePrimariesIsDeterministic|TestPrimaryComposeEntryIgnoresNonComposePrimary|TestWarnMultiplePrimaryCompose|TestPostBuild"

## Design Record

### 1. Optional Dir (optional/dir-exists) — **채택**

**결정**: `LifecycleEntry.Optional bool` 필드 추가. plan resolution 단계에서 `optional: true`이고 지정된 디렉토리가 없으면 엔트리를 건너뛰고 warning trace 출력.

**근거**: 
- dripter 프로젝트에서 미체크아웃 subproject로 인해 plan 전체가 실패하는 문제 해결
- Stack 레벨에서 선언적 속성으로 두면 plan 작성자가 명시적으로 의도 표시 가능
- Resolver 단계에서 체크하면 plan 실행 전 조기 차단 가능, dry-run에도 반영됨

**구현**: `internal/lifecycle/resolver.go`의 `ResolvePlan()`에서 `os.Stat`로 디렉토리 존재 여부 체크, 없으면 `continue`로 엔트리 제외

---

### 2. Post-Build Step — **채택**

**결정**: `NativeRunnerConfig.PostBuild string` 필드 추가. `dva build` 실행 시 build 성공 후 동일 디렉토리/환경에서 post-build 명령 실행.

**근거**:
- db-orchestrator에서 `make build-api`는 빌드 후 아티팩트 복사까지 수행하나 `dva build`는 빌드만 수행하던 갭 해소
- Build와 PostBuild가 같은 소프트웨어에 대한 연속된 단계이므로 동일 dir/env 공유가 자연스러움
- Compose는 별도 빌드 시스템이므로 native에만 우선 적용

**구현**: `internal/cli/build.go`의 `buildNativeTarget()`에서 build 성공 후 `target.postBuild` 실행

---

### 3. Primary Compose Entry — **채택**

**결정**: `LifecycleEntry.Primary bool` 필드 추가. `PrimaryComposeEntry()`가 explicit `Primary: true` 우선 반환, 없으면 기존 order/name fallback. 다중 primary는 validation warning.

**근거**:
- primeno1에서 다중 compose 엔트리 중 어느 것이 primary인지 암묵적(order/name순)으로만 결정되어 provision compose_up과 service 지정 interaction에서 혼선 발생
- 명시적 표기로 의도 명확화, 마이그레이션 시 안정성 확보
- Validation warning으로 설정 실수 조기 발견

**구현**: `internal/config/lifecycle_helpers.go`의 `PrimaryComposeEntry()` 수정, `internal/config/validate_warnings.go`에 `warnMultiplePrimaryCompose()` 추가

---

### 4. Native Env Documentation — **보류 (이미 완료)**

**결정**: 별도 코드 변경 불필요. `schema.json`의 `native_runner_config.env` 필드에 이미 문서화되어 있음 (TASK-323 문서화 범위와 연계).

**근거**: 
- Schema에 `env: { type: "object", additionalProperties: { type: "string" } }`로 정의되어 있고 description도 있음
- postkit에서 문서화 부족으로 Makefile 선택했다는 이슈는 문서 보완으로 해결 가능 (TASK-323)

---

## Implementation Record

설계 결정을 구현한 뒤 **구현 자체에서 발견된 결함 두 건**을 같은 범위에서 고쳤다.
둘 다 전체 스위트가 통과하는 상태에서 잠복해 있었다 — 해당 경로를 덮는 테스트가
하나도 없었기 때문이다.

1. **`PrimaryComposeEntry()` 비결정성** — `Config.Stack`은 map이고 Go의 map 순회는
   무작위다. "처음 만난 primary를 쓴다"는 초기 구현은 `primary: true`가 둘 이상일 때
   실행마다 다른 엔트리를 돌려줬고, `warnMultiplePrimaryCompose`는 "알파벳순 첫
   번째가 쓰인다"고 안내하고 있었다 — 경고가 실제 동작과 어긋났다. 이름 기준 최소값
   선택으로 바꿔 경고문과 동작을 일치시켰다.
   회귀 가드: `TestPrimaryComposeEntryMultiplePrimariesIsDeterministic` (50회 반복 호출),
   `TestWarnMultiplePrimaryCompose/names_the_entry_actually_used` (경고가 지목한
   엔트리와 실제 선택 엔트리가 같은지 대조).

2. **optional dir 체크의 `process` 분기 死코드** — `decodeRunnersMap`은 runner를
   YAML에 적힌 이름 그대로 키로 넣고, `runners.process`는 `*ProcessPluginConfig`로
   디코드된다. 초기 구현은 그 값을 `*NativeRunnerConfig`로 타입 단언해 절대 성립하지
   않았고, 결과적으로 process 러너 엔트리는 `optional: true`를 달아도 항상 유지됐다.
   두 타입을 모두 받는 `optionalRunnerDir` 헬퍼로 교체했다.
   회귀 가드: `TestResolvePlanSkipsOptionalProcessEntryWithMissingDir`.

디렉토리를 선언하지 않은 optional 엔트리는 **검사할 대상이 없으므로 유지**한다 —
무관한 경로를 근거로 건너뛰지 않는다 (`TestResolvePlanKeepsOptionalEntryWithoutDir`).

`post_build`는 build 성공 뒤 같은 dir/env에서 실행되고, build 실패 시 건너뛰며,
자신이 실패하면 빌드를 실패시킨다 (`TestPostBuild*` 5건).

게이트: `make lint` 0 issues, `make test` (-race) 전체 통과, `make doc-check` OK.
