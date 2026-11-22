---
id: TASK-634
title: "ci: decide repository-reuse checkout containment by file identity"
type: bug
priority: P2
effort: S
exec-tier: strong
status: todo
created: 2026-11-22
---

## Summary

`pathInside` in `internal/cirun/repository_reuse.go` decided checkout containment by comparing
path strings, so on a case-insensitive volume (macOS default APFS) a case-variant spelling of a
checkout path was judged outside the checkout, and a stat failure counted as not inside
(fail-open). Containment is now decided by file identity (`os.SameFile` over the canonical
ancestor chain), every caller refuses when containment cannot be decided, and the dead `..`
branch of `resolveMissingPath` is gone, with the dot refusal kept once upstream.

## Completion Criteria

- [ ] A case-variant spelling of a checkout path on a case-insensitive volume is refused as checkout-local, and identity cases (sibling prefix, outside symlink, missing dotted path) hold. | verify: `go test ./internal/cirun -run PathInside -count=1`
- [ ] Containment that cannot be decided (stat failure) refuses reuse and names the cause; the dotted missing entry is refused upstream. | verify: `go test ./internal/cirun -run RepositoryReuse -count=1`
