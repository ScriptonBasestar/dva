---
id: TASK-430
title: "Stop done-finalize from treating evidence paths as ownership"
type: bug
priority: P2
effort: M
exec-tier: strong
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-024](../issue/024-done-finalize-blocks-cards-that-name-artifacts-in-review-evidence.md)는
그대로다. `done-finalize`의 ownership 정합성 검사가 증거에 적힌 저장소 경로와
`depends-on` 같은 구조 참조를 소유 위반으로 읽는다. TASK-410의 증거 경로와
TASK-411의 `depends-on: [TASK-410]`이 같은 방식으로 다시 막혔다. 증거와
참조를 지워 통과시키는 것은 봉인 카드를 고치는 일이다. 소유는 ce-agent-kit#4.

방향: 정합성 검사가 전용 바인딩 필드만 보게 한다. 증거 산문과 정당한 태스크
참조는 검사 대상이 아니다.

## Completion Criteria

- [ ] 증거에 저장소 경로를 적은 카드의 `done-finalize --dry-run`이 ownership reconciliation으로 BLOCKED되지 않는다 | verify: human — ce-agent-kit cleanup 검사가 전용 바인딩만 보는지, 그리고 이 저장소 dry-run이 그 사유로 막히지 않는지 확인한다

## Out of scope

- 막힌 카드의 증거 문장이나 `depends-on`을 지우는 일.

## Sources

- [ISSUE-024](../issue/024-done-finalize-blocks-cards-that-name-artifacts-in-review-evidence.md)
