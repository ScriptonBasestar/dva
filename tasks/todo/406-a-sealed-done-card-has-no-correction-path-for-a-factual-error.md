---
id: TASK-406
title: "A receipt-sealed done card has no correction path for a factual error in its own record"
type: feature
priority: P2
status: todo
created: 2026-09-17
---

## Summary

`blocks:`를 선언한 done 카드는 `quality-review-receipt`의 sha256 digest로 봉인되어 카드 내용 수정 시 게이트가 실패한다. 사후에 발견된 단순 사실관계 오류(예: TASK-376의 리뷰 지적 반영 범위 서술)를 위조 없이 정식으로 정정할 수 있는 절차를 정립하고, TASK-376의 기록을 정정한다.

## Completion Criteria

- [ ] 봉인된 done 카드의 사실관계 정정 절차가 문서로 존재한다 | verify: human — 절차 문서를 읽고, 위조(리뷰 없이 digest 갱신)와 구분되는 지점이 명시돼 있는지 확인
- [ ] TASK-376의 `quality-review-evidence`가 Low 한 건의 전반부 미적용을 반영한다 | verify: human — 376의 quality-review-evidence를 읽고, '전부 수정했다'가 사라졌는지와 미적용 한 건이 어느 지적인지 문장 안에서 읽히는지 확인

## Sources

- ISSUE-010 — tasks/issue/010-a-sealed-done-card-has-no-correction-path-for-a-factual-error.md
