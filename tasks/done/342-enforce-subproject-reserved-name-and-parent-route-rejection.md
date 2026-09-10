---
id: TASK-342
title: "Enforce TASK-263's subproject reserved-name and parent-route rejection rules"
type: feature
priority: P1
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-reviewed-at: 2026-09-10
quality-review-evidence: "Independent TASK-367 review: reserved-name and all three parent-route rejection rules remain enforced by the named tests."
created: 2026-09-07
source: "tasks/done/263 done-review (PLAN-007 Tier A batch 1)"
---

## Summary

TASK-263 documented three subproject rules that the code never enforces. The done-review
confirmed each one is stated in the design but unbound in `internal/config`:

1. A subproject name that collides with a reserved built-in surface is accepted today.
   `dva config validate` must reject it and name the reserved word in the error.
2. A key the child's own validator rejects stays reachable through the parent routes
   (`--project <name>`, `<name>:<key>`, `<name>/<key>`). The parent must not offer an
   address the child would refuse.
3. Both errors must state their basis — which rule fired and which declaration triggered
   it — so a user can fix the config without reading the source.

Filed from the TASK-263 done-review; the card was left `conditional` for exactly this gap.

## Completion Criteria

- [x] `dva config validate` rejects a subproject whose name collides with a reserved built-in, and the message names the reserved word | verify: `/usr/bin/grep -rq "func TestSubprojectReservedNameRejected" internal/config`
- [x] a key the child validator rejects is unreachable through all three parent address forms (`--project`, `:`, `/`) | verify: `/usr/bin/grep -rq "func TestSubprojectParentRouteRejectsChildInvalidKey" internal`
- [x] both rejection paths report the rule and the offending declaration | verify: `/usr/bin/grep -rq "func TestSubprojectRejectionNamesRuleAndDeclaration" internal`

## Resolution (2026-09-08)

세 규칙 모두 구현하고 각각의 강제 지점을 mutation으로 검증했다.

### 측정한 결함 (v0.1.48, 커밋 `f94d090`)

부모가 `subprojects: {up: {path: child}}`와 `interaction: {"up:web": ...}`를 선언한 설정에서:

- `dva up:web` → `CHILD-WEB`. 같은 실행이 `ConflictAdvice`의 "no invocation reaches this
  key ... it fails with subproject 'up' not found"를 WARN으로 찍었다. 불가능하다고
  안내한 호출이 다른 프로젝트의 다른 명령에 도달했다.
- `dva run --project up status` → `CHILD-STATUS`. 자식 디렉토리에서 `dva config
  validate`를 돌리면 같은 `status` 키를 reserved conflict로 거부한다. 부모만이 그 키가
  동작하는 유일한 장소였다.

### 구현

| 규칙 | 강제 지점 |
|---|---|
| (a) 예약어 서브프로젝트 이름 | `Config.Validate` ← `ReservedSubprojectNames` / `SubprojectConflictAdvice` (`internal/config/reserved.go`) |
| (b) `--project`, `<p>:<k>` | `runSubprojectCommand` (`internal/cli/run.go`) |
| (b) `<p>/<k>` import | `resolveSubprojectImports` (`internal/config/subproject.go`) |
| (c) 규칙·선언 명시 | `SubprojectKeyRejection` — 세 형식이 공유하는 단일 생성자 |

(b)는 **키 단위**다. 자식에 결함 키가 하나 있어도 나머지 키의 import와 실행은 그대로다
(`TestSubprojectImportAcceptsKeysTheChildValidatorAllows`). 거부 판정은
`ValidateReservedCommands`에 위임해서, 부모가 거부하는 집합이 자식 validator가 거부하는
집합과 구조적으로 같도록 했다 — 재유도하면 hook 예외나 예약어 집합이 움직일 때 어긋난다.

(a)는 `config validate`에서만 거부한다(런타임 라우팅은 그대로). TASK-263 §3이 명시한
범위이고, 이로써 `UnroutableNamespacePrefix`의 설명이 "validate를 통과하는 모든 설정"에
대해 참이 된다.

