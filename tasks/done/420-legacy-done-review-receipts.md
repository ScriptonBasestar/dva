---
id: TASK-420
title: "Accept legacy done-card evidence in the review receipt contract"
type: bug
priority: P2
effort: M
exec-tier: strong
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent done-review by /root/review_433_434: verified commit 988e7de6 round-trips all three legacy completion-evidence forms through NewValidator while preserving missing reviewer-evidence rejection; exact full CI, source integration, and task-branch cleanup pass."
---

## Summary

[ISSUE-001](../issue/001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md)의
DVA 쪽은 끝났다. 영수증은 `tasks/done/evidence/`에 있고, 게이트는 READY다.
남은 것은 상류다. **레거시 completion evidence**가 없거나 prose이거나 파일 경로인
카드가 새 독립 리뷰를 거친 뒤 CE가 받아들이는 영수증으로 마이그레이션되지 않는다.
이는 `quality-review-evidence`와 다른 입력이다. 리뷰어의 실제 근거는 먼저
non-empty `quality-review-evidence`로 기록하고, 그 카드 내용으로 영수증을 계산해야
한다. 보고는 ce-agent-kit#7.

방향: 발급기와 validator가 missing/prose/file 형태의 legacy completion evidence를
같은 canonical digest round-trip으로 다루게 한다. `quality-review-evidence` 없는
승인을 허용하지 않는다. 이 저장소에 두 번째 검사나 수동 digest 우회를 더하지 않는다.
상류 구현은 `988e7de6319beaea2d8705448bce930f950756ed`에서 `master`에 통합됐다.
아래 구현 증거를 붙였으며, 카드 자체의 독립 done-review를 마친 뒤에만 `done`으로
이동한다.

## Completion Criteria

- [x] Missing, prose, and file-backed legacy completion evidence each round-trip through the CE canonical receipt and validator after fresh non-empty `quality-review-evidence` is recorded | verify: human — upstream round-trip tests attach each generated receipt to a final done-card fixture and `NewValidator.Validate` returns no errors; a negative fixture keeps missing reviewer evidence rejected

## Upstream implementation evidence

- Commit `988e7de6319beaea2d8705448bce930f950756ed` was rebased onto current
  CE `master`, integrated with `ce task run-finish`, and is now `master` at the
  source remote. The task worktree and both task branch refs were reclaimed.
- `internal/usecase/task/review_receipt_roundtrip_test.go` covers absent,
  prose, and file-backed legacy completion evidence. Every passing fixture first
  records fresh non-empty `quality-review-evidence`, issues a receipt, attaches
  the receipt and verdict fields, and requires `NewValidator.Validate` to pass.
  A separate negative fixture confirms that missing reviewer evidence remains
  rejected.
- Full commit CI `ci-20260925-092659-10441` passed on that exact commit with a
  clean worktree. The independent code review of the exact commit returned
  PASS; it identified only a non-blocking opportunity to clarify in CLI docs
  that reviewer evidence must exist before receipt issuance.

The implementation criterion is complete and the upstream code is integrated.
This DVA card is ready for its separate board `done-review`; it remains in
`review/` until that review records its verdict.

## Out of scope

- `tmp/` 아래 영수증. TASK-388이 이미 닫았다.
- 기대 digest를 영수증에 손으로 베끼는 절차.

## Sources

- [ISSUE-001](../issue/001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md)
