---
id: TASK-629
title: "process: detect native dev servers that daemonize and escape tracking"
type: feature
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-11-22
allowed-paths: [internal/lifecycle/, USAGE.md, CHANGELOG.md]
---

## Summary

In the TASK-538 pilot (airadartoday-web), Astro's dev server backgrounded itself by default.
The tracked pid exited, and the real server kept running untracked until it was killed by PID.
`ASTRO_DEV_BACKGROUND=0` worked around it. dva should notice when the tracked process exits
while its health check endpoint still answers, and report it instead of claiming the entry
stopped or failed.

## Completion Criteria

- [ ] When the tracked process exits but the entry's health endpoint keeps answering, `up`/`status` report an "escaped from tracking" error naming the address. | verify: human — internal/lifecycle에 추가한 Escaped 테스트 통과를 독립 리뷰가 확인
- [ ] USAGE.md documents the case and the foreground setting for known dev servers (Astro `ASTRO_DEV_BACKGROUND=0`). | verify: human — USAGE.md의 해당 설명을 독립 리뷰가 확인
