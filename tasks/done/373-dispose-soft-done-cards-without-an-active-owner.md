---
id: TASK-373
title: "Review soft-done cards without an active owner"
type: chore
priority: P2
effort: M
exec-tier: standard
status: done
needs-human: false
created: 2026-09-10
quality-review: pass
quality-review-date: 2026-09-11
quality-review-session: 1438093b-d243-4c4f-8985-48682b9b9802
source: "TASK-367 2026-09-10 done-backlog inventory — six independent cards have no disposition owner"
---

## Summary

Six soft-done cards have no plan that will review and dispose them: ISSUE-002 plus TASK-341,
352, 355, 369, and 372. PLAN-008 remains the canonical parent of TASK-358 through TASK-365,
even though it has completed all eight children, so this card does not take their ownership.
Leaving the six independent cards in `tasks/done/` recreates the backlog that PLAN-007 was
meant to prevent.

## Work

Independently review each listed card against its acceptance criteria and current repository
state. Record `quality-review: pass`, `conditional`, `waived`, or route a failed card back to
`todo`/`issue`. A waiver must state that card's specific reason in `quality-review-evidence`.
After the controller's disposition path is available, archive reviewed cards; do not fabricate
the receipt contract tracked by ISSUE-001.

## Scope

- ISSUE-002, TASK-341, 352, 355, 369, 372

## Completion Criteria

- [x] Every scoped card has a quality-review verdict or has left `tasks/done/` | verify: human — inspect the six scoped paths and their frontmatter
- [x] Every waived verdict carries card-specific quality-review evidence | verify: human — inspect each waived scoped card
- [x] A failed review is routed to `todo` or `issue` with an actionable finding | verify: human — inspect any non-pass/conditional outcome
- [x] Documentation gate passes | verify: `make doc-check` (regression-guard)

## Notes

- TASK-367 retains ownership of PLAN-007 descendants. PLAN-006, PLAN-008, and PLAN-009 retain
  their children.
- ISSUE-001 blocks durable receipt issuance for legacy cards; it does not authorize inventing
  evidence or calling an unreviewed card reviewed.
