---
id: TASK-381
title: "Check plan prose against plan frontmatter in planprogress"
type: feature
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-13
quality-review: conditional
quality-reviewed-at: 2026-09-14
quality-review-evidence: "review-381(독립 리뷰어, 작업 미참여)이 네 기준을 전부 재실행하고 결함 세 형태를 독립적으로 재파종해 셋 다 rc=1로 발화함을 확인했다. 판정은 conditional — 게이트는 공허하지 않으나 범용 수량사(개·건)를 카드 계수로 읽는 오탐 클래스가 남아 있고, 그 형태의 산문이 PLAN-007·009에 이미 존재한다. 미해결 발견은 ISSUE-016·017·018로 분리했다."
quality-review-receipt: tasks/done/evidence/TASK-381/done-review-747206df0e6d69d0a11a3c2e3c17ecd7451aad7f4bb1c772c0267596fd6fe45a.json
reviewed-card-sha256-algorithm: "plain-file-sha256 of the reviewed revision (27c4dfe). TASK-381 declares no blocks:, so ce task validate never reaches the receipt check and no CE canonical digest is obtainable for this card."
source: "PLAN-009의 산문/frontmatter 불일치(TASK-380 D-2)를 사람이 눈으로 발견. 같은 형태를 보는 기계가 없다"
depends-on: []
---

## Summary

`tools/planprogress`는 plan 카드의 `total-tasks` / `completed-tasks` / `progress`를
`## Children` 존과 대조한다. **산문은 보지 않는다.** 그래서 PLAN-009처럼 `scope:`와
`## Goal`이 "여섯 장"이라 쓰고 여섯 ID를 나열하는데 `children:`은 일곱 장인 상태가
어떤 게이트에도 걸리지 않았다.

일회성 오타가 아니라 재발하는 형태다: plan에 자식을 더할 때 사람은 `children:`과
`total-tasks`를 고치고 산문은 그대로 둔다. 두 곳이 같은 사실을 서로 다른 표현으로
말하고 있고, 그중 한쪽만 검사된다.

## 제안하는 규칙 두 개

1. `scope:` 및 `## Goal` 본문에 나타나는 `TASK-NNN` 집합이 `children:`의 부분집합이어야
   한다. (초과분만 오류로 본다 — 산문이 자식 전부를 열거할 의무는 없다.)
2. 산문에 한글 수사(`N장` / `N개`)가 나타나고 그 문장이 자식을 세는 문장이면
   `total-tasks`와 일치해야 한다.

2번은 오탐 위험이 있다. 자식을 세지 않는 "여섯 줄" 같은 표현과 구별해야 하므로,
`scope:` 필드와 `## Goal`의 첫 문단으로 검사 범위를 좁히는 편을 권한다.

## 2026-09-14 두 번째 실측 사례 — 제안한 규칙 둘 다 잡지 못한다

[[TASK-385]]가 PLAN-006 §Session handoff에서 같은 계열의 결함을 찾았다. 문장은
이랬다.

> 318·323까지 닫혀 §Order의 기계적 항목은 전부 소진됐고 … **남은 것은 10a
> 실기동(TASK-328) 하나뿐**이고, 사람이 실제로 돌려야 닫힐 카드다.

같은 문서가 아래에서 TASK-348과 TASK-329를 열린 것으로 적고, frontmatter는
26/29(잔여 3)다. 문장은 §Order 범위에서는 참이지만 자기 범위를 말하지 않아 계획 전체의
잔여로 읽힌다. TASK-385가 범위를 명시하는 것으로 고쳤다.

**이 사례는 위 §제안하는 규칙 두 개 중 어느 것으로도 잡히지 않는다.**

- 1번은 산문의 `TASK-NNN` 집합이 `children:`을 **초과**하는 경우만 본다. 여기서는
  집합이 부분집합이다 — 초과가 아니라 누락이고, 누락은 정상적인 서술이므로 규칙 1번은
  구조적으로 이것을 볼 수 없다.
- 2번은 한글 수사(`N장`/`N개`)를 센다. 이 문장은 수사를 쓰지 않고 **"하나뿐"**이라고
  쓴다. 사전에 `하나`를 추가해도 오탐이 폭증한다 — "방법은 하나뿐이다" 같은 문장이
  plan 산문에 흔하다.

