---
id: TASK-423
title: "Keep needs-human cards out of the agent-runnable count"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-006](../issue/006-preflight-reports-needs-human-cards-as-runnable.md)은
그대로다. preflight는 기준과 `verify:`가 있는지만 보고 runnable을 센다.
`needs-human: true`와 `verify: human —`은 그 판정에 닿지 않는다. 사람만 할 수
있는 큐가 READY로 보이고, 그 판정을 믿는 루프는 끝날 곳을 모른다.
소유는 ce-agent-kit#1.

방향: runnable과 agent가 실행할 수 있는 수를 나눈다. 사람 전용 카드를
unrunnable로 합치지 않는다. 사람은 그 카드를 계속 볼 수 있어야 한다.

## Completion Criteria

- [ ] `needs-human: true`인 카드가 agent 실행 가능 수에 들어가지 않는다 | verify: human — preflight JSON이 두 모집단을 구분하고, 사람 전용 카드만 있는 큐에서 agent 실행 가능 수가 0이다

## Out of scope

- DVA 카드에서 `needs-human`을 지워 preflight를 맞추는 일.

## Sources

- [ISSUE-006](../issue/006-preflight-reports-needs-human-cards-as-runnable.md)
