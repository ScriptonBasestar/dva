---
id: TASK-291
title: "Implement composition runtime and LIFO rollback"
type: feature
priority: P1
effort: L
exec-tier: strong
created-at: 2026-09-04T10:00:00+09:00
source: "PLAN-005 implementation of TASK-260's frozen composition contract"
scope: "wave-sequential cross-project execution for every lifecycle verb, automatic LIFO rollback with original-error preservation, the --no-rollback opt-out, and partial-state reporting"
status: done
quality-review: conditional
quality-reviewed-at: 2026-09-07T18:00:00+09:00
depends-on: [TASK-290]
---

# Task 291: implement composition runtime and LIFO rollback

## Summary

Execute a resolved `CompositionPlan`'s children sequentially by wave for every lifecycle verb, and
implement the automatic LIFO rollback-on-failure behavior TASK-260 §5.1 explicitly introduces as new
(single-project `Up` at `internal/lifecycle/orchestrator.go:75` has no such rollback today, and this task
must not silently change that single-project behavior while adding it for composition).

## Recommended direction

Add a composition-aware execution path that, for `up`, calls each wave's children in declaration order,
sequentially (TASK-260 §4.1 — wave is an ordering concept, not a concurrency model; do not introduce
parallel execution). Between waves, wait for each just-started child's own readiness/health check unless
`--no-wait` is set (§4.5).

On any child's `up` failure: stop starting further children, then tear down every already-succeeded child
in strict LIFO order via that child's own `down` (plain teardown — never with `--volumes`/`--purge`, per
§5.2). If a rollback `down` itself fails, preserve the original failure as the primary error unchanged
("original-error preservation", §5.2) and record the rollback failure as a secondary diagnostic naming
the still-possibly-up child.

Add the `--no-rollback` flag (TASK-260's "Open question — resolved 2026-09-04": the user approved this
opt-out after the initial draft; TASK-260 §4.4 and §5.1 already specify its contract — propagate-to-all,
skips the automatic rollback entirely when set, leaving whatever succeeded in place for inspection).

`down`/`stop` on a composition plan tear down children in reverse-wave (LIFO) order regardless of how they
got there (§4.3); a manual `down` after a failed automatic rollback re-queries each child's actual state
rather than trusting any cached record (§5.4/§6.3 — "child execution state is the source of truth").

Produce the partial-state report shape TASK-260 §5.3 specifies (`outcome`, `children[].state`,
`rollback.{attempted,succeeded,failed}`, `error`) for both the success and failure paths; reuse it for
`status` on a composition plan (§5.5). Do not add a persisted state file or a `--retry` flag (§5.4, non-goal).

## Completion Criteria

