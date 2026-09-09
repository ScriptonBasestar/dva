---
id: PLAN-009
title: "Work the tools/doccheck defect bundle in dependency order"
type: plan
scope: "TASK-344, 350, 343, 354, 338 — 다섯 장 모두 tools/doccheck(또는 그 Makefile 연결)를 고친다. 선언된 의존은 TASK-343 depends-on TASK-344 하나뿐이다"
progress: 0
total-tasks: 5
completed-tasks: 0
children: [TASK-344, TASK-350, TASK-343, TASK-354, TASK-338]
target-date: "2026-09-30"
created: 2026-09-09
---

## Goal

다섯 장이 고아 상태로 보드에 흩어져 있었다 — 전부 `tools/doccheck` 하나(또는 그 Makefile
연결)를 고치는데, 서로 간에 선언된 의존은 TASK-343의 `depends-on: [TASK-344]` 하나뿐이다.
이 문서가 이 다섯 장의 소유자이고, 소유하는 것은 우선순위가 아니라 **순서**다 —
`PLAN-008 §Order`가 세운 형식을 그대로 따른다.

## 왜 병렬로 돌리면 안 되는가 — PLAN-008과 같은 모양

[PLAN-008](008-migrate-section-order-defects.md)이 이미 이 형태로 한 번 다쳤다: 여덟 장이
`internal/config/migrate_section_order.go` 한 파일에 동시에 걸렸고, TASK-363이 slot-separator
로직을 `:184-188`로 인용했는데 실제 위치는 파일이 자란 뒤의 `:234-244`였다 —
"여덟 장이 동시에 움직이면 이 일이 여덟 번 난다"고 그 문서는 적었다.

이 다섯 장도 정확히 같은 모양이다. `tools/doccheck/check.go`의 `Result` 구조체(19-51행,
필드 32개)와 `tools/doccheck/main.go`의 출력 블록(60-78행)이 그 파일이다 — TASK-350은
역전(`grep -L`)·공허(bare 스위트 타깃) 두 카운터를, TASK-338은 `stale_link_paths` 카운터를
**같은 구조체, 같은 출력 블록**에 추가한다. 두 카드가 동시에 열리면 각자 옳은 필드를 넣고
서로의 줄 번호를 밀어낸다 — PLAN-008이 겪은 것과 같은 사고다. TASK-344는 별도로
`cardstatus.go`/`archive.go`의 공유 헬퍼(`frontmatterField`)를 고치고, TASK-343은 그 헬퍼가
반환하는 값을 그대로 읽는다(`cardids.go:60`) — 파일은 다르지만 결합은 같은 종류다.

## Order

