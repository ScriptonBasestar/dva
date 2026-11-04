---
id: TASK-575
title: "Unsealed env warning follow-ups"
type: feature
priority: P3
exec-tier: standard
status: backlog
created: 2026-11-02
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

## Completion Criteria

- [ ] Each gap is fixed or recorded as accepted | verify: human — review the four items at release planning
