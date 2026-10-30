---
id: TASK-539
title: "Enforce internal package import boundaries"
type: feature
priority: P2
exec-tier: standard
status: todo
created: 2026-10-31
---

## Summary

ARCHITECTURE.md의 의존 방향은 문서에만 있어 위반해도 실패하지 않았다. 패키지별 허용
import 목록을 `tools/archcheck`에 두고 `make lint`에서 검사한다.

## Completion Criteria

- [x] 현재 import 그래프가 허용 목록을 통과한다. | verify: `go run ./tools/archcheck`
- [x] 금지 import와 미선언 패키지를 위반으로 보고한다. | verify: `go test ./tools/archcheck`
