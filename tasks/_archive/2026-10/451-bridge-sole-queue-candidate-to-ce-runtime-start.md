---
id: TASK-451
title: "Bridge sole queue candidate to CE task runtime start"
type: feature
priority: P1
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-reviewed-at: 2026-09-28
quality-review-evidence: "Independent done-review session /root/task240_done_review PASS: verified explicit type and sole candidate gating, fixed tasks board, normalized TASK/ISSUE key, exact CE argv/cwd and no CE calls for other states, CE success and nonzero structured recovery output, board invariance, and W07b2b terminal boundary. Independent code review PASS after two defects fixed. Targeted tests/vet/config/doc/gate PASS; full DVA CI 85309aa644a6d1103d85de07477c8ef8 succeeded."
created: 2026-09-28
execution-mode: implementation
allowed-paths: [tools/taskqueueverdict, internal/integration, dva.yml, README.md]
archived-at: 2026-10-03
---

## Summary

W07b2a. DVA uses the strict read-only queue verdict and calls CE `task
run-start` only when an operator explicitly supplies a branch type and the
agent queue has exactly one candidate. The adapter maps TASK/ISSUE identity
deterministically to a CE runtime key, forwards CE success output and treats
CE failures as nonzero (the Go launcher may normalize the exact exit code),
and keeps CE as the sole Git/worktree and card-state writer. It neither
claims through TaskChain nor moves or completes a card. W07b2b owns actual
success, blocked retry, discard, and terminal/rollback host evidence.
CE's structured failure response remains visible because a failed start may
have already created worktree or runtime state; operators inspect CE status
before retrying. Start mode is bound to this repository's `tasks` board.

## Completion Criteria

- [x] Only an explicit operator type and exactly one validated candidate can invoke CE run-start | verify: human — adapter unit tests and independent code review PASS
- [x] Empty, human-only, multi-candidate, invalid queue/type, and alternate board invoke no CE writer | verify: human — negative DVA interaction fixture PASS
- [x] DVA interaction works from unrelated cwd, forwards CE success and blocked response, and does not change the board itself | verify: human — nested-cwd DVA integration fixture PASS

## Evidence (2026-09-28)

`go test ./tools/taskqueueverdict -count=1`, the targeted tagged DVA
integration test, `go vet ./tools/taskqueueverdict`, `dva config validate`,
`make doc-check`, and `ce task gate` passed. Full DVA CI run
`85309aa644a6d1103d85de07477c8ef8` succeeded. Independent code review
PASS after fixing the alternate-board and CE failure-response defects.
The done-review observation about three unit fixtures was corrected before
this final CI run: each now reaches its intended verdict state.
The real product binary `9e8fac7` from a nested cwd yielded an empty queue;
`dva task-queue-start feat` exited 1 with zero stdout and invoked no CE writer.
The synthetic fixture proves TASK/ISSUE identity mapping, exact CE argv/cwd,
single invocation, and observable BLOCKED JSON with nonzero exit. It does
not prove real CE finish/discard or terminal card transitions, which remain
W07b2b. ISSUE-004/006 remain open.
