---
id: TASK-573
title: "Warn when an encrypted env_file target is missing"
type: bug
priority: P2
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-573/independent-review.json
---

# TASK-573: Warn when an encrypted env_file target is missing

## Summary

An optional `env_file` entry with `sops_source` was skipped silently when its target was absent.
In flow-taskchain that started postgres clients with the wrong password and no signal. `dva up`
now warns and `dva doctor` reports a row; neither changes loading.

## Completion Criteria

- [x] Composition warns for the child owner | verify: `go test -count=1 -v ./internal/cli -run '^TestCompositionWarnsForChildOwner$' | /usr/bin/grep -q '^--- PASS: TestCompositionWarnsForChildOwner'`
- [x] Absent, present and plain cases pass | verify: `go test -count=1 -v ./internal/cli -run '^TestDoctorEnvUnsealed$' | /usr/bin/grep -q '^--- PASS: TestDoctorEnvUnsealed'`
- [x] Every up route warns once | verify: `go test -count=1 -v ./internal/cli -run '^TestUpWarnsOnEveryRouteWhenTargetMissing$' | /usr/bin/grep -q '^--- PASS: TestUpWarnsOnEveryRouteWhenTargetMissing'`
