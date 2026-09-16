---
id: TASK-371
title: "Align planprogress with the shared task progress contract"
type: bug
priority: P1
effort: S
exec-tier: standard
status: done
created: 2026-09-10
source: "ISSUE-002 — PLAN-006 16/26 is accepted as 61 by ce task validate and 62 by planprogress"
needs-human: false
blocks: [TASK-354]
quality-review: conditional
quality-reviewed-at: 2026-09-14
quality-review-receipt: tasks/done/evidence/TASK-371/done-review-1b0bc79fe4489d6918cfbe63ade7ff42c9c050e59a60fe332e70dbd0fa006df9.json
quality-review-evidence: "독립 리뷰 review-371(Claude Opus 5, 저자 아님, 2026-09-10 작업 미참여). 완료 기준 셋을 bare로 재실행해 go test ./tools/planprogress 0, make doc-check 0, ce task validate --all 1(이 카드 자신의 receipt 부재 하나뿐). 절삭 계약은 tools/planprogress/check.go의 truncatedPercent를 소스에서 읽고 check_test.go의 TestTruncatedPercent 표로 확인했다 — 15/16=93, 16/26=61, 7/19=36, 2/3=66은 전부 반올림과 갈리는 값이고 0/0=0이 0 나눗셈을 막는다. conditional인 이유는 코드가 아니라 기록이다: 카드의 verification-evidence가 근거 없는 소급 독립 리뷰를 주장했고, 이 커밋이 그것을 철회한다"
verification-evidence: "2026-09-10: go test ./tools/planprogress, make doc-check, and ce task validate --all passed. 2026-09-14 정정 — 같은 줄이 주장하던 'independent review confirmed shared truncation contract'는 철회한다. 뒷받침하는 산출물이 없고, 카드를 연 커밋(36594bb2 12:33:49)과 닫은 커밋(5fc87d79 12:49:49)이 16분 간격 동일 저자이며 그 문장 자체가 구현자의 닫는 커밋 안에서 추가됐다. 실재하는 독립 리뷰는 2026-09-14의 것이 최초이고 아래 receipt가 그것이다."
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

- [x] 16/26 같은 비정수 비율에서 planprogress가 공유 validator와 같은 절삭 값을 요구한다 | verify: `go test ./tools/planprogress`
- [x] PLAN-006과 모든 현재 plan이 두 검사기에서 같은 진행률로 유효하다 — 2026-09-14 재측정에서 exit 1의 사유는 진행률 불일치가 아니라 이 카드 자신의 receipt 부재 하나뿐이었고, 그것을 이 커밋이 닫는다 | verify: `ce task validate --all`
- [x] 저장소 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard)

## 2026-09-14 독립 리뷰 — 이것이 최초의 독립 리뷰다

[[TASK-384]]가 이 카드를 재검토했다. 판정은 `conditional`이고, 조건은 코드가 아니라
**기록**에 걸렸다.

- **High — 소급 독립 리뷰 주장에 근거가 없다.** `verification-evidence`가 "independent
  review confirmed shared truncation contract"라고 적었으나 뒷받침이 하나도 없다.
  `tasks/receipts/TASK-371/`이 존재하지 않고, 카드를 연 커밋 `36594bb2`(12:33:49)와 닫은
  커밋 `5fc87d79`(12:49:49)는 16분 간격 동일 저자이며, 그 문장은 구현자 자신의 닫는
  커밋에서 추가됐다. AGENTS.md가 독립 리뷰 요구를 얻은 것은 같은 날 17:50이다. 문장을
  그대로 둔 채 receipt를 붙이면 **있지도 않았던 리뷰를 새 receipt가 추인하는 모양**이
  되므로, 붙이기 전에 철회한다.
- **코드 쪽은 확인됐다.** `tools/planprogress/check.go`의 `truncatedPercent`가 `total == 0`을
  먼저 걸러낸 뒤 `completed * 100 / total`로 절삭한다. 리뷰어는 6/7을 임시 주입해
  확인했지만, 더 오래 가는 근거는 이미 저장소 안에 있다 — `check_test.go`의
  `TestTruncatedPercent` 표가 15/16=93, 16/26=61, 7/19=36, 2/3=66을 고정하는데 **넷 다
  반올림과 값이 갈리는 지점**이고, 0/0=0이 0 나눗셈을 막는다. 임시 주입은 실행이
  끝나면 사라지지만 이 표는 `go test`가 매번 다시 검사한다. 저장소 안에서 백분율을
  반올림하는 다른 코드는 없다.

기록해 둘 구분이 하나 있다. 이 카드의 `reviewed-card-sha256`은 **CE canonical digest**다 —
이 카드가 `blocks: [TASK-354]`를 선언하므로 `ce task validate`가 실제로 그 값을 비교한다.
`blocks:` 없는 카드에 붙는 plain sha256 pin([[ISSUE-012]])과 다르며, 그래서 이 pin은
검사에 닿는다.

## Notes

- 이 카드는 공유 엔진의 계산 규칙을 변경하지 않는다. 그 엔진은 여러 저장소의 task
  메타데이터 계약을 소유하며 이미 정수 나눗셈을 사용한다.
- TASK-354의 공유 게이트 연결은 이 카드가 닫힌 뒤에만 두 게이트가 동시에 green인지
  주장할 수 있다.
