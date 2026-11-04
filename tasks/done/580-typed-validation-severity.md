---
id: TASK-580
title: "Classify validation severity independently of warning text"
type: refactor
priority: P2
exec-tier: strong
status: done
created: 2026-11-04
quality-review: pass
quality-reviewed-at: 2026-11-04T11:02:26Z
quality-review-evidence: tasks/done/evidence/TASK-580/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-580/done-review-cac93b9a6ba1970037c5b5eb0bf76249ad3227b019dd6625c81af3705f0314a3.json
---

## Summary

validate의 strict 예외가 메시지의 ℹ 접두사에 의존하던 판정을 생산자별 DiagnosticSeverity로 옮긴다. ValidateDiagnostics가 정본 순서와 심각도를 소유하고, ValidateWarnings는 기존 문자열 목록을 투영한다. no-plans 안내만 정보로 분류하고 나머지·알 수 없는 심각도는 actionable로 처리한다. 텍스트 문구·순서와 기존 semantic JSON 형식은 유지한다.

## Completion Criteria

- [x] Strict validation uses producer-assigned severity while legacy messages and JSON remain compatible | verify: `go test ./internal/config ./internal/cli -run 'Test(ValidationDiagnostic.*|StrictIgnoresInformationalHints|ValidatePlanDriftWarnings.*)$'`

## Review Attempts

- 2026-11-04T11:02:26Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root` (strong). Criterion PASS after guarded `-count=1 -v` verification; TASK-581 retained `-race`; required named tests executed. No blocking findings; source `6ef16cbc45dfd070472e0e8e7cb7e5affad6e0dd`. Evidence: `tasks/done/evidence/TASK-580/independent-review.json`; runtime `review-stop-warning-bundle` (review). Outcome: pass; next transition: done.
