---
id: TASK-631
title: "health: verify the tracked process owns the probed port after start"
type: feature
priority: P3
effort: M
exec-tier: standard
status: todo
created: 2026-11-22
allowed-paths: [internal/lifecycle/, USAGE.md, CHANGELOG.md]
---

## Summary

The TASK-538 pre-start guard (3e2d3927) only refuses ports already served before `up`. If
another process takes the port after start, or the entry fails to bind while something else
answers, the health check still passes against the wrong owner. Where the OS allows it
(lsof/`/proc` on unix), check that the listening socket belongs to the tracked process group.
Otherwise state explicitly that ownership was not verified.

## Completion Criteria

- [ ] When the health endpoint's listener is not in the tracked process group, health reports a port-owner mismatch naming the address and owning pid. | verify: human — internal/lifecycle에 추가한 PortOwner 테스트 통과를 독립 리뷰가 확인
- [ ] Platforms without owner lookup report "owner unverified" rather than a pass or failure. | verify: human — internal/lifecycle에 추가한 PortOwner 테스트 통과를 독립 리뷰가 확인
