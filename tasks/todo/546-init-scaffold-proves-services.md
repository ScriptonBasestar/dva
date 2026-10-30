---
id: TASK-546
title: "Scaffold only interactions the project proves"
type: bug
priority: P2
status: todo
created: 2026-10-31
---

## Summary

<!-- One paragraph: what changes and why. -->

## Completion Criteria

- [ ] no interactions without an app service | verify: `go test -count=1 -run TestGenerateConfigOmitsInteractionsWithoutAppService ./internal/cli`
- [ ] undeclared npm scripts are dropped | verify: `go test -count=1 -run TestGenerateConfigKeepsOnlyDeclaredNpmScripts ./internal/cli`
