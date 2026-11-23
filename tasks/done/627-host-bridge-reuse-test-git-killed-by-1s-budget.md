---
id: ISSUE-627
title: "test: TestHostBridgeReuseHitDoesNotClaim git index listing killed by 1s run budget under load"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
delivered-by: 4fd2d62c
quality-review: pass
quality-review-evidence: "ce review-evidence 08a971a229c9c601e36d9c34e8673a0809f0e1276727d7bc32e26e92e054ce2b"
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

- [x] TestHostBridgeReuseHitDoesNotClaim passes repeatedly under a loaded full make test run. | verify: human — 부하 중 make test 반복 결과를 독립 리뷰가 확인

## Outcome

- 원인: 두 "does not claim" 테스트(`ReuseHitDoesNotClaim`, `ConditionFalseDoesNotClaim`)가 1초
  `context` 마감을 "슬롯을 잡지 않았다"의 탐지기로 썼다. 슬롯 점유는 이미 step 단언(`reused`/
  `condition-false`가 아니라 admission 대기 후 `timed_out`)이 잡으므로, 1초는 탐지에 필요 없는 속도
  예산이 되어 부하 중 `git` 자식을 죽였다.
- 수정: 두 테스트의 마감을 30초 hang guard(`noClaimDeadline`)로 바꾸고, 실패 메시지가 hang guard와
  admission 대기를 구분해 말하게 했다. 고정 슬롯 획득 마감도 다른 테스트와 같은 3초로 맞췄다.
- 검증: `go test ./... -count=1`를 동시에 돌리는 부하 중 두 테스트를 `-count=20`(40회) 실행해 전부
  통과. reuse hit를 끈 변이에서는 테스트가 `timed_out` 보고로 실패해 회귀를 여전히 잡는다.

## Review Attempts

### 1. 2026-11-24 — reviewer: 독립 opus 서브에이전트(읽기 전용), executor: 구현 실행(4fd2d62c)

- 기준 pass. 근거: `ce review-evidence 08a971a2…`.
- 리뷰어가 reuse hit·condition-false 조기 반환을 각각 끄는 변이로 두 테스트가 약 5초에 "did not skip the held heavy slot"로 실패함을 확인했다. 부하 중 40회 반복 실행은 구현 실행의 기록이며 리뷰어가 재실행하지는 않았다.
- 다음 전환: done.
