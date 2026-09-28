---
id: ISSUE-046
title: "Unpushed env-reseal commit has no card or owner confirmation"
type: bug
priority: P2
effort: XS
status: done
severity: medium
discovered-in: "2026-09-28 board refresh worktree inventory"
discovered-at: 2026-09-28
ownership: local
created: 2026-09-28
resolution: fixed
resolved-at: 2026-09-28T10:42:42Z
resolution-summary: "Resolved as fixed."
---

## Summary

Worktree `~/worktrees/misc/dva/grok__mst__fix__env-reseal` holds branch
`dev/grok/mst/fix/env-reseal` with one commit not on master: `a11b519e`
"fix(cli): re-encrypt an existing env source in place" (new `dva config env reseal`,
`internal/cli/config_env_reseal.go` plus test, USAGE and schema-reference lines,
dated 2026-09-27). It is not pushed, no card cites it, and `ce task run-list` shows no
execution for it. The board could not see this work; only `git worktree list` did.

## Reproduction

1. `git worktree list` in the primary checkout.
2. `git log --oneline master..dev/grok/mst/fix/env-reseal` → 1 commit.
3. `git branch -r --contains a11b519e` → empty.

## Expected vs Actual

- Expected: agent work in progress has a card or a lifecycle execution, so the board shows it.
- Actual: a finished-looking secrets-path feature sits only in a local branch of another agent.

## 소유권 — 이 저장소다

Decision (2026-09-28): ask the owning session or the user before acting, to avoid the
parallel-session duplication recorded elsewhere. If the work is abandoned, register a
task, run an independent review (it touches the secrets path), and integrate it through
the lifecycle. Do not modify or remove the worktree before that answer.

## Board-refresh lesson

A board refresh should include `git worktree list` and `ce task run-list`, because
work without a card is invisible to `tasks/`.

## Resolution evidence (2026-09-28)

The original `a11b519e` object, `dev/grok/mst/fix/env-reseal` branch, and
grok worktree are no longer present. Current `master` contains
`b42d0a915f8469f9f97494940371d625e4c82ef4`, whose feature subject and
five changed paths match the original report, including
`config_env_reseal.go` and its test. `git merge-base --is-ancestor b42d0a91
master` exits zero. The missing original object prevents a patch-id or
content-equivalence check, and the available evidence does not identify who
removed the grok worktree or when. The reported unpushed branch/worktree
condition is no longer present; this reconciliation changed only this issue
record and the board index.
