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
