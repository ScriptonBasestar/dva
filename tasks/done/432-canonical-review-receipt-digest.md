---
id: TASK-432
title: "Make the review receipt pin the canonical card digest"
type: bug
priority: P2
effort: M
exec-tier: strong
status: done
created: 2026-09-24
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "독립 done-review PASS: 원시 파일 해시 receipt는 canonical digest를 표시하며 거부되고, canonical 핀은 통과하며 이후 본문 변경은 다시 거부된다. 통합 commit 80efba9561773b8dcea54c2ba47d58992eb90e2a와 commit CI 6b88756eacac6c7e04f6ec57bdab50d2 PASS."
---

## Summary

[ISSUE-028](../_archive/issue/028-the-review-receipt-pins-a-digest-the-closed-card-can-never-match.md)의
이 보드는 TASK-401이 canonical digest로 다시 찍어 게이트가 초록이다.
상류 validator도 이제 canonical review-subject SHA-256을 요구하고, legacy raw
file hash 같은 비정규 receipt와 실제 검토 대상 변경을 설명 가능한 digest 오류로
거부한다. 소유는 ce-agent-kit#7.

방향: 핀해야 하는 digest가 canonical digest임을 스키마와 오류 메시지가
말하게 한다. 틀린 핀은 기대하는 canonical 값을 오류에 포함한다. 본문이
실제로 바뀐 카드는 계속 실패한다.

## Completion Criteria

- [x] plain 파일 해시를 핀한 영수증은 canonical digest를 지목하며 실패하고, canonical 핀은 통과한다 | verify: human — 상류 테스트가 두 핀을 구분하고, 본문을 바꾼 카드는 여전히 실패한다

## Out of scope

- 이 보드에 이미 찍힌 canonical 핀을 다시 발행하는 일.

## Sources

- [ISSUE-028](../_archive/issue/028-the-review-receipt-pins-a-digest-the-closed-card-can-never-match.md)

## Current evidence

- Regression `TestValidateChecksReviewReceipt/review-time_raw_file_hash_fails_with_the_required_canonical_digest`: raw file hash fails with the required canonical digest; canonical pin passes; later body change fails.
- Focused checks: receipt regression and `go test ./internal/usecase/task -count=1` pass; `make validate-docs` and `git diff --check` pass.
- DVA commit CI run `6b88756eacac6c7e04f6ec57bdab50d2` passed (6m10.230s; native-ci succeeded).
- Independent done-review: PASS; documentation contract and noncanonical-versus-changed error wording were rechecked.
- Integrated `ce-agent-kit` master commit `80efba9561773b8dcea54c2ba47d58992eb90e2a`; `ce task run-finish` readiness and integration passed; task worktree and local/remote branch reclaimed.
