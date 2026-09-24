---
id: TASK-428
title: "Count Resolution Criteria in preflight"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-020](../issue/020-preflight-counts-zero-criteria-on-a-card-whose-criteria-live-under-resolution-criteria.md)은
재현된다. 2026-09-24 `ce task preflight --zone issue`는 Resolution Criteria만
있는 이슈를 `criteria: 0`, `bound: 0`으로 셌다. 인식하는 제목은 Completion
Criteria, Acceptance Criteria와 한국어 완료 조건·완료 기준뿐이다.
`Resolution Criteria`는 없다. 소유는 ce-agent-kit#1.

방향: preflight가 issue 서식이 쓰는 기준 제목을 센다. 카드에 Completion
Criteria를 한 벌 더 붙여 검사를 통과시키지 않는다.

## Completion Criteria

- [ ] Resolution Criteria만 있는 issue 카드의 preflight 기준 수가 0보다 크다 | verify: human — 이 저장소 issue 존 preflight JSON에서 그 카드의 criteria가 0이 아니다

## Out of scope

- 이슈 본문에 `## Completion Criteria`를 추가하는 일.

## Sources

- [ISSUE-020](../issue/020-preflight-counts-zero-criteria-on-a-card-whose-criteria-live-under-resolution-criteria.md)
