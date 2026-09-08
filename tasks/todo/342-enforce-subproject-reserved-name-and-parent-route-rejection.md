---
id: TASK-342
title: "Enforce TASK-263's subproject reserved-name and parent-route rejection rules"
type: feature
priority: P1
effort: M
exec-tier: strong
status: todo
created: 2026-09-07
source: "tasks/done/263 done-review (PLAN-007 Tier A batch 1)"
---

## Summary

TASK-263 documented three subproject rules that the code never enforces. The done-review
confirmed each one is stated in the design but unbound in `internal/config`:

1. A subproject name that collides with a reserved built-in surface is accepted today.
   `dva config validate` must reject it and name the reserved word in the error.
2. A key the child's own validator rejects stays reachable through the parent routes
   (`--project <name>`, `<name>:<key>`, `<name>/<key>`). The parent must not offer an
   address the child would refuse.
3. Both errors must state their basis — which rule fired and which declaration triggered
   it — so a user can fix the config without reading the source.

Filed from the TASK-263 done-review; the card was left `conditional` for exactly this gap.

## Completion Criteria

- [ ] `dva config validate` rejects a subproject whose name collides with a reserved built-in, and the message names the reserved word | verify: `/usr/bin/grep -rq "func TestSubprojectReservedNameRejected" internal/config`
- [ ] a key the child validator rejects is unreachable through all three parent address forms (`--project`, `:`, `/`) | verify: `/usr/bin/grep -rq "func TestSubprojectParentRouteRejectsChildInvalidKey" internal`
- [ ] both rejection paths report the rule and the offending declaration | verify: `/usr/bin/grep -rq "func TestSubprojectRejectionNamesRuleAndDeclaration" internal`
