---
id: TASK-367
title: "Dispose the done backlog that refilled after PLAN-007 triage"
type: chore
priority: P2
effort: M
exec-tier: standard
status: done
needs-human: false
quality-review: pass
quality-reviewed-at: 2026-09-10
quality-review-evidence: "Independent /root/review_task367_plan007 review passed after checking all 46 rows, nine PLAN-007 verdicts, and TASK-373 scope."
created: 2026-09-09
source: "PLAN-007 마무리 점검(2026-09-09) — 목표 1이 이미 다시 거짓임을 실측"
---

## Summary

2026-09-10 재측정에서 `tasks/done/`은 **46장**이고, PLAN-007 자식 8장의 독립 review를
기록한 뒤 `quality-review`를 가진 카드는 9장이다. 나머지 **37장**은 soft-done이다.
PLAN-007이 58장을 비운 지 이틀 만에 측정한 19장은 더 이상 현재 재고가 아니다.

PLAN-007 §Goal 1은 "58장이 처리되어 `tasks/done/`에 Tier C 9장만 남는다"였다. 지금
2026-09-09 당시의 19장 중 Tier C는 5장(308·310·312·313·317)뿐이고 나머지 14장은 그 뒤에
들어온 것이었다.
**PLAN-007이 진단한 것은 재고가 아니라 유량이었고**(§Goal: "유입이 배출보다 빠르다"),
재고만 비웠으므로 유량은 그대로 다시 채웠다.

`ce task archive`는 `quality-review: pass|conditional|waived` 없는 work 카드를 거부한다.
이 재측정 전에는 46장 중 TASK-368만 review verdict가 있었다. 이제 PLAN-007 자식 9장은
verdict를 갖고, 나머지 37장의 review/disposition은 각 소유자에게 남아 있다.

## 무엇이 들어와 있는가 — 다섯 묶음, 소유자가 다르다

| 묶음 | 카드 | 소유자 / 처분 |
|---|---|---|
| PLAN-006 활성 자식 | 308, 310, 312, 313, 314, 315, 317, 318, 320, 322, 323, 324, 339, 340, 345, 346, 347, 351 | PLAN-006. 그 계획이 닫는 시점에 리뷰한다 |
| PLAN-009 활성 자식 | 338, 343, 344, 350, 371 | PLAN-009. TASK-354는 ISSUE-001 receipt 계약에 막혀 있으므로 이 카드가 가로채지 않는다 |
| **PLAN-007 자신의 자식** | 325, 327, 332, 333, 334, 335, 337, 342, 368 | 이 카드가 review 기록을 소유한다. 368은 이미 독립 review PASS |
| 완료 PLAN-008 자식 | 358, 359, 360, 361, 362, 363, 364, 365 | PLAN-008. 8/8 canonical parent가 review/disposition을 소유한다 |
| 독립 soft-done 카드 | ISSUE-002, 341, 352, 355, 369, 372 | TASK-373. 어느 계획도 소유하지 않는다 |

셋째 묶음이 이 카드의 핵심이다. PLAN-007 §Rules는 "리뷰가 관찰만 남기면 손실"이라며
결함을 카드로 만들라고 요구했고 실제로 그 여섯 장이 그렇게 태어났다. 그런데 **그 카드들이
닫힌 뒤 무엇을 하는지는 아무 데도 적혀 있지 않다.** 부모가 자식의 처분을 규정하지 않은
것이 이 backlog의 직접 원인이다.

## `waived`가 생겼다 — PLAN-007 §External 2번은 상류에서 해결됐다

PLAN-007 §External은 두 항목을 "카드 없음, 만들지 않는다"로 남겼다. 2026-09-09 재확인:

**2번(`waived` 값 추가)은 해결됐다.** `ce task archive --help`가 지금
`quality-review: pass|conditional|waived` 를 받는다고 명시하고, `waived`는
`quality-review-evidence`를 함께 요구한다. PLAN-007은 값이 없던 시절 Tier B 36장을
`conditional` + evidence 문구로 우회했다.

