---
id: ISSUE-029
title: "The receipt canonicalization migration left 16 done cards below the review-stage schema"
status: open
priority: P1
created: 2026-09-16
source: "TASK-404 카드 통합 시도(2026-09-16) — run-finish가 게이트 red로 블록"
---

## Summary

영수증 정준 위치 이관 작업(`06948b7` restore sealed receipts, `bc571e7` 보드 존 정규화,
TASK-401 계열)이 done 카드 16건에서 `quality-review-evidence` 필드를 지웠다. 이
워크스테이의 선언된 검증기(`ce` 0.8.4, kit 소스 `internal/usecase/task/validator_review_stage.go:37`)는
done zone에서 `quality-review: pass|conditional`에 비어 있지 않은
`quality-review-evidence`를 요구한다. 결과: **마스터 `72072a5`에서 `ce task gate`가
16 invalid로 NOT READY** — 통합 게이트가 red라 어떤 새 작업의 run-finish/branch-integrate도
블록된다(TASK-404 카드 브랜치가 첫 피해자).

TASK-328/329/402/403을 닫은 세션(host mbp)은 같은 보드 상태로 run-finish를 통과했다 —
그 호스트의 ce는 이 규칙을 다르게 읽는다. 호스트 간 스키마 왜곡(도구 버전) 또는 이관의
필드 삭제가 실수인지의 구별이 열려 있다.

## Evidence (2026-09-16 실측)

- `ce task gate` (primary, clean, 72072a5) → NOT READY, `16 task(s) failed validation`
- `ce task validate --all` → 107 valid / 16 invalid, 16건 전부 동일 오류:
  `accepted quality-review verdict requires quality-review-evidence` —
  307, 328, 329, 341, 348, 352, 355, 357, 373, 379 등 done 카드
- 이전·후 대조: TASK-379 카드는 2026-09-15 시점 `quality-review-evidence` 3항목을
  갖고 있었고, pull 후 그 필드가 사라졌다(내용은 `tasks/done/evidence/TASK-379/` 영수증에
  존재 — 삭제가 정보 소실이 아니라 중복 제거 의도로 보인다)
- 규칙 근거: kit 소스 dabaf0d5(체크아웃 0 behind origin) `validator_review_stage.go`
  — done: verdict가 pass/conditional이면 evidence 비어 있으면 오류

## Resolution Criteria

- [ ] 16건의 done 카드가 현행 선언 스키마를 통과하고 `ce task gate`가 READY를 회복한다
- [ ] 어느 쪽이 정본인지 기록된다 — 필드 복원(보드 수리) 또는 검증기 완화(kit 변경) 중 무엇을 했는지
- [ ] 블록된 통합(TASK-404 브랜치 등)이 회복된다

## 소유권 — 이 저장소다 (2026-09-16 명시)

보드 상태(카드 frontmatter)는 이 저장소 소유고, 게이트도 이 저장소가 선언한 계약을
따른다. 검증기 완화를 택하더라도 그 결정과 기록은 이 보드에 남는다 — 상류 작업은
요구되지 않는다.
