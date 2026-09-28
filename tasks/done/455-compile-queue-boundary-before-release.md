---
id: TASK-455
title: "Compile the TaskChain queue start boundary before release"
type: feature
priority: P1
effort: M
exec-tier: strong
execution-mode: implementation
allowed-paths: [internal/taskqueue, internal/cli, internal/integration, tools/taskqueueverdict, tasks, README.md, USAGE.md, CHANGELOG.md, ARCHITECTURE.md]
status: done
created: 2026-09-29
quality-review: pass
quality-reviewed-at: 2026-09-29
quality-review-evidence: "Independent session /root/compiled_queue_boundary done-review PASS: reviewed all four criteria against the final code including migrated tests, unchanged unauthorized manifest, exact queue/CE root binding, read-only output golden, and W07c2a boundary. DVA full CI 5ebc8dd34d4327b67fb62586e5a84134 succeeded."
---

## Summary

공개 승인 전에 DVA의 TaskChain 큐·pin 검증·CE 시작 로직을 컴파일된 명령에
연결한다. 승인된 배포 산출물은 아직 없으므로 production pin은 비활성 상태를
유지하고, 실제 CE 시작은 계속 거부한다. W07c2a에서만 출처 검증 후 활성화한다.

## Completion Criteria

- [x] Compiled `task-queue-start` owns the only CE mutation path and a Go/PATH tool wrapper cannot start CE | verify: `go test ./internal/taskqueue ./tools/taskqueueverdict ./internal/cli`
- [x] The production manifest rejects before queue, CE, or PATH Go invocation; an injected approved pin uses the hashed snapshot and binds queue and CE subprocesses to the DVA config root | verify: `go test ./internal/taskqueue && go test -tags=integration ./internal/integration -run TestTaskQueueStartInteractionFromUnrelatedDirectory`
- [x] Read-only queue and verdict commands preserve their exact stdout contract and remain explicitly unpinned | verify: `go test -tags=integration ./internal/integration -run 'TestTaskQueue(Interaction|VerdictInteraction)FromUnrelatedDirectory'`
- [x] ISSUE-453 and documentation describe the remaining release approval and positive host checks accurately | verify: human — inspect issue and release boundary text

## Dependency

TASK-454 supplies the staged manifest and fail-closed command. This preparation
may run before W07c1. It does not authorize publication or close ISSUE-453.

## Evidence

- Independent code review and separate done-review: PASS (`/root/compiled_queue_boundary`).
- `dva ci full`: `5ebc8dd34d4327b67fb62586e5a84134`, succeeded, 3m5.23s.
- Targeted unit and integration tests, exact verdict golden, and `git diff --check`: PASS.
- TASK-454 historical verify binding was retargeted to the moved pin tests; its criterion and review verdict remain unchanged.
