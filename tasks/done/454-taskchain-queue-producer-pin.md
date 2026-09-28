---
id: TASK-454
title: "Stage fail-closed TaskChain queue producer pin"
type: chore
priority: P1
effort: M
exec-tier: strong
execution-mode: implementation
allowed-paths: [tools/taskqueueverdict, internal/cli, internal/config, internal/integration, dva.yml, README.md, USAGE.md]
status: done
created: 2026-09-29
quality-review: pass
quality-reviewed-at: 2026-09-29
quality-review-evidence: "Independent session /root/pin_code_review done-review PASS: verified all four criteria against final diff, compiled fail-closed start, approved-pin snapshot tests, unchanged read-only interactions, and README/USAGE/ISSUE-453 staging boundary. Full DVA CI d55ba50ae7f6b14a748384305047ecf6 succeeded; earlier STALE run excluded."
---

## Summary

TaskChain 후보의 출처 정보를 DVA에 기록하고, 승인된 바이너리 핀이 없으면
`task-queue-start`가 큐·CE를 호출하기 전에 실패하게 한다. 승인된 핀이 생기면
첫 `PATH` 바이너리의 열린 파일을 해시 검증해 만든 스냅샷만 실행하는
검증용 구현을 준비한다. 공개 릴리스 승인 후 컴파일된 DVA 명령에 연결하는
작업은 ISSUE-453과 W07c2에 남긴다.

## Completion Criteria

- [x] Production task-queue-start rejects when no approved platform pin exists before queue or CE invocation | verify: `go test -tags=integration ./internal/integration -run TestTaskQueueStartInteractionFromUnrelatedDirectory`
- [x] An injected approved pin executes the exact hashed first PATH binary snapshot; mismatch and replacement fail closed | verify: `go test ./tools/taskqueueverdict -run 'Test(Pin|ExecuteStartTypeBridge)'`
- [x] Read-only queue interactions keep their prior output contract | verify: `go test -tags=integration ./internal/integration -run 'TestTaskQueue(Interaction|VerdictInteraction)FromUnrelatedDirectory'`
- [x] User documentation identifies the unverified read-only producer and staged start restriction | verify: human — inspect README and interaction descriptions

## Evidence

- Independent code review and separate done-review: PASS (`/root/pin_code_review`).
- `dva ci full`: `d55ba50ae7f6b14a748384305047ecf6`, succeeded, 57.84s.
- `make doc-check`, `ce task gate`, targeted unit/integration tests: PASS.
- ISSUE-453 remains open for published artifact approval and production activation.
