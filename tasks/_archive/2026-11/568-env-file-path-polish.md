---
id: TASK-568
title: "Name the real fragment in env_file path diagnostics"
type: bug
priority: P3
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-568/independent-review.json
---

# TASK-568: Name the real fragment in env_file path diagnostics

## Summary

Closes the non-blocking findings of TASK-566 and TASK-567. Diagnostics name the empty prefix,
keep nested unsupported references whole and list an unset reference next to an unsupported one.
The default rule has one owner, and USAGE states the expansion forms and remaining limits.

## Completion Criteria

- [x] Diagnostics name the real fragment | verify: `go test -count=1 -run 'TestUnresolvedPathRefsNamesTheCause' -v ./internal/config 2>&1 | /usr/bin/grep -c -- '^--- PASS' | /usr/bin/grep -qx 1`
- [x] Empty root fails through ApplyEnvFiles | verify: `go test -count=1 -run 'TestEnvFilePathUnsupportedOrEmptyIsUnresolved' -v ./internal/config 2>&1 | /usr/bin/grep -c -- '^--- PASS' | /usr/bin/grep -qx 1`
