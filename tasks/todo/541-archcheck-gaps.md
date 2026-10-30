---
id: TASK-541
title: "Close archcheck review gaps"
type: bug
priority: P3
exec-tier: standard
status: todo
created: 2026-10-31
---

## Summary

TASK-539 독립 리뷰의 비차단 지적을 처리한다. 모든 대상 OS의 import를 검사하고, 사라진
패키지의 허용 항목을 위반으로 보고하며, `go list` 실패 원인을 출력한다.

## Completion Criteria

- [x] 세 OS의 그래프가 허용 목록을 통과한다. | verify: `go run ./tools/archcheck`
- [x] 낡은 허용 항목을 위반으로 보고한다. | verify: `go test ./tools/archcheck`
