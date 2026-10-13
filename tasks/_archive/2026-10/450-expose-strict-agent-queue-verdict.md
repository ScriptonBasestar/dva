---
id: TASK-450
title: "Expose strict read-only agent queue verdict"
type: feature
priority: P1
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-reviewed-at: 2026-09-28
quality-review-evidence: "Independent done-review session /root/task240_done_review PASS on final diff: verified four-state versioned verdict and candidate/null contract, strict fail-closed queue validation, exact DVA nested-cwd interaction with no board mutation, README and ISSUE-004/006 open boundaries; independent code review PASS; targeted tests and vet PASS; DVA full CI run 98a6ccf3bb5212415dc89cd8ea724325 succeeded."
created: 2026-09-28
execution-mode: implementation
allowed-paths: [tools/taskqueueverdict, internal/integration, dva.yml, README.md]
archived-at: 2026-10-03
---

## Summary

W07b1의 읽기 전용 판단 경계다. 제품 queue의 사람용 전체 집합과 에이전트용
부분 집합을 엄격히 검증하고 0건·사람 전용·후보 1건·후보 여러 건을 서로 다른
버전 있는 결과로 낸다. 한 건은 실행 후보일 뿐 선택·claim·완료가 아니다.
실제 host lifecycle, terminal 전이 및 rollback은 W07b2에 남기며 기존 CE가
유일한 수명주기 writer로 유지된다.

## Completion Criteria

- [x] A versioned verdict distinguishes empty, human_required, candidate, and selection_required without selecting or claiming | verify: human — 과거 검증 명령 1 (아래 제거 후 기록 참조) covers all four states and candidate/null output
- [x] Malformed or inconsistent upstream queue output fails closed with no success JSON | verify: human — parser negative cases and DVA interaction failure cases passed
- [x] DVA interaction works from nested cwd and preserves the board while ISSUE-004/006 remain open | verify: human — 과거 검증 명령 2 (아래 제거 후 기록 참조) passed

## Evidence (2026-09-28)

`dva config validate` passed. From `docs/`, `dva task-queue-verdict` with the
locally built taskchain-task-manager `9e8fac7` binary (SHA-256
`d2ca31f2880a1420efef3503dba2333f62c0ddd84648b06f4c75b5a62ad6ecd0`)
returned `{"verdictVersion":1,"state":"empty","runnableCount":0,"agentRunnableCount":0,"runnable":[],"agentRunnable":[],"candidate":null}`.
The current DVA board has no P0 issue or ready todo, so this is an empty live
case, not human-only terminal evidence. The interaction fixture checks board
digest and lock absence across success and failure; ISSUE-004/006 remain open
until the later lifecycle and terminal criteria are met.

Independent done-review PASS: verified all three criteria against the diff,
including the four-state candidate/null contract, fail-closed upstream
validation, nested-cwd DVA interaction, unchanged board, and open
ISSUE-004/006 boundary. Independent code review PASS after fixes. Targeted
unit/integration tests and vet passed; `dva ci full` run
`98a6ccf3bb5212415dc89cd8ea724325` succeeded; `ce task gate` was READY.

## 기능 제거 후 기록 (2026-10-13)

TaskChain 전용 연동을 DVA 제품 경계에서 제거했다. 위의 검증 결과는 도입 당시
기록이며 현행 실행 바인딩이 아니다. 원래 명령과 과거 리뷰 영수증은 보존한다.
제품 판단은 [제품 경계](../../../PRODUCT.md#product-boundaries)를 따른다.

과거 검증 명령 1:

```sh
go test ./tools/taskqueueverdict
```

과거 검증 명령 2:

```sh
go test -tags=integration ./internal/integration -run '^TestTaskQueueVerdictInteractionFromUnrelatedDirectory$' -count=1
```
