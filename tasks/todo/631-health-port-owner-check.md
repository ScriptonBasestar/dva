---
id: TASK-631
title: "health: verify the tracked process owns the probed port after start"
type: feature
priority: P3
effort: M
exec-tier: standard
status: todo
created: 2026-11-22
allowed-paths: [internal/lifecycle/, USAGE.md, CHANGELOG.md, tasks/]
---

## Summary

The TASK-538 pre-start guard (3e2d3927) only refuses ports already served before `up`. If
another process takes the port after start, or the entry fails to bind while something else
answers, the health check still passes against the wrong owner. Where the OS allows it
(lsof/`/proc` on unix), check that the listening socket belongs to the tracked process group.
Otherwise state explicitly that ownership was not verified.

## Completion Criteria

- [x] When the health endpoint's listener is not in the tracked process group, health reports a port-owner mismatch naming the address and owning pid. | verify: human — internal/lifecycle에 추가한 PortOwner 테스트 통과를 독립 리뷰가 확인
- [x] Platforms without owner lookup report "owner unverified" rather than a pass or failure. | verify: human — internal/lifecycle에 추가한 PortOwner 테스트 통과를 독립 리뷰가 확인

## Decision (2026-11-22, user)

- C1: macOS uses `lsof`, Linux reads `/proc`. Where the owner cannot be looked up the result
  is "owner unverified"; a listener owned by another process group fails health.

## Outcome

- internal/lifecycle/port_owner.go: `checkHealthPortOwners` resolves each local http/tcp
  health address, looks up its listening pids (port_owner_darwin.go `lsof -t -sTCP:LISTEN`,
  port_owner_linux.go `/proc/net/tcp{,6}` inode → `/proc/<pid>/fd`), and compares each pid's
  process group with the tracked pid (which leads its own group, `Setpgid`). Other OSes,
  remote hosts, a failed lookup or an unreadable group are unverified.
- `up` (composition_wait_ready.go): after the probes pass with the tracked pid alive, a
  mismatch fails with "health port owner mismatch"; unverified addresses log a warning.
- `status` (ProcessPlugin.Status, running): mismatch → `error` + `extra.port_owner`;
  unverified → `extra.port_owner_unverified`; the table prints `!` / `?` lines.
- Tests: internal/lifecycle/port_owner_test.go (`-run PortOwner`), including a real-lookup
  test that finds the test process's own listener on darwin/linux. Mutation: disabling the
  group comparison fails the mismatch and real-lookup tests.
- Limit: a listener inside the group is accepted even if it is a different child than
  intended; `up`/`status` wiring is covered by the shared helper's tests, not end to end.