### 비공허성 근거

`go test -overlay`로 세 강제 지점을 각각 무력화했다 (repo 무수정):

| mutation | 실패한 테스트 |
|---|---|
| A: `Validate`의 블록 제거 | `TestSubprojectReservedNameRejected` (reserved 2건), `TestSubprojectRejectionNamesRuleAndDeclaration/reserved_subproject_name` |
| B: `run.go`의 체크 제거 | `.../--project_form`, `.../colon_shorthand_form` (각각 단독 실행으로 확인) |
| C: `subproject.go`의 체크 제거 | `.../slash_import_form`, `.../child_key_rejected_through_the_import_route` |

### 문서

`USAGE.md`의 `compose:ps` 도달 불가 설명에 이 규칙이 전제라는 문장을 추가하고,
`### subprojects`에 두 규칙을 기술했다. `ConflictAdvice`와
`warnLiteralKeyShadowsSubproject`의 주석에도 상호 참조를 남겼다.

### 독립 리뷰가 잡은 회귀와 후속 수정 (2026-09-08)

구현과 다른 세션의 리뷰가 위 커밋(`76a177b`)이 **읽기 표면 두 곳을 거짓으로 만들었다**고
측정했다. 규칙 (b)가 부모 라우트를 막으면서, 그 라우트를 광고하던 목록이 갱신되지 않았다.

`engine` 자식이 `status`(예약어)와 정상 키를 함께 선언한 부모에서:

- `dva manifest` → `"status": {"usage_example": "dva engine:status"}`. 같은 바이너리가
  `dva engine:status`를 exit 1로 거부한다. `manifest.go`의 `UsageExample` 주석이 그
  불변식을 그대로 적어 놓았다 — *"usage_example carries an implicit promise that running
  it invokes the entry it sits inside."*
- `dva ls --project engine` → `status`가 표시 없이 출력, `-f json`에도 표식 없음. 루트
  목록은 같은 조건의 키를 이미 표시하므로, 두 범위를 비교한 독자는 서브프로젝트 키만
  멀쩡하다고 결론짓게 된다.

**회귀이지 기존 결함이 아니다.** `76a177b` 이전에는 `dva engine:status`가 실제로 자식의
명령을 실행했으므로 두 표면 다 참이었다. 그래서 후속 카드가 아니라 같은 브랜치에서 고쳤다.

수정 범위:

| 표면 | 변경 |
|---|---|
| `internal/cli/list.go` `subprojectUsage` | 네 번째 상태 반환 — `unroutable` + 자식 자신의 진단. 거부를 먼저 검사(도달 호출이 없는 키는 shadowed로 설명될 수 없다) |
| `printSubprojectTable` | `(unreachable: subproject '...' rejects '...' — '...' is a reserved DVA command)` 표시. `printTable`과 달리 rename 제안 없음 — 평범한 예약어 키에서는 `RenameSuggestion`이 입력을 그대로 돌려줘 "rename to 'status'"가 된다 |
| `buildSubprojectCommandEntries` | `unroutable` / `unroutable_reason` — 루트 목록과 동일한 presence-is-the-signal 계약 |
| `buildManifestSubprojectCommands` | `UsageExample` 생략, `Unroutable`/`UnroutableReason` 설정 |
| `Config.RejectsInteractionKey` | 충돌한 내장 커맨드 이름을 함께 반환. 호출자가 재유도하면 이 함수가 "문제없음"이라 답한 키에 대해 내장 이름을 찍는 일이 가능해진다 |

거짓이 된 주석 네 곳(`list.go:103-105`, `subprojectUsage` 독스트링, `manifest.go`의
`Unroutable`·`ShadowedByLiteralKey` 필드)도 함께 고쳤다. `Unroutable`은 새 필드를 만들지
않고 재사용했다 — `ShadowedByLiteralKey`가 `ShadowedByBuiltin`에서 갈라져 나온 이유는 그
필드가 `static_commands` 표를 **가리키기** 때문인데, `Unroutable`은 어떤 표도 가리키지
않고 "왜 `usage_example`이 없는가"라는 한 질문의 답이다.

