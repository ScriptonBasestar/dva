---
id: ISSUE-461
title: "bare ce task run-finish fails when it could auto-select the single active task"
type: bug
priority: P3
status: done
severity: low
discovered-in: "2026-10-02 TASK-459 run-finish"
discovered-at: 2026-10-02
ownership: upstream
created: 2026-10-02
upstream-ref: "ce-agent-kit 1f3f9a74be0cbe9cbb9aa8de943331eb05bdac2e (TASK-378). 설치본은 이 개정으로 자동 갱신되지 않는다. 이전 값은 report pending이었다."
resolution: fixed
resolved-at: 2026-10-03T14:12:06Z
resolution-summary: "Resolved as fixed by TASK-484."
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
메시지를 우회할 수 없다. 당시 DVA 측 할 일은 없었고, 기록과 업스트림 보고
대기가 그 시점의 전부였다. 재발 시 메모리
(ce-run-finish-needs-explicit-task-id)의 절차로 우회한다.

## 제안 수정 (upstream)

1. 단일 활성 태스크 자동 선택(가장 명확한 해법), 또는
2. 오류 메시지에 run-status가 알려주는 것과 동일한 nextAction 문자열 포함.

## DVA 측 할 일 (2026-10-02 당시)

당시에는 기록과 업스트림 보고 대기였다. 재발 시 메모리
(ce-run-finish-needs-explicit-task-id)의 절차로 우회한다고 적었다.

## 2026-10-03 상류 재측정 — 당시 재현

ce-agent-kit 현행 소스(설치본 `vcs.revision=f3a8ba78`과 같은 계열)에서 경로가
그대로다. `TaskRunFinish`는 인자가 없으면 빈 task로 `Service.Finish`를 부르고,
`Finish`는 `s.status(ctx, "", …)`가 돌려준 목록 응답(`listResponse`, "task execution
states returned" / "select a task")에 `Execution`이 없으므로 그대로 exit 1을 낸다
(`internal/usecase/taskruntime/service_lifecycle.go`). 단일 ACTIVE 자동 선택이나
오류 안의 `nextAction` 노출은 없다. 상류 카드에도 이 요구를 추적하는 항목이 없다.
당시에는 upstream-waiting이었다.

## 2026-10-03 정본 소스 — TASK-484

위 재측정은 당시 설치본 `vcs.revision=f3a8ba78` 관찰이다. 그 줄에 적힌 HEAD `9b0b0305a553aec3faceeefd12bd6db6fd5a312d`는 당시 값이다.

현재 HEAD는 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`이다. `9b0b0305a553aec3faceeefd12bd6db6fd5a312d`와 `1f3f9a74be0cbe9cbb9aa8de943331eb05bdac2e`는 그 조상이다. bare finish는 같은 소유자의 활성 실행이 하나일 때만 고르고, 0개와 여러 개는 거부한다. 설치 CLI의 프로덕션 워크플로는 실행하지 않았다. 회귀는 실제 OS 프로세스로 돌았다. 이 카드에는 사람 기준 체크박스가 원래 없다.
