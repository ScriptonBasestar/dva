---
id: TASK-585
title: "Resolve encrypted env targets with loader path scope"
type: bug
priority: P2
exec-tier: strong
status: review
created: 2026-11-05
---

## Summary

Fix TASK-575's reproduced mismatch without changing the declared selector or expanding sops_source metadata. Share the loader's path resolver and pre-file scope with unseal/seal plaintext I/O, retain containment and anchored writes, suppress expanded path values in diagnostics and quote selector hints. Source-only operations keep their existing behavior. Main thread is the sole implementation writer; independent strong review evaluates both criteria before the frozen candidate's native full CI gate. This safety fix is its own task integration unit. The older v0.4.5 preparation baseline and its accepted macOS exception are historical, not fresh test evidence for this change.

## Completion Criteria

- [ ] Plaintext bridge reads writes and Git guards use the loader-resolved target while declared selectors and diagnostics remain stable | verify: human — independent source review and focused env bridge regressions
- [ ] Unresolved unsafe collision and symlink targets fail before secret operations without exposing expanded values or changing unrelated files | verify: human — focused fake SOPS Git and path safety regression evidence
