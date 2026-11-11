---
id: TASK-602
title: "compare readiness의 unknown health 처리"
type: feature
priority: P2
status: todo
created: 2026-11-11
---

## Summary

TASK-597의 `readinessText`는 health가 비어 있거나 `healthy`가 아니면 not ready로 본다. kubectl·vagrant·helm·skaffold 플러그인은 health를 `unknown`으로 내므로(`plugin.go:59`가 문서화한 값) 실행 중인 파드·VM·릴리스가 not ready로 표시된다. 수정 전에는 `-`였다. 카드 명세와는 일치하지만 거짓 not ready라 `unknown`의 의미를 정해야 한다(예: 실행 중이면 `-`). TASK-597 2차 리뷰에서 나왔다.

## Completion Criteria

- [ ] kubectl·vagrant·helm 등 health를 unknown으로 내는 플러그인의 실행 중 서비스가 not ready가 아닌 값으로 표시된다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
