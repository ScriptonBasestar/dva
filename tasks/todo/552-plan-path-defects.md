---
id: TASK-552
title: "Pass plan compose profiles to down and stop"
type: bug
priority: P2
exec-tier: standard
status: todo
created: 2026-11-01
---

# TASK-552: Pass plan compose profiles to down and stop

## Summary

`dva down <plan>` and `dva stop <plan>` dropped the plan's profiles, so compose ignored
profile-gated services and exited 0 with their containers still running.

## Completion Criteria

- [x] Teardown argv carries the plan profiles | verify: `go test -count=1 -run 'TestPlanTeardownArgvActivatesThePlanProfiles' -v ./internal/cli 2>&1 | /usr/bin/grep -c -- '--- PASS: TestPlanTeardownArgvActivatesThePlanProfiles/' | /usr/bin/grep -qx 2`
