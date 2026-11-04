---
id: TASK-580
title: "Classify validation severity independently of warning text"
type: refactor
priority: P2
exec-tier: strong
status: review
created: 2026-11-04
---

## Summary

validate의 strict 예외가 메시지의 ℹ 접두사에 의존하던 판정을 생산자별 DiagnosticSeverity로 옮긴다. ValidateDiagnostics가 정본 순서와 심각도를 소유하고, ValidateWarnings는 기존 문자열 목록을 투영한다. no-plans 안내만 정보로 분류하고 나머지·알 수 없는 심각도는 actionable로 처리한다. 텍스트 문구·순서와 기존 semantic JSON 형식은 유지한다.

## Completion Criteria

- [ ] Strict validation uses producer-assigned severity while legacy messages and JSON remain compatible | verify: `go test ./internal/config ./internal/cli -run 'Test(ValidationDiagnostic.*|StrictIgnoresInformationalHints|ValidatePlanDriftWarnings.*)$'`