표시 대상은 **선언된 키**(`cmd.Path[0]`)다. `runSubprojectCommand`는 `args[0]`만 검사하고
나머지는 보지 않으므로, 거부된 `status` 아래 `status build` 행도 같이 죽는다.

#### 회귀 수정의 비공허성

`subprojectUsage`의 거부 분기를 제거하면(on-disk mutation) 세 테스트가 정확히 그 결함을
재현하며 실패한다:

```
--- FAIL: TestManifestSubprojectRejectedChildKeyOffersNoInvocation
    usage_example = "dva guard:status", want empty
--- FAIL: TestLsProjectMarksRejectedChildKey
--- FAIL: TestSubprojectUsage
    rejected case: usage="dva engine:status" ... unroutable=""
```

#### 픽스처가 드러낸 것

기존 `subproject_usage_test.go`의 `engine` 자식은 `build` 키를 선언하고 있었다 — `build`는
예약어이므로 자식 자신의 validate가 거부하고, 규칙 (b)에 의해 부모 라우트도 거부한다.
"정상 키" 역할을 못 하므로 `compile`로 바꿨다. 규칙이 평범한 설정에도 실제로 문다는 증거다.

`run`이라는 이름의 서브프로젝트를 쓰던 `TestManifestSubprojectNamedAfterReservedCommandIsUnmarked`는
삭제했다. 규칙 (a)가 그 이름을 하드 에러로 만들었으므로 그 테스트가 고정하던 명제
("예약어 이름 서브프로젝트도 정상 라우팅된다")는 더 이상 정책이 아니다. 커버리지는
`TestSubprojectReservedNameRejected`로 옮겨갔다.

#### 게이트

`make test` / `make lint` / `make doc-check` / `make check-generate` 모두 exit 0.

### 2차 독립 리뷰: 주석 정확성과 순서 결정 (2026-09-08)

같은 리뷰어가 `76a177b`를 끝까지 읽고 BLOCK을 냈다. must-fix 1(읽기 표면 회귀)은
`0c4a4ef`가 이미 닫았고 — 리뷰는 그 커밋 이전 트리를 본 것이다 — must-fix 2와
should-fix 셋이 남았다. 코드 규칙 자체에는 결함이 없다.

#### 측정한 것

`reserved.go`가 스스로 세운 기준은 "여기 적힌 모든 invocation은 바이너리에 대해
실행됐다. 거부당하는 커맨드를 이름 대는 조언은 조언이 없느니만 못하기 때문"이다.
1e73a99에서 빌드한 바이너리로 네 주장을 전부 측정했다.

| 픽스처 | 실행 | 결과 |
|---|---|---|
| subproject `up` + 자식 `web` | `dva up:web` | `CHILD-WEB`, rc 0 |
| 같은 파일 | `dva config validate` | rc 1 |
| subproject `up` + 자식 `web` + interaction `up:web` | `dva up:web` | 조언을 WARN으로 남기며 `CHILD-WEB`, rc 0 |
| interaction `up:web`만 (subproject 없음) | `dva config validate` | rc 1 |
| subproject `engine` + 자식 `status` | `dva run --project engine status` | 거부, rc 1 |
| 같은 상황, 조언이 시킨 대로 | 부모 디렉토리에서 `dva run status` | `not recognized`, rc 1 |
| 같은 상황 | 자식 디렉토리에서 `dva run status` | `CHILD-STATUS`, rc 0 |
| subproject `compose` + 자식 `ps` | `dva compose:ps` / `config validate` | rc 0 / rc 1 |

#### 고친 주석 셋 — 모두 측정으로 반증됨

