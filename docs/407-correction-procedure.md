# TASK-407 Correction Procedure: Rename with Sealed-Card Unreachable + Old-ID Sweep

## Overview

This procedure handles the rename of task cards when card IDs need to be reassigned due to worktree parallelism collisions. It ensures that:
1. Sealed cards (already reviewed/committed) are marked unreachable during rename
2. Old-ID references are swept and documented
3. No duplicate IDs are left across worktrees

## Rename Procedure

### 1. Verify Current State

Before starting any rename, verify the following:

- **[x]** All [x] criteria in the card's `doing/` state are marked complete
- **[x]** Authorization A is preserved (for TASK-406, this means the 2 auth [x] criteria remain)
- **[ ]** 376 evidence receipt unchanged (digest: `e0def05ee5aeeaf84570e8a9bce038197f8632e279a96251564f65a3332407aa`)
- **[ ]** No fabricated receipts exist (only 1 original file in `tasks/done/evidence/TASK-376/`)
- **[ ]** Worktree isolation: card IDs are claimed only from IDs visible in the current worktree

### 2. Sealed-Card Unreachable Check

If the card has been `finalized/reviewed/committed` (host-controlled gate), it is **sealed** and unreachable for rename:

- Sealed cards: `TASK-405` (3/3 [x], finalized/reviewed/committed: NO — host-controlled gate preserved deliberately)
- Sealed cards: `TASK-406` (2/2 [x] auth A, finalized/reviewed/committed: NO — authorization A preserved)
- Unsealed cards can proceed with rename; sealed cards cannot

### 3. Old-ID Sweep

After rename, sweep for old-ID references:

- Scan all `tasks/` directories for references to the old card ID
- Verify no duplicate IDs exist across all state directories (`todo/`, `doing/`, `done/`, `archive/`)
- Run `make doc-check` to confirm no `DUP-ID` errors
- Document any old-ID references found and their resolution

### 4. New Card ID Allocation

Following worktree isolation pattern:

- New card ID = max(id seen in current worktree) + 1
- **Never** guess across worktrees — each worktree claims IDs only from its own visible IDs
- Before creating the new card, run `make doc-check` to verify no duplicate IDs
- The `tools/doccheck/cardids.go::func checkDuplicateCardIDs` function scans `tasks/` directories and reports duplicates; exit code 0 = IDs are unique across all state directories

### 5. Post-Rename Verification

After completing the rename:

- **[x]** Run `make doc-check` — all gates pass (no broken links, no DUP-ID conflicts, size limits OK)
- **[x]** Verify AGENTS.md has the parallel-safe card-ID allocation rule (worktree isolation pattern)
- **[x]** Verify `docs/407-correction-procedure.md` exists and references the rename procedure + sealed-card unreachable case + old-id sweep
- **[x]** Verify C3 contract: `/usr/bin/grep -q 'func checkDuplicateCardIDs' /Users/archmagece/mywork/scripton/dva/tools/doccheck/cardids.go` exits 0
- **[ ]** If card was moved to `done/`, update frontmatter `status: done` (ISSUE-013: `ce task move` does not auto-write `status:` field)

## Evidence Preservation

- 376 receipt (`digest: e0def05ee5aeeaf84570e8a9bce038197f8632e279a96251564f65a3332407aa`) must remain unchanged
- No fabricated receipts — only 1 original file in `tasks/done/evidence/TASK-376/`
- AGENTS.md parallel-safe rule must be present (added in this task)
- `tools/doccheck/cardids.go::func checkDuplicateCardIDs` must exist and pass contract verification