그래서 이 카드를 열 때 **실제로 필요한 규칙은 세 번째**일 가능성이 높다: 잔여를
단언하는 문장이 자기가 세는 집합(§Order / §Needs-human / 계획 전체)을 명시하는가.
이것은 정규식으로 판정할 수 있는 형태가 아니다 — 기계화 범위 자체를 다시 봐야 한다는
뜻이고, 이 카드의 P3 유지 근거를 약화시키는 것이 아니라 **범위 재검토가 착수 전 첫
작업임을 알리는 신호**로 기록한다.

## 순서 — P3이고 지금 열지 않는다

이 저장소에는 backlog 존이 없다(`ce task move`가 아는 존은 todo·doing·review·blocked·done
다섯이고 `ce-tasks.yaml` dialect 선언도 없다). 그래서 카드는 `todo`에 있지만 우선순위
P3으로 큐의 맨 뒤에 둔다. 뒤로 미루는 근거는 셋이다.

- `tools/planprogress`의 계산은 [[TASK-371]]이 shared task progress contract와 맞춰
  놓은 것이고, 그 계약 영역은 [[TASK-354]]가 아직 열어 두고 있다. 지금 손대면 두 카드가
  같은 자리를 건드린다.
- 현재 plan은 네 장뿐이라 재발률이 낮다. [[TASK-380]]이 유일한 실측 사례를 닫는다.
- 이 카드는 `blocks:` 간선을 만들지 않으므로 [[ISSUE-001]]과 무관하게 언제든 열 수 있다.
  서두를 이유가 없다.

## 2026-09-14 범위 재검토 결과 — 착수 시 첫 작업으로 수행

위 §제안하는 규칙 두 개를 **그대로 구현하면 현재 plan 네 장이 깨진다**(수용기준 3 위반).
착수 전 네 장을 전수 대조한 결과다.

| 위치 | 텍스트 | 제안 규칙대로면 |
|---|---|---|
| PLAN-008 `scope:` | `TASK-358..365 — TASK-318 재리뷰가…` | 규칙 1이 TASK-318을 `children:` 초과로 신고 — 부모 원인 언급이지 자식 주장이 아니다 |
| PLAN-007 `## Goal` | `done 카드 58장 전부` | 규칙 2가 58 ≠ 13 신고 |
| PLAN-006 `## Goal` | `23개 devbox 저장소` | 규칙 2가 23 ≠ 29 신고 |
| PLAN-009 `## Goal` | `처음 여섯 장이`, `다섯 장(…)` | 규칙 2가 6·5 ≠ 7 신고 (부분집합을 세는 정상 서술) |

**판별자는 "산문이 세는가"가 아니라 "산문이 열거하는가"다.** 고립된 `TASK-318` 언급은
맥락이고, `TASK-371, 344, 350, …` 같은 구분자 런이나 `TASK-358..365` 범위는 멤버십
주장이다. 그리고 `scope:`는 정의상 계획 전체를 서술하므로 거기서 열거에 계수가 붙으면
소진 주장이지만, `## Goal`은 부분집합을 세는 것이 정상이라 같은 잣대를 쓸 수 없다.

구현한 규칙 (`tools/planprogress/prose.go`):

1. **멤버십** — `scope:`/`## Goal`의 *열거*(범위, 또는 `,`·`·`로 이어진 2개 이상의 런.
   연속 항목은 `TASK-` 접두사를 생략할 수 있다 — 라이브 카드가 그렇게 쓴다)에 나타난 ID는
   전부 `children:`에 있어야 한다. 고립된 단일 언급은 검사하지 않는다.
2. **국소 계수** — 열거 옆 계수구(`8장` / `일곱 장`)는 그 열거의 크기와 같아야 한다.
   (`scope:`·`## Goal` 양쪽)
3. **소진 계수** — `scope:`에서만, 그 계수는 `total-tasks`와도 같아야 한다.

`## Goal`의 §남은 세 번째 규칙(잔여 단언 문장이 자기 범위를 명시하는가, TASK-385 사례)은
정규식으로 판정 불가라는 카드 본문의 결론을 유지하고 **구현하지 않았다**. 기계화하지 않은
이유를 `prose.go` 파일 주석에 남겼다 — 조용한 공백 하나를 시끄러운 오탐으로 바꾸는 거래이기
때문이다.

