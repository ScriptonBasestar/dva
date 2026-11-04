---
id: TASK-582
title: "Propagate cancellation through status queries and probes"
type: bug
priority: P2
exec-tier: standard
status: review
created: 2026-11-04
---

## Summary

status의 전체 스택·단일 plan·composition 조회에 호출자 context와 신호 취소를 연결한다. Compose·Podman Compose·Docker·Kubectl·Helm·Kustomize·Vagrant·Multipass의 실제 runtime 조회, 구성 health check·endpoint probe·조회 전 터널 인증 및 준비 대기를 같은 취소에 종속시키고, 취소 시 부분 조회 결과를 보존하며 이후 조회를 시작하지 않는다. 기존 HealthChecker·endpoint·터널 API는 호환 wrapper로 유지한다. 일반 조회 오류의 관대한 처리와 정상 text/JSON 형식은 유지하며, 관리 대상 애플리케이션의 시작·정지·제거는 수행하지 않는다. 조회용으로 시작한 자식 프로세스 그룹만 취소·회수하며, 관리 대상 애플리케이션의 기존 TERM 정책은 바꾸지 않는다. 실제 외부 접근이나 인증 변경 없이 임시 로컬 fixture로 검증한다.

## Completion Criteria

- [ ] Status cancels runtime queries and HTTP TCP command probes without starting later queries or modifying managed application resources | verify: `go test -race ./internal/cli ./internal/lifecycle -run 'Test(StatusCommandReceivesSignals|StatusHealth.*|StatusTunnelCleanup.*|BackendStatusQueriesCancelTheirProcessGroups|CloudflaredAuthCancellationTerminatesItsProcessGroup|StatusRoutesHonorCallerCancellation|PreCanceledStatusRoutesRenderWithoutQueries|EndpointCancellationRendersCompletedResultsOnly|OrchestratorStatusCancellationReturnsPartialResults|CompositionStatusCancellationPreservesPriorRows)$'`
- [ ] Normal status output and ordinary query failure tolerance remain compatible | verify: `go test ./internal/cli ./internal/lifecycle -run 'Test(StatusPlanPathEmitsJSON|StatusAllReachesWholeStackPastDefaultPlan|StatusWithoutConfigStaysTolerant|StatusAcceptsKnownPlanName|CompositionStatusReportsFailedChild|CheckHTTPAndTCPAllowZeroTimeout|BackendStatusQueriesCancelTheirProcessGroups)$'`
