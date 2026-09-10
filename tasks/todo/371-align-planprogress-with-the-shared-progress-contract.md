---
id: TASK-371
title: "Align planprogress with the shared task progress contract"
type: bug
priority: P1
effort: S
exec-tier: standard
status: todo
created: 2026-09-10
source: "ISSUE-002 — PLAN-006 16/26 is accepted as 61 by ce task validate and 62 by planprogress"
needs-human: false
blocks: [TASK-354]
---

# Task 371: planprogress의 진행률 계산을 공유 task 계약과 맞춘다

## Summary

`ce task validate`는 완료 자식 비율의 백분율을 정수 나눗셈으로 계산한다. 그러나 DVA의
`tools/planprogress`는 반올림을 사용한다. PLAN-006의 16/26은 이 차이 때문에 각각
61과 62로 평가되어 카드 값 하나로 두 게이트를 모두 통과할 수 없다.

공유 task-management runtime이 보드 메타데이터의 정본이므로, DVA의 로컬 검사기를
그 계약에 맞춘다. 상세 재현과 영향은 [[ISSUE-002]]에 기록한다.

## Scope

- `tools/planprogress`의 비율 계산과 테스트
- 현재 모든 plan의 저장된 진행률 재검증
- PLAN-006의 persisted `progress`를 공유 계약 값으로 현행화

## Completion Criteria

- [ ] 16/26 같은 비정수 비율에서 planprogress가 공유 validator와 같은 절삭 값을 요구한다 | verify: `go test ./tools/planprogress`
- [ ] PLAN-006과 모든 현재 plan이 두 검사기에서 같은 진행률로 유효하다 | verify: `ce task validate --all`
- [ ] 저장소 문서 게이트가 통과한다 | verify: `make doc-check`

## Notes

- 이 카드는 공유 엔진의 계산 규칙을 변경하지 않는다. 그 엔진은 여러 저장소의 task
  메타데이터 계약을 소유하며 이미 정수 나눗셈을 사용한다.
- TASK-354의 공유 게이트 연결은 이 카드가 닫힌 뒤에만 두 게이트가 동시에 green인지
  주장할 수 있다.
