---
id: TASK-467
title: "Split lifecycle orchestrator.go and resolver.go clusters"
type: refactor
priority: P2
effort: M
exec-tier: standard
allowed-paths: [internal/lifecycle, tasks]
status: todo
created: 2026-10-02
---

## Summary

`internal/lifecycle/orchestrator.go`(743 코드라인)와 `resolver.go`가 config
kind 한도를 넘는다. 클러스터 봉합으로 나눈다. 함수·타입 이동만 허용한다.

봉합 계획(실행 시 실측 조정):

- `orchestrator_filter.go`(신규) — 필터·태그 클러스터: filterEntries,
  validateDeclaredTags, filterByNames, filterByTags, partitionPlanServices,
  serviceLooksRunning, hasAnyTag.
- `orchestrator_mode.go`(신규) — 모드 프로세스 클러스터: startModeProcesses,
  haltModeProcesses, stopModeProcesses, signalModeProcesses.
- orchestrator.go 잔존 — Orchestrator 본체, UpOptions/DownOptions/StopOptions,
  Up/Down/Stop/Restart/Status, NewOrchestrator.
- `resolver_plan.go`(신규) — 플랜 해석·extends 병합: ResolvePlanName,
  ResolvePlanAlias, MergePlanExtends, clonePlanConfig, mergePlanConfigs,
  ExecutionPlan OwnerConfig 포함 타입군.
- `resolver_runner.go`(신규) — 러너 헬퍼: mergeStringMap, copyStringSlice,
  normalizeRunnerName, runnerDeclared, runnerConfigDir, optionalSkipDir.
- resolver.go 잔존 — ResolvePlan 본체, CalculateWaves, trace/warn 계열.

## Completion Criteria

- [x] `internal/lifecycle/orchestrator.go`가 한도 안에 있다 | verify: `ce validate filesize internal/lifecycle/orchestrator.go` (observed: 2026-10-02 — 421 물리, exit 0, Medium 1건은 300 경고선 고지)
- [x] `internal/lifecycle/resolver.go`가 한도 안에 있다 | verify: `ce validate filesize internal/lifecycle/resolver.go` (observed: 2026-10-02 — 408 물리)
- [x] 새 파일 전부가 한도 안에 있다 | verify: `ce validate filesize internal/lifecycle/orchestrator_filter.go internal/lifecycle/orchestrator_mode.go internal/lifecycle/resolver_plan.go internal/lifecycle/resolver_runner.go` (observed: 2026-10-02 — filter 167 / mode 189 / plan 232 / runner 100 물리, High 0)
- [x] 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/lifecycle/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 15.2s)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/lifecycle/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — main-thread 리뷰: numstat 두 원본 모두 0 insertions, 이동/잔존 블록 전체 cmp 바이트 동일, 추가분은 package+import뿐)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — run-finish 게이트 확인)

## Evidence

분할 결과(물리): orchestrator.go 759→421, resolver.go 724→408, 신규
orchestrator_filter 167 / orchestrator_mode 189 / resolver_plan 232 /
resolver_runner 100. 보존 정산: 759 = 421 + 331 이동 + 7(import 4 + 구분
공백 3), 724 = 408 + 311 이동 + 5. 이동·잔존 블록 전체 cmp 바이트 동일.

게이트: `go build ./...` ok · `go test ./internal/lifecycle/ -count=1` ok ·
vet silent · gofmt clean · filesize 6파일 High 0 (Medium 1건: orchestrator
323 코드라인 고지).
