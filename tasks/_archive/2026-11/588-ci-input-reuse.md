---
id: TASK-588
title: "CI 입력 기반 결과 재사용과 fresh 실행"
type: feature
priority: P2
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-588/independent-review.md
created: 2026-11-06
---

## Summary

동일 입력의 native 검사 PASS를 프로젝트별 캐시 래퍼 없이 재사용한다. 선언 입력·도구·
환경·probe·산출물과 유효기간을 검증하며 fresh 실행은 실제 검사와 하위 CI까지 전파한다.
설정과 사용 계약은 docs/75-ci-result-reuse.md가 소유한다.

## Completion Criteria

- [x] 선언 입력 변이와 fresh 실행 회귀 검사가 통과한다 | verify: `go test ./internal/config ./internal/cirun ./internal/cli -run 'Test.*Reuse|TestReusable|TestCIFreshPreview'`