**그 우회의 흔적이 남아 있다.** `tasks/_archive/done/`의 `quality-review: conditional`
**48장** 중 상당수는 리뷰된 카드가 아니라 우회된 waiver다. 지금은 정직한 값이 존재하므로
그 48장은 **읽는 사람을 오도한다** — `conditional`은 "조건부로 통과했다"로 읽히지
"리뷰하지 않기로 했다"로 읽히지 않는다.

**이 카드는 그 48장을 고치지 않는다.** 아카이브된 카드를 소급 수정하는 것은 이 카드의
범위 밖이고, evidence 문구(`waived: PLAN-007 backlog triage, no independent review`)가
각 카드에 남아 있어 정보 자체는 손실되지 않았다. 기록만 남긴다 — 누가 그 48장을 세다가
"48장이 리뷰됐다"고 결론 내리는 것을 막기 위해서다.

**1번(닫기 시점 강제)은 여전히 열려 있고 여전히 엔진 소유다.** 이 backlog가 그 항목이
비어 있는 동안 무슨 일이 일어나는지에 대한 증거다. DVA 쪽에서 할 수 있는 유일한 절반은
문서화이고 TASK-368이 그것을 맡는다.

## 작업

1. 46장을 위 표대로 다섯 묶음으로 나누고 각 행의 소유자를 `## 처분 기록`에 적는다.
2. PLAN-006 소유 18장과 PLAN-009 소유 5장: **여기서 처분하지 않는다.** 각 부모가 닫는
   시점에 리뷰한다는 기존 결정을 그대로 두고, 그 사실만 기록한다. PLAN-007이 Tier C를
   범위 밖으로 둔 것과 같은 근거다 — 활성 소유자가 있는 카드를 다른 계획이 가로채면
   리뷰가 두 번 나거나 아무도 안 한다.
3. PLAN-007 자식 9장: 각각 `pass` / `conditional` / `waived` 중 하나를 정한다.
   `waived`를 고르면 `quality-review-evidence`에 **왜 리뷰하지 않는지**를 적는다 —
   "backlog triage" 같은 일괄 문구가 아니라 그 카드에 대한 이유여야 한다.
4. `pass`/`conditional`로 정한 카드는 실제로 리뷰한다. 리뷰어는 그 카드를 구현한 세션과
   달라야 한다(PLAN-007 §Tier A 규칙).
5. PLAN-008의 8/8 child는 PLAN-008이 review/disposition을 소유한다. 독립 soft-done 카드
   6장만 TASK-373에 등록한다. 이 카드가 다른 부모의 처분을 흡수해 scope를 넓히지 않는다.
6. 처분이 끝난 카드는 controller가 가능해졌을 때 `ce task archive`로 옮긴다.

## 처분 기록

