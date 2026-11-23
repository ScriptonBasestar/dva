---
id: TASK-632
title: "status: do not report escaped after dva stop when another process takes the port"
type: bug
priority: P3
effort: S
exec-tier: standard
status: done
delivered-by: 5f249f27
quality-review: pass
quality-review-evidence: "ce review-evidence 061dd6064a592c71233b63b9558d6bec4aeb5c6dc8b20b6118c620033590f0b5"
created: 2026-11-22
allowed-paths: [internal/lifecycle/, USAGE.md, CHANGELOG.md, tasks/]
---

## Summary

TASK-629 (3a3c64a6) reports a process entry as `error`/escaped when its pidfile pid is dead
but a health address answers. `haltProcess` keeps the pidfile after `dva stop`, so if the user
then runs the same server by hand (or anything else takes the port), `status` says the entry
escaped and advises stopping it by PID — a process dva never started. Found by the TASK-629
independent review.

## Completion Criteria

- [x] After `dva stop`, an unrelated listener on the entry's health port does not produce an escaped report. | verify: human — internal/lifecycle에 추가한 stop 후 테스트 통과를 독립 리뷰가 확인
- [x] A server that daemonized without a stop is still reported as escaped. | verify: `go test ./internal/lifecycle -run Escaped -count=1`

## Notes

- Options from review: record a halted marker in `haltProcess` and skip the check after it,
  or soften the wording to "something answers on X; it may be an escaped server".

## Outcome

- 첫 선택지(정지 표식)를 택했다. `haltProcess`가 프로세스를 직접 종료했을 때만 `<name>.halted`를
  남기고, `Status`의 stopped 분기는 표식이 있으면 escaped 검사를 건너뛴다. 이미 죽은 pid를 stop한
  경우에는 표식을 남기지 않아, 그 전에 빠져나간 서버는 계속 escaped로 보고된다.
- 표식은 `startLocalProcess`가 새 프로세스를 추적할 때와 `removeEntryState`(down)에서 지운다.
- 테스트: `TestStatusNotEscapedAfterStop`, `TestStatusEscapedAfterRestartClearsHalted`,
  `TestStopOfDeadPidDoesNotMarkHalted`(기존 Escaped 테스트 유지). 변이 4종(skip 제거, mark 제거,
  start에서 clear 제거, 죽은 pid에도 mark) 모두 테스트가 실패했다.

## Review Attempts

### 1. 2026-11-24 — reviewer: 독립 opus 서브에이전트(읽기 전용), executor: 구현 실행(5f249f27)

- 기준 전부 pass. 근거: `ce review-evidence 061dd606…`.
- 리뷰어가 stop 시간 초과·죽은 pid·Restart·다른 starter(tilt, sam, serverless)·instance 경로를 확인하고, 자체 변이 3종이 모두 테스트에 잡힘을 확인했다.
- 비차단 지적: stop 뒤 `startLocalProcess`가 실패하면 표식이 남는다(dva가 시작하지 않은 프로세스의 escaped 보고만 가려 무해). stop 시간 초과 경로는 테스트가 없다(단순 조기 반환).
- 다음 전환: done.
