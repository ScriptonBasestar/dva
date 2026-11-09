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

`internal/lifecycle`의 `TestStatusTunnelCleanupEscalatesOnlyItsOwnedProcessGroup`가 `-count=50`에서 `kill process group …: operation not permitted`(`status_owned_tunnel_unix_test.go:25`)로 간헐 실패한다. 원인을 찾아 테스트 또는 정리 로직을 고친다. TASK-592 리뷰 중 재현했다.

## Completion Criteria

- [ ] TestStatusTunnelCleanupEscalatesOnlyItsOwnedProcessGroup이 반복 실행(-count=50)에서 실패하지 않는다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
