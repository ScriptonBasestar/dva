---
id: TASK-434
title: "Report list-form quality-review-evidence honestly"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-031](../issue/031-list-form-quality-review-evidence-is-misreported-as-missing.md)은
그대로다. `quality-review-evidence`가 리스트면 scalar 읽기가 빈 값을 돌려주고,
validator는 필드가 없다고 말한다. 필드는 있다. 형식이 다를 뿐이다.
이 보드의 해당 카드는 이미 스칼라로 고쳤고, 이 카드는 같은 보고가 다시
나오는 것을 막는다. 소유는 ce-agent-kit#9.

방향: 리스트를 만나면 한 줄 스칼라여야 한다고 말하거나, 리스트를 수용한다.
어느 쪽이든 "필드가 없다"고 말하지 않는다.

## Completion Criteria

- [ ] 리스트형 `quality-review-evidence`가 형식 오류로 보고되거나 수용된다 | verify: human — 상류 반영 후 리스트형 카드의 validate 출력이 필드 부재가 아니다

## Out of scope

- 이미 스칼라로 고친 이 보드 카드의 내용 변경.

## Sources

- [ISSUE-031](../issue/031-list-form-quality-review-evidence-is-misreported-as-missing.md)
