---
id: TASK-602
title: "compare readiness의 unknown health 처리"
type: feature
priority: P2
status: done
quality-review: pass
quality-reviewed-at: 2026-11-11T09:00:00Z
quality-review-evidence: tasks/done/evidence/TASK-602/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-602/done-review-0e6cb2d1092c6aed3168c7774431f31a7b3f7789f11cc15a8d2eba7e9105527e.json
created: 2026-11-11
---

## Summary

TASK-597의 `readinessText`는 서비스 state가 `running`이 아니거나 health가 비어 있지 않은데 `healthy`도 아니면 not ready로 본다(`compare_result.go:277`). kubectl(`k8s_helpers.go:66`)·vagrant(`vagrant.go:147`)는 실행 중인 파드·VM의 health를 `unknown`(`plugin.go:59`가 문서화한 값)으로 내므로 not ready로 표시된다. helm은 state를 `deployed`나 원시 helm status로, skaffold는 state를 `unknown`으로 내므로(`helm.go:52-57`, `skaffold.go:67`) health 규칙만 고쳐서는 not ready로 남는다. 수정 전에는 `-`였다. 카드 명세와는 일치하지만 거짓 not ready라, 플러그인별로 `unknown`·`deployed` 같은 값을 어떻게 볼지 정해야 한다. TASK-597 2차 리뷰와 마감 리뷰에서 나왔다.

## Completion Criteria

- [x] health·state가 `unknown`이거나 helm `deployed`처럼 판단할 수 없는 서비스는 not ready가 아니라 `-`로 표시되고, 확실히 나쁜 신호가 함께 있으면 not ready로 남는다 | verify: `go test ./internal/cli/ -run '^(TestReadinessTextUnjudgeableServices|TestReadinessTextSummarizesHealth|TestCompareTableRowShowsDerivedReadiness)$' -count=1`

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (독립), executor: claude-opus-5-5

- 대상: 8090fec4. 판정: fail. `status.phase`가 없는 kubectl/kustomize 리소스는 state가 `""`인데 여전히 not ready로 표시됐다. 9e565d2c에서 수리했다(코디네이터 요약, 리뷰 원문 미보존).

### 2. 2026-11-11 — reviewer: claude-code-subagent (독립, read-only), executor: claude-opus-5-5

- 대상: 9e565d2c. 판정: pass. 검증 바인딩이 통과했다. 판단 불가 집합 {`""`, `unknown`, helm `deployed`}은 플러그인이 내는 값과 맞고 거짓 ready를 만들 수 없다. 변이(되돌림, `""` 제거, 판단 불가 분기 끄기, ToLower, 순서)를 모두 잡았다. go vet, windows vet, `go test` cli가 통과했다. `make doc-check`는 master와 같은 기존 대형 문서 docs/75로만 실패한다. 테스트 이동에서 빠진 경우가 없고 USAGE 항목 합치기에서 빠진 문구가 없다.
- 비차단: 커밋 메시지의 "no other plugin emits empty state" 주장은 추정이다. 주석 한 줄이 길다. 서비스가 없는 항목이 ready를 막지 않는 기존 공백이 있다(후속 TASK-603). 다음 전환: done/.
