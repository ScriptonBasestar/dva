---
id: TASK-557
title: "Warn when a subproject target has no dva.yml"
type: feature
priority: P3
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-04T07:09:40Z
quality-review-evidence: tasks/done/evidence/TASK-557/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-557/done-review-da649d7c5654fcca102245a71f7150ba057a391f311c30c1d0730def3adc584f.json
---

# TASK-557: Warn when a subproject target has no dva.yml

## Summary

Validate says nothing when a `subprojects:` target has no dva.yml (observed in sigdock-idp). Source: DVA feedback plan of 2026-10-31, measured on 0.4.2 across 52 projects.

## Completion Criteria

- [x] Validate warns and names the subproject whose target lacks dva.yml | verify: `go test ./internal/config -run 'Test(LoadConfigWithUnimportedMissingSubproject|WarnMissingSubprojectConfigDoesNotLoadChild|LoadConfigWithImportedMissingSubproject)$'`

## Review Attempts

- 2026-11-04T07:09:40Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root/config_missing_and_profiles` (standard). Criterion PASS after the declared guarded test ran with `-count=1 -v`; required named tests executed. No blocking findings; source `9962c14f51e3345fa9d472b19686f96b0882dd85`. Evidence: `tasks/done/evidence/TASK-557/independent-review.json`; review runtime `review-execution-stability-diagnostics` (review). Outcome: pass; next transition: done.
