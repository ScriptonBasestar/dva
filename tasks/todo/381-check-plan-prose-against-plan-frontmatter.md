---
id: TASK-381
title: "Check plan prose against plan frontmatter in planprogress"
type: feature
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-13
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

## Completion Criteria

- [ ] `scope:`/`## Goal`의 `TASK-NNN` 집합이 `children:`을 초과하면 planprogress가 보고한다 | verify: `go test ./tools/planprogress/`
- [ ] 자식 수를 세는 산문 수사가 `total-tasks`와 어긋나면 보고한다 | verify: `go test ./tools/planprogress/`
- [ ] 현재 plan 네 장이 새 규칙 아래 전부 통과한다 | verify: `make doc-check` (regression-guard)
- [ ] 규칙을 어기는 fixture와 지키는 fixture가 둘 다 테스트에 있다 | verify: `go test ./tools/planprogress/`
