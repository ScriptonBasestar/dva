---
id: TASK-346
title: "Warn on a plan profile the compose file does not define"
type: feature
priority: P2
effort: M
exec-tier: strong
status: todo
created: 2026-09-07
needs-human: true
---

## Summary

TASK-315 gave plan entries a `profiles:` field. There is no validation behind it, so a typo
validates clean: `profiles: [rustt]` passes `dva validate`, and `dva up` starts only the
ungated base services. The plan appears to work and quietly does less than it says — no error,
no warning, and a zero exit code.

`services:` already has a guard for exactly this failure mode
(`warnPlanServicesNotDeclared`, `internal/config/validate_warnings_refs.go`), whose own
comment names the reason: a plan selecting a name outside the declared surface "starts nothing
at runtime". A profile typo is the same class of silent under-delivery, and it is now reachable
because this field exists.

## Why this is not "add the same guard"

The `services:` guard works because it has a declared surface *inside dva.yml* to check
against — `stack.{entry}.runners.compose.services`. It never reads the compose file; its
failure message even offers "the services map is stale" as one of two readings.

Profiles have no such surface, and that is not an oversight — TASK-315 deliberately did not
add a `Profiles` field to `ComposePluginConfig` (see the rationale comment there). So the only
source of truth for which profiles exist is the compose file itself, and reading it would be a
new capability for this validator rather than a copy of an existing one.

That is the fork this card has to resolve, and it is a design decision rather than an
implementation detail:

1. **Parse the compose file** to collect `services.*.profiles`. Accurate, and it works with no
   new dva.yml surface — but it makes validation depend on a file dva does not currently read
   at validate time, and it must degrade quietly when that file is absent, unreadable, or uses
   `extends`/interpolation the parser cannot resolve.
2. **Declare profiles in dva.yml** and check against that, mirroring `services:`. Consistent
   with the existing guard and cheap to implement — but it adds the exact field TASK-315
   argued against, and a declared list can go stale against the compose file, which is the
   second reading the `services:` message already has to hedge for.

Option 1 is the recommendation: it validates against the thing that actually decides the
outcome, and it does not reopen a decision TASK-315 made on other grounds. But the degradation
behaviour is the hard part and should be settled before implementation, not during it.

Whichever is chosen, the warning must stay a warning. An unresolvable compose file is not a
broken config.

## Completion Criteria

- [ ] The source-of-truth decision is recorded on this card with its rationale | verify: human — 카드 하단에 "Design record" 절이 추가되어 option 1(compose 파일 파싱)과 option 2(dva.yml 선언) 중 채택안과 근거, 그리고 compose 파일이 없거나 파싱 불가할 때의 degradation 동작이 기록되었는지 확인
- [ ] A plan profile absent from the resolved profile set produces a warning naming the plan, the entry index, and the available profiles | verify: `/usr/bin/grep -rq 'func TestWarnPlanProfilesNotDefined(' internal/config`
- [ ] An unresolvable or absent compose file degrades to silence rather than a warning or an error | verify: `/usr/bin/grep -rq 'func TestPlanProfileWarningStaysQuietWithoutAResolvableComposeFile(' internal/config`
- [ ] Gates stay green | verify: `make test`
