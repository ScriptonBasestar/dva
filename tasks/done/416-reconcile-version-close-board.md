---
id: TASK-416
title: "Reconcile version-close evidence and current task state"
type: docs
priority: P2
effort: M
exec-tier: standard
status: done
quality-review: pass
quality-reviewed-at: 2026-09-23
quality-review-evidence: "Independent ce-judge review410: actual 11-file docs/board diff inspected, all 29 PLAN-006 children found once in archive, TASK-328 failures preserved, TASK-411/413 independent verdicts checked. Issue archive evidence wording corrected. git diff --check, card validate, make doc-check and ce task gate passed. No source changes; full CI not repeated."
created: 2026-09-23
depends-on: [TASK-415]
---

## Summary

PLAN-006과 tasks/README.md를 durable 증거에 맞게 현행화한다. ISSUE-037을 동기화 증거로 종결하고 ISSUE-024에 TASK-410 재현을 연결한다. TASK-413의 잘못 넓은 AC1은 ci.profiles 범위로 고쳐 독립 재리뷰한다. TASK-411은 복구된 두 게이트로 재리뷰한다. 역사 receipt와 사람 전용 결정을 보존한다.

## Completion Criteria

- [x] 현재 작업 목록이 PLAN-010과 연결되고 역사와 현재를 구분한다 | verify: human — tasks/README.md와 PLAN-006의 현재 상태를 TASK-328/329/348 durable receipt와 대조
- [x] TASK-413 조건부 판정과 TASK-411 차단 원인을 실제 게이트에 맞춰 정리한다 | verify: human — 새 독립 리뷰 기록과 ISSUE-038 해결 근거를 확인
- [x] 문서 및 보드 게이트 통과 | verify: `make doc-check && ce task gate` (regression-guard)

## Verification

TASK-411 네 기준과 TASK-413 정정 기준은 독립 ce-judge가 재검증해 pass했다.
ISSUE-035/038은 해결·아카이브하며 기존 본문을 보존했다. ISSUE-037에는 날짜가 있는
해결 확인을 추가한 뒤 아카이브했으며, archive 명령 자체가 그 본문을 바꾸지 않았음을 확인했다.
TASK-328의 실제 실패 exit와 역사 receipt는 수정하지 않았다.
