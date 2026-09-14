---
id: TASK-385
title: "Currentize the plan ownership record for the unreviewed done backlog"
type: docs
priority: P2
effort: S
exec-tier: standard
status: doing
created: 2026-09-14
source: "2026-09-14 보드 점검. PLAN-007의 마지막 재측정은 2026-09-10의 46장/9장이고 현재는 68장/29장이다. 그 사이에 닫힌 PLAN-008이 자식 8장을 리뷰되지 않은 채 done에 남겼다"
depends-on: []
completion-summary: "리뷰되지 않은 done 39장의 소유권을 2026-09-14 기준으로 재측정해 PLAN-007에 표로 남겼다. PLAN-008이 진척 100%로 닫히면서 자식 8장을 리뷰 없이 남긴 사실을 그 계획 본문에 기록했고, 소유자 없는 9장과 합쳐 17장을 레거시로 선언했다(소급 리뷰 없음). PLAN-006 §Session handoff의 잔여 단언에 범위를 명시했고, 그 사례를 TASK-381이 제안한 규칙 둘 중 어느 것으로도 잡지 못한다는 분석과 함께 그 카드에 기록했다"
verification-status: verified
verification-evidence: "기계 바인딩 4개 exit 0, make doc-check rc=0. 39/68 수치는 첫 프론트매터 블록만 읽는 awk로 재측정했고 카드 id 39개가 초안과 완전히 일치했다. PLAN-008 자식 8장의 quality-review 부재는 grep -c로 8건 모두 0을 확인했다. §Session handoff 문장은 육안 확인 — 수정 후 문장이 §Order 범위임을 명시하고 frontmatter 26/29와 나머지 두 장(TASK-348, TASK-329)을 함께 적는다"
---

## Summary

`tasks/done/`에 68장이 있고 그중 **39장**에 `quality-review` 키가 없다. [[PLAN-007]]의
마지막 재측정(2026-09-10)은 "46장, `quality-review` 보유 9장"이고 소유권 배분도 그 시점
것이다. 숫자와 배분이 둘 다 낡았다.

이 카드는 소급 리뷰를 하지 않는다. **누가 무엇을 소유하는지만 현행화한다.**

## 측정 (2026-09-14)

`quality-review` 없는 39장의 소유 plan:

| 소유 | 장수 | 카드 |
|---|---|---|
| PLAN-006 (활성, 89%) | 17 | 309, 312, 313, 314, 315, 317, 318, 320, 322, 323, 324, 339, 340, 345, 346, 347, 351 |
| PLAN-008 (**닫힘**, 100%) | 8 | 358, 359, 360, 361, 362, 363, 364, 365 |
| PLAN-009 (활성, 85%) | 5 | 338, 343, 344, 350, 371 |
| PLAN-007 (닫힘, 100%) | 0 | — |
| 소유 plan 없음 | 9 | ISSUE-003, 308, 310, 331, 349, 356, 366, 374, 375 |

39장 중 `parent:` 프론트매터를 가진 카드는 **하나도 없다.** 소유권은 전부 plan 쪽
`children:`에서만 읽힌다 — 카드 한 장만 보고는 자기 소유자를 알 수 없다는 뜻이다.

## 드러난 것 둘

**1. PLAN-008은 100%로 닫혔는데 자식 8장 전부가 리뷰 없이 done에 있다.** PLAN-007이
세운 규칙 — "리뷰 없이 아카이브한 카드는 `quality-review` 값으로 그 사실이 정직하게
남는다" — 이 적용되지 않은 채 계획이 닫혔다. 진척 100%는 **자식이 done이라는 뜻이지
리뷰됐다는 뜻이 아니다.** 이 구분이 어디에도 적혀 있지 않았다.

**2. 소유 plan이 없는 9장은 어느 계획의 종료 조건에도 들어 있지 않다.** PLAN-007이
2026-09-10에 "독립 6장은 TASK-373으로 분리한다"고 한 그 자리가 다시 비었고, 이번에는
9장이다. 필요를 문장으로 적는 것과 카드로 만드는 것이 다르다는 것을 PLAN-007 자신이
이미 기록했다.

## 처리 방침 — 소급 리뷰하지 않는다

39장을 지금 사후 리뷰해도 provenance가 진짜가 아니다. 리뷰어가 변경 시점의 맥락 없이
판정을 쓰는 것은 `waived`를 `pass`로 위장하는 것과 같다. 대신:

- 활성 plan(006·009)의 22장은 **각 부모가 닫는 시점에** 리뷰된다. PLAN-007 §Order 5가
  Tier C에 대해 이미 정한 방식 그대로다.
- PLAN-008의 8장과 소유자 없는 9장은 **레거시로 선언한다.** 리뷰되지 않았다는 사실이
  이 카드와 PLAN-007의 재측정 절에 남는다.
- 앞으로는 `blocks:`를 가진 카드가 validator에 걸린다. [[TASK-383]]이 durable 경로를
  열었으므로 그 걸림은 이제 실제로 해소 가능하다. 즉 **리뷰 부채의 유입은 게이트가
  막고, 재고는 선언으로 닫는다.**

## PLAN-006 §Session handoff의 범위가 모호하다

`tasks/plan/006-devbox-dogfood-followup.md`의 "다음 착수" 항목이 **남은 것은 10a
실기동(TASK-328) 하나뿐**이라고 굵게 적는다. §Order 안에서는 참이지만 같은 문서가
아래에서 TASK-348(잔여 사람 작업)과 TASK-329(§남은 순서 미지정 1장)를 열린 것으로
적는다. frontmatter의 26/29(잔여 3)가 맞고, 굵은 문장은 자기 범위를 말하지 않는다.

범위를 명시하는 것으로 고친다 — 사실을 바꾸는 것이 아니라 어느 집합을 세는지 밝힌다.

이 형태는 [[TASK-381]]이 기계화하려는 것과 같은 계열이지만 **381이 제안한 규칙 둘
중 어느 것도 이것을 잡지 못한다.** 1번은 산문의 `TASK-NNN` 집합이 `children:`을
초과하는 경우만 보고, 2번은 수사(`N장`)를 세는데 이 문장은 "하나뿐"이라고 쓴다.
그 사실을 TASK-381에 기록한다.

## Completion Criteria

- [x] PLAN-007에 2026-09-14 재측정과 소유권 표가 있다 | verify: `/usr/bin/grep -rq --include='007-*.md' '2026-09-14' tasks`
- [x] PLAN-008이 자식 8장을 리뷰 없이 닫았다는 사실이 기록돼 있다 | verify: `/usr/bin/grep -rq --include='008-*.md' 'quality-review' tasks`
- [x] PLAN-006의 굵은 문장이 자기 범위를 밝힌다 | verify: `human — §Session handoff를 읽고 §Order 범위임이 문장 안에서 읽히는지 확인`
- [x] TASK-381이 자기 규칙으로 잡지 못하는 사례를 알고 있다 | verify: `/usr/bin/grep -rq --include='381-*.md' 'PLAN-006' tasks`
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)
