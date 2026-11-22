---
id: TASK-628
title: "validate: warn when a process entry health check probes another entry's port"
type: feature
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-11-22
allowed-paths: [internal/config/, internal/cli/, USAGE.md, CHANGELOG.md]
---

## Summary

Since 3e2d3927 (TASK-538), `ProcessPlugin.Up` refuses to start when an http/tcp health check
address already accepts connections. A process/native entry whose health check targets a
dependency (for example a DB on 5432) therefore fails `up` whenever that dependency runs. The
behaviour is documented in USAGE.md but only surfaces at `up` time. `dva validate` should warn
before then.

## Completion Criteria

- [ ] `dva validate` emits a warning naming the entry and address when a process/native entry's http/tcp health check resolves to a port that another stack entry declares or probes. | verify: human — internal/config 또는 internal/cli에 추가한 HealthPort 테스트 통과를 독립 리뷰가 확인
- [ ] Entries whose health check targets their own port produce no warning. | verify: human — internal/config 또는 internal/cli에 추가한 HealthPort 테스트 통과를 독립 리뷰가 확인
