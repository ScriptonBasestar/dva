---
id: TASK-342
title: "Enforce TASK-263's subproject reserved-name and parent-route rejection rules"
type: feature
priority: P1
effort: M
exec-tier: strong
status: done
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
