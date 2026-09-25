---
id: ISSUE-044
title: "Document canonical status written by issue resolve"
type: bug
priority: P3
effort: XS
exec-tier: cheap
status: todo
severity: medium
ownership: upstream
upstream-ref: "ce-agent-kit#3 (related strict-status contract; this docs/test follow-up is not separately filed)"
discovered-in: "Document canonical status written by issue resolve"
discovered-at: 2026-09-25
created: 2026-09-25
---

## Summary

CE's task-command reference says `ce task resolve` writes three resolution fields, but
`Resolver.Resolve` also writes `status: done`. The resolver test is still named
`TestResolve_stampsAllThreeFields` even though it asserts all four fields. This omission hides
the strict validator's intentional rule that a resolved issue may remain temporarily in
`issue/` with `status: done` only when `resolution:` is nonempty, making the canonical writer
output look like a zone-status violation to repository users.

## Reproduction

1. Compare `docs/10-ce-cli/26-cli-reference-task-commands.md`'s resolve section with
   `internal/usecase/task/resolver.go` and `resolver_test.go`.
2. Run `ce task resolve` on an issue and inspect the diff; the command writes
   `resolution`, `resolved-at`, `status: done`, and `resolution-summary`.
3. Under strict status policy, validate the result. CE accepts the output; `ce task lint`
   then recommends archiving the resolved issue from `issue/`.

## Expected vs Actual

- Expected: CLI reference and test name describe all fields the resolver writes, and direct
  validator coverage rejects `status: done` on an unresolved issue.
- Actual: reference and test name describe three fields, while the resolver writes four; no
  direct negative fixture was found for `issue/ + status: done` without `resolution:`.

## Resolution Criteria

- [ ] Update the resolve reference and resolver test name to include `status: done`; add a
  negative strict-validator fixture for done-without-resolution | verify: human — compare the
  reference, resolver output, and strict validator tests in ce-agent-kit

## Related

- [TASK-424](../done/424-validate-owns-zone-status.md)
- [TASK-426](../done/426-write-frontmatter-status-on-move-and-close.md)

## 소유권 — 상류

DVA records the observed status-policy comparison and local reproduction. ce-agent-kit owns
the resolver reference, resolver test name, and direct negative validator fixture because all
three live in its source. The upstream reference points to the related strict-status contract;
this documentation/test-name follow-up has not been separately filed in CE.