### 게이트가 실제로 작동하는지 확인 (공허한 통과 배제)

라이브 보드는 `planprogress: OK`다. 초록이 규칙이 **작동한다**는 뜻은 아니므로 보드 사본에
위반을 심어 확인했다.

```
# TASK-380 D-2 원본 결함을 PLAN-009 scope에 재현 (ID 6개 + "여섯 장", children 7장)
ERROR: PLAN-009 (…/009-….md): scope: enumerates and counts 6 card(s), but total-tasks=7
       ("TASK-371, 344, 350, 343, 354, 338")
planprogress: FAIL  (rc=1)

# PLAN-008 scope 범위에 비자식 ID를 붙임
ERROR: PLAN-008 (…/008-….md): scope: enumerates TASK-999, which children: does not list
       ("TASK-358..365, 999")
ERROR: PLAN-008 (…/008-….md): scope: counts 8 card(s) beside an enumeration of 9
       ("TASK-358..365, 999")
planprogress: FAIL  (rc=1)
```

두 번째 프로브가 실제 결함을 하나 잡았다: 범위 분기가 뒤따르는 구분자 런을 소비하지 않아
`TASK-358..365, 999`의 `999`가 통째로 무시됐다. 수정하고 회귀 테스트로 고정했다
(`TestFindEnumerations/range_continued_by_a_run`).

## 독립 리뷰 (2026-09-14, `review-381`)

작업에 참여하지 않은 리뷰어가 판정했다. **판정: `conditional`, final.**
영수증: `tasks/receipts/TASK-381/done-review-d2dfbf1c….json`.

### 기준 네 개 — 전부 PASS, 단 바인딩이 서로를 구별하지 못한다

리뷰어가 네 기준을 모두 재실행했고 넷 다 rc=0이다. 그러나 기준 1·2·4가 **같은 명령
하나**(`go test ./tools/planprogress/`)에 묶여 있어 서로를 구별하지 못한다 — 기준 2의
fixture만 지워도 기준 2는 여전히 통과한다. 리뷰어가 테스트 본문을 직접 읽고 각 규칙을
보드 사본에 따로 재파종해 그 틈을 닫았다. 기준 3의 `make doc-check`가 넷 중 가장 강한
바인딩이다. 같은 계열의 지적이 TASK-344 리뷰(finding 4)에서도 나왔으므로 이것은 이
카드 한 장의 실수가 아니라 카드 작성 관행의 문제다.

### 게이트가 공허하지 않다는 것은 제3자 측정으로 섰다

리뷰어가 직접 파종한 네 형태가 전부 발화했다 — 저자의 프로브를 다시 돌린 것이 아니라
독립적으로 만든 것이다:

| 파종 | 관측 |
|---|---|
| PLAN-009 scope를 `여섯 장` + 6 id로, children 7 | `scope: enumerates and counts 6 card(s), but total-tasks=7` rc=1 |
| PLAN-008 scope를 `TASK-358..365, 999 — …8장` | 비자식 id + 계수 불일치 2건 rc=1 |
| PLAN-009 Goal을 `네 장(…5개 id)` | `## Goal counts 4 card(s) beside an enumeration of 5` rc=1 |
| PLAN-008 범위를 `TASK-358..364`로 좁히고 `8장` 유지 | `counts 8 card(s) beside an enumeration of 7` rc=1 |

저자가 자체 보고한 `TASK-358..360, 999` 결함도 실제로 고쳐졌고 회귀 테스트로 고정된
것이 독립 확인됐다.

### conditional의 근거 — 미해결 발견 여섯

셋을 카드로 분리했다:

- [[ISSUE-016]] (medium) — `개`·`건`·`장`은 한국어 **범용** 수량사인데 같은 문장의 id
  열거와 무조건 짝지어진다. `필드 32개`, `관련 문서 8건`이 올바른 열거에 대해 오탐을
  낸다. **PLAN-009 본문에 `필드 32개`가, PLAN-007 scope에 `결함 3건`이 이미 있다** —
  오늘 통과하는 것은 규칙이 옳아서가 아니라 숫자와 열거가 우연히 같은 문장에 없기
  때문이다. `prose.go`의 파일 주석은 이 클래스를 피했다고 주장하지만 피하지 못했다.
  같은 카드에 계수된 부분 열거 거부(PLAN-006의 실제 scope 형태)와 한 문장 안 두 열거의
  오탐도 묶었다.
