# Sealed done-card factual correction (TASK-406)

A `blocks:` done card whose `quality-review-receipt` is readable is sealed:
`ce task validate` pins `reviewed-card-sha256` to the card bytes. Editing the
card without a new independent review makes the board invalid. Rewriting the
receipt digest from the validator error, without reviewing, is forgery.

This procedure is the non-forgery path. It does not weaken the seal.

## 1. Decide whether the card is sealed

All three must hold (`ISSUE-010`):

1. The file sits under `tasks/done/` (`zoneFromPath`, not the `status:` field).
2. Frontmatter declares `blocks:`.
3. `quality-review-receipt` points at a file that this checkout can read.

If any one is missing, `ce task validate` never reaches the digest check.
`quality-review-receipt` itself is excluded from the digest: changing only
that pointer does not break a seal.

## 2. Forgery (forbidden)

- Copy the sha256 from a validate error into a new or existing receipt.
- Backdate `reviewed-at` or name a reviewer who did not read this revision.
- Edit a historical receipt body. Receipts are immutable
  (`tasks/done/evidence/README.md`).

## 3. Formal path for a live sealed card

1. Edit the factual error on the card in a task worktree. The seal will fail
   until the next step finishes. Do not mint a digest to silence that.
2. An independent reviewer (not the todo-resolve executor) re-runs every
   required criterion against the new bytes.
3. That reviewer writes `quality-review`, `quality-reviewed-at`, and
   `quality-review-evidence` for this revision.
4. Hash the card, write a **new** receipt at
   `tasks/done/evidence/<TASK-ID>/done-review-<sha>.json` with
   `reviewed-card-sha256` equal to that hash. Keep the old receipt file.
5. Point `quality-review-receipt` at the new file (this field is not in the
   digest).
6. Run `ce task validate` on the card. It must pass because the new receipt
   matches, not because anyone copied the error string.

TASK-384 is the first worked example of independent re-review plus a new
receipt. Use it as the shape, not as a backdate.

## 4. Archived cards

Once the file leaves `tasks/done/`, step 1 of the seal is false. Editing
`tasks/archive/` does not forge a done receipt. Still do not rewrite the
historical receipt. Record the correction on the archived card and leave
`quality-review-receipt` pointing at the old file.

TASK-376 is this case: archived 2026-09-17, `quality-review-evidence` claimed
every Low finding was fixed. The first half of one Low (`:441/:446` EXIT trap
on log/preview temps) was deliberately not applied (TASK-387). The archived
card's evidence line is the place to say so.

## 5. After the correction

Run `ce task validate` on the edited card (and `make doc-check` if the edit
is under a size-gated tree). Do not run validate expecting a done-zone seal
on an archive path.
