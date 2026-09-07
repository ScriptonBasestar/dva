---
id: TASK-337
title: "Bind GatedCommands to the command tree so a gated command cannot ship ruleless"
type: feature
priority: P1
effort: M
exec-tier: strong
status: todo
created: 2026-09-07
source: "tasks/done/286 done-review (PLAN-007 Tier A batch 2)"
depends-on: [TASK-286]
---

## Summary

TASK-286's criterion 1 states its purpose as "so a new gated command cannot ship without a
rule". Nothing enforces that. `make check-generate` only proves `docs/agent-deny-rules.md`
matches `GatedCommands` — the generated doc agreeing with the list it was generated from. No
check compares `GatedCommands` against the cobra command tree or against the `env_bridge` gate
set, so adding a gated command and forgetting its deny rule stays green end to end. The only
consumers of the package are `tools/agentdenygen` and `internal/cli/agentdeny.go`.

TASK-286 deferred this because TASK-282 had not landed and there was no gate set to check
against. TASK-282 has since landed, so the stated blocker is gone.

The check has to fail when a command that the `env_bridge` gate covers has no entry in
`GatedCommands` — the direction that matters. The reverse (a stale rule for a removed command)
is worth reporting too, but it fails safe and need not block.

## Completion Criteria

- [ ] a check compares `GatedCommands` against the gated surface and fails when a gated command has no rule | verify: `/usr/bin/grep -rq "func TestGatedCommandsCoverEveryGatedSurface" internal/cli`
- [ ] the check runs inside an existing `make` gate that CI already invokes, not as a manual step | verify: human — the card names the gate and shows it failing on the mutation below
- [ ] adding a gated command without a rule fails the gate — proven by mutation, not by assertion alone | verify: human — the card records the mutation and the failure message it produced
