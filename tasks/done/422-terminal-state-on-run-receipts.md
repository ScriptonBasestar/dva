---
id: TASK-422
title: "Record a terminal state on ce task run receipts"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent TASK-422 review (ce-explorer, not implementer): PASS; verified the integrated TaskReceipt status field, finish/abort append paths, terminal status precedence, and DONE/ABORTED persistence tests at ce-agent-kit master 05b0b4ae3c18517df2d7635f3b185da4ee3d9baf."
---

## Summary

[ISSUE-005](../_archive/issue/005-ce-task-run-receipts-never-reach-a-terminal-state.md)의
후속 기준이다. 영수증은 자체 `status`를 보존하고, `run-finish`와 `run-abort`가
각각 `DONE`과 `ABORTED`를 기록한다. 상태 조회는 종단 영수증을 워크트리 존재 여부보다
우선한다.

## Completion Criteria

- [x] run 영수증에 종단 상태 필드가 있고 `run-finish`와 `run-abort`가 그 필드를 쓴다 | verify: human — 상류 구현과 영속성 테스트 확인

## Out of scope

- 이미 충족된 `run-list` 보고. 그 기준은 ISSUE-005를 닫은 이유다.
- 기존 영수증을 소급 수정하는 마이그레이션. 상류가 순서를 고른다.

## Sources

- [ISSUE-005](../_archive/issue/005-ce-task-run-receipts-never-reach-a-terminal-state.md)

## Current evidence

상류 `ce-agent-kit` master의 통합 커밋 `05b0b4ae`에서 확인했다.

- 영속 필드와 terminal status: `internal/domain/task_runtime.go:96-120`
- finish/abort append 경로: `internal/usecase/taskruntime/service_lifecycle.go:132-285`, `service_abort.go:69-78`
- 조회 시 저장된 terminal status 우선: `internal/usecase/taskruntime/state.go:24-29`
- finish 후 DONE 영수증 보존 테스트: `internal/usecase/taskruntime/integration_no_fetch_test.go:63-86`
- abort 상태 테스트: `internal/usecase/taskruntime/service_abort_test.go:31-62`
