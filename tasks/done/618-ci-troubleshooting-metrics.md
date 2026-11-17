---
id: TASK-618
title: "CI 트러블슈팅 기록과 실행 메트릭 수집안 분석"
type: docs
priority: P1
effort: S
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-618/review.txt
created: 2026-11-17
---

## Summary

사용자 요청에 따라 기존 문제의 원인·조치·검증·미확정을 기록하고,
CLI 테스트와 lint 도구별 메트릭을 실행마다 수집할 때의 문제와 대응을 제안한다.
구현·도입을 요청받은 작업은 아니므로 분석과 구현 상태를 구분한다.

## Completion Criteria

- [x] 기존 원문을 보존하며 해결·미해결을 구분한 기록을 남긴다 | verify: human — 기존 TASK-612/613/615/617 문서·증거와 사실 대조; source에 새 트러블슈팅 문서 없음 (observed: 2026-11-17)
- [x] 현재 수집 정보와 추가 제안을 코드·workflow에 대조한다 | verify: human — run/step receipt·hosted JSONL·로컬 test/lint 경계 검토 (observed: 2026-11-17)
- [x] 수집 비용·실패·보존·정보 노출·해석 한계와 구현 순서를 설명한다 | verify: human — 기존 검사 판정·timeout을 보존하는 제안인지 독립 검토 (observed: 2026-11-17)
- [x] 미채택 제안을 현재 기능처럼 서술하거나 실행 설정을 바꾸지 않는다 | verify: human — source 대비 diff와 미구현 상태 확인 (observed: 2026-11-17)

## Verification

[트러블슈팅 정본](../../docs/ci-troubleshooting.md),
[메트릭 검토안](../../docs/ci-metrics-proposal.md).
새 성능 측정을 실행한 작업은 아니며 기존 측정과 실패의 의미를 바꾸지 않는다.

## Delivery

독립 source/done 리뷰와 문서·카드 검사 후 확정 커밋의 dev/full·원격 CI를 검증한다.
CE 통합·push·작업 트리 회수 결과는 실행 receipt로 확인한다.
