---
id: TASK-578
title: "Close reviewed cards 540 545-549 551 577 in done zone"
type: chore
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tasks]
status: done
created: 2026-11-04
quality-review: pass
quality-reviewed-at: 2026-11-04
quality-review-evidence: "Main-thread self-close execution (TASK-478 pattern): 9 cards moved in one commit, every moved card carries quality-review fields, 8 evidence JSONs written from review-batch and review-577 verdicts, doc-check green in the worktree."
---

## Summary

TASK-540·545·546·547·548·549·551의 구현은 이미 master에 통합됐고(각 리뷰 커밋 a432429b+dbfb0dd7, 97fd271b, 79cbefd5, 59e301bb, 4616935a, c6bf5a3f, 2e677c73), TASK-577은 3638b938로 통합됐다. 8장 모두 독립 리뷰 PASS 판정을 받았으나 카드가 todo 존에 남아 큐가 미완료로 표시한다. TASK-478 자기-종결 패턴대로 이 카드(578)가 자기 브랜치에서 8장과 자기 자신을 함께 done 존으로 이관하고 quality-review 필드와 독립 리뷰 증거 8건(`tasks/done/evidence/TASK-*/independent-review.json`)을 기록한다.

## Sources

- 배치 리뷰 보고서(2026-11-04, review-batch 서브에이전트): 7장 전부 PASS, 차단 없음, non-blocking 기록 각 카드 evidence에 보존.
- TASK-577 리뷰(review-577 서브에이전트): FAIL(문서 범위) → 3638b938 수정 → PASS. 검증 커밋 97ebd574·3638b938.
- 사용자 지시(2026-11-04 세션): "워크트리가 분기되어 있으면 통합해가면서 작업하면 된다." — 4개 세션 인수인계 카드의 종결을 이 세션이 소유.

## Completion Criteria

- [x] 이관 대상 카드 8장이 전부 done 존에 있고 todo에는 없다 | verify: `for f in 540-installed-skill-initialization-path 545-skill-downgrade-guard 546-init-scaffold-proves-services 547-strict-ignores-plan-hint 548-skill-am-optional-wording 549-skill-post-init-completion 551-skill-rules-match-validator 577-enforce-single-plan-readiness-bounds-with-signal-aware-cancellation; do test -f tasks/done/$f.md && test ! -f tasks/todo/$f.md || exit 1; done`
- [x] 이관 카드 전부(마감 카드 포함)가 quality-review: pass 필드를 가진다 | verify: `for f in 540-installed-skill-initialization-path 545-skill-downgrade-guard 546-init-scaffold-proves-services 547-strict-ignores-plan-hint 548-skill-am-optional-wording 549-skill-post-init-completion 551-skill-rules-match-validator 577-enforce-single-plan-readiness-bounds-with-signal-aware-cancellation; do /usr/bin/grep -q '^quality-review: pass' tasks/done/$f.md || exit 1; done && /usr/bin/grep -q '^quality-review: pass' tasks/done/578-*.md`
- [x] 독립 리뷰 증거 파일 8건이 evidence 루트에 존재한다 | verify: `for id in 540 545 546 547 548 549 551 577; do test -f tasks/done/evidence/TASK-$id/independent-review.json || exit 1; done`
- [x] 문서·카드 검증이 통과한다 | verify: `make doc-check`
