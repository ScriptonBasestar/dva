---
id: ISSUE-006
title: "Preflight reports needs-human cards as runnable"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: medium
discovered-in: "2026-09-13 task:run-all loop termination"
discovered-at: 2026-09-13
created: 2026-09-13
---

## Summary

`ce task preflight` decides runnability from card structure — criteria present,
`verify:` bindings present, no blocking advisories — and never from whether a
runner exists that can satisfy them. A card whose every remaining criterion is
`verify: human — …` is structurally perfect, so it is reported runnable to an
agent that cannot run it.

DVA's queue is currently entirely this shape. `ce task preflight --zone
doing,todo --json` returns `verdict: READY`, `total: 4`, `runnable: 4`,
`unrunnable: 0`. All four cards carry `needs-human: true` in frontmatter, and
their remaining criteria are human-bound. Agent-runnable work is zero.

The information preflight would need is already in front of it, twice. It reads
the frontmatter, where `needs-human: true` is declared. It parses the `verify:`
bindings, since it counts them (`criteria: 3, bound: 3`) — and the `human —`
prefix is in the text being counted. Neither signal reaches the runnable
verdict, and the per-card record it emits carries no field for either.

Owner is external: preflight belongs to ce-workbook/task_management. Filed here
on the ISSUE-004 precedent. It is closely related to [[ISSUE-004]], which is
the same gap seen from the other side — that card is about admission refusing
human-only work, this one is about runnability admitting it.

## Reproduction

1. Run `ce task preflight --zone doing,todo --json` in this repository.
2. Observe `verdict: READY`, `runnable: 4`, `unrunnable: 0`.
3. `grep -l 'needs-human: true' tasks/todo/*.md` returns all four of those cards.
4. Read the emitted `cards[]` records: no field reports `needs-human` or the
   human binding, though `bound` counts the very lines that carry it.

## Expected vs Actual

- Expected: a card an agent cannot advance is not counted toward the runnable
  total, and the verdict distinguishes "nothing left" from "nothing an agent
  can do".
- Actual: READY with 4 runnable, while agent-runnable work is 0.

## Impact

An automated loop that trusts the verdict — which is the loop's contract with
preflight — cannot terminate on it. It reaches READY, finds no card it can
advance, and must either stop on a judgement preflight did not authorize or
attempt human-only work. This session terminated on its own reading of the
cards instead of on the verdict, which is exactly the coupling preflight exists
to remove.

The alternative failure is worse than a wrong count: a loop that resolves READY
by attempting the work fabricates evidence for a human-bound criterion.

## Recommended Resolution

Report the two populations separately rather than reclassifying. Human-only
cards are not unrunnable — a human runs them, and a human-facing queue should
still list them. Collapsing them into `unrunnable` would trade this defect for
its mirror image. A distinct count (`runnable` vs `agent-runnable`, or an
explicit `needs_human` on each card record) lets each consumer read what it
needs, and lets an agent loop terminate on the verdict.

`needs-human: true` in frontmatter should be the discriminator, since it is
declared rather than inferred; the `verify: human —` prefix is corroborating
evidence and a reasonable source for a warning when the two disagree.

## Resolution Criteria

- [ ] Preflight distinguishes agent-runnable cards from human-only cards in its verdict and its per-card records | verify: human — upstream tests cover a queue that is entirely human-only
- [ ] A DVA preflight run reports agent-runnable 0 for the current four cards | verify: human — fresh output from this repository is linked here
- [ ] An automated loop can terminate on the verdict alone | verify: human — the run-all skill's exit condition reads the new field, not the card bodies
