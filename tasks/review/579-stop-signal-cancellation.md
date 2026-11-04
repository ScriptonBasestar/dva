---
id: TASK-579
title: "Propagate cancellation through stop execution paths"
type: bug
priority: P2
exec-tier: standard
status: review
created: 2026-11-04
---

## Summary

전체 스택·단일 plan·composition의 stop 호출에 호출자 context와 SIGINT·SIGTERM·SIGHUP 취소를 전달한다. 취소 전에는 엔트리를 실행하지 않고, 실행 중 취소되면 이후 대상을 중지하지 않으며 이미 끝난 정지는 유지한다. down·purge·rollback 정책과 status는 이번 범위 밖이다. 소유 구현은 internal/cli의 stop 경로와 internal/lifecycle의 Stop 전용 취소 검사다.

## Completion Criteria

- [ ] Stop preserves caller cancellation across whole-stack, plan and composition routes without starting later entries | verify: `go test ./internal/cli ./internal/lifecycle -run 'Test(StopRoutes.*|CompositionStopHonorsCancellationWithoutStartingLaterChildren|OrchestratorCancelledCallerDoesNotStartOrStopEntries|LifecycleCommandContextReceivesSignals)$'`
