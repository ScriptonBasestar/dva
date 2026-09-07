---
id: TASK-347
title: "Teach the agent-mesh library plan-level profile selection"
type: docs
priority: P3
effort: S
exec-tier: standard
status: todo
created: 2026-09-07
---

## Summary

The agent-mesh flow library still teaches only the pre-TASK-315 route for activating compose
profiles, so generated configs will keep reaching for the shape that feature replaced.

Three files carry the drift:

- `agent-mesh-flows/shared/library/shared-guardrails.md` — the plan-scope guardrail says
  "Service selection belongs in `plans.*.entries[].services`". True and now incomplete: profile
  selection belongs in `plans.*.entries[].profiles`.
- `agent-mesh-flows/shared/library/shared-checklist.md` — "Compose service subsets use
  `plans.entries[].services`" has no profiles counterpart, so a review using this checklist
  cannot notice the omission.
- `agent-mesh-flows/shared/library/dva-schema.md` — advises `profiles: ["rust"]` on optional
  app services in the compose-file section, but the only *selection* mechanism it documents is
  `compose_profiles` under `modes:`. That is the legacy route: the guardrails file above
  already says to "preserve legacy `modes:` only until an explicit migration". So the library
  tells a generator to gate services and then offers only the deprecated way to reach them.

Net effect: a flow that follows the library produces either a mode-based config it was told
not to prefer, or profile-gated services no plan can activate.

Note on scope: these are hand-maintained source files under `WF_LIBRARY`, not `make generate`
output. `library_reference.txt` is the generated artifact; editing these `.md` files is the
correct fix, and `make generate` should follow so the reference picks the change up.

## Completion Criteria

- [ ] The plan-scope guardrail names profile selection alongside service selection | verify: `/usr/bin/grep -q 'entries\[\].profiles' agent-mesh-flows/shared/library/shared-guardrails.md`
- [ ] The checklist has a profiles line under Plans & runner strategy | verify: `/usr/bin/grep -q 'profiles' agent-mesh-flows/shared/library/shared-checklist.md`
- [ ] The schema reference documents plan-entry profiles as the current route, with `compose_profiles` marked legacy | verify: `/usr/bin/grep -q 'entries\[\].profiles' agent-mesh-flows/shared/library/dva-schema.md`
- [ ] Generated artefacts are regenerated and gates stay green | verify: `make generate && make doc-check`
