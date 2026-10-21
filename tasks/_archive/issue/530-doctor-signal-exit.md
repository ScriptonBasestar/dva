---
id: ISSUE-530
title: "doctor records exit_code -1 without the signal for a killed command"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
severity: low
ownership: local
discovered-in: "v0.4.0 release candidate real-use verification (2026-10-21)"
discovered-at: 2026-10-21
created: 2026-10-21
resolution: fixed
resolved-at: 2026-10-21T15:55:10Z
resolution-summary: "Resolved as fixed."
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

## 2026-10-22 해결

- `runDoctorCommand`가 신호로 끝난 명령의 wait status에서 종료 신호를 읽어
  `DoctorCommandEvidence.Signal`(JSON `command.signal`)에 `SIGTERM` 같은 이름으로 기록한다.
  이름 표에 없는 신호는 번호로 남긴다. Windows wait status에는 신호가 없으므로 비워 둔다.
- `doctorCommandFinding`에 `signal` 분기를 추가해 finding이 `command terminated by
  SIGTERM: <command>`(신호 이름이 없으면 `command terminated by a signal: <command>`)로
  표시된다. `exit_code: -1`과 `error_classification: "signal"`은 바꾸지 않았고, timeout의
  process group 종료는 범위 밖이라 그대로다. USAGE.md doctor 절과 CHANGELOG Unreleased에
  반영했다.
- 검증: `TestRunDoctorCommandNamesTerminatingSignal`이 재현 절차의 `kill -TERM $$`를
  실행해 `signal: SIGTERM`, `exit_code: -1`, finding 문구, JSON 필드를 확인하고,
  `TestDoctorCommandFindingSignal`이 이름 있음·없음·일반 non-zero 문구를 고정한다.
  `make test`, `make lint`, `make fmt-check`, `make doc-check`가 exit 0이고,
  `GOOS=windows go vet ./internal/cli/`로 Windows 빌드를 확인했다.
