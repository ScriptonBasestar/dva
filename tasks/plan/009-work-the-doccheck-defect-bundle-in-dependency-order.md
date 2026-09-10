---
id: PLAN-009
title: "Work the task and doccheck defect bundle in dependency order"
type: plan
scope: "TASK-371, 344, 350, 343, 354, 338 — 여섯 장이 shared task progress contract, tools/doccheck, 또는 그 Makefile 연결을 고친다. 선언된 의존은 TASK-343 depends-on TASK-344와 TASK-354 depends-on TASK-371이다"
progress: 83
total-tasks: 6
completed-tasks: 5
children: [TASK-371, TASK-344, TASK-350, TASK-343, TASK-354, TASK-338]
target-date: "2026-09-30"
created: 2026-09-09
---

## Goal

여섯 장이 고아 상태로 보드에 흩어져 있었다. TASK-371은 shared task progress contract와
DVA `planprogress`를 맞추고, 나머지 다섯 장은 `tools/doccheck` 또는 그 Makefile 연결을
고친다. 선언된 의존은 TASK-343의 `depends-on: [TASK-344]`와 TASK-354의 `depends-on: [TASK-371]`다. 이
문서가 이 여섯 장의 소유자이고, 소유하는 것은 우선순위가 아니라 **순서**다 —
`PLAN-008 §Order`가 세운 형식을 그대로 따른다.

## 왜 병렬로 돌리면 안 되는가 — PLAN-008과 같은 모양

[PLAN-008](008-migrate-section-order-defects.md)이 이미 이 형태로 한 번 다쳤다: 여덟 장이
`internal/config/migrate_section_order.go` 한 파일에 동시에 걸렸고, TASK-363이 slot-separator
로직을 `:184-188`로 인용했는데 실제 위치는 파일이 자란 뒤의 `:234-244`였다 —
"여덟 장이 동시에 움직이면 이 일이 여덟 번 난다"고 그 문서는 적었다.

TASK-344·350·343·354·338 다섯 장은 정확히 같은 모양이다. `tools/doccheck/check.go`의 `Result` 구조체(19-51행,
필드 32개)와 `tools/doccheck/main.go`의 출력 블록(60-78행)이 그 파일이다 — TASK-350은
역전(`grep -L`)·공허(bare 스위트 타깃) 두 카운터를, TASK-338은 `stale_link_paths` 카운터를
**같은 구조체, 같은 출력 블록**에 추가한다. 두 카드가 동시에 열리면 각자 옳은 필드를 넣고
서로의 줄 번호를 밀어낸다 — PLAN-008이 겪은 것과 같은 사고다. TASK-344는 별도로
`cardstatus.go`/`archive.go`의 공유 헬퍼(`frontmatterField`)를 고치고, TASK-343은 그 헬퍼가
반환하는 값을 그대로 읽는다(`cardids.go:60`) — 파일은 다르지만 결합은 같은 종류다.

## Parallel lane — join before TASK-354

TASK-371 is P1 and owns `tools/planprogress` plus the persisted `progress` in PLAN-006.
It may proceed in parallel with TASK-344 through TASK-343 because those cards do not
modify that plan. TASK-354 also changes PLAN-006, so it explicitly `depends-on:
[TASK-371]`; TASK-371 must integrate and TASK-354 must rebase before TASK-354 starts.
This aligns the local calculation with the shared task contract; details are in
[[ISSUE-002]].

## Order

| # | 카드 | P | 왜 이 자리인가 |
|---|---|---|---|
| 1 | TASK-344 프론트매터 값 파서 강화 | P3 | **가장 먼저.** `cardstatus.go`의 `frontmatterField`와 `archive.go`의 `hasCanonicalField`가 공유하는 파서를 고친다. TASK-343이 `depends-on: [TASK-344]`로 이미 선언했고, 그 카드 자신의 §Ordering이 "먼저 안 하면 여기서 쓴 테스트가 344 이후 다시 쓰인다"고 적어 뒀다. 보드에서 가장 잘 명세된 카드이기도 하다 — 검증 바인딩 4개가 오늘 전부 정확히 실패하고, 공허한 기준이 0개다 |
| 2 | TASK-350 역전/공허 바인딩 거부 | P2 | **343보다 먼저.** 350은 공허한 verify 바인딩을 거부하는 카드다. 근거는 **잔존 건수가 아니라 발생률**이다 — 2026-09-08~09 이틀 사이에 세 건이 새로 심어졌다: TASK-369의 release-notes 바인딩, TASK-338의 `make doc-check`, TASK-341의 `make test`. **셋 다 지금은 처리됐다**(각각 재작성·삭제·`(regression-guard)` 표기), 그러니 오늘 이 셋을 실행해 재현하려 하지 말 것. 그런데도 카드가 닫히지 않는 이유가 이 자리의 논거다: **세 건 전부 사람이 리뷰에서 붙잡았고 검사기는 아직 없다.** 350이 먼저 들어와야 다음 필링 배치가 같은 형태를 또 넣지 못한다 |
| 3 | TASK-343 파일명 번호 충돌 가드 | P2, `depends-on: [TASK-344]` | done 2026-09-10. 344가 닫히면서 언블록됐고, TASK/ISSUE namespace별 filename-number 중복 가드를 추가했다. |
| 4 | TASK-354 보드를 `ce task validate` 통과시키고 게이트 연결 | P2, `depends-on: [TASK-371]` | 공유 `ce task gate`를 저장소 게이트에 연결한다. TASK-371이 PLAN-006 진행률을 두 validator가 합의하는 값으로 맞춰야 하며, 두 카드가 같은 plan 파일을 병렬로 바꾸지 않는다. **직접 CE 경계 조사**는 2026-09-10에 끝났다(`TASK-329`의 stale verify-binding 경고를 확인). 그러나 board-ready와 gate 연결 완료는 [[ISSUE-001]]의 CE-compatible review-receipt 발급 경로가 없어 외부 차단됐으므로 TASK-354는 todo로 남는다. |
| 5 | TASK-338 링크의 written path 소멸 보고 | P2 | **문서화된 예외로 완료.** TASK-354의 직접 조사 결과로 shared CE validator가 verify 바인딩을 소유한다는 범위는 확인됐지만, ISSUE-001 때문에 그 카드의 gate 완료·통합을 기다릴 수 없었다. 그래서 TASK-338은 markdown link만 다루고 verify-binding을 복제하지 않는 범위로 한정해 진행했다. live docs 25건은 0으로 고쳤고 archive 84건은 기록 보존으로 남겼다. |

