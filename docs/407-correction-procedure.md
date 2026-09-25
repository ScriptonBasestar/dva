# Card rename procedure: fixing a duplicate card ID (TASK-407)

When `make doc-check` reports a `DUP-ID` conflict — two card files claiming one
task id after parallel worktrees allocated the same number — rename exactly one
of the two cards following this procedure.

## 1. Allocate the replacement ID

- New card ID = max(id visible in the current worktree) + 1.
- **Never** guess across worktrees; each worktree claims IDs only from its own
  visible tree (rule: `Parallel-safe card-ID allocation (worktree isolation)`
  in `AGENTS.md`).
- Before creating the renamed card, run `make doc-check` to confirm the
  `DUP-ID` it reports is the one being fixed and no other duplicate exists.

## 2. Rename the card

1. Rename the file `tasks/<state>/NNN-<slug>.md` to the new number and update
   its frontmatter `id:` to match.
2. After any `ce task move`, verify that frontmatter `status:` matches the
   destination. CE source at `8f2ed451` and later writes it automatically; an older
   installed binary may leave it stale, in which case correct it by hand. Then run
   `make doc-check`, which enforces zone/status agreement.
3. Commit only the renamed card path plus this procedure doc when it changes;
   never bundle unrelated cards into the same commit.

## 3. Sealed-card unreachable case

A sealed card cannot be renamed — treat it as unreachable and rename the other
side instead:

- Sealed means the card sits in `tasks/done/` with a `quality-review` verdict
  in its frontmatter, optionally with a review receipt under
  `tasks/done/evidence/<TASK-ID>/`. Done cards and their receipts are
  immutable: do not move, edit, or re-id them.
- If the duplicate pair is one sealed card and one live card, the live card
  is always the one renamed.
- If both cards claiming the id are sealed, stop and escalate to the host:
  resolving it requires a human decision about which history stands. Record
  the `DUP-ID` output and both card paths in the escalation; do not force a
  rename.

## 4. Old-ID sweep

After the rename, sweep every reference to the retired ID:

1. Search all card zones for the old id string:
   `/usr/bin/grep -rq --include='*.md' 'OLD-ID' tasks/` (absolute path keeps
   the invocation reproducible in an ordinary shell).
2. Repoint each match to the new ID, or to whichever state directory actually
   holds the card — doccheck resolves `tasks/<state>/NNN-…` links and the same
   path inside inline code (where `verify:` bindings live) to the single
   directory holding `NNN-…` (TASK-143), so one match resolves and zero is a
   genuine broken link.
3. Re-run `make doc-check` and confirm zero `DUP-ID` lines; also run
   `go test ./tools/doccheck/` when the sweep touched checker-adjacent files.
4. Confirm the duplicate-ID guard is still wired:
   `/usr/bin/grep -q 'func checkDuplicateCardIDs' tools/doccheck/cardids.go`.
