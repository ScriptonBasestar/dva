---
id: TASK-553
title: "Emit JSON from status on the plan path"
type: bug
priority: P2
exec-tier: standard
status: todo
created: 2026-11-01
---

# TASK-553: Emit JSON from status on the plan path

## Summary

`dva status --json` printed the human table when a plan was routed.

## Completion Criteria

- [x] Plan-path status stdout parses as JSON | verify: `go test -count=1 -run 'TestStatusPlanPathEmitsJSON' -v ./internal/cli 2>&1 | /usr/bin/grep -q -- '--- PASS: TestStatusPlanPathEmitsJSON'`
