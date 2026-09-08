---
id: TASK-335
title: "Close the seal create-only TOCTOU window before the final rename"
type: bug
priority: P1
effort: M
exec-tier: strong
status: todo
created: 2026-09-07
source: "tasks/done/282 done-review (PLAN-007 Tier A batch 2)"
depends-on: [TASK-282]
---

## Summary

`dva config env seal` is specified as create-only: TASK-281 §2-1 states the lost update is
"발생 자체가 불가능". A narrow race reopens it.

`source_exists` (seal matrix row 16) is checked once, in `sealPreflight`. Two things happen
after that check and before the write:

1. the TTY confirmation prompt, which waits on human input for an unbounded time, and
2. the `sops encrypt` run.

The final write, `safeWriter.Commit`, is a plain `Rename` after `stillAnchored()` — it never
re-checks that the target is still absent, so it overwrites whatever appeared in the window.
A concurrent `seal`, a `git pull`, or an `edit` landing during the prompt is silently
overwritten.

The card's criterion "an existing source is never opened for write" is literally true — the
path renames rather than opens — which is why the matrix tests pass while the guarantee is
broken. The fix has to bind the absence check to the write itself, not to preflight.

## Completion Criteria

- [ ] the create-only guarantee is enforced at the write, not only at preflight — the commit path refuses when the target appeared after the check | verify: `/usr/bin/grep -rq "func TestConfigEnvSealRefusesSourceCreatedAfterPreflight" internal/cli`
- [ ] the refusal reuses `codeSourceExists` rather than adding a code — `envBridgeCodes` stays identical to the frozen list asserted in `internal/cli/config_env_grammar_test.go` | verify: `make test`
- [ ] the seal fault matrix gains a row covering the post-preflight-appearance case | verify: `/usr/bin/grep -q 'source appears between preflight and commit' internal/cli/config_env_seal_test.go`
