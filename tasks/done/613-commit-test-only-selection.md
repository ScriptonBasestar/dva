---
id: TASK-613
title: "테스트 전용 staged 변경의 직접 패키지 검사와 일반 변경의 역의존 검사를 구분"
type: feature
priority: P2
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-613/review.txt
created: 2026-11-16
---

## Summary

기존 실측의 `_test.go` 변경도 CLI 소비자까지 전파돼 테스트 실행이 약 39초 걸렸다.
테스트 전용 변경은 해당 패키지의 테스트를 검사하고 일반 Go 변경은 기존의 production·
내부/외부 test import 역의존 폐쇄를 유지한다. 공통 dev 영향 선택기는 변경하지 않는다.

## Scope

`tools/commitverify/main.go`, `main_test.go`, CI 파일럿·실측 문서와 CHANGELOG.
공유 입력·삭제·이동·타입 변경·매핑 불가의 core/full 경계와 최종 전체 검사를 유지한다.
재사용은 입력 완전성·절감 효과 검토 후 판단하고 보호 규칙은 읽기 전용으로 확인한다.

## Completion Criteria

- [x] 테스트 전용 root와 일반 변경 root를 구분한다 | verify: `/usr/bin/grep -q 'productionDirs' tools/commitverify/main.go` (observed: 2026-11-17)
- [x] 내부/외부 test·일반·혼합·미매핑 변경과 각 phase의 argv를 검증한다 | verify: `go test ./tools/commitverify -run 'TestTestOnlyAndMixedChangesSelectPackageGraph|TestTestOnlySelectionUsedInAllGoPhases|TestBroadChangesFailBeforeExpensiveChecks|TestEmbeddedTestFilesRequireCore'` (observed: 2026-11-17)
- [x] 동일 staged 사례의 source·cache·명령·성공/실패·시간과 채택 결정을 보존한다 | verify: human — raw measurement와 독립 리뷰 대조 (observed: 2026-11-17)

## Delivery

측정은 최종 인수 검사를 대체하지 않는다. 독립 리뷰 후 관련 기록을 확정하고 한 batch의
최종 full·CE 통합·설치·정리를 수행한다. 기존 측정·실패 증거는 수정하지 않는다.
