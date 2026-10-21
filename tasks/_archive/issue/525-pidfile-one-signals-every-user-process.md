---
id: ISSUE-525
title: "A pidfile holding 1 makes dva stop signal every process the user owns"
type: bug
priority: P2
effort: S
exec-tier: standard
status: done
severity: medium
ownership: local
discovered-in: "ISSUE-515 test cleanup fix review (2026-10-21)"
discovered-at: 2026-10-21
created: 2026-10-21
resolution: fixed
resolved-at: 2026-10-21T03:21:11Z
resolution-summary: "Resolved as fixed."
---

## Summary

`signalableProcessGroupPID`는 `pid > 0`만 확인한다(`internal/lifecycle/process_group_contract.go`).
그래서 pidfile 값이 1이면 그 값이 그대로 `terminateProcessGroup(pid)`에 전달되고, 이 함수는
`syscall.Kill(-1, SIGTERM)`을 보낸다. `kill(-1)`은 호출한 사용자가 신호를 보낼 수 있는 모든
프로세스에 신호를 보낸다. 손상되거나 손으로 고친 pidfile 하나 때문에 `dva down`/`stop`이
사용자의 다른 프로세스까지 종료시킬 수 있다. `TestSignalableProcessGroupPID`는 pid 1이 신호
대상이라는 동작을 오히려 고정하고 있다.

## 소유권 — 이 저장소다

pidfile을 해석하고 프로세스 그룹에 신호를 보내는 코드는 DVA `internal/lifecycle`이 소유한다.

## Reproduction

1. native mode health check나 tilt background 항목의 pidfile
   (`.dva/pids/<name>.pid`)에 `1`을 쓴다.
2. 그 항목을 멈추는 `dva` 명령을 실행한다. 이 명령은 `orchestrator_mode.go`의 health check
   정지 경로, 또는 `tilt.go`의 `stopBackgroundProcess`를 탄다.
3. 두 경로 모두 `signalableProcessGroupPID(1)`이 true라서 `terminateProcessGroup(1)`,
   곧 `kill(-1, SIGTERM)`까지 간다. 이 판단은 코드 읽기로만 확인했고 실행으로 재현하지는 않았다.
   재현할 때는 버려도 되는 사용자나 컨테이너에서 한다.

## Expected vs Actual

- Expected: pid ≤ 1이나 자기 프로세스 그룹은 신호 대상에서 거부하고, 오류로 드러낸다.
- Actual: pid 1을 신호 대상으로 받아들여 `kill(-1)`을 보낸다.

## 범위

`signalableProcessGroupPID`의 경계, 그 테스트, 두 호출 경로의 오류 처리를 고친다. ISSUE-515의
hosted 정지는 다른 원인으로 확인됐다(그 카드의 2026-10-20 그룹 신호 진단). 이 카드는 그
조사에서 기각된 가설의 실제 결함 부분만 다룬다. 테스트 cleanup의 같은 경계는
`internal/lifecycle/process_group_kill_unix_test.go`에 이미 들어갔다.

## 2026-10-21 해결

- 판정을 모든 호출부가 신호 전에 거치는 순수 함수 `processGroupPIDError`(`internal/lifecycle/process_group_contract.go`)로 모았다. pid 1과 DVA 자신의 process group은 모든 플랫폼에서 `errUnsafeProcessGroup` 오류가 된다. 그래서 stop·down·status, orchestrator health-check stop(dry-run 포함), sam·serverless·tilt가 모두 신호 없이 오류를 돌려주고 pidfile을 남긴다.
- unix `terminateProcessGroup`은 require를 거치지 않는 경로(시작 직후 정리, tunnel)를 위한 마지막 방어로 `pid <= 1`과 자기 group을 다시 거부한다.
- pid 0과 음수는 이전처럼 "프로세스 없음"으로 다루며 신호를 보내지 않는다. 오류로 바꾸지 않은 이유는 이 값이 kill 대상이 되지 않기 때문이다.
- 검증: `TestProcessGroupPIDError` 판정표, `TestTerminateProcessGroupRefusesUnsafePIDs`, pidfile `1`로 Stop/Down/Status를 실행하는 `TestRetainedPIDOneIsReportedNotSignalled`, make gate 전부, `GOOS=windows|linux go vet`.
- 남은 개선 여지(차단 아님): 오류 메시지에 pidfile 경로가 없어 사용자가 항목 이름으로 파일을 찾아야 한다.