1. **`ConflictAdvice`** — "'no invocation reaches this key'가 참인 것은
   `ReservedSubprojectNames`가 거부하기 때문". 거짓이다. 그 규칙은 `Validate()`에서만
   돌고 라우팅 경로에 없어서, 규칙이 들어온 지금도 `dva up:web`은 자식을 실행한다.
   문장을 참으로 만드는 것은 interaction 키 자체를 거부하는 `ValidateReservedCommands`다.
   "validate를 통과한 config에 한정"으로 범위를 명시하고 두 측정을 함께 기록했다.

2. **`ReservedSubprojectNames`** — "여기서 거부하는 것이
   `UnroutableNamespacePrefix`의 서술을 모든 validate 통과 config에 대해 참으로 만든다".
   공허하다. `up:web`은 subproject `up`의 존재와 무관하게 이미 거부되므로(측정), 그
   서술은 이 규칙 없이도 공허하게 참이었다. 규칙이 실제로 사는 것은 **선언**을 막는
   것이라고 고쳐 쓰고, 닫지 않는 divergence(`compose:ps`)를 §3 (a)가 라우팅을 동결한
   결과로 명시했다.

3. **`SubprojectKeyRejection`** — "자식의 진단을 실어 부모 출력만으로 조치 가능".
   거짓이다. 전달되는 진단은 `dva run status`를 가리키는데 부모 디렉토리에서 그것은
   exit 1이고 `cd` 후에야 동작한다. 진단은 **원인 설명이지 여기서 실행할 커맨드가
   아니다**로 고쳤다 — 부모의 언어로 다시 쓰는 것은 이 생성자가 피하려던 중복 진단이다.

커밋 메시지 본문에도 같은 주장이 있었으나 `f268ae7`은 이미 통합·푸시됐으므로 정정은
여기에 남긴다.

#### should-fix 셋

- **`exclude_tags` 대 자식 거부의 순서 — 기록이 아니라 이동을 택했다.** 부모가
  `exclude_tags`로 제외한 키를 `ls --project`는 없는 것으로 보여주는데 `run --project`는
  "있지만 거부"라고 답했다. 한 철자에 두 답 — `LiteralKeyWins` 주석이 금지하는 형태다.
  거부 검사를 `FilterInteractions` **뒤로** 옮겨 양쪽 모두 "not found"가 되게 했다.
  `TestSubprojectParentRouteExcludedKeyStaysNotFound`가 두 레그로 고정한다: 제외된 키는
  not-found이고 자식 진단을 유출하지 않으며, 제외되지 않은 같은 키는 여전히 거부된다.
  두 번째 레그가 없으면 "검사를 지웠다"와 "필터가 먼저 온다"를 구별할 수 없다.
- **거부 레그 둘에 `dryRun = true`.** 검사가 `dryRun`보다 먼저 돌므로 단언을 약화시키지
  않는다. 측정: 검사를 지우면 `dryRun` 없이는 `--project` 레그만 보고되고
  `dvaexec.ExecReplace`의 panic이 바이너리를 끝내 colon 레그는 실행조차 되지 않았다.
  설정 후에는 두 레그 모두 깨끗한 FAIL이다.
- **CHANGELOG `[Unreleased]`**에 breaking change 둘과 순서 변경을 기록했다.

#### 반증 테스트 (mutation)

- 거부 검사를 필터 앞으로 되돌림 → `..._ExcludedKeyStaysNotFound/excluded_reserved_key_reports_not_found`가
  두 단언 모두에서 FAIL.
- 거부 검사 삭제 + `dryRun` 유지 → 거부 레그 둘 다 FAIL (panic 없음).
- 거부 검사 삭제 + `dryRun` 제거 → `--project` 레그만 FAIL 후 panic으로 바이너리 종료.

#### 리뷰가 옳지만 고치지 않은 것

`dva compose:ps`가 라우팅되면서 `config validate`는 exit 1인 divergence는 실재한다.
TASK-263 §3 (a)가 규칙을 validate에 두고 라우팅을 동결하기로 한 결과이며, 닫으려면
검사를 라우팅 경로로 옮기는 별도 결정이 필요하다. 코드는 그대로 두고 주석과
CHANGELOG에 명시했다.
