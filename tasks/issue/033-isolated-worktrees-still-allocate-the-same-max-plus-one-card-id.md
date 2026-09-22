---
id: ISSUE-033
title: "Isolated worktrees still allocate the same max-plus-one card id"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: medium
discovered-in: "TASK-407 independent review — AGENTS.md Parallel-safe rule still collides on identical snapshots"
discovered-at: 2026-09-22
ownership: local
created: 2026-09-22
---

## Summary

`AGENTS.md` Parallel-safe card-ID allocation tells each worktree to take
`max(id visible here) + 1` and never guess across worktrees. Two worktrees
cut from the same board snapshot therefore both see the same max and both
claim the same next id. The live guard is post-hoc: `make doc-check` DUP-ID
and the rename procedure in `docs/407-correction-procedure.md`. TASK-407
wrote that convention; it did not make isolated allocators pick different
numbers.

## Reproduction

1. From the same `origin/master` tip, create two worktrees.
2. In each, compute `max(id seen in this tree) + 1` without looking at the other.
3. Both numbers are equal.

## Expected vs Actual

- Expected: two isolated worktrees cannot mint the same new card id, or the
  written rule states that detection (DUP-ID) is the guard rather than
  allocation uniqueness.
- Actual: the rule reads as if isolation prevents collision; isolation plus
  `max+1` is exactly the collision.

## Related

- TASK-407 — tasks/done/407-card-ids-are-allocated-by-scanning-the-board-so-parallel-worktrees-collide.md
- ISSUE-012 — tasks/issue/012-card-ids-are-allocated-by-scanning-the-board-so-parallel-worktrees-collide.md
