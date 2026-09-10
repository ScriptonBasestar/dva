---
id: TASK-362
title: "Make the flow-style fixture cover the blocked root shape it claims"
type: test
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview)"
status: done
depends-on: []
needs-human: false
verification-evidence: "2026-09-10: focused flow-style contract test, ce task validate, make doc-check, and independent review passed."
---

# Task 362: flow-style root fixture가 현재 차단 계약을 검증하게 한다

## Summary

`TestMigrateSectionOrderBailsOnUnrepresentableShapes`의 flow-style root mapping은 줄 범위를
분할할 수 없는 YAML 모양을 다룬다. 배너 주석을 포함한
`"# banner\\n{stack: b, version: a}\\n"`을 써서 comment-extension 경로까지 거친 뒤에도,
현재 계약대로 원본 바이트를 반환하고 `report.Blocked` 사유를 남기며 패닉하지 않는지를
검증한다.

카드가 처음 인용한 "가드를 제거하면 배너 fixture만 패닉한다"는 현상은 과거 revision의
관측이며 현재 source에서는 재현되지 않는다. 가드를 제거한 일회성 overlay에서 두 fixture
모두 report assertion으로 실패할 뿐 패닉하지 않은 결과와 처리 방향은
[[ISSUE-003]]에 남겼다. 따라서 그 역사적 overlay 조건을 완료 기준으로 쓰지 않는다.

## 무엇을

`flow-style root mapping` 서브테스트의 fixture를 배너 주석이 있는 값으로 유지하고,
`MigrateSectionOrder`가 다음 현재 observable contract를 지키는지 확인한다.

- 입력 원본을 바꾸지 않는다.
- 변경 보고를 만들지 않는다.
- flow-style root mapping의 구체적인 `Blocked` 사유를 하나 남긴다.
- 오류나 패닉 없이 focused test를 통과한다.

## Completion Criteria

- [x] flow-style root mapping 서브테스트의 fixture가 배너 주석을 포함한다 | verify: `/usr/bin/grep -qF "# banner\\n{stack: b, version: a}" internal/config/migrate_section_order_test.go`
- [x] 배너 fixture가 원본 보존·변경 없음·flow-style `Blocked` 사유를 함께 검증한다 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderBailsOnUnrepresentableShapes`
- [x] 과거 overlay 패닉 주장이 현행 계약에서 제거되고 [[ISSUE-003]] disposition이 링크됐다 | verify: `/usr/bin/grep -rqF --include='362-make-the-flow-style-fixture-reproduce-the-panic-it-claims.md' '[[ISSUE-003]]' tasks`

## 참고

- [[ISSUE-003]] — historic guard-removal panic의 현재 source 재현 결과와 disposition
- TASK-318 재리뷰 (t318-rereview) Q4 — flow-style root mapping 서브케이스
- TASK-350 (inverted/vacuous verify binding 거부) — fixture와 실제 계약을 함께 검증하는 같은 범주의 사례
- `internal/config/migrate_section_order_test.go` (`TestMigrateSectionOrderBailsOnUnrepresentableShapes`), `internal/config/migrate_section_order.go` (flow-style root guard)

## 검증 기록

- 2026-09-10: `go test ./internal/config/ -run TestMigrateSectionOrderBailsOnUnrepresentableShapes`가 배너 fixture의 원본 보존, 변경 없음, 정확한 `Blocked` 사유, 무패닉을 확인했다.
- 2026-09-10: `ce task validate tasks/done/362-make-the-flow-style-fixture-reproduce-the-panic-it-claims.md`, `make doc-check`, `dva ci commit`이 통과했다.
- 역사적 guard-removal overlay의 무패닉 결과와 disposal은 [[ISSUE-003]]에 기록했다.
- 2026-09-10: 독립 리뷰가 fixture의 원본 보존·변경 없음·정확한 `Blocked` 사유·무패닉 계약과 PLAN-008 수치를 확인했다. 기준의 volatile-zone binding은 done 경로 직접 검사로 수정했다.
