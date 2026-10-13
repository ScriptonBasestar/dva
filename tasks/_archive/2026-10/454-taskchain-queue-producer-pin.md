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
archived-at: 2026-10-03
---

## Summary

TaskChain 후보의 출처 정보를 DVA에 기록하고, 승인된 바이너리 핀이 없으면
`task-queue-start`가 큐·CE를 호출하기 전에 실패하게 한다. 승인된 핀이 생기면
첫 `PATH` 바이너리의 열린 파일을 해시 검증해 만든 스냅샷만 실행하는
검증용 구현을 준비한다. 공개 릴리스 승인 후 컴파일된 DVA 명령에 연결하는
작업은 ISSUE-453과 W07c2에 남긴다.

## Completion Criteria

- [x] Production task-queue-start rejects when no approved platform pin exists before queue or CE invocation | verify: human — 과거 검증 명령 1 (아래 제거 후 기록 참조)
- [x] An injected approved pin executes the exact hashed first PATH binary snapshot; mismatch and replacement fail closed | verify: human — 과거 검증 명령 2 (아래 제거 후 기록 참조)
- [x] Read-only queue interactions keep their prior output contract | verify: human — 과거 검증 명령 3 (아래 제거 후 기록 참조)
- [x] User documentation identifies the unverified read-only producer and staged start restriction | verify: human — inspect README and interaction descriptions

## Evidence

- Independent code review and separate done-review: PASS (`/root/pin_code_review`).
- `dva ci full`: `d55ba50ae7f6b14a748384305047ecf6`, succeeded, 57.84s.
- `make doc-check`, `ce task gate`, targeted unit/integration tests: PASS.
- ISSUE-453 remains open for published artifact approval and production activation.
- TASK-455 moved the pin and its regression tests to `internal/taskqueue`; the criterion's verify binding now targets the moved tests. The approved-pin requirement and prior review finding did not change.

## 기능 제거 후 기록 (2026-10-13)

TaskChain 전용 연동을 DVA 제품 경계에서 제거했다. 위의 검증 결과는 도입 당시
기록이며 현행 실행 바인딩이 아니다. 원래 명령과 과거 리뷰 영수증은 보존한다.
제품 판단은 [제품 경계](../../../PRODUCT.md#product-boundaries)를 따른다.

과거 검증 명령 1:

```sh
go test -tags=integration ./internal/integration -run TestTaskQueueStartInteractionFromUnrelatedDirectory
```

과거 검증 명령 2:

```sh
go test ./internal/taskqueue -run 'Test(Start|Pin)'
```

과거 검증 명령 3:

```sh
go test -tags=integration ./internal/integration -run 'TestTaskQueue(Interaction|VerdictInteraction)FromUnrelatedDirectory'
```
