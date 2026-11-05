---
id: TASK-575
title: "Unsealed env warning follow-ups"
type: feature
priority: P2
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-05T01:12:07Z
quality-review-evidence: tasks/done/evidence/TASK-575/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-575/done-review-2a94889d94e4559ed72cd78ea1b3f6b424ae2f520f34f6bb0a22c65f93bc812a.json
---

# TASK-575: Unsealed env warning follow-ups

## Summary

Gaps left after TASK-573, none seen in real use yet. Review at the next release.

- `dva doctor` checks only the root config; child encrypted targets go unreported.
- `dva restart` does not warn.
- An `env_file` path with interpolation gets an unseal hint naming the raw path.
- An unknown plan name warns before the refusal.

## Investigation

2026-11-04: disposable overlay fixtures reproduced the four cases at source
`cdb85caba74dcbf5272d805dd86026eaff49a897`. In particular, an interpolated declaration
loaded the expanded file but fake-SOPS unseal created a literal target. Durable facts,
raw evidence and limits are in [the reproduction record](../../docs/env-warning-reproduction.md).
The v0.4.5 candidate documents these limits; this is not an accepted exception or completion.
Priority is raised to P2 because the interpolation case is an observed loader/write-target
mismatch, rather than only warning presentation. No real-use incident is claimed.

TASK-585 implements the plaintext target resolution and selector-hint correction in source
after the v0.4.5 preparation baseline. This execution addresses the other three items:
owner-correct warnings on all restart routes, warnings after target/flag validation, and
direct-child encrypted env diagnostics with explicit unavailable rows. Child inspection
preserves modules/overrides, version checks and env ownership, without recursively resolving
imports or running child backend/user checks. Regression coverage lives in
`internal/cli/doctor_envsops_test.go` and `internal/cli/doctor_subproject_env_test.go`.

The existing TASK-kind card was incorrectly placed in the BACKLOG-kind zone. The shared
move/promote commands rejected that mismatch; placement was normalized through todo while
preserving its identity and historical text, then moved to doing through the shared tool.

## Completion Criteria

- [x] Each gap is fixed or recorded as accepted | verify: human — independent review of TASK-585 and this change against the four original gaps

## Review Attempts

- 2026-11-05T01:12:07Z — first formal review: independent strong `/root/doctor_flake_diagnosis`, executor strong `/root`. All four gaps pass at `5a12a23fd918e96d49b56497da67c97d1fb10e7f`; named guarded CLI/config tests and review-card validation passed. Evidence: `tasks/done/evidence/TASK-575/independent-review.json`. No repair cycle. Next transition: done. Final full CI and installation remain delivery gates.
