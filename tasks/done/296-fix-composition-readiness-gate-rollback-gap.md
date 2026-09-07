---
id: TASK-296
title: "Fix composition readiness-gate failure leaving a started child running with no rollback"
type: bug
priority: P1
effort: S
exec-tier: standard
created-at: 2026-09-04T00:00:00+09:00
source: "Independent review of TASK-260 composition rollback implementation (reviewer finding, rated MEDIUM)"
scope: "internal/lifecycle/composition_orchestrator.go CompositionOrchestrator.Up wave-boundary readiness gate only"
status: done
quality-review: pass
quality-reviewed-at: 2026-09-07T18:05:00+09:00
depends-on: []
---

# Task 296: fix composition readiness-gate failure leaving a started child running with no rollback

## Summary

`internal/lifecycle/composition_orchestrator.go`'s `CompositionOrchestrator.Up` deliberately excludes
the readiness-failed child from its own rollback list:

```go
// The failing child keeps state "failed" and is not itself rolled back: TASK-260 §5.2
// rolls back the children that succeeded, and a child's own partial state after a
// failed up stays the child's business (§5.1).
report.Outcome = CompositionOutcomeFailed
report.Error = primary.Error()
report.Children[failedAt].State = ChildStateFailed
report.Children[failedAt].Error = primary.Error()
// A readiness failure lands on a child whose up already succeeded; it is the failure,
// not a rollback target.
succeeded = dropIndex(succeeded, failedAt)
```

Confirmed directly (composition_orchestrator.go:193–235). The wave loop runs two passes per wave: an
`exec.Up` pass that appends `i` to `succeeded` only when `Up` returns nil, then (unless `NoWait`) a
`WaitReady` pass over the same wave. When `WaitReady` fails, `failedAt = i` is set for a child that was
already appended to `succeeded` in the first pass — its `up` genuinely succeeded, only its readiness
check timed out. `dropIndex(succeeded, failedAt)` then removes that index from the rollback list, so the
LIFO rollback loop at line 248 never calls `exec.Down` on it.

For the sibling case — a child whose `exec.Up` itself fails — `failedAt` is set *before* the append to
`succeeded` happens (the loop `break`s first), so that index was never in `succeeded` to begin with;
`dropIndex` is a no-op there. The comment's justification ("not a rollback target") only actually holds
for that case. It is being applied uniformly to a second, distinct case — readiness failure on an
already-started child — where it silently drops a live resource from rollback instead of being a no-op.

TASK-260 §5.2 step 2 requires: "Wave 0..N에서 이미 성공적으로 `up`된 모든 child를 LIFO 순서로 `down`한다"
— every child whose `up` succeeded in the affected waves, with no carve-out for a child whose subsequent
readiness check is what triggered the rollback.

**Concrete failure scenario**: wave 0 = `[api/deploy, web/deploy]`. Both `Up` calls succeed. `web/deploy`'s
`WaitReady` times out. `api/deploy` is torn down (rolled back) via the LIFO loop, but `web/deploy` is left
running with its containers/ports still held — even though `report.Children[web].State` reads `"failed"`,
which everywhere else in the composition status contract (§5.3) means "did not come up," not "came up
and is still running." The operator sees a `"failed"` child in the JSON report with no indication that
live resources exist behind it, and must separately know to run `dva down release` by hand to reclaim
them. This is exactly the "some children may still be up" situation the contract already has a documented
opt-out for — `CompositionUpOptions.NoRollback` (§4.4) exists so an operator can *choose* to preserve
failure state for inspection — except here it happens implicitly, only for readiness failures, regardless
of `NoRollback`.

Also note: the test fake already carries a `readyErr` hook for exactly this
(`internal/lifecycle/composition_orchestrator_test.go:29`, `WaitReady` at line 86), but no existing test
in the file ever sets `f.readyErr[...]` — this path is currently untested.

## Recommended direction

`dropIndex(succeeded, failedAt)` should not run unconditionally for every failure. Per §5.2, rollback's
job is to tear down every child whose `up` succeeded in the affected waves — and a readiness-failed child
qualifies, since it is only in `succeeded` when its `up` returned nil in the first place. Two ways to get
there, either acceptable:

- Stop dropping `failedAt` from `succeeded` at all. For the `exec.Up`-failure case this is already a
  no-op (the index was never appended), so removing the `dropIndex` call changes behavior only for the
  readiness-failure case — which is the one that needs to change. The LIFO rollback loop then naturally
  calls `exec.Down` on the readiness-failed child alongside its wave-mates, and on success its
  `report.Children[idx].State` transitions from `ChildStateFailed` to `ChildStateRolledBack` the same way
  any other rolled-back child's does (the `Error` field the rollback loop leaves untouched on success
  still shows why the rollback happened).
