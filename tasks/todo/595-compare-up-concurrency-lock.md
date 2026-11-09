---
id: TASK-595
title: "compare up 동시 실행 락"
type: bug
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-11-09
---

## Summary

<!-- One paragraph: what changes and why. -->

## Completion Criteria

- [ ] 같은 비교 이름의 compare up 두 개가 겹쳐 실행되면 둘째는 아무것도 시작하기 전에 명확한 오류로 실패한다 | verify: human — 구현 카드가 추가한 테스트 이름을 go test -run 바인딩으로 바꾼다
- [ ] 락은 프로세스가 죽어도 남지 않는다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
