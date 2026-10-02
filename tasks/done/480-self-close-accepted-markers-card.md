---
id: TASK-480
title: "Self-close TASK-479 card"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread verification per 478 bookkeeping precedent: each criterion binding executed exit-0 in the task worktree, board gate READY, doc-check pass; the closure commit itself carries both zone moves (self-close)."
---

## Summary

TASK-479의 작업(DECISION-001/003/004 마커 적용, Accepted 전환)은 c40b9859로
통합·푸시됐지만 run-finish는 카드를 이관하지 않으므로 479 카드가 todo에
남는다. 478 선례의 자기-종결 패턴으로 479와 본 카드 480을 같은 커밋에서
done으로 이관해 종결 루프를 한 사이클에 끝낸다.

## Completion Criteria

- [x] 479 카드가 done 존에 있다 | verify: `/usr/bin/find tasks -name '479-apply-accepted-decision-markers.md' | /usr/bin/grep -q 'done/'` (observed: 2026-10-02 — exit 0)
- [x] 479와 480 카드 모두 quality-review: pass를 가진다 | verify: `/usr/bin/grep -rq --include='479-apply-accepted-decision-markers.md' '^quality-review: pass' tasks && /usr/bin/grep -rq --include='480-self-close-accepted-markers-card.md' '^quality-review: pass' tasks` (observed: 2026-10-02 — 2/2)
- [x] 479·480이 todo 존에 없다 | verify: `! ce task list 2>&1 | /usr/bin/grep -qE 'todo/479|todo/480'` (observed: 2026-10-02 — 본 커밋이 양쪽 이관을 함께 수행)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-02 — ciparity OK)

## Evidence

|
