---
id: TASK-420
title: "Accept legacy done-card evidence in the review receipt contract"
type: bug
priority: P2
effort: M
exec-tier: strong
status: todo
created: 2026-09-24
---

## Summary

[ISSUE-001](../issue/001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md)의
DVA 쪽은 끝났다. 영수증은 `tasks/done/evidence/`에 있고, 게이트는 READY다.
남은 것은 상류다. evidence가 없거나 prose이거나 파일인 레거시 done 카드를
CE가 받아들이는 리뷰 영수증으로 만들지 못한다. 보고는 ce-agent-kit#7.

방향: 발급기와 validator가 그 세 형태를 하나의 canonical digest 계약으로
다루게 한다. 이 저장소에 두 번째 검사나 수동 digest 우회를 더하지 않는다.

## Completion Criteria

- [ ] 없는 evidence, prose evidence, 파일 evidence가 모두 CE canonical digest 영수증으로 검증된다 | verify: human — ce-agent-kit 테스트 또는 이 저장소 레거시 카드의 재검증 로그가 이 카드에 링크된다

## Out of scope

- `tmp/` 아래 영수증. TASK-388이 이미 닫았다.
- 기대 digest를 영수증에 손으로 베끼는 절차.

## Sources

- [ISSUE-001](../issue/001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md)
