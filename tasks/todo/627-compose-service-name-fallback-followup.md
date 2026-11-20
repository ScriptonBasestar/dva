---
id: TASK-627
title: "compose status: warn on unknown Labels shape and review Names[0] fallback"
type: improvement
priority: P3
effort: XS
exec-tier: standard
status: todo
created: 2026-11-20
allowed-paths: [internal/lifecycle/]
---

## Summary

These are non-blocking follow-ups left after TASK-606 shipped podman-compose service names
(`internal/lifecycle/compose.go`, `composeLabels` / `serviceNameOf`):

1. If `Labels` decodes to neither an object nor a comma-separated string, the parser
   silently returns nil. Status output then shows an empty name and gives no hint why.
   Emit a stderr `[warn]` that names the unrecognised shape. Keep returning nil, which is
   still the safe choice.
2. When there is no compose service label, `serviceNameOf` falls back to `Names[0]`. That
   returns the container name (for example `proj_web_1`), not the service name. Decide
   whether that fallback stays and document the choice, or drop it in favour of empty.

## Completion Criteria

- [ ] An unknown `Labels` shape writes one stderr `[warn]` and does not fail status. | verify: `go test ./internal/lifecycle -run ServiceName -count=1`
- [ ] The `Names[0]` fallback decision is recorded in the `serviceNameOf` comment and covered by a test. | verify: `go test ./internal/lifecycle -run ServiceName -count=1`
