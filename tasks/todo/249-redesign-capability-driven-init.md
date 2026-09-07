---
id: TASK-249
title: "Redesign init around verified capabilities instead of a fixed three-plan template"
type: chore
priority: P1
effort: M
exec-tier: strong
created-at: 2026-09-01T19:25:00+09:00
source: "PLAN-002 tracked D8-compatible scaffold ruling"
scope: "init discovery contract, capability preset integration, plan naming/default rules, human-agent parity, census ownership"
status: todo
needs-human: true
decision-status: decided
decided-at: 2026-09-03T21:35:00+09:00
---

# Task 249: redesign capability-driven init

## Summary

Replace the rejected fixed three-plan template with an evidence-driven generation contract aligned
with D8 and the repository's capability preset policy. The design decision and the rejected baseline
live in [docs/58](../../docs/58-capability-driven-init-design.md); the TASK-233 conflict analysis is
in docs/58 §4-5; the criterion-by-criterion verification against TASK-250's implementation is in
[docs/59](../../docs/59-capability-driven-init-verification.md); the label/evidence inventory is in
[docs/60](../../docs/60-capability-driven-init-label-inventory.md). This card only tracks what remains
open.

## Status

9 of the original 10 completion criteria are decided and verified against TASK-250's implementation;
see [docs/59 §2](../../docs/59-capability-driven-init-verification.md#2-완료기준-매핑) for the full
mapping. Only census ownership (criterion below) is still open. This card stays in `todo/` until that
is answered — `decision-status: decided` covers the design direction (docs/58 §5), not this card's
completion.

## Completion Criteria

- [ ] Define census owner, canonical repository IDs/revisions, input inventory, cadence, and the change threshold that can revise defaults | verify: human — a bare count without revision is insufficient

## Non-goals

The following are out of scope for this card because they are already decided and verified — see
[docs/59 §1](../../docs/59-capability-driven-init-verification.md#1-task-250-대조-검증-2026-09-04)
for the record and evidence of each:

- Discovery evidence/output contract for compose-only, native-only, hybrid, and no-discovery fixtures
  (docs/59 §1 기준 1).
- Reusing the capability-driven preset policy so generation omits plans lacking evidence (docs/59 §1
  기준 2).
- Separating human-facing example names from verified provider facts across the repository's preset/
  flow/generated-library projections ([docs/60](../../docs/60-capability-driven-init-label-inventory.md)).
- Keeping `local-infra`, `local-dev`, `full-stack` out of Go `init` generator defaults while coexisting
  with TASK-233's `am` preset corpus decision (docs/58 §5, docs/59 §1 기준 4).
- Single-plan implicit default vs. explicit `default_plan` for multi-plan output (docs/59 §1 기준 5).
- No-overwrite, preview/dry-run, idempotence, and invalid partial-discovery behavior (docs/59 §1 기준 6).
- One canonical generator/preset shared by human `init` and agent workflows (docs/59 §1 기준 7).
- Backward-compatibility matrix for the five templates, four flags, `config init`, and the top-level
  `init` alias (docs/59 §1 기준 8).
- Recording the selected contract and rejected alternatives (docs/58, this card's own record).

## Related

- Design and decision record: [docs/58-capability-driven-init-design.md](../../docs/58-capability-driven-init-design.md)
- Verification against implementation and criterion mapping: [docs/59-capability-driven-init-verification.md](../../docs/59-capability-driven-init-verification.md)
- Label/evidence inventory: [docs/60-capability-driven-init-label-inventory.md](../../docs/60-capability-driven-init-label-inventory.md)
- Implementation: [TASK-250](../done/250-implement-capability-driven-init.md) (`status: done`, commit `4cc0fdc`)
- Parent plan: [PLAN-002](../plan/002-command-surface-delivery.md)
- Superseded-surface note: [TASK-233](../_archive/233-capability-driven-plan-presets.md) (`am` preset
  corpus default — coexists with this card's Go `init` generator scope; see docs/58 §5)
