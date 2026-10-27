---
id: TASK-537
title: "Compose opted-in native CI steps across independent roots"
type: feature
priority: P1
effort: L
exec-tier: strong
status: todo
created: 2026-10-27
---

## Summary

Implement the adopted native-only child CI contract while keeping current nested rejection as the default. DVA owns lifecycle, budgets and result edges; child projects retain their own checks. The source design is docs/ci-child-composition-proposal.md. No release, deployment or OCI child composition is included.

## Completion Criteria

- [ ] Native step opt-in admits independent roots while defaults, cycle checks and old/new compatibility fail visibly. | verify: human — independent review compares contract, schema and admission regressions
- [ ] Parent deadline, cooperative jobs and normal cancellation/descendant cleanup are verified through real process tests without weakening ordinary native or OCI behavior. | verify: human — compare finite integration outputs and cleanup evidence
- [ ] Manifest, dry-run and linked receipts describe actual parent/child results; required checks and source delivery are verified before consumer adoption. | verify: human — compare CLI evidence, checks and native delivery receipt

## Sources

- [ROADMAP](../../ROADMAP.md).
- [Contract](../../docs/ci-child-composition-proposal.md).

## Scope and acceptance

Use the contract acceptance matrix, including exact old/new binary behavior,
all-ancestor admission, canonical root cycles, effective jobs and normal
multi-level cancellation/process-group cleanup. Resolve the lock-store wire
identity in implementation review. Parent group termination alone is not proof.

Preserve OCI's independent deadline/cleanup/resource behavior. Update config,
schema, manifest/dry-run, runtime receipts, docs and canonical skill references
as one coherent source change; regenerate only through the owning generator.
Do not erase parent tokens or copy consumer checks into DVA.

## Execution notes

2026-10-27: user goal delegates clear SOUL/product judgments and continued work.
Main adopted the bounded scope after independent factual exploration and strong
architecture judgment; no current feature support is claimed. This design delivery
only creates the open implementation card; it does not complete it.