| # | 카드 | P | 왜 이 자리인가 |
|---|---|---|---|
| 1 | TASK-344 프론트매터 값 파서 강화 | P3 | **가장 먼저.** `cardstatus.go`의 `frontmatterField`와 `archive.go`의 `hasCanonicalField`가 공유하는 파서를 고친다. TASK-343이 `depends-on: [TASK-344]`로 이미 선언했고, 그 카드 자신의 §Ordering이 "먼저 안 하면 여기서 쓴 테스트가 344 이후 다시 쓰인다"고 적어 뒀다. 보드에서 가장 잘 명세된 카드이기도 하다 — 검증 바인딩 4개가 오늘 전부 정확히 실패하고, 공허한 기준이 0개다 |
| 2 | TASK-350 역전/공허 바인딩 거부 | P2 | **343보다 먼저.** 350은 공허한 verify 바인딩을 거부하는 카드다. 근거는 **잔존 건수가 아니라 발생률**이다 — 2026-09-08~09 이틀 사이에 세 건이 새로 심어졌다: TASK-369의 release-notes 바인딩, TASK-338의 `make doc-check`, TASK-341의 `make test`. **셋 다 지금은 처리됐다**(각각 재작성·삭제·`(regression-guard)` 표기), 그러니 오늘 이 셋을 실행해 재현하려 하지 말 것. 그런데도 카드가 닫히지 않는 이유가 이 자리의 논거다: **세 건 전부 사람이 리뷰에서 붙잡았고 검사기는 아직 없다.** 350이 먼저 들어와야 다음 필링 배치가 같은 형태를 또 넣지 못한다 |
| 3 | TASK-343 파일명 번호 충돌 가드 | P2, `depends-on: [TASK-344]` | 344가 닫히면서 언블록된다. 순서 자체는 하드 제약(§순서가 하드 제약인 곳 참조) |
| 4 | TASK-354 보드를 `ce task validate` 통과시키고 게이트 연결 | P2 | 공유 `ce task gate`를 저장소 게이트에 연결한다. 이 연결이 들어와야 그 엔진이 이미 무엇을 커버하는지(예: verify 바인딩의 stale 경로 경고, `ce task validate tasks/todo/329-*.md`가 오늘도 실제로 낸다) 눈에 보인다 — 5번 TASK-338이 자기 범위를 "엔진이 안 보는 부분"으로 정확히 그으려면 이 표면이 먼저 드러나 있어야 한다 |
| 5 | TASK-338 링크의 written path 소멸 보고 | P2 | **마지막.** 이 카드는 인용 증거(docs/53·58·61의 PLAN-002 링크)가 사라진 채였고 C3 기준이 닫힌 기록을 건드릴지 여부를 묻지 않고 숨겨 두고 있었다 — **둘 다 2026-09-09에 카드를 다시 쓰면서 처리했다**(TASK-338 §열린 결정, §TASK-354와의 경계). 남은 것은 그 §열린 결정에 사람이 `tasks/_archive/` 85건의 포함 여부를 적는 일이고, 그것이 이 카드 `needs-human: true`의 내용이다. 자리가 마지막인 이유는 354가 드러낸 엔진 표면(verify 바인딩 stale 경로)과 겹치지 않는 범위(`docs/`의 마크다운 링크)로 다시 그은 뒤에 착수해야, 엔진이 이미 하는 일을 `tools/doccheck` 안에서 한 겹 아래 다시 구현하는 함정(TASK-354 §게이트 연결이 스스로 경고한 바로 그 함정)을 피한다 |

## 순서가 하드 제약인 곳 — 한 곳뿐

나머지 네 자리는 전부 권고다. 아래 하나만 카드 본문에 명시적으로 적혀 있다.

- **1 → 3** (TASK-344 → TASK-343). TASK-343 §Ordering: "the tests written here would be
  rewritten once TASK-344 changes what the helper returns." `ce task preflight`는
  `unmet-dependency`를 advisory가 아니라 **blocking**으로 센다 — 344가 닫히기 전에는 343이
  runnable 집합에서 아예 빠진다.

2번(350)이 3번(343)보다 앞서는 것은 하드 제약이 아니라 **긴급도**다: 344와 350 사이에는
아무 결합이 없으므로 350을 1번 자리에 넣어도 깨지지 않는다. 다만 350이 막으려는 결함이
바로 지금(TASK-338·341) 재생산되고 있으므로, 344 직후로 당겨 더 심어지는 것을 막는다.
4번(354)과 5번(338)의 결합은 코드 결합이 아니라 **정보 결합**이다 — 354가 노출하는 엔진의
커버리지를 보지 않고 338의 범위를 그으면, TASK-354 §게이트 연결이 경고한 "엔진이 하는 일을
저장소 안에서 다시 구현"하는 함정을 그대로 밟는다.

## Children

- TASK-344 — 프론트매터 값 파서를 트레일링 주석·중복 키에 강화 (P3, 공유 파서를 먼저 고정)
- TASK-350 — doccheck의 역전/공허 verify 바인딩 거부 (P2, 지금도 심어지고 있어 344 다음으로 당김)
- TASK-343 — 파일명 번호 충돌 가드 완성 (P2, `depends-on: [TASK-344]`)
- TASK-354 — 보드를 `ce task validate` 통과시키고 공유 게이트에 연결 (P2, 엔진 커버리지를 드러냄)
- TASK-338 — 링크의 written path 소멸을 보고 (P2, 354가 드러낸 엔진 범위와 겹치지 않게 재기술 후 착수)

## Rules

- 한 번에 한 장. 앞 장이 master에 통합된 뒤 다음 장을 연다.
- **TASK-350과 TASK-338을 병렬로 열지 말 것.** 둘 다 `tools/doccheck/check.go`의 `Result`
  구조체와 `tools/doccheck/main.go`의 출력 블록에 카운터를 추가한다 — 같은 자리다.
- 카드 상태는 `tasks/todo` → `tasks/done`이 정본이고, 이 문서는 순서·의존·기각 기록만 갖는다.
