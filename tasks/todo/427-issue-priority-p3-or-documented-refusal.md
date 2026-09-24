---
id: TASK-427
title: "Give issue cards a P3 priority or a documented refusal"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-019](../issue/019-issue-cards-cannot-express-p3-while-task-cards-can.md)는
그대로다. issue의 `priority`는 P0–P2만 허용하고 task는 P3도 허용한다.
급하지 않은 이슈를 적을 칸이 없어 severity로 새고, 두 필드가 서로 다른 축을
재게 된다. 거부는 `canonical_validator.go`의 issue enum이다. 소유는
ce-agent-kit#3.

방향: issue도 P3를 허용하거나, 거부 이유를 스키마 문서에 한 줄로 고정한다.
이 저장소에 priority 우회 검사를 만들지 않는다.

## Completion Criteria

- [ ] issue 카드가 P3를 허용하거나, 거부 이유가 상류 스키마 문서에 있다 | verify: human — `canonical_validator.go`의 issue priority enum 또는 그 문서가 이 카드에 링크된다

## Out of scope

- 이 보드의 기존 이슈를 P3로 일괄 변경하는 일. 어휘가 열린 다음에 한다.

## Sources

- [ISSUE-019](../issue/019-issue-cards-cannot-express-p3-while-task-cards-can.md)
