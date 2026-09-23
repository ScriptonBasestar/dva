---
id: TASK-416
title: "Reconcile version-close evidence and current task state"
type: docs
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-23
depends-on: [TASK-415]
---

## Summary

PLAN-006과 tasks/README.md를 durable 증거에 맞게 현행화한다. ISSUE-037을 동기화 증거로 종결하고 ISSUE-024에 TASK-410 재현을 연결한다. TASK-413의 잘못 넓은 AC1은 ci.profiles 범위로 고쳐 독립 재리뷰한다. TASK-411은 복구된 두 게이트로 재리뷰한다. 역사 receipt와 사람 전용 결정을 보존한다.

## Completion Criteria

- [ ] 현재 작업 목록이 PLAN-010과 연결되고 역사와 현재를 구분한다 | verify: human — tasks/README.md와 PLAN-006의 현재 상태를 TASK-328/329/348 durable receipt와 대조
- [ ] TASK-413 조건부 판정과 TASK-411 차단 원인을 실제 게이트에 맞춰 정리한다 | verify: human — 새 독립 리뷰 기록과 ISSUE-038 해결 근거를 확인
- [ ] 문서 및 보드 게이트 통과 | verify: `make doc-check && ce task gate` (regression-guard)
