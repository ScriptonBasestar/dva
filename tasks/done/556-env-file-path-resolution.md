---
id: TASK-556
title: "Expand env_file paths and fail on unresolved references"
type: fix
priority: P2
exec-tier: standard
status: done
created: 2026-11-01
quality-review: pass
quality-reviewed-at: 2026-11-01
quality-review-evidence: tasks/done/evidence/TASK-556/independent-review.json
---

# TASK-556: Expand env_file paths and fail on unresolved references

## Summary

env_file paths were opened literally, so a `${VAR}` selector on an optional entry was silently
skipped. Paths now expand first; an unresolved reference is an error regardless of `required`,
reported by the runtime, doctor and `validate --strict`. Unresolved `${X}` in values keeps its
runtime behaviour and only fails `doctor --strict`.

## Completion Criteria

- [x] Paths expand and unresolved paths fail closed | verify: `go test -count=1 -run 'TestEnvFilePathExpandsAgainstEnvironment|TestEnvFileUnresolvedPathFailsRegardlessOfRequired|TestEnvFileValueReferenceKeptVerbatimAndCounted|TestValidateWarnsOnUnresolvedEnvFilePath|TestEnvFilePathNestedDefaults' -v ./internal/config 2>&1 | /usr/bin/grep -c -- '^--- PASS' | /usr/bin/grep -qx 5`
- [x] Doctor rows are advisory and fail under --strict | verify: `go test -count=1 -run 'TestDoctorEnvFileUnresolvedRows|TestValidateStrictFailsOnUnresolvedEnvFilePath' -v ./internal/cli 2>&1 | /usr/bin/grep -c -- '^--- PASS' | /usr/bin/grep -qx 2`
- [x] USAGE.md documents the error | verify: `/usr/bin/grep -q 'unresolved_path' USAGE.md`
