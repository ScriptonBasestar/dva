---
id: TASK-319
title: "native entries: optional dir, post-build steps, explicit primary compose entry"
type: feature
priority: P2
effort: M
exec-tier: strong
created-at: 2026-09-05T10:30:00+09:00
source: "docs/dogfood/{dripter,scripton-db-orchestrator,primeno1,funbricks-postkit}.md"
status: done
quality-review: pass
quality-review-evidence: "독립 리뷰 2라운드(review-319). 1라운드 conditional — findings 9건. 해소: B1 존재하지 않는 --trace 플래그 문서 오류, B2 lifecycle.go:50/schema.json:955의 거짓 'with a warning' 약속, F3 평면 선언 형태에서 optional 무동작(코드 수정), F6 PrimaryComposeEntry 헤더 주석 드리프트, F7 실패 불가능한 교차검증 단언, F9 done 카드의 needs-human 잔존. 분리: F4·F5·실행경로 경고 → TASK-374. 게이트(리뷰어 직접 실행): make lint 0 issues / go test ./... FAIL 0 / make doc-check OK (yamlcheck 5 examples, 0 err, 0 warn) / make check-generate 드리프트 없음."
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

**결정**: 코드 변경 없이 문서로만 해소. 단, 최초 기록의 근거가 사실과 달라 정정한다.

**정정**: 최초 기록은 "schema에 description도 있음"을 근거로 보류했으나, 확인 결과
`internal/config/schema.json:244`의 `env`에는 `description` 키가 **없다**
(`{ "type": "object", "additionalProperties": { "type": "string" } }` 뿐이며 :258, :1022도 같다).
보류 결론 자체는 유지하되 근거를 실제 문서 위치로 바꾼다.

**실제 근거**:
- native `env:`의 유일한 사용자 문서는 `USAGE.md`의 마이그레이션 예시(`PORT: "8080"`)이며,
  우선순위에 대한 산문 설명은 없다. 값 우선순위는 CLAUDE.md/USAGE.md의 env 우선순위 절이 소유한다.
- postkit이 Makefile을 택한 원인은 이 산문 부재이므로 코드가 아니라 문서로 닫는 것이 맞다 (TASK-323).
- 이 카드 범위에서는 새로 추가한 `optional`/`post_build`/`primary` 3개 필드를 USAGE.md에
  문서화하는 것까지 수행했다. 세 예시 모두 fence info string 을 `yaml dva.yml` 로 달아
  yamlcheck(TASK-357)가 실제로 검증한다 — `yaml_examples_checked: 5, errors 0, warnings 0`.
  표시를 붙이자 `primary` 예시가 stack 레벨 `order:` 를 가르치고 있다는 사실이 드러났다.
  그 필드는 validator 가 `plans.*.entries[].order` 로의 이전을 권고하는 deprecated
  필드이므로 예시에서 걷어내고 산문에 그 사실을 명시했다. native `env:` 산문 보강은
  TASK-323이 계속 소유한다.

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

### 독립 리뷰 후 추가 수정

리뷰어가 optional 검사의 **무동작 범위**를 지적했고, 확인 결과 사실이었다.
`decodeRunnersMap`은 리터럴 `runners:` 키가 있을 때만 `Runners`를 채우는데,
`LifecycleEntry`는 `process:`/`tilt:`/`vagrant:`/`kustomize:`/`serverless:` 평면 필드라는
**별개의 선언 형태**도 갖는다. 그 형태로 쓴 엔트리는 `Runners`가 비어 있어
`optional: true`가 스키마를 통과하고도 아무 일도 하지 않았다 — 선언적 플래그가 가질 수
있는 최악의 실패 형태다.

`optionalEntryDir`로 교체해 세 선언 형태를 모두 본다: `runners` 맵(**러너 이름
사전순** — 맵 순회 무작위성 때문에 "먼저 만난 러너"는 실행마다 다른 디렉토리를
고른다), 평면 타입 필드, `source.path`. `dir`를 가진 러너 6종(native, process,
kustomize, tilt, vagrant, serverless)을 모두 인식한다.

