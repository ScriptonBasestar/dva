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

`dva task-queue` and the read-only verdict bridge still use the first
`taskchain-task-manager` on `PATH`. The earlier YAML start bridge could pass a
protocol-compatible stale binary's candidate into CE. TASK-454 replaced that
production route with a compiled command that disables CE start while no
artifact is approved. TASK-455 connected the verified producer path to that
command; the embedded candidate remains unauthorized. The remaining work is
approval of a published artifact, pin activation, and positive/negative host
checks. DVA owns this consumer-side provenance check; TaskChain release
preparation is tracked in `task-manager-devbox/task/list.md` W07c1/W07c2a.

## Reproduction

1. Read `dva.yml`: only read-only `task-queue` and `task-queue-verdict` remain
   as PATH-based interactions.
2. Read `internal/cli/task_queue_start.go`: the compiled start command rejects
   before queue or CE invocation. Its production activation remains pending.
3. Read `internal/taskqueue/pin.go`: the verifier binds the first PATH result
   to a hashed snapshot and is connected to the compiled start command. The
   production manifest remains unauthorized. Positive host evidence belongs to
   W07c2a.

## Expected vs Actual

- Expected: the consumer checks the selected binary against a reviewed,
  pinned source/artifact identity before a CE start; mismatch fails closed.
- Actual: the compiled path verifies a pinned snapshot when an approved pin
  exists, but production CE start remains disabled because the embedded pin is
  unauthorized. Read-only queue outputs remain unpinned.

## Impact

The local W07b1/W07b2a evidence used a binary built from product commit
`9e8fac7`; this establishes that run's provenance, not future invocations.
The gap blocks a truthful W07c2a release activation claim. It does not invalidate
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
  the product candidate identity, W07c2a supplies the release/pin decision.
- `next_action`: after W07c1 and public approval, record the reviewed published
  artifact pin in DVA, activate it, and prove the selected binary's hash and
  CE call counts on an actual host. The compiled verifier is already connected.
- `next_check`: an alternate binary with valid `outputVersion: 1` is rejected
  with no CE invocation, while the matching artifact proceeds.

## Resolution Criteria

- [ ] A reviewed manifest records TaskChain source commit/tree, build toolchain and platform, build command, and artifact SHA-256 | verify: human — compare W07c1/W07c2a release evidence with the selected binary
- [ ] DVA verifies the exact selected binary against its pinned artifact identity before any CE runtime mutation | verify: human — matching and mismatching integration runs show CE call counts and binary hashes
- [ ] A stale or alternate protocol-compatible binary fails closed, and read-only queue inspection remains explicit about provenance | verify: human — run the negative and positive consumer checks and inspect the resulting diagnostics

## Staged mitigation (TASK-454, 2026-09-29)

The compiled `dva task-queue-start` command now fails before TaskChain or CE while
no product artifact is approved. A DVA-owned candidate manifest records the
internal build with `mutationAuthorized: false`; flipping that flag alone cannot
authorize the candidate. Snapshot/hash execution is covered by injected-pin
tests and the compiled DVA command (TASK-455). The read-only Go verdict tool
cannot initiate CE. This issue remains open until W07c2a provides a published,
reviewed artifact, activates the pin, and records positive and negative
host-level CE call counts. The compiled DVA root normalizes a failed child CE
exit to 1; W07c2a host checks must assert nonzero failure and inspect CE
status rather than expect the child's exact exit code.
