---
id: TASK-597
title: "compare status·down 출력 보완"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-11-09
---

## Summary

<!-- One paragraph: what changes and why. -->

## Completion Criteria

- [ ] 사람용 status 표에 pid, log, readiness 열이 있고 skipped 행과 실패한 Status 호출 행에도 pid·log가 채워진다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
- [ ] 기록된 비교안이 없을 때 down --json과 status --json이 variants를 null이 아닌 빈 배열로 낸다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