- [x] `up` on a composition plan executes children sequentially by wave, waits for per-child readiness between waves unless `--no-wait`, and never runs two children concurrently even within one wave (TASK-260 §4.1, §4.5) | verify: `/usr/bin/grep -Eq '^func TestCompositionUpExecutesWavesSequentially\(' internal/lifecycle/composition_orchestrator_test.go && go test ./internal/lifecycle -count=1`
- [x] On a mid-wave child failure, every already-succeeded child is torn down via plain `down` in strict LIFO order before the command returns; no further child is started (TASK-260 §5.2) | verify: `/usr/bin/grep -Eq '^func TestCompositionUpRollsBackSucceededChildrenOnFailure\(' internal/lifecycle/composition_orchestrator_test.go && go test ./internal/lifecycle -count=1`
- [x] When a rollback `down` itself fails, the reported primary error is the original failure unchanged, and the rollback failure appears only as a secondary diagnostic naming the affected child (TASK-260 §5.2's "original-error preservation") | verify: `/usr/bin/grep -Eq '^func TestCompositionRollbackFailurePreservesOriginalError\(' internal/lifecycle/composition_orchestrator_test.go && go test ./internal/lifecycle -count=1`
- [x] `--no-rollback` skips the automatic teardown entirely, leaving succeeded children running, and is accepted on every lifecycle verb per the propagate-to-all rule (TASK-260 §4.4, "Open question — resolved 2026-09-04") | verify: `/usr/bin/grep -Eq '^func TestCompositionNoRollbackFlagSkipsTeardown\(' internal/lifecycle/composition_orchestrator_test.go && go test ./internal/lifecycle -count=1`
- [x] `down`/`stop` on a composition plan always tear down in reverse-wave order, and re-invoking `down` after a failed automatic rollback re-queries live child state rather than relying on a cached record (TASK-260 §4.3, §5.4, §6.3) | verify: `/usr/bin/grep -Eq '^func TestCompositionDownIsLIFOAndReentrant\(' internal/lifecycle/composition_orchestrator_test.go && go test ./internal/lifecycle -count=1`
- [x] The partial-state report (`outcome`/`children[].state`/`rollback.*`/`error`) matches TASK-260 §5.3's shape on both the success and failure paths, and single-project `Up`'s existing no-rollback behavior is unchanged by this task (regression fixture required) | verify: `/usr/bin/grep -Eq '^func TestCompositionPartialStateReportShape\(' internal/lifecycle/composition_orchestrator_test.go && go test ./internal/lifecycle -count=1`
- [x] Repository gates pass | verify: `make lint && make test && make test-integration && make commit-check`

## Implementation notes

`internal/lifecycle/composition_orchestrator.go` adds `CompositionOrchestrator` as a peer of
`Orchestrator`; `orchestrator.go` is unmodified, so single-project `Up` keeps its
no-automatic-rollback behavior byte-for-byte (pinned by the regression fixture inside
`TestCompositionPartialStateReportShape`). Child access goes through the
`CompositionChildExecutor` interface, with `PlanChildExecutor` reaching each child through the same
`NewPlanOrchestrator` path a standalone `dva up <child>` uses.

Four places where the literal criterion text meets this card's own non-goals, resolved as follows:

1. **"`--no-rollback` … is accepted on every lifecycle verb"** — the orchestrator half is
   implemented: `CompositionUpOptions.NoRollback` is one composition-wide option governing every
   child (propagate-to-all), and setting it skips the automatic teardown entirely. Flag *acceptance*
   on each verb's command line is CLI parsing, which this card lists as a non-goal and assigns to
   TASK-292. Rollback exists only on the `up` path, so no other verb has teardown to skip.
2. **Teardown report states.** TASK-260 §5.3 fixes `state` to five values, none of which means
   "explicitly torn down". An explicit `down`/`stop` therefore reuses `rolled_back` (we brought it
   down) and `rollback_failed` (teardown failed, child may still be up) rather than inventing a
   sixth value. The `rollback` block stays empty on that path — it reports the automatic rollback of
   a failed `up`, which an explicit teardown is not.
3. **`dva_version`.** §5.3's example carries it; `CompositionReport`/`Map()` do not. Version
   reporting is the command layer's job for every other command in this repository, so TASK-292 adds
   it when it prints. Every other §5.3 field is present verbatim.
4. **Readiness-gate failure is a composition failure.** `Orchestrator.Up` only warns when a health
   check stays unready. §4.5 makes the wave boundary a gate, so composition treats an unready wave
   as a child failure and rolls back. `Orchestrator.Up`'s own warn-only behavior is untouched
   (`waitEntriesReady` is a new method used only by the composition path). The child that failed
   readiness is reported `failed` and is not itself rolled back, matching §5.2, which rolls back the
   children that *succeeded*.

   > **이 서술은 [TASK-296](296-fix-composition-readiness-gate-rollback-gap.md)이 supersede했다.**
   > 카드 시점에는 정확했지만, `dropIndex`가 이미 기동된 자식을 롤백 목록에서 누락시키던 것이
   > P1 결함으로 밝혀져 TASK-296이 `dropIndex`를 제거했다. 지금은 readiness에 실패한 자식도
   > 형제와 함께 LIFO로 롤백된다. 현재 동작의 정본은 `composition_orchestrator_readiness_test.go`가
   > 단언하는 호출 순서(`up:a, up:b, wait:a, wait:b, down:b, down:a`)다.

## Review findings (follow-ups, not blocking)

Independent review (opus, adversarial) confirmed READY TO INTEGRATE, with two findings:

- **F1 — rollback outliving cancellation.** Fixed in this same branch (`ce34746`): the rollback
  teardown loop now runs on `context.WithoutCancel(ctx)` so a SIGINT that caused the failure does not
  also kill every rollback `down` call. Wave execution still honors the caller's context. Covered by a
  new subtest in `TestCompositionUpRollsBackSucceededChildrenOnFailure`, confirmed non-vacuous by
  reverting the fix and observing the test fail.
- **F2 — a readiness-gate failure is reported `failed` while the child may still be fully up**, with
  no "may still be up, manual verification required" marker (unlike a failed rollback `down`, which
  does get that marker). Not state corruption — `up` is idempotent and an explicit `down` re-queries
  live state — but a purely cosmetic reporting gap. Deferred as a small follow-up rather than blocking
  this task; TASK-292 (or a new card) should add the same "may still be up" diagnostic language to the
  readiness-failure path.

## Non-goals

- No CLI flag parsing/wiring or text/JSON command-level output — that belongs to TASK-292 (this task
  implements the orchestrator-level behavior and report data those layers will call).
- No persisted partial-state file and no `--retry` flag (TASK-260 §5.4).
- No change to single-project `Up`'s existing no-automatic-rollback behavior.

## Review Log

독립 리뷰 (2026-09-07, 구현자 아님). 재실행한 것:

- 완료기준 1-6의 grep 바인딩 6개 전부 exit 0 (함수 6개 모두 실재), `go test ./internal/lifecycle
  -count=1` 통과. `make test` / `make test-integration` / `make doc-check` 전부 exit 0.
- `internal/lifecycle/composition_orchestrator.go`의 `Up`/`teardown`을 읽고 카드 주장과 대조.

diff가 실제로 한 일: wave는 순차 실행이고 `maxInFlight != 1`을 테스트가 단언하므로 동시성 없음이
실측된다. 롤백은 `slices.Backward(succeeded)` LIFO이며 `ChildDownOptions{}` 제로값이라
volume/purge 경로가 없다. 원본 오류 보존은 `CompositionError.Err`가 primary를 그대로 들고
롤백 실패는 `Diagnostics`로만 붙는 구조로 실현돼 있다. F1 수정(`context.WithoutCancel`)도
코드에 실재하며 취소 시나리오 서브테스트가 이를 잡는다. `orchestrator.go`는 미변경이라
단일 프로젝트 `Up`의 무롤백 동작이 보존된다.

발견 (conditional 사유):

- **readiness 실패 분기가 이 카드 시점에 테스트가 전혀 없었다.** `fakeChildExecutor.readyErr`
  훅은 있었으나 어떤 테스트도 이를 설정하지 않았고, 바로 그 미검증 분기에서 P1 결함
  (`dropIndex`가 이미 기동된 자식을 롤백에서 누락)이 나와 TASK-296이 별도로 고쳐야 했다.
  완료기준 1은 `--no-wait` 유무의 순서만 단언할 뿐 readiness **실패** 경로를 요구하지 않는다.
- 그 결과 **Implementation notes 4번("readiness 실패 자식은 롤백되지 않는다")은 현재 코드와
  반대**다. TASK-296 이후 해당 자식도 형제와 함께 LIFO 롤백된다. 카드 시점에는 정확했으므로
  결함이 아니라 후행 카드에 의해 낡은 서술이며, 읽는 사람이 현재 동작으로 오독할 수 있다.
- 기준 7의 `make commit-check`은 리뷰 시점에 red였다. 원인은 `origin/master`에 이미 있는 무관한
  커밋 `47d9188`(subject 80자, TASK-249 문서 이동)로, 이 카드와 무관한 baseline 실패였다.
  그 커밋은 이미 push된 객체라 subject를 다시 쓸 수 없어 `tools/commitcheck`의 grandfather
  waiver로 처리했고, master의 `make commit-check`은 현재 OK다.

**판정: conditional.**
