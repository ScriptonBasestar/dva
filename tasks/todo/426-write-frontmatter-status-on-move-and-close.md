---
id: TASK-426
title: "Write frontmatter status when a card changes zone or closes"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-013](../issue/013-ce-task-move-does-not-sync-the-frontmatter-status-field.md)은
그대로다. `ce task move`는 본문 Status 셀만 고친다. `resolve`와 `archive`도
frontmatter `status:`를 쓰지 않는다. 2026-09-24에 ISSUE-014를 닫자 카드는
`status: todo`인 채로 아카이브되었고, 아카이브 존은 `done`만 허용해서 그
필드를 손으로 맞췄다. 소유는 ce-agent-kit#4.

방향: 존을 바꾸거나 이슈를 닫는 명령이 frontmatter `status:`를 도착 상태로
쓴다. 본문 셀이 없는 카드도 포함한다.

## Completion Criteria

- [ ] `move`, `resolve`, `archive`가 도착 상태의 frontmatter `status:`를 쓴다 | verify: human — 본문 Status 셀이 없는 카드에 대한 상류 테스트가 이 카드에 링크된다

## Out of scope

- DVA doccheck의 zone 표를 이 명령의 대용으로 넓히는 일. 그 순서는 TASK-424가 소유한다.

## Sources

- [ISSUE-013](../issue/013-ce-task-move-does-not-sync-the-frontmatter-status-field.md)
