---
id: ISSUE-448
title: "planprogress fails after the last live plan is archived"
type: bug
priority: P1
effort: XS
status: done
quality-review: pass
quality-review-evidence: "Independent review /root/planprogress_review PASS after focused tests, make doc-check, ce task validate, and dva ci commit run 5e1f0ebdd11c63b942a5c11b0b4cf48c (6/6)."
severity: medium
discovered-in: "2026-09-28 ISSUE-004 status update validation"
discovered-at: 2026-09-28
ownership: local
created: 2026-09-28
resolution: fixed
resolved-at: 2026-09-28T03:10:55Z
resolution-summary: "Resolved as fixed."
---

## Summary

All plan cards now live under `tasks/_archive/plan/`, so `tasks/plan/` does not
exist. `make doc-check` fails at `go run ./tools/planprogress` with
`read tasks/plan: open tasks/plan: no such file or directory`. The same failure
reproduces on clean master; the ISSUE-004 documentation change did not cause it.

## Reproduction

1. Confirm `tasks/plan/` is absent after PLAN-010 and PLAN-011 were archived.
2. Run `make doc-check` on clean master.
3. Observe the `planprogress: read tasks/plan` error and exit 2.

## Expected vs Actual

- Expected: zero live plans produces `plans_checked: 0` and a successful document gate.
- Actual: `os.ReadDir` returns `ENOENT`, which aborts the gate before it can
  report the valid empty plan set.

## 소유권 — 이 저장소다

The failing `tools/planprogress` command and its `make doc-check` invocation are
both owned by DVA. No upstream tool or external board transition is required.

## Resolution

- [x] An absent live-plan directory is a valid empty plan set | verify: human — run the focused planprogress regression and confirm `plans_checked: 0`
- [x] Other directory read failures remain visible | verify: human — inspect the focused regression for non-ENOENT errors
- [x] The repository document gate passes | verify: `make doc-check`

The fix belongs to `tools/planprogress`. No empty placeholder plan card is needed.

## Verification (2026-09-28)

The focused tests cover absent `tasks/plan` and an existing regular file that
must still return `ENOTDIR`. `make doc-check` reports `plans_checked: 0` and exits
zero. Independent review PASS; `dva ci commit` run
`5e1f0ebdd11c63b942a5c11b0b4cf48c` passed format, docs, vet, lint, test,
and build with unchanged input attestation.
