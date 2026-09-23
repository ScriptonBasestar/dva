---
id: ISSUE-033
title: "Isolated worktrees still allocate the same max-plus-one card id"
type: bug
status: done
priority: P2
effort: S
exec-tier: standard
severity: medium
discovered-in: "TASK-407 independent review — AGENTS.md Parallel-safe rule still collides on identical snapshots"
discovered-at: 2026-09-22
ownership: local
created: 2026-09-22
resolution: wontfix
resolved-at: 2026-09-23T03:05:28Z
resolution-summary: "Resolved as wontfix."
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

## 2026-09-23 확정 — TASK-407이 정확히 이 이슈가 요구한 대로 고쳤다

이 카드의 "Expected" 두 번째 절("또는 할당 유일성 대신 탐지가 가드라고 규칙이
스스로 적는다")이 정확히 `origin/master` 커밋 `fc7c41ef`("docs(agents): stop
claiming max+1 prevents card-ID collisions")가 한 일이다 — `AGENTS.md`
Parallel-safe 절을 다시 써서 "`max(id)+1` 자체는 같은 스냅샷 충돌을 막을 수
없고, DUP-ID 탐지가 실제 가드"라고 명시했다(같은 수정이 이 세션에서도 독립적으로
재현되어 [[ISSUE-039]]를 낳았을 만큼 두 세션이 같은 결론에 도달했다). 할당기
자체를 충돌-불가로 만드는 것은 여전히 별 설계 과제로 남지만, 이 카드가 실제로
요구한 것(정직한 문서화)은 이미 upstream에 있다. `wontfix`로 닫는다 — 설계
변경 거부가 아니라, 요구된 대안(문서 정정)이 이미 충족됐다는 뜻이다.

## 소유권 — 이 저장소다

충돌은 이 저장소 `AGENTS.md` Parallel-safe 규칙과 `max(id)+1` 할당 관례에서
난다. DUP-ID 가드는 이미 있고, 할당기 자체 수정은 별 설계다. 지금은 승격하지
않는다.

## Related

- TASK-407 — tasks/todo/407-card-ids-are-allocated-by-scanning-the-board-so-parallel-worktrees-collide.md
- ISSUE-012 — tasks/issue/012-card-ids-are-allocated-by-scanning-the-board-so-parallel-worktrees-collide.md
