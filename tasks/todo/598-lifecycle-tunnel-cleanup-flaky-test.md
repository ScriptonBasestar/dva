---
id: TASK-598
title: "lifecycle 터널 정리 테스트 불안정"
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

- [ ] TestStatusTunnelCleanupEscalatesOnlyItsOwnedProcessGroup이 반복 실행(-count=50)에서 실패하지 않는다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
