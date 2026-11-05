---
id: TASK-564
title: "Stop reporting parent-owned compose as drift"
type: bug
priority: P3
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-04T07:09:40Z
quality-review-evidence: tasks/done/evidence/TASK-564/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-564/done-review-fbd21a8e34dd2344b69b015fd9560284d989b94a90b21282ea0638fc49c483bf.json
---

# TASK-564: Stop reporting parent-owned compose as drift

## Summary

A compose file owned by a parent project is reported as drift in the child. Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [x] Drift detection does not report a compose file the parent owns, and still reports a real orphan | verify: `go test ./internal/cli -run '^TestDetectConfigDriftWarnings_(Parent|AncestorWithout|UnloadableParent|UnsupportedParent|SymlinkedSubproject)'`

## Review Attempts

- 2026-11-04T07:09:40Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root/parent_compose_drift` (standard). Criterion PASS after the declared guarded test ran with `-count=1 -v`; required named tests executed. No blocking findings; source `9962c14f51e3345fa9d472b19686f96b0882dd85`. Evidence: `tasks/done/evidence/TASK-564/independent-review.json`; review runtime `review-execution-stability-diagnostics` (review). Outcome: pass; next transition: done.
