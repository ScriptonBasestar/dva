---
id: TASK-422
title: "Record a terminal state on ce task run receipts"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-005](../_archive/issue/005-ce-task-run-receipts-never-reach-a-terminal-state.md)의
운영 증상은 닫혔다. `ce task run-list`는 rc 0으로 살아 있는 실행과 settled
건수를 낸다. 남은 것은 스키마다. raw `executions[]`에는 종단 상태 필드가 없고
`updatedAt`은 `startedAt`과 같다. `states[]`는 그 순간 워크트리가 있는지로
다시 계산되므로, 워크트리가 사라진 뒤의 과거를 보존하지 못한다.
소유는 ce-agent-kit#2.

방향: 영수증이 스스로 DONE 또는 ABORTED를 갖게 하고, `run-finish`와
`run-abort`가 그 필드를 쓴다. 워크트리 부재로 끝남을 추론하지 않는다.

## Completion Criteria

- [ ] run 영수증에 종단 상태 필드가 있고 `run-finish`와 `run-abort`가 그 필드를 쓴다 | verify: human — 상류 스키마와 테스트가 이 카드에 링크된다

## Out of scope

- 이미 충족된 `run-list` 보고. 그 기준은 ISSUE-005를 닫은 이유다.
- 기존 영수증을 소급 수정하는 마이그레이션. 상류가 순서를 고른다.

## Sources

- [ISSUE-005](../_archive/issue/005-ce-task-run-receipts-never-reach-a-terminal-state.md)
