---
id: ISSUE-004
title: "Controller scope admission cannot represent external or human-only cards"
type: bug
status: todo
priority: P1
effort: S
exec-tier: standard
severity: medium
discovered-in: "2026-09-10 direct queue-run preflight"
discovered-at: 2026-09-10
created: 2026-09-10
---

## Summary

The direct queue preflight requires every non-decomposable `todo` and the selected P0 `issue` to
have non-empty `allowed-paths`. The controller rejects `tasks/` paths,
absolute paths, parent traversals, and glob patterns. This is correct for an
executor that modifies product files, but it cannot represent cards whose
truthful work is exclusively external, human-operated, or a controller-owned
state transition.

DVA currently has each shape:

- ISSUE-001 is a P0 external receipt-contract blocker owned by ce-agent-kit
  and ce-workbook/task_management.
- TASK-329 and TASK-348 require external project readiness or a real build.
- TASK-307, TASK-309, TASK-319, TASK-321, and TASK-351 await an explicit
  human design decision.

Adding a synthetic DVA file scope would authorize an implementation the card
does not own. Adding its task-card path is rejected because `tasks/` is
controller-owned. The queue therefore exits 65 after selecting the card but before provider/action
execution or a controller-owned route transition can occur.

## Reproduction

1. Run the bound `queue_preflight` command for this repository.
2. Observe `task-allowed-paths-missing` for ISSUE-001 and active todos.
3. Add a task-card path as a hypothetical scope and observe
   `task-allowed-path-invalid`: `tasks/` is forbidden by
   `entry_selection._allowed_paths()`.

## Expected vs Actual

- Expected: cards that have no repository implementation scope can receive a
  controller-owned external/manual disposition without granting a fake executor
  scope.
- Actual: preflight selects the P0 issue, then rejects it before provider/action
  execution and route transition.

## P1 Blocker

- `reason`: DVA cannot resume its direct lifecycle queue truthfully while the
  scope contract treats external/manual cards as executable implementation
  units.
- `owner`: ce-workbook/task_management owns queue preflight, selection, and
  allowed-path validation. DVA owns only its adopted card states after the
  shared contract supports them.
- `next_action`: define and test an explicit machine-readable disposition-only
  admission contract. It must name the authoritative discriminator, permitted
  controller-owned routes, and fail-closed behavior before it can classify an
  external or human-only card ahead of implementation-scope validation.
- `next_check`: bound `queue_preflight` exits 0 for DVA without permitting
  `tasks/`, absolute paths, parent traversals, or globbed executor scopes.

## Resolution Criteria

- [ ] Preflight distinguishes executor-owned implementation cards from external
  and human-only disposition cards | verify: human — upstream selection tests
  cover P0 issue, external todo, and decision todo shapes
- [ ] A DVA queue run routes ISSUE-001 and the external/manual cards without a
  synthetic product-file scope | verify: human — fresh controller evidence is
  linked here
- [ ] Scope validation remains strict for real implementation cards | verify: human — upstream regression rejects `tasks/`, absolute, parent, and glob paths
