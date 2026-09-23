---
id: TASK-415
title: "Cover doing and blocked cards in repository validation"
type: bug
priority: P1
effort: S
exec-tier: standard
created: 2026-09-23
status: done
archived-at: 2026-09-23
verified-at: 2026-09-23
verification-summary: "Re-verified 2026-09-23. cardZones permits doing and blocked. TestCardZonesDeclareDoingAndBlocked and TestDuplicateCardIDAcrossDoingAndBlocked passed. go test ./tools/doccheck and make doc-check passed. No follow-up."
quality-review: pass
quality-reviewed-at: 2026-09-23
quality-review-evidence: "Independent ce-judge review410, not implementer: source diff and PLAN-010/TASK-370/416/417/418 metadata reviewed. go test ./tools/doccheck, make doc-check and ce task gate PASS. CI 02a44afdadb3434c6f9cf82e0e36754e succeeded in 2m13.219705583s with matching attestation. Three queue metadata findings corrected and revalidated. No remaining findings."
---

## Summary

`tasks/blocked/`의 TASK-411 때문에 현재 `make doc-check`가 실패한다.
정식 생명주기 상태 doing/blocked를 zone 표에 선언하여 상태·ID 검사의 사각지대를 없앤다.
2026-09-23 사전 검사: duplicate IDs 0, 문서 게이트는 undeclared blocked 한 건으로 실패.
새 카드 등록 전 검사 자체는 실행했으며 이 기존 실패를 숨기지 않는다.

통합 선행 조건인 TASK-410의 누락 receipt는 기존 구현과 분리된 독립 리뷰로 복구한다.
이는 코드 수정이 아니라 이미 구현된 선행 카드의 검증 증거 복구이며, 기존 증거는 보존한다.

## Completion Criteria

- [x] Canonical lifecycle zones participate in status and identity checks | verify: `go test ./tools/doccheck`
- [x] Repository documentation gate passes | verify: `make doc-check` (regression-guard)

## 마감 큐 등록

사용자 요청에 따라 PLAN-010과 TASK-416~418을 이 작업에서 등록했다.
실제 후속 구현은 각 task worktree에서 순차 진행한다.

## Verification

- `dva ci commit`: 02a44afdadb3434c6f9cf82e0e36754e, succeeded, 2m13.219705583s.
- `make doc-check`: PASS; unknown board directories 0, duplicate IDs 0.
- TASK-410 independent review and canonical receipt restored `ce task gate` READY.

## Archive

- [x] 아카이브
- 후속: 없음
