---
id: TASK-344
title: "Harden frontmatter value parsing against trailing comments and repeated keys"
type: fix
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-07
---

## Summary

`frontmatterField` (`tools/doccheck/cardstatus.go`) returns everything after the colon with
only whitespace trimmed. Two YAML forms therefore parse into a value no reader intends:

- A trailing comment is kept. `id: TASK-331 # duplicate of the other one` yields the value
  `TASK-331 # duplicate of the other one`, which matches no other card, so the card silently
  leaves the id space. Probed: adding such a card gives `card_ids: 345 (duplicate: 0)` and
  `doc-check: OK` — a comment is enough to walk past the duplicate-id guard.
- A repeated key takes the first occurrence and ignores the rest, with no report. YAML itself
  treats a duplicate mapping key as an error in strict parsers; here it resolves quietly.

Both predate `checkDuplicateCardIDs` and affect `cardStatus` identically, so a status value
carrying a comment has always been compared verbatim against the allowed set. The reason to
fix it now is that the duplicate-id check turned a lenient parse into a *bypass*: before, a
malformed value produced a mismatch someone would notice; now it produces silence.

`archive.go:130-148`'s `hasCanonicalField` uses the same line-scan shape, so any fix belongs in
the shared helper rather than at one call site, and the two must not drift apart.

Scope note: this is deliberately not a YAML parser. The corpus is machine-written cards with a
fixed field set, and adopting a real parser would change what the whole sweep accepts. Strip a
trailing ` #` comment, and report a repeated key rather than resolving it — nothing wider.

## Completion Criteria

- [ ] A trailing `#` comment is stripped from a frontmatter value, so `id: TASK-331 # note` collides with `id: TASK-331` | verify: `/usr/bin/grep -rq 'func TestFrontmatterValueDropsTrailingComment(' tools/doccheck`
- [ ] A `#` inside a quoted value survives, so titles containing `#` are unchanged | verify: `/usr/bin/grep -rq 'func TestQuotedValueKeepsItsHash(' tools/doccheck`
- [ ] A repeated frontmatter key is reported rather than silently resolved to the first | verify: `/usr/bin/grep -rq 'func TestRepeatedFrontmatterKeyIsReported(' tools/doccheck`
- [ ] `cardStatus` and `hasCanonicalField` share the hardened helper | verify: `make test`
