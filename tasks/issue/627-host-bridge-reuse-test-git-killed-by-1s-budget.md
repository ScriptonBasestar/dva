---
id: ISSUE-627
title: "test: TestHostBridgeReuseHitDoesNotClaim git index listing killed by 1s run budget under load"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
severity: medium
ownership: local
discovered-in: "test: TestHostBridgeReuseHitDoesNotClaim git index listing killed by 1s run budget under load"
discovered-at: 2026-11-22
created: 2026-11-22
---

## Summary

Candidate finding, not fixed. During the TASK-634 integration gate, one full `make test`
run failed `TestHostBridgeReuseHitDoesNotClaim` (internal/cirun) with
`list git index inputs: signal: killed`: the test's 1-second run budget killed the `git`
child that lists index inputs. The test passed 5 times alone, and a full `make test` rerun
exited 0, so the cause is inferred to be host load against a fixed 1s budget, not a logic
defect. It makes the integration gate flaky for every dva change.

## Reproduction

1. Run the full `make test` in a dva worktree while the host is loaded (observed once on
   2026-11-22 at fd706b16 plus the TASK-634 stack; not reproduced deterministically).

## Expected vs Actual

- Expected: the test's budget tolerates normal gate load, or the budget failure is told apart
  from a reuse-claim failure.
- Actual: `signal: killed` from the git index listing under the 1s budget fails the test.

## 소유권 — 이 저장소다 (2026-11-23 명시)

실패하는 것은 dva 자체 테스트 `TestHostBridgeReuseHitDoesNotClaim`의 1초 실행 예산이고, 고칠 곳도
이 저장소의 테스트 또는 예산 처리다. 상류 도구에 넘길 몫은 없다.

## Completion Criteria

- [ ] TestHostBridgeReuseHitDoesNotClaim passes repeatedly under a loaded full make test run. | verify: human — 부하 중 make test 반복 결과를 독립 리뷰가 확인
