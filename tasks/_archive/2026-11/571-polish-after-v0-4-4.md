---
id: TASK-571
title: "Clear the follow-ups left by the v0.4.4 publication"
type: docs
priority: P3
exec-tier: standard
status: done
created: 2026-11-02
quality-review: pass
quality-reviewed-at: 2026-11-02
quality-review-evidence: tasks/done/evidence/TASK-571/independent-review.json
---

# TASK-571: Clear the follow-ups left by the v0.4.4 publication

## Summary

Marks child CI as shipped, states the provision owner scope in USAGE, and stops the flow-taskchain
dogfood harness from purging a target whose AGENTS.md forbids it.

## Completion Criteria

- [x] Harness no longer purges flow-taskchain | verify: `/usr/bin/grep -q '|$DVA down local-dev$' tools/dogfoodrun/dogfood_steps.sh`
- [x] USAGE states the provision owner scope | verify: `/usr/bin/grep -q 'child 소유 profile은 root 실패로' USAGE.md`