회귀 가드: `TestResolvePlanSkipsOptionalEntryAcrossDeclarationShapes`(7개 서브테스트).
수정 전 코드 대비 **6/7 실패**를 실측했다 — 평면 선언 5종과 `runners.tilt`가 실패하고,
`source.path`는 수정 전에도 통과한다. b765d11의 switch는 각 case 식에 `!= ""` 가드를
두어(`case optionalRunnerDir(...) != "":`) 빈 Dir이면 다음 case로 흘렀고, 그래서
`source.path` 폴백에 정상적으로 도달했다. 이 카드는 그 동작을 유지한다.

`TestOptionalEntryDirIsDeterministicAcrossRunners`(50회 반복)와
`TestOptionalEntryDirIgnoresNilFlatFields`(any에 담긴 타입 nil 포인터는 nil
인터페이스가 아니므로 가드 없이 `Dir`을 읽으면 panic)는 새 함수를 직접 부르는 단위
테스트라 pre-fix 대비 실측 대상이 아니다.

문서 쪽 거짓 진술 3건도 함께 고쳤다.

- `lifecycle.go`의 `Optional` 주석과 `schema.json`의 `optional` description이 둘 다
  "skipped **with a warning**"이라고 약속했으나, 실제로는 `resolved.trace(...)`만
  남기고 `printPlanResolution`은 dry-run 경로에서만 호출된다. 실행 경로에는 경고가
  없다. 두 문구를 실제 동작(dry-run 트레이스)으로 정정했다.
- USAGE.md가 존재하지 않는 `--trace` 플래그를 안내하고 있었다(`internal/cli/` 전체에
  등록 없음). `--dry-run`으로 정정했다.
- `PrimaryComposeEntry` 주석이 explicit primary를 "returned **immediately**"라고
  설명했으나 실제로는 루프를 끝까지 돌아 사전순 최소값을 고른다. 정정했다.

범위 제한은 USAGE.md에 명시했다 — `compose`/`docker`/`helm`/`script`/`kubectl`은
디렉토리가 아니라 파일로 대상을 지정하므로 `optional`이 항상 유지로 동작한다는 점,
그리고 검사가 plan이 선택한 러너가 아니라 우선순위상 첫 디렉토리를 본다는 점.
경고 자체의 실행 경로 노출과 러너 선택 반영은 TASK-374가 이어받는다.

### 테스트 경로 자체의 결함

리뷰어가 findings 가 아니라 **evidence gap** 으로 지적한 것이 가장 값졌다: optional
테스트가 전부 `*config.Config` 구조체 리터럴로 만들어져 `config.Load()` 를 통과하는
경로가 하나도 없었다. 손으로 만든 `Runners` 맵은 항상 채워져 있지만 YAML 디코더는
리터럴 `runners:` 키에서만 채운다 — 평면 선언 형태 무동작이 눈에 띄지 않은 이유가
정확히 이것이다. 테스트가 프로덕션 디코더를 우회하면 디코더의 결함은 구조적으로
보이지 않는다.

`TestResolvePlanSkipsOptionalEntryLoadedFromYAML` 을 추가해 세 선언 형태를 YAML 로
쓰고 `config.Load()` 로 태운다. 추가하자마자 구조체 리터럴 테스트가 통과시키던
제약 위반이 드러났다 — `source:` 는 v1 에서 compose 러너만 허용한다
(`stack.side.source: v1 does not support runner "process"`). 구조체 리터럴은 이
검증을 건너뛰므로, 실제로는 불가능한 설정을 테스트하고 있었던 셈이다.

또한 `TestWarnMultiplePrimaryCompose/names_the_entry_actually_used` 의 교차검증이
실패할 수 없는 형태였다. 경고문이 충돌하는 모든 primary 를 나열하므로
`strings.Contains(got[0], used.Name)` 는 어느 엔트리가 반환되든 그 나열에 걸려
통과한다. 승자만 등장하는 `"(<name>) is used"` 절을 겨냥하도록 바꿨고, 선택 로직을
사전순 최대값으로 뒤집어 실제로 실패하는 것을 실측했다.

게이트: `make lint` 0 issues, `make test` (-race) 전체 통과, `make doc-check` OK,
`make check-generate` 드리프트 없음.
