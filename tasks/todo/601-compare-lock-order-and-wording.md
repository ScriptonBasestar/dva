---
id: TASK-601
title: "compare 락 순서 고정과 문구 보완"
type: feature
priority: P2
status: todo
created: 2026-11-11
---

## Summary

TASK-600 리뷰에서 나온 낮은 심각도 후속이다. (1) `compare down`의 락이 기록 읽기보다 먼저 잡혀야 한다는 순서를 고정하는 테스트가 없어, 락을 `ReadRecord` 뒤로 옮기는 변이가 살아남는다. (2) `compare up`의 거절 문구가 아직 "another `dva compare up`"이라 락을 쥔 쪽이 down일 때 안내가 틀린다. (3) TASK-599의 코드 주석과 카드가 EPERM 비율을 약 0.65%로 적었으나 리뷰어 실측은 부하에 따라 8~40%였다. 문구와 서술을 고치고 순서 테스트를 추가한다.

## Completion Criteria

- [ ] 락을 기록 읽기보다 먼저 잡는 순서를 고정하는 테스트가 있고 compare up의 거절 문구가 down이 락을 쥔 경우에도 맞다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
