---
id: TASK-445
title: "Correct host pin and worktree reclaim blockers"
type: docs
priority: P1
effort: S
exec-tier: strong
status: done
created: 2026-09-27
quality-review: pass
quality-reviewed-at: 2026-09-27
quality-review-evidence: "Independent review session /root/review_task443_board_refresh PASS on staged tree after 61ead57b handoff: TASK-431 host/project pins and remaining install evidence, TASK-436 existing reclaim authorization, PLAN-011/README counts, W14 independence; target task-manager origin/master 21c9a29 contains W13/W14. git diff --cached --check, make doc-check, ce task validate --all, and ce task gate passed."
---

## Summary

2026-09-27 재측정에서 TASK-431의 DVA 제품 핀과 전역 호스트 핀을 분리했다.
TASK-436은 회수 승인이 이미 있어 CE 수명주기 명령만 남았다. TASK-421/423의
제품 소유권은 동시 통합된 `61ead57b`의 task-manager-devbox W13/W14 이관을 따른다.

## Completion Criteria

- [x] TASK-431 distinguishes DVA project lint pin from host pin and names the worktrunk installation evidence still required | verify: human — inspect .mise.toml, CI, corrected card
- [x] TASK-436 records existing reclaim authorization and the missing CE discard lifecycle capability | verify: human — inspect corrected card and ISSUE-039
- [x] PLAN-011 and tasks README preserve the product-owner handoff and corrected blockers | verify: human — inspect board summaries and source cards
