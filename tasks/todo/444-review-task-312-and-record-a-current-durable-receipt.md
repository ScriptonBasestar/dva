---
id: TASK-444
title: "Review TASK-312 and record a current durable receipt"
type: chore
priority: P2
effort: M
exec-tier: strong
status: todo
created: 2026-09-27
---

## Summary

Migrate archived TASK-312 to the current DVA-owned completion-review contract while treating its old waived review record as historical context, not present-day evidence. Obtain a fresh independent review of the implementation, then record current non-empty review evidence before generating and tracking the canonical receipt; keep ISSUE-001 open if the review is not acceptable.

## Completion Criteria

- [ ] An independent reviewer rechecks archived TASK-312 against its original acceptance criteria, current implementation, and regression test, then returns a fresh verdict and non-empty evidence; the 2026-09-14 waived rationale remains historical context and is not treated as current proof | verify: human — reviewer evidence names the archived card, source/test paths, check result, and verdict
- [ ] After an independent PASS, preserve the 2026-09-14 waived rationale in a historical note, then set TASK-312 `quality-review: pass`, the current `quality-reviewed-at`, and non-empty `quality-review-evidence` before `ce task review-receipt` emits the canonical digest; store and link its first JSON receipt under the Git-tracked `tasks/done/evidence/TASK-312/` path without rewriting any pre-existing receipt | verify: human — inspect review-field/history update, evidence-before-receipt ordering, canonical digest, tracked file, and both links
- [ ] Run `ce task validate --all` and `ce task gate --json` as board non-regression checks, explicitly noting that archive cards are skipped as history; then rerun `ce task review-receipt` against the final card and compare canonical `reviewed-card-sha256` fields with the tracked JSON | verify: human — validator output, READY gate JSON, and canonical-digest round trip are linked here
