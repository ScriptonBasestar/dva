---
id: TASK-367
title: "Dispose the done backlog that refilled after PLAN-007 triage"
type: chore
priority: P2
effort: M
exec-tier: standard
status: todo
needs-human: true
created: 2026-09-09
source: "PLAN-007 마무리 점검(2026-09-09) — 목표 1이 이미 다시 거짓임을 실측"
---

## Summary

`tasks/done/`에 **19장**이 있고 **`quality-review`를 가진 카드는 0장**이다. PLAN-007이
58장을 비운 지 이틀 만이다.

PLAN-007 §Goal 1은 "58장이 처리되어 `tasks/done/`에 Tier C 9장만 남는다"였다. 지금
19장 중 Tier C는 5장(308·310·312·313·317)뿐이고 나머지 14장은 그 뒤에 들어온 것이다.
**PLAN-007이 진단한 것은 재고가 아니라 유량이었고**(§Goal: "유입이 배출보다 빠르다"),
재고만 비웠으므로 유량은 그대로 다시 채웠다.

`ce task archive`는 `quality-review: pass|conditional|waived` 없는 work 카드를 거부한다.
따라서 **19장 전부가 지금 아카이브 불가**다.

## 무엇이 들어와 있는가 — 세 묶음, 소유자가 다르다

| 묶음 | 카드 | 소유자 |
|---|---|---|
| Tier C 잔여 | 308, 310, 312, 313, 317 | PLAN-006. PLAN-007 §Tier C가 명시적으로 범위 밖으로 뒀다 |
| PLAN-006 후속 | 314, 315, 318, 320, 322, 323, 324 | PLAN-006. 그 계획이 닫는 시점에 리뷰하기로 돼 있다 |
| **PLAN-007 자신의 자식** | 327, 332, 333, 334, 335, 337, 342 | **없음.** 이 카드가 만들려는 것이 그 소유자다 |

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

1. 19장을 위 표대로 세 묶음으로 나누고 각 행의 소유자를 `## 처분 기록`에 적는다.
2. PLAN-006 소유 12장(첫째·둘째 묶음): **여기서 처분하지 않는다.** PLAN-006이 닫는
   시점에 리뷰한다는 기존 결정을 그대로 두고, 그 사실만 기록한다. PLAN-007이 Tier C를
   범위 밖으로 둔 것과 같은 근거다 — 활성 소유자가 있는 카드를 다른 계획이 가로채면
   리뷰가 두 번 나거나 아무도 안 한다.
3. PLAN-007 자식 7장: 각각 `pass` / `conditional` / `waived` 중 하나를 정한다.
   `waived`를 고르면 `quality-review-evidence`에 **왜 리뷰하지 않는지**를 적는다 —
   "backlog triage" 같은 일괄 문구가 아니라 그 카드에 대한 이유여야 한다.
4. `pass`/`conditional`로 정한 카드는 실제로 리뷰한다. 리뷰어는 그 카드를 구현한 세션과
   달라야 한다(PLAN-007 §Tier A 규칙).
5. 처분이 끝난 카드는 `ce task archive`로 옮긴다.

## 처분 기록

<!-- 19장 각각 한 행: 카드 ID | 묶음 | 소유자 또는 quality-review 값 | 근거 -->

## Completion Criteria

- [ ] PLAN-007 자식 7장이 `quality-review` 값을 갖거나 tasks/done을 떠났다 | verify: `! /usr/bin/find tasks/done -maxdepth 1 -name '327-*.md' -o -name '332-*.md' -o -name '333-*.md' -o -name '334-*.md' -o -name '335-*.md' -o -name '337-*.md' -o -name '342-*.md' | /usr/bin/xargs -r /usr/bin/grep -L '^quality-review:' | /usr/bin/grep -q .`
- [ ] `waived`로 표기한 카드는 `quality-review-evidence`를 함께 갖는다 | verify: `! /usr/bin/grep -rl '^quality-review: waived$' tasks/done --include='*.md' | /usr/bin/xargs -r /usr/bin/grep -L '^quality-review-evidence:' | /usr/bin/grep -q .`
- [ ] 19장 각각에 소유자와 처분 근거가 기록됐다 | verify: human — 이 카드의 `## 처분 기록` 표에 19장 전부가 소유 plan 또는 처분 사유와 함께 한 행씩 있는지 확인
- [ ] 게이트 통과 | verify: `make doc-check`

## Notes

- **첫 수용기준을 `tasks/done` 전체가 아니라 PLAN-007 자식 7장으로 좁힌 것은 의도적이다.**
  전체로 쓰면 기준이 참이 되는 시점이 PLAN-006이 12장을 닫는 때이고, 이 카드는 그 12장에
  값을 적을 권한이 없다. 그런 기준은 카드를 무기한 세워 두면서 그 사실을 숨긴다 — 바인딩만
  보면 이 카드가 게으른 것처럼 보이지 남의 일을 기다리는 줄은 안 보인다.
- 좁힌 대가는 나머지 12장을 감시하는 **기계 기준이 없다**는 것이고, 그것을 세 번째
  기준(`## 처분 기록` 19행 전수)이 사람 검사로 받는다. 12장은 값이 아니라 **소유자 이름**을
  갖는 것이 옳은 상태다 — 이 카드가 아니라 PLAN-006이 닫을 때 리뷰된다.
- 이 카드가 세는 19라는 수는 착수 시점에 다시 세야 한다. 유량이 문제인 대기열이므로
  그 사이에 늘어난다.