- [[ISSUE-017]] (low) — 역방향 범위가 뒤따르는 런을 통째로 버리고(고쳐진 결함의 반대
  방향), id가 중복 제거되지 않아 `TASK-1, 1, 2 — 세 장`이 3자식 plan을 통과한다.
  `TestFindEnumerations/reversed_range_is_ignored`가 현재 동작을 **정답으로 고정**하고
  있어 수정하려면 그 테스트의 기대값부터 바꿔야 한다. 셋째 항목이 제일 아프다 —
  `[[TASK-N]]` 위키링크 형태는 `runTailRE`가 id 바로 뒤의 `,`·`·`를 요구하는데 `]]`가
  끼어들어 **열거 자체가 보이지 않는다**(`[[TASK-1]], [[TASK-9]] 두 장` → 결함 0건,
  TASK-9가 비자식인데도). 그런데 그 형태가 **이 보드의 상호참조 표기**다.
  실측(2026-09-14): `[[TASK-N]]`은 이 카드 본문에 7건, PLAN-007에 2건 있고,
  PLAN-006에는 위키링크가 아예 없으며 PLAN-008·009는 `[[ISSUE-N]]`·`[[PLAN-N]]`만 쓴다.
  **결정적으로, 검사가 실제로 읽는 두 구역(`scope:`·`## Goal`)에는 오늘 위키링크가
  0건이다** — 즉 노출은 현재가 아니라 잠재다. 그러나 본문에서 그렇게 쓰는 저자가
  scope에도 그렇게 쓰는 것은 자연스럽고, 그때 검사는 아무 말 없이 꺼진다.
- [[ISSUE-018]] (low, 그러나 영향은 severity보다 크다) — 파싱 층의 조용한 스킵 셋: `scope: >` 접힌 블록은 리터럴
  `">"`로 파싱돼 scope 규칙 둘이 다 꺼지고, `## Goal (2026)`처럼 접미사가 붙은 제목은
  빈 Goal이 되며, h1은 Goal 절을 끝내지 않는다(`extractSection`의 자체 doc comment와
  모순). 진단 없이 검사가 꺼지는 쪽이 실패하는 쪽보다 나쁘다 — 카드는 여전히 green이다.

카드로 만들지 않고 여기 남기는 둘:

- (informational) 한국어 수량사만 인식하므로 영어로 쓰인 plan 카드는 규칙 2·3을
  공짜로 통과한다. PLAN-006 scope는 이미 부분적으로 영어다.
- (informational) 범위 확장 오류 메시지가 한 줄에 최대 199개 id를 나열할 수 있다.
  `maxRangeSpan`은 멤버십 주장을 제한하지 취지 메시지를 제한하지 않는다.

### 리뷰어가 확인하지 않은 것

`make build`, `make test-integration`, CI 워크플로가 실제로 `make doc-check`를 부르는지,
`tasks/_archive/plan/`의 보관 plan(`loadPlans`가 읽지 않는다), 퍼징·성능, 그리고 이 카드
자체의 lifecycle(존·frontmatter). 마지막 항목은 이 절과 영수증이 대신 채운다.


## Completion Criteria

- [x] `scope:`/`## Goal`의 `TASK-NNN` 집합이 `children:`을 초과하면 planprogress가 보고한다 (범위 재검토: *열거*에 한정 — §2026-09-14 참조) | verify: `go test ./tools/planprogress/`
- [x] 자식 수를 세는 산문 수사가 `total-tasks`와 어긋나면 보고한다 (`scope:`의 열거 옆 계수에 한정) | verify: `go test ./tools/planprogress/`
- [x] 현재 plan 네 장이 새 규칙 아래 전부 통과한다 | verify: `make doc-check` (regression-guard)
- [x] 규칙을 어기는 fixture와 지키는 fixture가 둘 다 테스트에 있다 | verify: `go test ./tools/planprogress/`
