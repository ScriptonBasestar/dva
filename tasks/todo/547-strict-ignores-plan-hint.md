---
id: TASK-547
title: "Keep the no-plans hint from failing strict validation"
type: bug
priority: P2
status: todo
created: 2026-10-31
---

## Summary

<!-- One paragraph: what changes and why. -->

## Completion Criteria

- [ ] informational hints do not fail strict | verify: `go test -count=1 -run TestStrictIgnoresInformationalHints ./internal/cli`
