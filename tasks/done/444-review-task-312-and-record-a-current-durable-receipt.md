---
id: TASK-444
title: "Review TASK-312 and record a current durable receipt"
type: chore
priority: P2
effort: M
exec-tier: strong
quality-review: pass
quality-reviewed-at: 2026-09-27
quality-review-evidence: "Independent final done-review by /root/review_task443_board_refresh passed after the historical ISSUE-001 snapshot was explicitly superseded. The reviewer verified the fresh TASK-312 PASS, preserved 2026-09-14 waiver history, first canonical receipt and final digest/tool-stamp round trip, archive-skip caveat, ISSUE-001 fixed/archive result, README/PLAN consistency, and doc-check/validate/gate outputs."
status: done
created: 2026-09-27
---

## Summary

Migrate archived TASK-312 to the current DVA-owned completion-review contract while treating its old waived review record as historical context, not present-day evidence. Obtain a fresh independent review of the implementation, then record current non-empty review evidence before generating and tracking the canonical receipt. Resolve ISSUE-001 only after an acceptable fresh PASS; otherwise leave it open.

## Completion Criteria

- [x] An independent reviewer rechecks archived TASK-312 against its original acceptance criteria, current implementation, and regression test, then returns a fresh verdict and non-empty evidence; the 2026-09-14 waived rationale remains historical context and is not treated as current proof | verify: human — reviewer evidence names the archived card, source/test paths, check result, and verdict; see Verification evidence
- [x] After an independent PASS, preserve the 2026-09-14 waived rationale in a historical note, then set TASK-312 `quality-review: pass`, the current `quality-reviewed-at`, and non-empty `quality-review-evidence` before `ce task review-receipt` emits the canonical digest; store and link its first JSON receipt under the Git-tracked `tasks/done/evidence/TASK-312/` path without rewriting any pre-existing receipt | verify: human — inspect review-field/history update, evidence-before-receipt ordering, canonical digest, tracked file, and both links
- [x] Run `ce task validate --all` and `ce task gate --json` as board non-regression checks, explicitly noting that archive cards are skipped as history; then rerun `ce task review-receipt` against the final card and compare canonical `reviewed-card-sha256` fields with the tracked JSON | verify: human — validator output, READY gate JSON, and canonical-digest round trip are linked here
- [x] Resolve ISSUE-001 as fixed and archive it after the acceptable PASS; leave it open if the fresh review is not acceptable | verify: human — ISSUE-001 resolution summary cites TASK-444, archived issue path exists, and tracked receipt remains linked

## Verification evidence

- Independent read-only review by `/root/implement_issue001_workbook` returned **PASS** on 2026-09-27. It matched archived [`TASK-312`](../_archive/2026-09/312-dry-run-up-blocks-on-native-health.md)'s original criterion to `internal/lifecycle/orchestrator.go:158-176` and `internal/lifecycle/orchestrator_dry_run_health_unix_test.go:14-56`; historical fix `4f267fc2` is reachable. The reviewer ran `go test ./internal/lifecycle -run '^TestUpDryRunSkipsEntryHealthWait$' -count=1` successfully (0.398s), confirmed the unreachable-HTTP dry-run regression asserts the would-wait output and no health-wait failure, and found no acceptance gap.
- The old 2026-09-14 `waived` verdict, date, and exact rationale remain under TASK-312's **Review history** section. Current frontmatter records the independent PASS, date, and non-empty evidence before receipt generation. This is TASK-312's first receipt; `tasks/done/evidence/TASK-312/` was empty beforehand.
- The source-built CE `v0.8.4-372-gf3cfa169` emitted canonical digest `1eab9b3d14798f6b3ad275caa251bdbcea5a4a2f7132577298f8a85af0b09acf` and revision `f3cfa16980b5cca5497e13ff85605f6cb5ca2ae6`. Re-running `ce task review-receipt` on the final archived card returns the same digest and stamps as [`the tracked receipt`](evidence/TASK-312/done-review-1eab9b3d14798f6b3ad275caa251bdbcea5a4a2f7132577298f8a85af0b09acf.json). The receipt path is excluded from CE's review subject; the non-empty review evidence is included.
- `ce task validate --all` reports **498 valid / 0 invalid** and marks archived TASK-312 skipped as historical; it is a board non-regression check, not validation of TASK-312's archived receipt. `ce task gate --json` returns **READY** at `dba2348b48618e671d133438cec496cc0853e7ab`. The independent test and final-card receipt round trip are the review evidence for TASK-312.
- After the acceptable PASS and receipt round trip, `ce task resolve ... fixed --by TASK-444` recorded `resolution-summary: "Resolved as fixed by TASK-444."`; ISSUE-001 was archived at [`tasks/_archive/issue/001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md`](../_archive/issue/001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md). Its resolution section distinguishes the completed DVA migration from the archived-card validator skip.
