---
id: ISSUE-453
title: "DVA queue consumer lacks pinned product binary provenance"
type: bug
priority: P1
effort: M
exec-tier: strong
status: todo
severity: medium
discovered-in: "DVA queue consumer lacks pinned product binary provenance"
discovered-at: 2026-09-29
ownership: local
created: 2026-09-29
---

## Summary

`dva task-queue` and the verdict/start bridge run whichever
`taskchain-task-manager` appears first on `PATH`. The bridge validates the
queue JSON contract (`outputVersion: 1`) but cannot tell whether that binary
was built from the reviewed TaskChain source. A stale or substituted binary
with the same JSON shape can therefore supply an agent candidate before
`dva task-queue-start` mutates CE runtime state. DVA owns this consumer-side
provenance check; TaskChain release preparation is tracked in
`task-manager-devbox/task/list.md` W07c1/W07c2.

## Reproduction

1. Read `dva.yml` interactions `task-queue`, `task-queue-verdict`, and
   `task-queue-start`; none names a binary path, source revision, or digest.
2. Read `tools/taskqueueverdict/main.go` `loadQueue`: it resolves
   `taskchain-task-manager` through `PATH` and checks JSON fields, including
   `outputVersion`, but no binary provenance.
3. The code path has no further identity check before accepting a compatible
   response; a negative integration fixture is part of the resolution criteria.

## Expected vs Actual

- Expected: the consumer checks the selected binary against a reviewed,
  pinned source/artifact identity before a CE start; mismatch fails closed.
- Actual: JSON contract validation alone accepts any binary on `PATH` that
  emits a compatible queue response.

## Impact

The local W07b1/W07b2a evidence used a binary built from product commit
`9e8fac7`; this establishes that run's provenance, not future invocations.
The gap blocks a truthful W07c2 release cutover claim. It does not invalidate
the read-only queue classification or the tested sole-candidate guard.

## 소유권 — 이 저장소다

The defect is in DVA's choice and verification of its queue producer. The
TaskChain product owns the artifact and release identity, while DVA owns which
binary its interactions execute and whether that binary may trigger CE. The
consumer check therefore belongs in this repository; W07c1/W07c2 evidence is
an input, not a substitute for the check.

## P1 Blocker

- `reason`: a mutable `PATH` selection can change the queue producer without
  changing the DVA consumer or its JSON version.
- `owner`: DVA owns the consumer check and its failure behavior; W07c1 supplies
  the product candidate identity, W07c2 supplies the release/cutover decision.
- `next_action`: pin the reviewed TaskChain artifact identity in DVA and check
  the exact binary selected for queue before `task-queue-start` reaches CE.
- `next_check`: an alternate binary with valid `outputVersion: 1` is rejected
  with no CE invocation, while the matching artifact proceeds.

## Resolution Criteria

- [ ] A reviewed manifest records TaskChain source commit/tree, build toolchain and platform, build command, and artifact SHA-256 | verify: human — compare W07c1/W07c2 release evidence with the selected binary
- [ ] DVA verifies the exact selected binary against its pinned artifact identity before any CE runtime mutation | verify: human — matching and mismatching integration runs show CE call counts and binary hashes
- [ ] A stale or alternate protocol-compatible binary fails closed, and read-only queue inspection remains explicit about provenance | verify: human — run the negative and positive consumer checks and inspect the resulting diagnostics
