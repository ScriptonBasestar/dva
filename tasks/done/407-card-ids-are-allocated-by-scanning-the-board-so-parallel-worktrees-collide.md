---
id: TASK-407
title: "Card ids are allocated by scanning the board, so parallel worktrees collide"
type: feature
priority: P2
status: done
reopened: 2026-09-22
reopen-reason: "2026-09-22 done-board re-verification: criterion 1 is unmet by measurement. AGENTS.md's rule is `max(id seen in current worktree) + 1`, so two worktrees cut from the same snapshot see the same max and mint the same id — exactly what the criterion asks be impossible. The rule also does not fall back to stating that DUP-ID detection is the guard rather than allocation uniqueness. Criteria 2 and 3 re-verified as met (see Re-verification). Residue is filed as ISSUE-033."
created: 2026-09-17
quality-review: pass
quality-reviewed-at: 2026-09-23
quality-review-evidence: "Independent review (separate session from the fc7c41ef implementer) in worktree dev/claude/mbp/feat/task-407. Criterion 1 (human): read AGENTS.md:318-337 verbatim — the section now states plainly that max(id)+1 per worktree is not collision-proof (two worktrees off one snapshot mint the same id), that isolation does not stop this, and that tools/doccheck/cardids.go::checkDuplicateCardIDs (gated by make doc-check) is the actual post-hoc guard; grepped AGENTS.md and docs/ for any remaining prevention claim — none found, and docs/407-correction-procedure.md:9-11 references the same rule without re-claiming prevention. Criterion 2: docs/407-correction-procedure.md read in full — covers rename steps, sealed-card-unreachable escalation, and old-ID sweep. Criterion 3: `/usr/bin/grep -q 'func checkDuplicateCardIDs' tools/doccheck/cardids.go` re-run, exit 0. `ce task gate 2>&1 | /usr/bin/grep -q '^READY —'` re-run, matched (READY — task_board_ready). `make doc-check` re-run clean (doc-check: OK, cilabels/flowcheck/planprogress/yamlcheck/changelogcheck all OK). Scope check: `git show fc7c41ef` touches only AGENTS.md (the Parallel-safe section) and this card file — no unrelated changes. Independence: commit fc7c41ef predates this review session and was already pushed to origin/dev/claude/mbp/feat/task-407 before review started."
---

## Summary

병렬 워크트리 환경에서 카드 번호 발급 시 각 워크트리가 동일한 최대 ID를 조회하여 번호 충돌(예: 383 중복 발급)이 발생하는 문제를 방지하기 위해, 병렬 충돌 방지 발급 규칙을 정립하고 개명 시 옛 참조 일괄 훑기 절차를 수립한다.

## Completion Criteria

- [x] 발급 규칙이 병렬 워크트리에서 충돌하지 않도록 정해지고 `AGENTS.md`에 적혀 있다 | verify: human — AGENTS.md의 발급 규칙을 읽고, 두 워크트리가 서로를 보지 못하는 상태에서도 같은 번호를 고르지 않는지 확인
  - 2026-09-22 measurement: the section exists and says exactly `New card ID = max(id seen in current worktree) + 1, **never** guess across worktrees`. Isolation plus `max+1` *is* the collision: two worktrees off one snapshot compute the same number. The written rule prevents nothing at allocation time; `make doc-check` DUP-ID catches it afterwards. Closing this criterion needs either an allocator that cannot collide (id derived from the worktree identity, or a claim file), or AGENTS.md rewritten to say detection is the guard — see ISSUE-033.
  - 2026-09-23 completion: rewrote AGENTS.md's "Parallel-safe card-ID allocation" section (`AGENTS.md:318-337`) to stop claiming that isolation plus `max+1` prevents a collision. It now says explicitly that two worktrees cut from the same board snapshot compute the same max and can mint the same id, that isolation does not stop this, and that `tools/doccheck/cardids.go::checkDuplicateCardIDs` (gated by `make doc-check`) — not allocation-time uniqueness — is the actual guard, to be resolved via the existing `docs/407-correction-procedure.md` rename procedure when it fires. This matches ISSUE-033's decision not to build a collision-free distributed allocator (out of proportion for a P2 with no shared-state mechanism) and closes criterion 1 honestly rather than by claiming a guarantee the design cannot keep. `go test ./tools/doccheck/` and `make doc-check` both pass with the new wording (no code changed, doc-only).
