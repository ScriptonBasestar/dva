---
id: TASK-617
title: "일반 Go commit 실측과 재사용 입력 경계 판정"
type: chore
priority: P1
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-617/review.txt
created: 2026-11-17
---

## Summary

기존 test-only 측정을 일반 Go 변경 두 사례로 넓히고 full lint 비용을 분리 관찰한다.
입력 완전성과 순절감이 입증되지 않으면 결과 재사용 미채택으로 종결한다.

## Completion Criteria

- [x] 일반 Go 두 사례의 cold/warm fresh·기본 실행과 실패를 보존한다 | verify: human — 측정 JSON, receipt와 원문 대조; source에는 TASK-617 측정 없음 (observed: 2026-11-17)
- [x] full lint 도구별 시간을 기존 32초 관측과 구분한다 | verify: human — 별도 진단의 도구 시간·원래 스크립트와 instrumentation 대조 (observed: 2026-11-17)
- [x] 재사용 후보의 입력 경계와 채택 여부를 판정한다 | verify: human — 실제 파일·환경·도구·외부 상태 경계 독립 검토 (observed: 2026-11-17)
- [x] 기존 검사·pin·예산을 유지하고 측정 주석을 제품에 넣지 않는다 | verify: human — source 대비 diff와 experiment 격리 확인 (observed: 2026-11-17)

## Verification

[측정과 채택 판정](../../docs/ci-production-measurements.md),
[원문 색인](evidence/TASK-617/measurements.json),
[lint 시간](evidence/TASK-617/lint-timings.json).
GOROOT 불일치 실패와 원인 미확정 test timeout은 성공으로 바꾸지 않는다.
기준 source는 `3fa577340ededf26fcb2275733be8b625e63b876`이며 진단은 최종 gate와 별개다.

## Delivery

reuse 미채택. 독립 source/done 검토 후 확정 커밋의 dev/full·원격 CI,
설치 검증·CE 통합/push/회수를 실행하고 실제 결과는 immutable receipt로 남긴다.
문서·증거만 바꾸므로 로컬 macOS 플랫폼 전용 suite의 재실행 조건에는 해당하지 않는다.
