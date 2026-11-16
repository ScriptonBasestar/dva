---
id: TASK-615
title: "reseal creation-rule 오류를 Wait 관측 순서와 독립적으로 분류"
type: bug
priority: P0
effort: S
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-615/review.txt
created: 2026-11-17
---

## Summary

호스팅 CI의 early reseal 사례가 creation-rule 오류 대신 decrypt 오류로 실패했다.
두 자식의 결과와 제한된 진단을 회수한 뒤 알려진 encrypt creation-rule 오류를 우선한다.
실제 OS 실패 시각을 복원한다는 보장은 하지 않는다.

## Scope

`internal/cli/secret_source_sops.go`와 전용 테스트, CHANGELOG, 원인·검증 기록.
최초 실패 시 취소·양쪽 Wait 회수, 평문 pipe, 진단 길이 제한과 고정 오류 코드,
실패 시 원본 보존을 유지한다. sleep·재시도·새 timeout으로 실패를 숨기지 않는다.
CI 예산 TASK-614와 별도 제품 task로 전달한다.

## Completion Criteria

- [x] 두 결과와 수집된 진단을 함께 분류한다 | verify: `/usr/bin/grep -q 'resealResultError' internal/cli/secret_source_sops.go` (source: absent; observed: 2026-11-17)
- [x] 종료 결과 순서가 반대인 경우와 일반 실패·성공·진단 경계를 검증한다 | verify: `go test ./internal/cli -run TestResealResultErrorPreservesCreationRule` (observed: 2026-11-17)
- [x] 기존 실제 pipe 사례의 원본 보존·민감 출력 비노출을 유지한다 | verify: `go test ./internal/cli -run TestSecretSourceResealFailurePreservesSource` (observed: 2026-11-17)
- [x] 독립 검토와 원격 실패·수정의 증거를 대조한다 | verify: human — 원인 기록과 raw 실패·리뷰 대조

## Verification

`go test -race ./internal/cli -run '^(TestResealResultErrorPreservesCreationRule|TestResealCancelsFailedStagePeer|TestSecretSourceResealFailurePreservesSource)$' -count=10`
통과(8.645초). lint·doc-check·check-generate 통과. 최종 committed-head 검사는 별도다.

## Delivery

독립 source/done 검토 뒤 확정된 커밋에서 dev/full과 원격 CI를 검증한다.
제품 수정 통합 후 TASK-614를 최신 source에 갱신해 최종 검증·통합·설치·정리한다.
