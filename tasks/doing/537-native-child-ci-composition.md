---
id: TASK-537
title: "Compose opted-in native CI steps across independent roots"
type: feature
priority: P1
effort: L
exec-tier: strong
status: doing
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
- [Evidence](../../docs/ci-child-composition-evidence.md).

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

2026-10-27 implementation progress: strict child_ci presence/type handling,
bounded composition JSON and verified all-ancestor admission, cooperative worker
resolution, Run/dispatch/receipt wiring and scope cancellation are implemented in
the task worktree. The config/cirun package diagnostic passed before the latest
worker provenance changes. Scoped process diagnostics proved A-B and A-B-C,
default nested rejection, indirect cycles, swallowed child failures, inherited
step deadline, SIGINT/SIGTERM cancellation, sibling failure and normal background
child waiting with descendant cleanup. These are partial diagnostics, not final
acceptance or installed feature support.

Independent architecture review accepted atomic scope admission, one live direct
child per step, normal waiting within the original deadline and a separate 15s
cancellation cleanup grace preserving ordinary OCI leaves. Scope metadata binds
the complete issued grant; acquisition/registration share one critical section;
parent observes the final child receipt after root lock release and completion
registration. Helper review caught two corrected blockers: unverified cycle
owner attribution and computed jobs mislabeled as explicit profile settings.

Current evidence and scope limitations are recorded in
[composition evidence](../../docs/ci-child-composition-evidence.md). Production
admission races, worker provenance, budget-aware preview, shared-resource busy,
concurrent-child busy, actual old/new CLI and ordinary OCI success/cancellation
have scoped passing diagnostics. Manifest schema 1.10 has a passing narrow check.

Final full gate 43f5efc5ce3e5fa9bebcc61953f6e273 passed all nine steps in
2m7.726s, with equal before/after inputs. The built binary hash equals the
actual final CLI/OCI evidence. Implementation manifest (34 files) is
e8d45612fd17c7896d4adc26887b7b5615d275c7768c2066cc77b527bb46e2d4.

Remaining: bind independent done-review to the committed implementation,
complete the canonical transition and native source delivery. No installation
or consumer predeploy attempt has been made.

## Attempts

- Full diagnostic 1 failed on the owning CI document's size limit; removed
  duplicated explanation in favor of the canonical contract, then doc-check passed.
- Full diagnostic 2 failed on modernize/unparam findings; preserved semantics
  using CutPrefix and an error-only path validator, then lint passed.
- Full diagnostic 3 passed; the reviewer identified a test setup timing risk,
  resolved with a readiness handshake and a passing focused race diagnostic.
- Final full verification passed after that refinement; see canonical evidence.
