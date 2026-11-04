---
id: TASK-585
title: "Resolve encrypted env targets with loader path scope"
type: bug
priority: P2
exec-tier: strong
status: done
created: 2026-11-05
quality-review: pass
quality-reviewed-at: 2026-11-04T16:33:00Z
quality-review-evidence: tasks/done/evidence/TASK-585/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-585/done-review-d681214ab1df1226cfb67d38f6fd3a32438d83bc6ddfb82f5c0b14b14c57a263.json
---

## Summary

Fix TASK-575's reproduced mismatch without changing the declared selector or expanding sops_source metadata. Share the loader's path resolver and pre-file scope with unseal/seal plaintext I/O, retain containment and anchored writes, suppress expanded path values in diagnostics and quote selector hints. Source-only operations keep their existing behavior. Main thread is the sole implementation writer; independent strong review evaluates both criteria before the frozen candidate's native full CI gate. This safety fix is its own task integration unit. The older v0.4.5 preparation baseline and its accepted macOS exception are historical, not fresh test evidence for this change.

## Completion Criteria

- [x] Plaintext bridge reads writes and Git guards use the loader-resolved target while declared selectors and diagnostics remain stable | verify: human — independent source review and focused env bridge regressions
- [x] Unresolved unsafe collision and symlink targets fail before secret operations without exposing expanded values or changing unrelated files | verify: human — focused fake SOPS Git and path safety regression evidence

## Review Attempts

- 2026-11-04T16:33:00Z — first formal review: independent strong `/root/doctor_flake_diagnosis`, executor strong `/root`. Both criteria pass at `d767fe59db274ccb45c4ff54f178e23b96f831f6`; focused guarded race and seal symlink cases passed. Extra seal-specific coverage was added before verdict; no post-verdict repair. Evidence: `tasks/done/evidence/TASK-585/independent-review.json`. Next transition: done. Native full CI and installation remain required at the final delivery boundary.
