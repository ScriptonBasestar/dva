---
id: TASK-596
title: "compare shared 항목의 보고와 down 범위"
type: bug
priority: P2
effort: M
exec-tier: strong
status: todo
created: 2026-11-09
---

## Summary

`compare up`은 이미 떠 있던 shared 항목도 `started`로 보고하고, 인자 없는 `compare down`은 다른 비교가 쓰는 shared 항목까지 내릴 수 있다(`compare_down.go`). shared `up`·`status`는 vars를 넘기지 않아 shared 결과의 per_variant endpoint URL도 비교안별 값으로 치환되지 않는다. TASK-592 리뷰 중 발견했다.

## Completion Criteria

- [ ] 이미 떠 있던 shared 항목은 started가 아니라 이미 실행 중으로 보고된다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
- [ ] 인자 없는 compare down이 다른 비교가 쓰는 shared 항목을 내리지 않거나 내리기 전에 알린다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
- [ ] shared 결과의 per_variant endpoint URL이 비교안별 값으로 치환된다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
