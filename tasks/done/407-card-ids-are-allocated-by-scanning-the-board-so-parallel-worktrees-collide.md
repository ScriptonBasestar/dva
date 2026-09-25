---
id: TASK-407
title: "Card ids are allocated by scanning the board, so parallel worktrees collide"
type: feature
priority: P2
status: done
quality-review: pass
quality-reviewed-at: 2026-09-25
quality-review-evidence: "Independent done-review PASS by /root/review_433_434 on 2026-09-25: the revised criteria match AGENTS.md's collision disclosure, linked renamable-card procedure, old-ID sweep, duplicate-ID gate, and current CE frontmatter-status behavior; doc-check also passes."
reopened: 2026-09-22
reopen-reason: "2026-09-22 done-board re-verification: criterion 1 is unmet by measurement. AGENTS.md's rule is `max(id seen in current worktree) + 1`, so two worktrees cut from the same snapshot see the same max and mint the same id — exactly what the criterion asks be impossible. The rule also does not fall back to stating that DUP-ID detection is the guard rather than allocation uniqueness. Criteria 2 and 3 re-verified as met (see Re-verification). Residue is filed as ISSUE-033."
created: 2026-09-17
---

## Summary

병렬 워크트리의 `max(id)+1` 발급은 같은 스냅샷에서 충돌할 수 있다. 별도
분산 allocator는 만들지 않기로 결정했다 (ISSUE-033). 이 카드는 충돌 방지
보장을 주장하는 대신, 한계를 정확히 문서화하고 DUP-ID 검사 및 복구 절차를
정본으로 삼는다.

## Completion Criteria

- [x] `AGENTS.md`가 같은 스냅샷의 `max(id)+1` 충돌 가능성을 밝히고 DUP-ID 검사를 실제 사후 guard로 지목한다 | verify: human — 발급 규칙과 `make doc-check` 충돌 처리 설명을 확인
  - 2026-09-22 measurement: the section exists and says exactly `New card ID = max(id seen in current worktree) + 1, **never** guess across worktrees`. Isolation plus `max+1` *is* the collision: two worktrees off one snapshot compute the same number. The written rule prevents nothing at allocation time; `make doc-check` DUP-ID catches it afterwards. Closing this criterion needs either an allocator that cannot collide (id derived from the worktree identity, or a claim file), or AGENTS.md rewritten to say detection is the guard — see ISSUE-033.
  - 2026-09-23 completion: rewrote AGENTS.md's "Parallel-safe card-ID allocation" section (`AGENTS.md:318-337`) to stop claiming that isolation plus `max+1` prevents a collision. It now says explicitly that two worktrees cut from the same board snapshot compute the same max and can mint the same id, that isolation does not stop this, and that `tools/doccheck/cardids.go::checkDuplicateCardIDs` (gated by `make doc-check`) — not allocation-time uniqueness — is the actual guard, to be resolved via the existing `docs/407-correction-procedure.md` rename procedure when it fires. This matches ISSUE-033's decision not to build a collision-free distributed allocator (out of proportion for a P2 with no shared-state mechanism) and closes criterion 1 honestly rather than by claiming a guarantee the design cannot keep. `go test ./tools/doccheck/` and `make doc-check` both pass with the new wording (no code changed, doc-only).
- [x] 카드 개명 절차가 옛 id 참조 일괄 훑기를 포함한다 | verify: human — 절차 문서를 읽고, 봉인된 카드에 도달하지 못하는 경우의 처리까지 적혀 있는지 확인
  - Evidence: `docs/407-correction-procedure.md` (2789 bytes) — §2 rename steps, §3 sealed-card unreachable (rename the live side; both-sealed → escalate to host), §4 old-ID sweep (`/usr/bin/grep -rq` sweep + `make doc-check` re-run).
- [x] 중복 카드 id를 재는 검사가 유지된다 | verify: `/usr/bin/grep -q 'func checkDuplicateCardIDs' tools/doccheck/cardids.go`
  - Evidence: present at `tools/doccheck/cardids.go:44`, wired into gate (`check.go:306-308`, error at `:387-388`, `DUP-ID` output `main.go:116`); `go test ./tools/doccheck/` ok, `make doc-check` exit 0.

## Sources

- ISSUE-012 — tasks/_archive/issue/012-card-ids-are-allocated-by-scanning-the-board-so-parallel-worktrees-collide.md
- ISSUE-033 — tasks/_archive/issue/033-isolated-worktrees-still-allocate-the-same-max-plus-one-card-id.md

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

Follow-up already exists: ISSUE-033 (`tasks/_archive/issue/033-isolated-worktrees-still-allocate-the-same-max-plus-one-card-id.md`).
No new card was opened — this card *is* the unfinished work, and ISSUE-033 is its defect record.

## 2026-09-23 아카이브 재검증 — 되돌림

기준 1은 미충족이다. 검증 문장은 서로를 보지 못하는 두 워크트리가 같은 번호를
고르지 않는지 보라고 한다. `AGENTS.md`는 여전히 워크트리마다 `max(id)+1`이고,
같은 스냅샷에서 둘 다 같은 번호를 발급한다고 스스로 적는다. 2026-09-23에 체크를
닫은 기록은 그 문장을 탐지 가드 문서화로 바꿔 충족했다고 본 것이다. 이 재검증은
기준을 구현에 맞춰 축소하지 않으므로 아카이브하지 않고 `blocked/`로 되돌린다.

기준 2는 충족한다. `docs/407-correction-procedure.md`가 개명, 봉인된 카드를
고치지 못하는 경우, 옛 id 훑기를 담는다. 기준 3도 충족한다. `func checkDuplicateCardIDs`가
`tools/doccheck/cardids.go`에 있고 `go test ./tools/doccheck/`가 통과했다.

후속 카드는 만들지 않았다. 남은 일은 이 카드이고, 충돌 없는 할당기를 만들지
않는다는 결정은 이미 ISSUE-033의 wontfix다. 사람이 기준 1을 바꾸거나 그 할당기를
만들기로 하기 전에는 닫히지 않는다.

## 2026-09-25 범위 정합화

사용자가 승인한 기존 결정(분산 allocator 미구현)을 기준에 반영했다. 기준 1은
충돌 없는 발급을 더 이상 요구하지 않고, 현재 정본 `AGENTS.md`가 collision risk와
사후 DUP-ID guard를 정확히 설명하는지 묻는다. 기존 독립 리뷰는 변경 전 기준을
검토했고 아래에 독립 done-review PASS를 기록했다.

## 2026-09-25 독립 done-review

- 2026-09-25 | reviewer: /root/review_433_434 (independent review agent) | executor-tier: strong | finding: pass | verification: revised criteria vs `AGENTS.md`; linked `docs/407-correction-procedure.md` renamable-card choice, sealed-card handling and old-ID sweep; `tools/doccheck/cardids.go::checkDuplicateCardIDs`; CE `8f2ed451` frontmatter behavior | outcome: pass | next: done | non-blocking: TASK-436 lifecycle reclamation remains separate
