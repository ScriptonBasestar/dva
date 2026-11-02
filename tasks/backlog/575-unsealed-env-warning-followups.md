---
id: TASK-575
title: "Unsealed env warning follow-ups"
type: improvement
priority: P4
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

## Completion Criteria

- [ ] Each gap is fixed or recorded as accepted | verify: human — review the four items at release planning