- Or keep `failedAt` distinguished from ordinary `succeeded` entries but still call `exec.Down` on it as
  part of the rollback pass, if the report wants to keep its state visibly different from a plain
  rolled-back sibling (e.g. a fifth `ChildState` for "rolled back after readiness failure"). This is more
  invasive for what looks like a should-be-simple fix — only worth it if reusing `ChildStateRolledBack`
  turns out to erase information a caller relies on.

Either way, update or remove the "not a rollback target" comment — it's currently wrong for the readiness
path and should either be scoped explicitly to the `exec.Up`-failure case or dropped once the code no
longer needs it to justify anything.

## Completion Criteria

- [x] A wave-boundary readiness (`WaitReady`) failure on a child whose `exec.Up` succeeded results in that
      child being included in the LIFO rollback alongside the rest of its wave's succeeded siblings — not
      dropped from the rollback list | verify: `/usr/bin/grep -Eq '^func TestCompositionReadinessFailureRollsBackSucceededSiblings\(' internal/lifecycle/composition_orchestrator_readiness_test.go && go test ./internal/lifecycle -count=1`
      (note: the test landed in a new sibling file `composition_orchestrator_readiness_test.go` rather
      than the existing `composition_orchestrator_test.go` — the repo's file-size hook blocks that file at
      600 code lines and it was already at the limit)
- [x] The existing `exec.Up`-failure rollback behavior is unchanged — a child whose `up` itself failed is
      still not treated as a rollback target (it was never in `succeeded`), confirmed by the pre-existing
      up-failure rollback tests in `composition_orchestrator_test.go` continuing to pass unmodified |
      verify: `go test ./internal/lifecycle -run 'TestCompositionRollback|TestCompositionUpRollsBackSucceededChildrenOnFailure' -count=1`
- [x] The `readyErr` hook on `fakeChildExecutor` (composition_orchestrator_test.go:29) is exercised by at
      least one test (`TestCompositionReadinessFailureRollsBackSucceededSiblings`,
      `composition_orchestrator_readiness_test.go`), closing the gap noted above | verify:
      `/usr/bin/grep -Eq '^func TestCompositionReadinessFailureRollsBackSucceededSiblings\(' internal/lifecycle/composition_orchestrator_readiness_test.go`
- [x] Repository gates pass | verify: `make lint && make test && make test-integration && make commit-check`

## Non-goals

- Not changing how an `exec.Up`-failure itself is reported or rolled back — only the readiness-failure
  path, which currently diverges from it incorrectly.
- Not introducing a new `ChildState` unless the simpler fix (reusing `ChildStateRolledBack`) turns out to
  lose information a caller needs — see "Recommended direction" above.
- Not changing `CompositionUpOptions.NoRollback` semantics — a readiness-failed child should still be
  preserved un-rolled-back when the operator explicitly passes `--no-rollback`, same as any other
  succeeded child today.
- Not touching `Orchestrator.Down`'s per-entry error swallowing — that's TASK-295, a separate defect in a
  different function.

## Review Log

독립 리뷰 (2026-09-07, 구현자 아님). 재실행한 것:

- 기준 1/3의 grep 바인딩 exit 0, `go test ./internal/lifecycle -count=1` 통과.
- 기준 2: `go test ./internal/lifecycle -run
  'TestCompositionRollback|TestCompositionUpRollsBackSucceededChildrenOnFailure' -count=1 -v`
  → 두 테스트 모두 PASS, 즉 `exec.Up` 실패 경로의 기존 동작이 변하지 않았다.
- `make test` / `make test-integration` / `make doc-check` 전부 exit 0.

diff가 실제로 한 일: `dropIndex`는 코드베이스에서 완전히 사라졌다
(`grep -n dropIndex internal/lifecycle/*.go` 0건) — "Recommended direction"의 첫 번째 안,
즉 더 단순한 쪽이 채택됐고 새 `ChildState`는 도입되지 않아 non-goal을 지킨다.
`composition_orchestrator_readiness_test.go`는 코드가 돌았다는 확인이 아니라 실제 동작을
단언한다: 호출 순서(`up:a,up:b,wait:a,wait:b,down:b,down:a`)로 readiness 실패 자식 b가 형제
a와 함께 LIFO로 내려가는 것을, 상태 맵으로 b가 `failed`→`rolled_back`으로 전이하며 원본
readiness 오류가 `child.Error`에 보존되는 것을, `errors.Is`로 primary 오류가 그대로 반환되는
것을 각각 잡는다. c-plan이 `not_started`로 남아 후속 wave 미기동도 확인된다.

적대적으로 확인한 것: `NoRollback` 경로는 롤백 루프 이전에 return하므로 readiness 실패 자식도
그대로 살려 두며, 이는 카드의 non-goal과 일치한다. 그때 상태가 `failed`로 남아 실제로는 기동된
자식을 가리키는 표기 격차는 TASK-291의 F2로 이미 별도 기록돼 있다.

**판정: pass.**
