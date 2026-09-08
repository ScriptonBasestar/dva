---
id: TASK-333
title: "Expose owner and canonical/alias identity for imported items"
type: feature
priority: P2
effort: M
exec-tier: standard
status: done
created: 2026-09-07
source: "tasks/done/259 done-review (PLAN-007 Tier A batch 1)"
---

## Summary

TASK-259 §5 items 1-2 were scoped out of that card and never refiled. An imported item
currently renders identically to a locally declared one, so a user cannot tell where it came
from or which of several addresses is the canonical one.

Two additions:

1. An `owner` field on every item in `dva ls --json` and in the manifest output, naming the
   subproject the item was imported from (or the root for locally declared items).
2. Canonical/alias markers, so an item reachable under more than one address states which
   address is canonical and which are aliases.

Shell completion must offer all three address forms (`--project <name>`, `<name>:<key>`,
`<name>/<key>`) rather than only the canonical one — an alias the user can type but cannot
complete is a worse surface than no alias.

## Completion Criteria

- [x] `dva ls --json` and the manifest output carry an `owner` field for every item | verify: `/usr/bin/grep -rq "func TestImportedItemOwnerField" internal/cli`
- [x] items reachable under multiple addresses mark one canonical and the rest aliases | verify: `/usr/bin/grep -rq "func TestImportedItemCanonicalAndAliasMarkers" internal/cli`
- [x] completion offers all three address forms | verify: `/usr/bin/grep -rq 'func TestCompletionOffersAllThreeAddressForms(' internal/cli`

## Result

세 기준 모두 `internal/cli/imported_item_identity_test.go`의 실제 temp-dir fixture로
고정했다. 구현은 서브에이전트가, 리뷰는 별도 에이전트가 맡았고(구현자 != 리뷰어),
리뷰 판정은 REQUEST-CHANGES → **APPROVE-WITH-NITS**로 이동했다.

### 무엇을 어디에 실었나

- `InteractionCommand`에 `SubprojectName`/`CanonicalAddress` 두 `yaml:"-"` 필드
  (`internal/config/config.go`). 논리 이름은 공개해도 되지만 `SubprojectPath`는 아니라는
  이유를 필드 주석에 남겼다 — 경로는 이 머신의 사실이고 이름은 config의 사실이다.
- 스탬프는 `subproject.go:186-192`에서 **clone 하나에 한 번**. canonical 키와 alias 키가
  같은 clone을 가리키므로 map 키에서 되유도할 필요가 없다.
- `mergeInteraction`은 두 필드를 **항상 parent에서** 가져온다: `cloneImportedInteraction`이
  스탬프보다 **먼저** `Subcommands`로 재귀하므로 자식은 config 안에서 zero value다.
- manifest `schema_version` 1.6 → 1.7. 추가 전용이므로 minor
  (근거: `tasks/_archive/done/272-freeze-manifest-route-identity.md:191`).
- `Owner`에 `omitempty`를 **일부러 붙이지 않았다**. 모든 항목에 owner가 하나씩 있으므로
  "없음"이 신호가 될 여지가 없다. 대신 `ManifestDynCmd` 생성자 전부를 확인해
  `"owner": ""`가 새지 않는 것을 확인했다.

### 리뷰가 잡은 실제 결함 (MUST-FIX 1)

`completionCandidates`가 `LoadSubprojects(dir, c.Subprojects)`로 **맵 전체를 한 번에**
로드했다. 그 함수는 all-or-nothing 계약이다(`internal/config/subproject.go:9-12`:
"On any load failure the function returns nil, err"). 즉 이 머신에 없는 sibling 체크아웃
**하나**가 다른 모든 subproject의 `p:k` 형태를 completion에서 지웠다 — 그런데
`dva run p:k`는 여전히 라우팅됐다. best-effort 자리에서 fail-fast API를 부른 것이다.

`-overlay`로 저자 코드를 되돌려 직접 재현했다: subproject 둘, 경로 하나 없음 →
`run completion = [engine/compile]`. 건강한 쪽의 colon 형태까지 사라졌다.

수정은 **subproject당 한 번씩 로드하고 실패한 것만 건너뛴다**. `run.go:123`이 단일 엔트리
맵으로 같은 함수를 부르는 저장소 관용구와 같은 모양이며, 이제 completion이 제안하는 집합이
`dva run`이 실제로 받는 집합과 일치한다.

**왜 강한 테스트를 통과했는가**: 원래 fixture의 subproject가 정확히 **하나**였다.
cardinality 1에서는 all-or-nothing과 per-item 의미를 구별할 수 없다. 그래서 회귀 방어를
별도 테스트가 아니라 `TestCompletionOffersAllThreeAddressForms`의 fixture에 `broken:`
subproject로 심었다 — 그 파일의 모든 단언이 동시에 회귀 테스트가 된다.

### `root` 충돌 (리뷰 brief 항목 1)

owner 센티널이 `root`인데 `subprojects.root`를 막는 것이 없었다. 측정: manifest 두 행
`local-task`와 `root/compile`이 **둘 다** `owner=root`. 필드로 필터링하는 소비자에게
대체 신호가 없어 문서화가 아니라 거부를 택했다.

리뷰는 `validate.go`에 `"root"`를 다시 쓰라고 했지만 **그렇게 하지 않았다**. 두 패키지에
컴파일러가 볼 수 없는 관계로 같은 문자열이 놓이면, 나중에 센티널을 개명해도 아무 테스트가
깨지지 않은 채 충돌이 되살아난다. `config.RootOwnerName` 상수 하나를 두고
`internal/cli/list.go`가 그것을 alias한다 — 관계를 타입 시스템에 넣었다.

`reservedCommands`에는 **일부러 넣지 않았다**. 그 집합은 이름을 interaction 키와 `p:k`
접두사로도 금지하는데 `dva run root`에는 모호한 것이 없다. 좁은 문제에는 좁은 규칙.
에러 메시지도 공유하지 않는다 — reserved-name 문구는 독자를 `reservedCommands`로 보내는데
`root`는 거기 없다. `subproject_reserved_test.go`의 새 subtest가 두 메시지가 갈라진 채로
있음을 고정한다(`"reserved subproject name"`을 **포함하지 않을 것**까지 단언).

### 반증 테스트 (mutation)

- `completion.go`를 저자의 whole-map 로드로 되돌림(`-overlay`) →
  `TestCompletionOffersAllThreeAddressForms`가 FAIL, 건강한 subproject의 colon 형태 소멸.
- `names = append(names, projectName+":"+key)` 삭제 → `key` 미사용으로 **컴파일 에러**;
  `_ = key`로 치환하면 깨끗한 FAIL. 저자가 "이 기준은 mutation 증명이 불가능하다"고 한 것은
  틀렸고, 독립적인 증명이 둘 있다.

### 리뷰가 옳지만 이 카드에서 고치지 않은 것

`plans:` import는 아직 세 필드를 싣지 않는다. 자식 plan `boot`를 `as: quickboot`로 import하면
manifest에 **바이트 단위로 동일한** 두 항목이 나오고 둘을 구분할 표시가 없다 — 이 카드가
interaction에서 없애려던 바로 그 모호함이 한 절 옆에 남아 있다.

`USAGE.md`에 공백을 명시하고(소비자를 오도하지 않게) **동시에** TASK-366로 카드를 냈다.
산문 disclosure만으로는 고쳐지지 않고, 주인 없는 알려진 결함은 리뷰마다 다시 발견되어 다시
기록되기 때문이다. TASK-366를 닫을 때 그 문단을 같은 커밋에서 지우도록 카드에 적어 뒀다.
