---
id: TASK-579
title: "Propagate cancellation through stop execution paths"
type: bug
priority: P2
exec-tier: standard
status: done
created: 2026-11-04
quality-review: pass
quality-reviewed-at: 2026-11-04T11:02:26Z
quality-review-evidence: tasks/done/evidence/TASK-579/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-579/done-review-bc5e9269d5f0bb4f1b200feea85557a6c83ef15136326420e120643a0ebb55a8.json
---

## Summary

전체 스택·단일 plan·composition의 stop 호출에 호출자 context와 SIGINT·SIGTERM·SIGHUP 취소를 전달한다. 취소 전에는 엔트리를 실행하지 않고, 실행 중 취소되면 이후 대상을 중지하지 않으며 이미 끝난 정지는 유지한다. down·purge·rollback 정책과 status는 이번 범위 밖이다. 소유 구현은 internal/cli의 stop 경로와 internal/lifecycle의 Stop 전용 취소 검사다.

## Completion Criteria

- [x] Stop preserves caller cancellation across whole-stack, plan and composition routes without starting later entries | verify: `go test ./internal/cli ./internal/lifecycle -run 'Test(StopRoutes.*|CompositionStopHonorsCancellationWithoutStartingLaterChildren|OrchestratorCancelledCallerDoesNotStartOrStopEntries|LifecycleCommandContextReceivesSignals)$'`

## Review Attempts

- 2026-11-04T11:02:26Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root/parent_compose_drift` (standard). Criterion PASS after guarded `-count=1 -v` verification; TASK-581 retained `-race`; required named tests executed. No blocking findings; source `6ef16cbc45dfd070472e0e8e7cb7e5affad6e0dd`. Evidence: `tasks/done/evidence/TASK-579/independent-review.json`; runtime `review-stop-warning-bundle` (review). Outcome: pass; next transition: done.
