---
id: ISSUE-530
title: "doctor records exit_code -1 without the signal for a killed command"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
severity: low
ownership: local
discovered-in: "v0.4.0 release candidate real-use verification (2026-10-21)"
discovered-at: 2026-10-21
created: 2026-10-21
---

## Summary

When a project `doctor` command is killed by a signal, the JSON result records
`exit_code: -1` with `error_classification: "signal"`, but no signal name or number, and the
finding text reads `command exited non-zero: <command>`. A reader cannot tell which signal
ended the command, and the finding text describes an exit that did not happen.

## 소유권 — 이 저장소다

The finding text and the command evidence come from `internal/cli/doctor_command.go`; its
classification switch has no `signal` case and falls through to the non-zero wording.

## Reproduction

1. Declare a doctor command `kill -TERM $$`.
2. Run `dva doctor --json` with dva 0.4.0 at `6911d747`.
3. Observed: `"finding": "command exited non-zero: kill -TERM $$"`, `"exit_code": -1`,
   `"error_classification": "signal"`, and no signal field.

## Expected vs Actual

- Expected: the result names the terminating signal and the finding says the command was
  terminated by it.
- Actual: `-1` and an "exited non-zero" finding.

## 범위

Only the doctor command result and finding text. Process-group kill on timeout already works
and is out of scope.
