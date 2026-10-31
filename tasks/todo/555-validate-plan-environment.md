---
id: TASK-555
title: "Fail validation on an undeclared plan environment"
type: bug
priority: P2
exec-tier: standard
status: todo
created: 2026-11-01
---

# TASK-555: Fail validation on an undeclared plan environment

## Summary

Validate accepted a plan naming an environment its owner does not declare, and every verb
then refused that plan at resolve time.

## Completion Criteria

- [x] Validate refuses the plan, checking the plan owner | verify: `go test -count=1 -run 'TestValidatePlanEnvironments|TestValidateFailsOnUnknownPlanEnvironment' -v ./internal/config 2>&1 | /usr/bin/grep -c -- '--- PASS' | /usr/bin/grep -qx 3`
