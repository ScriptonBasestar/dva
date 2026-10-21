---
id: ISSUE-527
title: "Composition children drop Compose --wait for entries without DVA health checks"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
severity: medium
ownership: local
discovered-in: "Stale run review of grok's composition-compose-wait execution (2026-10-21)"
discovered-at: 2026-10-21
created: 2026-10-21
resolution: fixed
resolved-at: 2026-10-21T03:48:56Z
resolution-summary: "Resolved as fixed."
---

## Summary

`dva up <composition>` starts every compose entry of a child with `--wait` removed, and the
wave boundary that should replace it waits only for entries that declare DVA
`health_checks`. A compose entry relying on Compose healthchecks alone therefore gets no
readiness gate at all, and the next wave starts while its services may still be starting.
Single-plan `dva up` is not affected: it keeps Compose's default `--wait`.

## 소유권 — 이 저장소다

`PlanChildExecutor.Up` (`internal/lifecycle/composition_orchestrator.go`) passes `Wait: false`
and `waitEntriesReady` in the same file skips entries without health checks. Both are this
repository's code.

The defect was first found by a grok session (execution `composition-compose-wait`, started
2026-09-24), which left an uncommitted fix in its worktree. That fix decided per child (any
entry with health checks turned `--wait` off for the whole child), so an entry without health
checks in a mixed child still lost it. The fix here decides per entry instead.

## Reproduction

1. Declare a composition whose order-0 child has a compose entry with no `health_checks`,
   and an order-1 child that depends on it.
2. Run `dva --dry-run up <composition>` and read the compose command logged for the entry.
3. Observed on master `dd499efd`: `up -d <service>`, without `--wait`. The wave boundary
   (`WaitReady`) returns at once for that entry because it has no health checks.

## Expected vs Actual

- Expected: an entry without DVA health checks keeps Compose's `--wait`, unless the
  composition runs with `--no-wait`. An entry with health checks keeps `--wait` off, because
  the wave boundary gates it.
- Actual: every entry of a composition child runs without `--wait`.

## 범위

Only the readiness of composition `up`/`restart` children. The single-plan path, the DVA
health-check gate itself, and a `runners.compose.up_options` override are out of scope.

## 2026-10-21 해결

- `UpOptions.ComposeWaitWithoutHealthChecks`(`internal/lifecycle/orchestrator.go`)를 추가해
  `Wait`가 false여도 DVA health check가 없는 entry는 Compose `--wait`를 유지한다. 판단은
  entry 단위이므로 health check가 섞인 child에서도 health check 없는 entry가 readiness
  신호를 잃지 않는다. 인라인 DVA health wait는 여전히 `Wait`에만 따른다.
- `PlanChildExecutor.NoWait`로 composition의 `--no-wait`를 전달해, 사용자가 대기를 끈 경우
  위 예외도 꺼진다. `newCompositionExecutor`가 `up`/`restart` 경로의 `flags.noWait`를 넘긴다.
- 검증: `TestCompositionUpKeepsComposeWaitOnlyForEntriesWithoutHealthChecks`
  (`internal/lifecycle/composition_compose_wait_test.go`)가 health check 없음·혼합 child·
  `--no-wait` 세 경우의 dry-run compose 명령을 확인한다. 수정을
  `ComposeWaitWithoutHealthChecks: false`로 되돌리는 변이에서 앞의 두 경우가 실패함을
  확인했다.
