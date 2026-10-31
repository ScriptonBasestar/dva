---
id: TASK-551
title: "Align skill schema rules with what validation enforces"
type: bug
priority: P3
status: todo
created: 2026-11-01
---

## Summary

`schema-reference.md` stops calling Compose `name:` and runner tags mandatory, explains the scaffolded `version:` floor, and `dva-config` covers projects outside Git.

## Completion Criteria

- [ ] schema rules state what validation does not enforce | verify: `/usr/bin/grep -q "does not enforce" skills/dva-config/references/schema-reference.md`
