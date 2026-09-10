---
id: TASK-346
title: "Warn on a plan profile the compose file does not define"
type: feature
priority: P2
effort: M
exec-tier: strong
status: done
created: 2026-09-07
needs-human: false
allowed-paths:
  - internal/config/validate_warnings.go
  - internal/config/validate_warnings_refs.go
  - internal/config/validate_warnings_refs_test.go
  - tasks/todo/346-warn-on-a-plan-profile-the-compose-file-does-not-define.md
  - tasks/plan/006-devbox-dogfood-followup.md
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

- [x] The source-of-truth decision is recorded on this card with its rationale | verify: human — 카드 하단에 "Design record" 절이 추가되어 option 1(compose 파일 파싱)과 option 2(dva.yml 선언) 중 채택안과 근거, 그리고 compose 파일이 없거나 파싱 불가할 때의 degradation 동작이 기록되었는지 확인
- [x] A plan profile absent from the resolved profile set produces a warning naming the plan, the entry index, and the available profiles | verify: `/usr/bin/grep -rq 'func TestWarnPlanProfilesNotDefined(' internal/config`
- [x] An unresolvable or absent compose file degrades to silence rather than a warning or an error | verify: `/usr/bin/grep -rq 'func TestPlanProfileWarningStaysQuietWithoutAResolvableComposeFile(' internal/config`
- [x] Gates stay green | verify: `make test`

## Design record

2026-09-10: Option 1을 채택한다. profile 정의의 정본은 dva.yml에 중복 선언하지 않고, 각
stack compose runner가 지정한 파일의 `services.*.profiles`로 둔다. 실제 실행 결과를
결정하는 compose 파일을 직접 확인하므로 별도 `ComposePluginConfig.Profiles` 목록이
compose와 어긋나는 두 번째 drift 경로를 만들지 않으며, TASK-315에서 정한 "profile은 plan의
실행 의도"라는 경계도 유지한다.

검사는 plan이 선언된 실제 owner config와 source entry의 기준 디렉터리에서 명시된 compose
파일 전부를 읽고, profile 합집합을 정렬해 비교한다. 유효한 compose service 집합에 profile이
하나도 없으면 `(available: none)`으로 경고한다. 반면 compose runner/file 선언이 없거나,
파일이 없거나 읽히지 않거나 YAML을 해석할 수 없거나, 경로·profile interpolation 또는
`include`/`extends`/compose 특수 merge tag 때문에 완전한 집합을 확정할 수 없으면 warning과
error를 모두 내지 않는다. 일부 파일에서 읽은 불완전한 집합으로 경고하는 것보다 조용히
runtime compose 진단에 맡기는 쪽이 이 비치명적 validator의 계약에 맞는다.

## Resolution (2026-09-10)

- Validation reads the configured compose files from the effective owner/source
  directory and warns deterministically for plan profiles absent from the union
  of `services.*.profiles`.
- It remains silent when the file or profile shape cannot be resolved, including
  interpolation, `include`, `extends`, Compose merge tags, and non-string YAML
  profile values.
- The two card bindings, focused and package tests, and an independent strong
  review passed.
