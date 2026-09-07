---
id: TASK-334
title: "Cover config env show against real sops by allocating a pty"
type: test
priority: P1
effort: M
exec-tier: strong
status: todo
created: 2026-09-07
source: "tasks/done/282 done-review (PLAN-007 Tier A batch 2)"
depends-on: [TASK-282]
---

## Summary

TASK-282's criterion "cover the real-sops path for both commands" is checked, but only
`seal` is actually covered (`TestConfigEnvSealRealSOPSRoundTrip`). No test exercises
`show`'s real `bridgeSops.Decrypt` path. The card's own evidence says so — "`show`: still
not covered" — so the gap is recorded, not hidden; this card carries it.

`show` writes decrypted bytes only to `bridgeOpenTTY()`'s real `os.OpenFile("/dev/tty", ...)`,
so proving the path end to end requires the child `dva` process to hold a controlling
terminal. Nothing in the harness allocates one, and `sops`/`age`/existing dependencies do not
provide one. Allocating a pty is OS-specific (darwin and linux differ in ioctls) and this repo
prefers direct syscalls over new dependencies, so the approach needs a decision before code:
either a small platform-split pty helper under `internal/integration/`, or a documented
decision that the fake-driven `TestConfigEnvShowFaultMatrix` plus the real-binary terminal-gate
subtest is the accepted ceiling.

Whichever way it goes, TASK-282's checkbox must stop overstating the coverage.

## Completion Criteria

- [ ] a decision is recorded on whether to allocate a pty or accept the fake-driven ceiling, with the reasoning | verify: human — the decision and its basis appear in this card or in a docs/ Decision Record, and TASK-282's Review Log links to it
- [ ] if a pty is allocated, `show` decrypts a real sops-encrypted secret end to end and the plaintext is asserted byte-identical | verify: `/usr/bin/grep -rq "func TestConfigEnvShowRealSOPS" internal/integration`
- [ ] the Linux leg is either verified in the config-env-platform CI job or explicitly excluded with a stated reason — an unvalidated Linux path must not ship silently | verify: human — CI result or a stated exclusion in this card
- [ ] TASK-282's real-sops criterion no longer claims coverage `show` does not have | verify: `/usr/bin/grep -q "TASK-334" tasks/done/282-implement-gated-env-bridge-commands.md`
