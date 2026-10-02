---
id: ISSUE-461
title: "bare ce task run-finish fails when it could auto-select the single active task"
type: bug
priority: P3
status: todo
severity: low
discovered-in: "2026-10-02 TASK-459 run-finish"
discovered-at: 2026-10-02
ownership: upstream
created: 2026-10-02
upstream-ref: "ce-agent-kit (report pending — next upstream wave, ISSUE-454와 동일)"
---

## Summary

활성 태스크가 정확히 하나여도 bare `ce task run-finish`는 실패한다:

```
$ ce task run-finish
READY: task execution states returned
next: select a task
Error: task execution states returned
```

정답은 `ce task run-status <task> --json`의 `nextAction`에만 있다
("run ce task run-finish task-459"). 오류 메시지는 상태가 여럿임을
암시하지만 실제로는 하나뿐이었고, 해결책(명시적 id)을 어디에도
말해주지 않는다. 에이전트 워크플로에서 매 태스크마다 한 번씩 재발하는
마찰이고, 나는 메모리로 우회했는데 이것이야말로 근본원인 해결이
아니라는 사용자 지적으로 발굴된 항목이다.

## Expected vs Actual

- Expected: 활성 태스크가 1개면 자동 선택해 진행하거나, 최소한 오류에
  `run ce task run-finish <task-id>`를 포함한다.
- Actual: "task execution states returned" 한 줄 — 상태 조회 결과를
  재해석해야만 다음 명령을 알 수 있다.

## Reproduction

1. 태스크를 하나 활성 상태로 만든다 (`ce task run-start`).
2. `ce task run-finish`를 인자 없이 실행한다.
3. `Error: task execution states returned`로 실패하는 반면,
   `ce task run-status <task> --json`의 `nextAction`은 정확한 명령을
   알려주는 것을 비교한다.

## 소유권 — upstream (2026-10-02 명시)

런타임이 상태 모호성을 처리하는 위치는 `ce`가 구현한다 — DVA가 오류
메시지를 우회할 수 없다. DVA 측 할 일은 없으며, 기록과 업스트림 보고
대기가 이 카드의 전부다. 재발 시 메모리
(ce-run-finish-needs-explicit-task-id)의 절차로 우회한다.

## 제안 수정 (upstream)

1. 단일 활성 태스크 자동 선택(가장 명확한 해법), 또는
2. 오류 메시지에 run-status가 알려주는 것과 동일한 nextAction 문자열 포함.

## DVA 측 할 일

없다 — 기록과 업스트림 보고 대기. 재발 시 메모리
(ce-run-finish-needs-explicit-task-id)의 절차로 우회한다.