## 순서가 하드 제약인 곳 — 두 곳

나머지 자리는 전부 권고다. 아래 두 관계는 frontmatter로 명시한다.

- **1 → 3** (TASK-344 → TASK-343). TASK-343 §Ordering: "the tests written here would be
  rewritten once TASK-344 changes what the helper returns." `ce task preflight`는
  `unmet-dependency`를 advisory가 아니라 **blocking**으로 센다 — 344가 닫히기 전에는 343이
  runnable 집합에서 아예 빠진다.

2번(350)이 3번(343)보다 앞서는 것은 하드 제약이 아니라 **긴급도**다: 344와 350 사이에는
아무 결합이 없으므로 350을 1번 자리에 넣어도 깨지지 않는다. 다만 350이 막으려는 결함이
바로 지금(TASK-338·341) 재생산되고 있으므로, 344 직후로 당겨 더 심어지는 것을 막는다.

- **TASK-371 → 4** (TASK-371 → TASK-354). TASK-371은 PLAN-006의 persisted progress를
  바꾸고 TASK-354는 그 plan에 `## Children`을 더한다. 둘을 병렬로 통합하면 충돌한다.
  371이 앞서 두 validator의 계산을 정렬한 뒤 354가 rebase한다.

4번(354)과 5번(338)의 결합은 코드 결합이 아니라 **정보 결합**이다 — 354의 직접 CE 경계
조사로 엔진 커버리지를 확인해야 338이 verify-binding을 복제하지 않는다. 이 조사는 완료됐지만
354의 gate 완료는 [[ISSUE-001]]에 외부 차단됐다. 따라서 338 진행은 ordered lane의 유일한 예외이며,
그 범위·근거·잔여 blocker를 이 문서와 TASK-338에 기록한다.

## Children

- TASK-371 — planprogress를 shared task progress contract와 맞춤 (P1, 두 보드 게이트가 같은 진행률을 읽게 먼저 고정)
- TASK-344 — 프론트매터 값 파서를 트레일링 주석·중복 키에 강화 (P3, 공유 파서를 먼저 고정)
- TASK-350 — doccheck의 역전/공허 verify 바인딩 거부 (P2, done 2026-09-10)
- TASK-343 — 파일명 번호 충돌 가드 완성 (P2, done 2026-09-10)
- TASK-354 — 보드를 `ce task validate` 통과시키고 공유 게이트에 연결 (P2, 직접 CE 경계 조사는 완료했으나 [[ISSUE-001]]에 외부 차단, todo 유지)
- TASK-338 — 링크의 written path 소멸을 보고 (P2, done 2026-09-10; live docs 25건을 0으로 고치고 archive 84건은 기록 보존)

## Rules

- 아래 ordered lane은 한 번에 한 장이다. 앞 장이 master에 통합된 뒤 다음 장을 연다. 단, TASK-338은
  TASK-354의 **직접 CE 경계 조사**가 끝난 뒤 ISSUE-001의 외부 차단 때문에 gate 완료를 기다리지 못한
  문서화된 예외다. 이 예외는 TASK-354 완료나 gate 연결을 주장하지 않으며, 338의 markdown-link 범위에만 적용한다.
- TASK-371은 TASK-344~343과 병렬로 진행할 수 있지만, PLAN-006을 함께 바꾸는 TASK-354보다
  먼저 통합하고 그 작업은 rebase한다.
- **TASK-350과 TASK-338을 병렬로 열지 말 것.** 둘 다 `tools/doccheck/check.go`의 `Result`
  구조체와 `tools/doccheck/main.go`의 출력 블록에 카운터를 추가한다 — 같은 자리다.
- 카드 상태는 `tasks/todo` → `tasks/done`이 정본이고, 이 문서는 순서·의존·기각 기록만 갖는다.
- **frontmatter에 `completed-children:`를 넣지 않는다.** PLAN-006·008은 갖고 있고
  PLAN-007·009는 없어 형태가 갈리지만, 맞춰야 할 쪽은 없는 쪽이다 — 이 필드를 읽는 코드가
  `tools/`와 `internal/` 어디에도 없다(2026-09-09 실측). `planprogress`가 세는 것은
  `total-tasks`/`completed-tasks`와 `## Children`이다. 형태를 맞추려고 죽은 필드를
  되살리지 말 것.
