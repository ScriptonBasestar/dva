---
id: TASK-612
title: "비교 fake-host 테스트의 포트 점유 조회를 격리하고 실제 운영 검사를 유지"
type: bug
priority: P1
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-613/review.txt
created: 2026-11-16
---

## Summary

TASK-611 진단에서 fake lifecycle 테스트가 실제 포트 점유 상태 때문에 의도한 오류
응답 대신 preflight 실패를 받았다. 임시 포트를 닫고 다시 확인하는 테스트 의존성을
기존 host 경계로 격리하고 실제 점유 검사의 보호 동작을 유지한다.

## Scope

`internal/cli/compare_runner.go`, `compare_up.go`, 비교 테스트와 CHANGELOG만 변경한다.
사용자 프로세스를 종료하거나 운영 포트 선택·점유 검사·실패 정책을 바꾸지 않는다.

## Completion Criteria

- [x] 포트 점유 조회가 기존 host 경계에 포함된다 | verify: `/usr/bin/grep -q 'PortBusy(port int)' internal/cli/compare_runner.go` (observed: 2026-11-17)
- [x] fake busy/error는 시작·파일 생성을 막고 skipped variant는 점유 조회를 하지 않는다 | verify: `go test ./internal/cli -run 'TestCompare(UpPreflightCreatesNothing|SkippedVariantDoesNotProbePorts)'` (observed: 2026-11-17)
- [x] 실제 listener를 유지한 운영 점유 검사가 실패를 검출한다 | verify: `go test ./internal/cli -run '^TestCompareProductionPortProbeSeesHeldListener$'` (observed: 2026-11-17)

## Delivery

독립 리뷰와 반복 진단, batch 최종 full 및 CE 통합·push·정리는 별도 증거로 확인한다.
