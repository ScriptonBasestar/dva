---
id: TASK-577
title: "Enforce single-plan readiness bounds with signal-aware cancellation"
type: feature
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-11-04
---

## Summary

`dva up <plan>`이 공용 readiness 게이트를 쓰도록 바꿔 ready_timeout 초과, 호출자 취소·마감이 실패로 전달되고 이후 엔트리가 시작되지 않게 한다. compose 루트가 signal.NotifyContext로 SIGINT/SIGTERM/SIGHUP을 plan up에 연결하고, 단일 plan 경로의 `orch.Up`이 준비 실패를 warn에서 오류로 승격한다. 게이트가 `orch.Up` 자체에 있으므로 legacy 전체 스택 `dva up`과 `dva restart`(스택·plan 양쪽, `Wait: true` 호출자)도 같은 실패 전달을 받아 종료 코드가 바뀐다 — 독립 리뷰가 이 범위를 지적해 문서와 CHANGELOG에 명시했다(TASK-117 fail-closed 철학과 일치). 구현과 테스트는 codex 세션이 2026-10-29에 `codex__mbp__fix__plan-readiness-bounds` 워크트리에 남긴 미커밋 작업(13파일 수정 + 신규 테스트 2개)에서 온 것이며, 이 카드는 그 통합을 소유한다.

## Sources

- 워크트리 `/Users/archmagece/worktrees/misc/dva/codex__mbp__fix__plan-readiness-bounds`의 미커밋 diff (2026-10-29 17:20, 브랜치 커밋 없음): `runPlanUp` ctx 매개변수화, `waitEntryReady` 경합 수정, `waitEntriesReady` 공용화.
- 사용자 지시 (2026-11-04 세션): "워크트리가 분기되어 있으면 통합해가면서 작업하면 된다."
- 구조개선 검토(2026-10-31 세션 3bd035d5) 권장안 #1 "signal context 전파"에 해당.
- CHANGELOG 항목: "단일 plan의 readiness 대기" (적용 시 재배치).

## Completion Criteria

- [ ] Single-plan up fails on the declared ready_timeout and bounds in-flight probes | verify: `go test -count=1 -run TestOrchestratorUpReadinessTimeout ./internal/lifecycle -v 2>&1 | /usr/bin/grep -q -- '--- PASS: TestOrchestratorUpReadinessTimeout'`
- [ ] Caller cancellation and deadline reach the readiness wait and stop later entries | verify: `go test -count=1 -run TestOrchestratorUpReadinessCallerCancellation ./internal/lifecycle -v 2>&1 | /usr/bin/grep -q -- '--- PASS: TestOrchestratorUpReadinessCallerCancellation'`
- [ ] A dead native entry fails readiness with its PID diagnostic | verify: `go test -count=1 -run TestOrchestratorUpReadinessDeadNativeProcess ./internal/lifecycle -v 2>&1 | /usr/bin/grep -q -- '--- PASS: TestOrchestratorUpReadinessDeadNativeProcess'`
- [ ] The cobra up command passes its caller context into plan up | verify: `go test -count=1 -run TestPlanUpCallerContextReachesReadiness ./internal/cli -v 2>&1 | /usr/bin/grep -q -- '--- PASS: TestPlanUpCallerContextReachesReadiness'`
