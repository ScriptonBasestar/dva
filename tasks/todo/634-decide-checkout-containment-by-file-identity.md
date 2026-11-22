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

## Review

Independent Opus review of cd1066ea and d7abf3c8 (rebased as 90ca7087..512a02fa) found no
fail-open across 12 probe kinds (case variant, NFD, symlinks in and out, missing tail under a
symlink, symlinked root, the /var alias, a firmlink, `//` and trailing slash, `/./`, `/`), all
three callers failing closed through `refuseCheckoutLocal`, and killed a SameFile-to-string
mutation; low notes: a hard-linked outside tool and a TOCTOU window outside the threat model.
VERDICT: PASS. The reviewed patch equals this branch's code commits: `git patch-id --stable`
of `git diff 401225f1 512a02fa` and of `git diff fd706b16 eacf8f19` are both `acfa9f9e…`, and
`git diff 512a02fa eacf8f19` over the four touched files is empty (the whole-tree diff is not
empty only because the base moved from 401225f1 to fd706b16).

## Completion Criteria

- [ ] A case-variant spelling of a checkout path on a case-insensitive volume is refused as checkout-local, and identity cases (sibling prefix, outside symlink, missing dotted path) hold. | verify: `go test ./internal/cirun -run PathInside -count=1`
- [ ] Containment that cannot be decided (stat failure) refuses reuse and names the cause; the dotted missing entry is refused upstream. | verify: `go test ./internal/cirun -run RepositoryReuse -count=1`
