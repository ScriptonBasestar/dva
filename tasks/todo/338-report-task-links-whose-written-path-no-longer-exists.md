---
id: TASK-338
title: "Report task links whose written path no longer exists"
type: feature
priority: P2
effort: S
exec-tier: standard
status: todo
created: 2026-09-07
source: "PLAN-002 archive, 2026-09-07"
---

## Summary

doccheck resolves a task-card link by card id regardless of directory (TASK-143). That is
deliberate and worth keeping — it is what lets a card move between `todo/`, `done/`, and
`_archive/` without breaking every inbound link.

The cost is that a link whose written path stopped existing stays green forever. Archiving
PLAN-002 left `docs/53`, `docs/58`, and `docs/61` pointing at
`../tasks/plan/002-command-surface-delivery.md`, a path with no file behind it;
`broken_links: 0` throughout. The same happened when PLAN-003/004/005 were archived. A human
following the link in an editor gets nothing, and `grep` for the path finds a file that is not
there.

The fix is not to make these hard errors — id resolution is the feature. Report them as their
own counted category (a stale written path, distinct from a genuinely unresolvable link) so
the drift is visible and can be swept, without turning every zone move into a broken build.

## Completion Criteria

- [ ] doccheck counts and lists links that resolve by id but whose written path does not exist | verify: `/usr/bin/grep -rq "stale_link_paths" tools/doccheck`
- [ ] the new category is reported separately from `broken_links` and does not by itself fail the gate | verify: `go test ./tools/doccheck/` (regression-guard)
- [ ] the existing stale paths in `docs/` and `tasks/` are swept to zero, or the remainder is listed with a reason | verify: `make doc-check`
