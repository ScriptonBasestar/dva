---
id: TASK-333
title: "Expose owner and canonical/alias identity for imported items"
type: feature
priority: P2
effort: M
exec-tier: standard
status: todo
created: 2026-09-07
source: "tasks/done/259 done-review (PLAN-007 Tier A batch 1)"
---

## Summary

TASK-259 §5 items 1-2 were scoped out of that card and never refiled. An imported item
currently renders identically to a locally declared one, so a user cannot tell where it came
from or which of several addresses is the canonical one.

Two additions:

1. An `owner` field on every item in `dva ls --json` and in the manifest output, naming the
   subproject the item was imported from (or the root for locally declared items).
2. Canonical/alias markers, so an item reachable under more than one address states which
   address is canonical and which are aliases.

Shell completion must offer all three address forms (`--project <name>`, `<name>:<key>`,
`<name>/<key>`) rather than only the canonical one — an alias the user can type but cannot
complete is a worse surface than no alias.

## Completion Criteria

- [ ] `dva ls --json` and the manifest output carry an `owner` field for every item | verify: `/usr/bin/grep -rq "func TestImportedItemOwnerField" internal/cli`
- [ ] items reachable under multiple addresses mark one canonical and the rest aliases | verify: `/usr/bin/grep -rq "func TestImportedItemCanonicalAndAliasMarkers" internal/cli`
- [ ] completion offers all three address forms | verify: `/usr/bin/grep -rq 'func TestCompletionOffersAllThreeAddressForms(' internal/cli`
