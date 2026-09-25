---
id: TASK-437
title: "Refresh upstream follow-up planning and ownership evidence"
type: chore
priority: P2
effort: M
exec-tier: standard
created: 2026-09-25
status: done
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent final review by /root/review_433_434 on 2026-09-25: board counts, PLAN-011 progress, CE integration, ISSUE-043 resolution, receipts, and gates match; no stale facts remain."
---

## Summary

Refresh the stale upstream follow-up summary without changing PLAN-010's v0.3.0
release scope. Record the live queue and current CE revision, give PLAN-011's
TASK-420–436 and TASK-438–439 children a separate dependency plan (TASK-437 is this
board-maintenance task and remains outside that child set), and correct controller
ownership and issue-promote evidence from current upstream source.

## Completion Criteria

- [x] Board index and PLAN-010 distinguish the actual todo, review, and blocked queues and current CE revisions | verify: human — compare both current-state summaries against ce task list
- [x] A separate plan lists every active upstream follow-up with explicit dependencies and parallel boundaries | verify: human — plan children and order match current task zones
- [x] Unresolved controller ownership and stale issue-promote behavior are recorded without weakening path admission or cleanup guards | verify: human — ISSUE-040 and ISSUE-032 state the current evidence and preserve the existing path and cleanup guards
- [x] Documentation and board gates pass after the update | verify: `make doc-check` (regression-guard)

## Current evidence

- `tasks/README.md` reports PLAN-011's 19 children as todo 0, doing 0, review 0, blocked 6, done 13 (13/19, progress 68); TASK-437 is explicitly outside that count. PLAN-010 is complete at 8/8 with TASK-411 done and ignored runtime residue preserved; `ce task gate --json` is READY at tool `v0.8.4-345-g5d70c9d8`.
- PLAN-011 includes every selected child and orders TASK-420, TASK-438, and dependent TASK-439 in sequence. TASK-438 and TASK-439 are integrated at CE `master`/`origin/master` `71b1d169` and `7eaf596a`; TASK-439 resolved and archived ISSUE-043 after full CI and independent done-review PASS. TASK-424 remains blocked on full upstream validator parity (ISSUE-041), while controller/human/reclamation blockers remain visible.
- ISSUE-040 records that the historical controller owner has no current product/CLI successor. ISSUE-032 and TASK-435 record the corrected source-branch issue-promote behavior and preserved cleanup guard; fix `05b0b4ae` is an ancestor of current CE `master` `7eaf596a`.
- ISSUE-041 records the validator contract gaps found by TASK-424 review; ISSUE-042 records the TASK-438 regression found by independent review and is fixed through TASK-438. ISSUE-043's post-cleanup Worktrunk recovery gap is fixed through TASK-439 and archived. The related tasks and plan rows reflect the current outcomes.
- Ten upstream-fixed issue cards are archived with `status: done`, including resolved ISSUE-008, ISSUE-042, and ISSUE-043; the active follow-up links target these archived locations, while older state paths remain resolvable by card ID.
- `ce task lint --json`: CLEAN; `ce task validate --all`: 39 valid, 0 invalid; `ce task gate --json`: READY (`v0.8.4-345-g5d70c9d8`, revision `5d70c9d84f34574488a6dd80ddd04d1ae2c9442f`). Current global board zones: blocked 6, done 14, review 2, doing 0, todo 0.
- `make doc-check`: PASS; 615 Markdown candidates, 610 checked, 892 links, 0 broken links, 0 status mismatches, and 0 duplicate card IDs. `cilabels`, `flowcheck`, `planprogress`, `yamlcheck`, `changelogcheck`, and `ciparity` all pass. PLAN-011 progress is 13/19 (68), matching its children.
