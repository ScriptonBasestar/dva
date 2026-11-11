---
id: TASK-602
title: "compare readiness의 unknown health 처리"
type: feature
priority: P2
status: todo
created: 2026-11-11
---

## Summary

TASK-597의 `readinessText`는 서비스 state가 `running`이 아니거나 health가 비어 있지 않은데 `healthy`도 아니면 not ready로 본다(`compare_result.go:277`). kubectl(`k8s_helpers.go:66`)·vagrant(`vagrant.go:147`)는 실행 중인 파드·VM의 health를 `unknown`(`plugin.go:59`가 문서화한 값)으로 내므로 not ready로 표시된다. helm은 state를 `deployed`나 원시 helm status로, skaffold는 state를 `unknown`으로 내므로(`helm.go:52-57`, `skaffold.go:67`) health 규칙만 고쳐서는 not ready로 남는다. 수정 전에는 `-`였다. 카드 명세와는 일치하지만 거짓 not ready라, 플러그인별로 `unknown`·`deployed` 같은 값을 어떻게 볼지 정해야 한다. TASK-597 2차 리뷰와 마감 리뷰에서 나왔다.

## Completion Criteria

- [ ] kubectl·vagrant·helm 등 health를 unknown으로 내는 플러그인의 실행 중 서비스가 not ready가 아닌 값으로 표시된다 | verify: human — 구현 카드가 go test -run 바인딩으로 바꾼다
