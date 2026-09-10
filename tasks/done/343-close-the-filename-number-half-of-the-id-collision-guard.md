---
id: TASK-343
title: "Close the filename-number half of the task id collision guard"
type: feature
priority: P2
effort: S
exec-tier: standard
status: done
created: 2026-09-07
needs-human: false
depends-on: [TASK-344]
---

## Summary

`checkDuplicateCardIDs` (`tools/doccheck/cardids.go`) closed the link-resolution failure mode:
doccheck resolves task links by frontmatter `id` regardless of directory (TASK-143), so two
cards claiming one id made the tree ambiguous, and that is now a hard error.

It closed the second failure mode only halfway. A worker that numbers a new card from the
highest *filename* it can see — which is how the 2026-09-07 TASK-331 collision actually
happened — can still land two cards on one number as long as their frontmatter ids differ.
Probed on the real corpus: writing a second card numbered `331` in the todo zone, carrying
`id: TASK-900` beside the existing `331-plan-scoped-lifecycle-hooks` card, yields
`card_ids: 345 (duplicate: 0)`, `doc-check: OK`, and `planprogress: OK`.

Neither tool catches it, for different reasons. doccheck keys on the frontmatter id, so two
distinct ids are two distinct cards no matter what the files are named. `planprogress` keys on
the leading filename digits off a filesystem walk, so it folds both files into a single
`taskRecord` and sees nothing unusual — and its own type comment scopes uniqueness out.

The gap matters because the filename number is what a human reads when picking the next id,
and what a stale-based agent reads when it invents one. An id space that is unique in
frontmatter but ambiguous on disk still misleads the next writer.

Implementation pointer: `taskRecord` already carries the data this check needs. `paths` is
appended to at `tools/planprogress/check.go:70` and read nowhere — a per-id list of the files
that folded into one record is exactly the collision evidence, so the check is closer to
consuming a dead field than to collecting new state.

Design note: the filename number and the frontmatter id are two separate namespaces that are
conventionally kept in sync, and nothing enforces the convention either. A card whose filename
says `331` and whose id says `TASK-900` is currently legal. Deciding whether that is a second
defect or an intentional degree of freedom belongs in this card, before writing the check —
enforcing agreement is a stricter rule than merely forbidding filename collisions, and would
reject the archive's `PLAN-00n` cards unless those zones are exempted.

## Design note resolution — do not require filename/id numeric agreement

The guard groups a leading filename number by the frontmatter ID namespace (`TASK`, `ISSUE`,
and so on), then rejects duplicate pairs only within that namespace. It does **not** require a
filename number to equal the numeric suffix of `id:`. That equality is a useful convention, but
the current corpus and task format treat filename numbering and the full frontmatter identity as
separate inputs; making it mandatory would reject compatible cards without evidence that their
freedom is a defect. This card's narrowly evidenced failure is two distinct `TASK-*` ids sharing
one visible filename number. `tasks/plan/` and archived `tasks/_archive/plan/` are excluded, and
the repository's intentional `ISSUE-001` / historical `TASK-001` overlap remains valid because
they are separate namespaces.

## Ordering — land TASK-344 first

This card adds a filename-number check beside `checkDuplicateCardIDs`, which reads
ids through `frontmatterField` (`tools/doccheck/cardids.go:60`). TASK-344 hardens
that same helper — stripping trailing `#` comments from values and reporting
repeated keys — and folds `hasCanonicalField` into it. So the value this card's
check sees is exactly what TASK-344 changes.

The work is possible in either order — it is just wasteful the other way, because the
tests written here would be rewritten once TASK-344 changes what the helper returns.
But note what declaring it costs: `ce task preflight` counts `unmet-dependency` as
**blocking**, not advisory, so this card leaves the runnable set entirely until
TASK-344 closes. That is deliberate. TASK-344 is P3 and small while this card is P2,
so without the declaration a runner picking by priority would reliably choose the
expensive order.

## Completion Criteria

- [x] Two task cards sharing a leading filename number in the same id space are reported, with distinct frontmatter ids | verify: `/usr/bin/grep -rq 'func TestDuplicateFilenameNumbersAreReported(' tools/doccheck`
- [x] The `PLAN-00n` / `TASK-00n` archive overlap and any zone deliberately exempted stay silent | verify: `/usr/bin/grep -rq 'func TestPlanAndTaskNamespacesDoNotCollideByNumber(' tools/doccheck`
- [x] The decision on whether filename number and frontmatter id must agree is recorded in this card before the check is written | verify: human — read the Design note resolution in this card
- [x] Gates stay green | verify: `make doc-check` (regression-guard)

## Verification evidence

2026-09-10: `go test ./tools/doccheck`, `make doc-check`, and `go run ./tools/planprogress`
passed. The real corpus reported `filename_numbers: 373 (duplicate: 0)` after excluding active
and archived plan paths; the intended historical `TASK-001` / `ISSUE-001` filename overlap
remained silent.
