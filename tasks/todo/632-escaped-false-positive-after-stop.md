---
id: TASK-632
title: "status: do not report escaped after dva stop when another process takes the port"
type: bug
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-11-22
allowed-paths: [internal/lifecycle/, USAGE.md, CHANGELOG.md, tasks/]
---

## Summary

TASK-629 (3a3c64a6) reports a process entry as `error`/escaped when its pidfile pid is dead
but a health address answers. `haltProcess` keeps the pidfile after `dva stop`, so if the user
then runs the same server by hand (or anything else takes the port), `status` says the entry
escaped and advises stopping it by PID — a process dva never started. Found by the TASK-629
independent review.

## Completion Criteria

- [ ] After `dva stop`, an unrelated listener on the entry's health port does not produce an escaped report. | verify: human — internal/lifecycle에 추가한 stop 후 테스트 통과를 독립 리뷰가 확인
- [ ] A server that daemonized without a stop is still reported as escaped. | verify: `go test ./internal/lifecycle -run Escaped -count=1`

## Notes

- Options from review: record a halted marker in `haltProcess` and skip the check after it,
  or soften the wording to "something answers on X; it may be an escaped server".
