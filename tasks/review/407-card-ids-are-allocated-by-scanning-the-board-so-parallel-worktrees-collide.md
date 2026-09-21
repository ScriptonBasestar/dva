---
id: TASK-407
title: "Card ids are allocated by scanning the board, so parallel worktrees collide"
type: feature
priority: P2
status: review
created: 2026-09-17
---

## Summary

병렬 워크트리 환경에서 카드 번호 발급 시 각 워크트리가 동일한 최대 ID를 조회하여 번호 충돌(예: 383 중복 발급)이 발생하는 문제를 방지하기 위해, 병렬 충돌 방지 발급 규칙을 정립하고 개명 시 옛 참조 일괄 훑기 절차를 수립한다.

## Completion Criteria

- [x] 발급 규칙이 병렬 워크트리에서 충돌하지 않도록 정해지고 `AGENTS.md`에 적혀 있다 | verify: human — AGENTS.md의 발급 규칙을 읽고, 두 워크트리가 서로를 보지 못하는 상태에서도 같은 번호를 고르지 않는지 확인
  - Evidence: `AGENTS.md` section `Parallel-safe card-ID allocation (worktree isolation)` — worktree naming pattern, per-worktree ID claim, `max+1 never guess across worktrees`, `make doc-check` pre-claim check.
- [x] 카드 개명 절차가 옛 id 참조 일괄 훑기를 포함한다 | verify: human — 절차 문서를 읽고, 봉인된 카드에 도달하지 못하는 경우의 처리까지 적혀 있는지 확인
  - Evidence: `docs/407-correction-procedure.md` (2789 bytes) — §2 rename steps, §3 sealed-card unreachable (rename the live side; both-sealed → escalate to host), §4 old-ID sweep (`/usr/bin/grep -rq` sweep + `make doc-check` re-run).
- [x] 중복 카드 id를 재는 검사가 유지된다 | verify: `/usr/bin/grep -q 'func checkDuplicateCardIDs' tools/doccheck/cardids.go`
  - Evidence: present at `tools/doccheck/cardids.go:44`, wired into gate (`check.go:306-308`, error at `:387-388`, `DUP-ID` output `main.go:116`); `go test ./tools/doccheck/` ok, `make doc-check` exit 0.

## Sources

- ISSUE-012 — tasks/issue/012-card-ids-are-allocated-by-scanning-the-board-so-parallel-worktrees-collide.md
