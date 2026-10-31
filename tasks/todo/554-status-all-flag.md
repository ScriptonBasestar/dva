---
id: TASK-554
title: "Reach whole-stack status past default_plan"
type: bug
priority: P2
exec-tier: standard
status: todo
created: 2026-11-01
---

# TASK-554: Reach whole-stack status past default_plan

## Summary

With `default_plan` set, no invocation reached whole-stack status. `dva status --all` does,
and USAGE.md documents it.

## Completion Criteria

- [x] --all reports the whole stack and refuses a NAME | verify: `go test -count=1 -run 'TestStatusAll' -v ./internal/cli 2>&1 | /usr/bin/grep -c -- '--- PASS: TestStatusAll' | /usr/bin/grep -qx 2`
- [x] USAGE.md documents the flag | verify: `/usr/bin/grep -q 'dva status --all' USAGE.md`