- [x] 카드 개명 절차가 옛 id 참조 일괄 훑기를 포함한다 | verify: human — 절차 문서를 읽고, 봉인된 카드에 도달하지 못하는 경우의 처리까지 적혀 있는지 확인
  - Evidence: `docs/407-correction-procedure.md` (2789 bytes) — §2 rename steps, §3 sealed-card unreachable (rename the live side; both-sealed → escalate to host), §4 old-ID sweep (`/usr/bin/grep -rq` sweep + `make doc-check` re-run).
- [x] 중복 카드 id를 재는 검사가 유지된다 | verify: `/usr/bin/grep -q 'func checkDuplicateCardIDs' tools/doccheck/cardids.go`
  - Evidence: present at `tools/doccheck/cardids.go:44`, wired into gate (`check.go:306-308`, error at `:387-388`, `DUP-ID` output `main.go:116`); `go test ./tools/doccheck/` ok, `make doc-check` exit 0.

## Sources

- ISSUE-012 — tasks/issue/012-card-ids-are-allocated-by-scanning-the-board-so-parallel-worktrees-collide.md
- ISSUE-033 — tasks/issue/033-isolated-worktrees-still-allocate-the-same-max-plus-one-card-id.md

## Review Attempts

- 2026-09-22T02:19:30Z | reviewer: opencode (independent of implementer) | executor-tier: standard | finding: pass | verification: AGENTS.md Parallel-safe section; docs/407-correction-procedure.md §3–§4; `func checkDuplicateCardIDs` at cardids.go:44; `go test ./tools/doccheck/` pass | outcome: pass | next: done | non-blocking: ISSUE-033 (isolated max+1 still races; DUP-ID is the live guard)
- 2026-09-23T00:00:00Z | reviewer: claude (independent of implementer; separate session from commit fc7c41ef) | executor-tier: standard | finding: pass | verification: AGENTS.md:318-337 re-read verbatim — now says max+1 is not collision-proof and names checkDuplicateCardIDs (make doc-check) as the actual guard, with no contradicting leftover sentence in the section or in docs/407-correction-procedure.md; `/usr/bin/grep -q 'func checkDuplicateCardIDs' tools/doccheck/cardids.go` exit 0; `ce task gate 2>&1 | /usr/bin/grep -q '^READY —'` matched; `make doc-check` clean; `git show fc7c41ef` scoped to AGENTS.md + this card only | outcome: pass | next: done | non-blocking: ISSUE-033 remains open as the pre-existing follow-up for the underlying race (unchanged by this doc-only fix)

## Re-verification (2026-09-22)

Done-board re-verification ahead of archiving. The 2026-09-22T02:19:30Z independent
review (opencode) recorded `pass` while noting in the same breath that isolated
worktrees still both do `max+1` and that DUP-ID is the live guard. That note is the
criterion's negation, so this re-verification reads the verdict as `conditional` and
returns the card rather than archiving it.

| Criterion | Verdict | Evidence |
|---|---|---|
| 1 — collision-free allocation rule in AGENTS.md | **unmet** | rule is `max+1` per worktree; identical snapshots yield identical ids (ISSUE-033) |
| 2 — rename procedure covers the old-id sweep | met | `docs/407-correction-procedure.md` §2 rename, §3 sealed-unreachable (rename the live side; both sealed → escalate), §4 sweep via `/usr/bin/grep -rq` + `make doc-check` re-run |
| 3 — duplicate-id check is kept | met | `func checkDuplicateCardIDs` at `tools/doccheck/cardids.go:44`; `go test ./tools/doccheck/` ok; `make doc-check` exit 0 |

Follow-up already exists: ISSUE-033 (`tasks/issue/033-isolated-worktrees-still-allocate-the-same-max-plus-one-card-id.md`).
No new card was opened — this card *is* the unfinished work, and ISSUE-033 is its defect record.
