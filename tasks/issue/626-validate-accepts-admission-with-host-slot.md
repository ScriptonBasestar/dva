---
id: ISSUE-626
title: "dva validate accepts profile admission combined with step host_slot"
type: bug
priority: P3
effort: S
status: todo
severity: low
discovered-in: "ce-devenv TASK-147 session 22aadb92: host_slot adoption analysis"
discovered-at: 2026-11-19
created: 2026-11-19
ownership: local
---

## Summary

A profile that declares `admission:` and has a step with `host_slot: true` passes
`dva validate`, but every `dva ci <profile>` run of it fails at start with
`profile admission and host_slot cannot be combined`
(`internal/cirun/host_slot_bridge.go`, `resolveHostSlotBridge`). The run-time mutex is
intended (TASK-621), but static validation does not report it, so a config stays green
under validation while it cannot run, even with no `DVA_RESOURCE_POLICY` set.

## Reproduction

1. In an empty Git repository, write `dva.yml`:

   ```yaml
   ci:
     profiles:
       dev:
         admission: {mode: when-policy-configured, class: heavy, priority: interactive, wait_timeout: 1m}
         timeout: 1m
         steps:
           - name: s
             run: "true"
             host_slot: true
   ```

2. Run `dva validate`, then `dva ci dev`, with `DVA_RESOURCE_POLICY` unset.

## Expected vs Actual

- Expected: `dva validate` rejects the profile with the same reason the run gives.
- Actual: `✅ dva.yml is valid` (rc 0), then `dva ci dev` fails with
  `ERROR: profile admission and host_slot cannot be combined`.

## Evidence

- Observed 2026-11-19 with dva built from master `f8373533` (`make build`,
  `dva version 0.5.2`); validate rc 0, run error as quoted.
- Fix direction (not presumed final): enforce the rule at the config-validation owner so
  `validate`, schema checks and the runner share one predicate; keep the run-time check.