| 카드 | 묶음 | 소유자 또는 quality-review | 근거 |
|---|---|---|---|
| 308 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 310 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 312 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 313 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 314 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 315 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 317 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 318 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 320 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 322 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 323 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 324 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 339 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 340 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 345 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 346 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 347 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 351 | PLAN-006 | 보류 | 활성 PLAN-006 child |
| 338 | PLAN-009 | 보류 | 활성 PLAN-009 child |
| 343 | PLAN-009 | 보류 | 활성 PLAN-009 child |
| 344 | PLAN-009 | 보류 | ISSUE-001 receipt 계약이 TASK-354를 막음 |
| 350 | PLAN-009 | 보류 | 활성 PLAN-009 child |
| 371 | PLAN-009 | 보류 | ISSUE-001 receipt 계약이 TASK-354를 막음 |
| 325 | PLAN-007 | pass | live flowcheck snapshot and doc-check reviewed |
| 327 | PLAN-007 | pass | supersession wording and tempName contract reviewed |
| 332 | PLAN-007 | pass | dead constant absent; rejection contract remains |
| 333 | PLAN-007 | conditional | interaction identity passes; TASK-366 retains plans/provision scope |
| 334 | PLAN-007 | pass | accepted fake-driven ceiling and exclusion reviewed |
| 335 | PLAN-007 | pass | atomic create-only placement and mutation evidence reviewed |
| 337 | PLAN-007 | conditional | current surface passes; TASK-353 retains AST-shape gap |
| 342 | PLAN-007 | pass | reserved name and parent route rejections reviewed |
| 368 | PLAN-007 | pass | `/root/review_task368` independent review evidence |
| 358 | PLAN-008 | 보류 | 8/8 canonical PLAN-008 child |
| 359 | PLAN-008 | 보류 | 8/8 canonical PLAN-008 child |
| 360 | PLAN-008 | 보류 | 8/8 canonical PLAN-008 child |
| 361 | PLAN-008 | 보류 | 8/8 canonical PLAN-008 child |
| 362 | PLAN-008 | 보류 | 8/8 canonical PLAN-008 child |
| 363 | PLAN-008 | 보류 | 8/8 canonical PLAN-008 child |
| 364 | PLAN-008 | 보류 | 8/8 canonical PLAN-008 child |
| 365 | PLAN-008 | 보류 | 8/8 canonical PLAN-008 child |
| ISSUE-002 | independent | TASK-373 | 활성 부모 없음 |
| 341 | independent | TASK-373 | 활성 부모 없음 |
| 352 | independent | TASK-373 | 활성 부모 없음 |
| 355 | independent | TASK-373 | 활성 부모 없음 |
| 369 | independent | TASK-373 | 활성 부모 없음 |
| 372 | independent | TASK-373 | 활성 부모 없음 |

## Completion Criteria

- [x] PLAN-007 자식 9장이 `quality-review` 값을 갖거나 tasks/done을 떠났다 | verify: `! /usr/bin/find tasks/done -maxdepth 1 \( -name '325-*.md' -o -name '327-*.md' -o -name '332-*.md' -o -name '333-*.md' -o -name '334-*.md' -o -name '335-*.md' -o -name '337-*.md' -o -name '342-*.md' -o -name '368-*.md' \) ! -exec /usr/bin/grep -q '^quality-review:' {} \; -print | /usr/bin/grep -q .`
- [x] `waived`로 표기한 카드는 `quality-review-evidence`를 함께 갖는다 | verify: `! /usr/bin/find tasks/done -maxdepth 1 -name '*.md' -exec /usr/bin/grep -q '^quality-review: waived$' {} \; ! -exec /usr/bin/grep -q '^quality-review-evidence:' {} \; -print | /usr/bin/grep -q .`
- [x] 46장 각각에 소유자와 처분 근거가 기록됐다 | verify: human — 이 카드의 `## 처분 기록` 표에 46장 전부가 소유 plan 또는 처분 사유와 함께 한 행씩 있는지 확인
- [x] 게이트 통과 | verify: `make doc-check` (regression-guard)

## Notes

- **첫 수용기준을 `tasks/done` 전체가 아니라 PLAN-007 자식 9장으로 좁힌 것은 의도적이다.**
  전체로 쓰면 기준이 참이 되는 시점이 PLAN-006과 PLAN-009이 닫는 때이고, 이 카드는 그 카드에
  값을 적을 권한이 없다. 그런 기준은 카드를 무기한 세워 두면서 그 사실을 숨긴다 — 바인딩만
  보면 이 카드가 게으른 것처럼 보이지 남의 일을 기다리는 줄은 안 보인다.
- 좁힌 대가는 나머지 37장을 감시하는 **기계 기준이 없다**는 것이고, 그것을 세 번째
  기준(`## 처분 기록` 46행 전수)이 사람 검사로 받는다. 활성 plan 자식은 값이 아니라
  **소유자 이름**을 갖는 것이 옳은 상태이며, 활성 부모 없는 6장은 TASK-373이 처리한다.
- 이 카드가 세는 46이라는 수는 착수 시점에 다시 세야 한다. 유량이 문제인 대기열이므로
  그 사이에 늘어난다.

## Completion record

- 2026-09-10: `/root/review_task367_plan007`가 PLAN-007 자식 8장을 독립 review하고,
  카드 현행화·TASK-373 분리도 재검토해 PASS했다. `make doc-check`와 개별 카드 검증이
  통과했다.
