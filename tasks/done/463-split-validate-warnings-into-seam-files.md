---
id: TASK-463
title: "Split validate_warnings.go and validate_warnings_test.go into seam files"
type: refactor
priority: P2
effort: M
exec-tier: standard
allowed-paths: [internal/config, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: validate_warnings.go 1575->650 + 6 seam files + 6 test files; numstat pure, test parity, build/vet/gofmt/test gates, High cleared. Integrated at 440d7698."
---

## Summary

`internal/config/validate_warnings.go`는 1,000 코드라인(1,575 물리)으로 경고
검증 전부를 한 파일에 담고 있고, `validate_warnings_test.go`(1,740 물리, 35
테스트)도 한도를 넘는다. 주제별 봉합으로 나눈다. 함수·타입 이동만 허용한다.

봉합 계획(실행 시 선언 경계 실측으로 조정):

- `interaction_walk.go`(신규) — `eachInteractionNode`, `inheritedExec`,
  `firstNonEmptyStr`. 이들은 validate.go·validate_warnings_refs.go도 호출하는
  공유 인프라라(실측 2026-10-02) warnings 파일에 두면 소유가 어긋난다.
- `section_order.go`(신규) — canonicalSectionOrder, canonicalOrderIndex, init,
  CanonicalSectionOrder, validateCanonicalOrder, validateCanonicalOrderFromBytes.
- `validate_warnings_health.go`(신규) — warnHealthCheckRedundancy,
  warnUnreachableHealthChecks.
- `validate_warnings_interaction.go`(신규) — warnDuplicateParentSubcommand,
  warnChildOverridesParentCritical, MaxSubcommandDepth,
  warnDeepSubcommandNesting, calculateSubcommandDepth, warnUnreachableCommands,
  hasUnsupportedBracedOperator.
- `validate_warnings_compose_split.go`(신규) — warnDuplicateStackOrder,
  entriesNamedByPlans, warnMultiStackComposeSplit, compose-split 판정 클러스터
  (composeEntriesAreIsolated, plansPartitionComposeServices 등), heavyInfra
  클러스터(heavyInfraServiceNames, heavyInfraTags, warnDefaultModeHeavyInfra,
  isHeavyInfra), warnMissingDefaultMode.
- validate_warnings.go 잔여 — ValidateWarnings 진입점, 레거시 경고
  (warnEquivalentReplaceHooks, warnLegacyModes, warnLegacyStackOrder,
  warnLegacyEnvironmentFields, warnNoPlansHint), 플랜 경고
  (warnInertProvisionSteps, warnIgnoredParallelSteps, warnIgnoredPlanFilters,
  warnDuplicatePlanDeclarations, plansHaveEqualDeclaration,
  compositionEntriesEqual, planEntriesEqual, warnMultiplePlansWithoutDefault),
  env 경고(warnUnresolvedEnvVars, warnSuspiciousEnvPatterns),
  warnLiteralKeyShadowsSubproject, warnMultiplePrimaryCompose.

테스트는 구현 주제를 따른다: TestCanonicalOrder_* → section_order_test.go,
TestWarnHealthCheck*/TestWarnUnreachableHealthChecks → health 테스트 파일,
TestWarnDuplicateParentSubcommand·TestWarnChildOverrides*·TestWarnDeepSubcommand*
·TestWarnUnreachableCommands·TestInteractionWarnings*·TestChildOverrideCompares*
→ interaction 테스트 파일, TestWarnMultiStackComposeSplit*(6) → compose-split
테스트 파일, TestWarnUnresolvedEnvVars·TestWarnSuspiciousEnvPatterns → env 테스트
파일, 나머지(replace-hook 후보, integration, order-stability 등)는
validate_warnings_test.go 잔존. 테스트 파일명은 구현 파일 + `_test` 규약.

## Completion Criteria

- [x] `internal/config/validate_warnings.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize internal/config/validate_warnings.go` (observed: 2026-10-02 — 650 물리/411 코드, exit 0)
- [x] `internal/config/validate_warnings_test.go`가 test kind 한도 안에 있다 | verify: `ce validate filesize internal/config/validate_warnings_test.go` (observed: 2026-10-02 — 362 물리)
- [x] 새로 생긴 봉합 파일 전부가 각 kind 한도 안에 있다 | verify: `ce validate filesize internal/config/interaction_walk.go internal/config/section_order.go internal/config/validate_warnings_health.go internal/config/validate_warnings_interaction.go internal/config/validate_warnings_compose_split.go` (observed: 2026-10-02 — High 0, Medium 2건은 300 경고선 고지)
- [x] 이동 후 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/config/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 1.2s)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/config/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다(시그니처·동작·주석 내용 변경 없음) | verify: human — split commit diff review (observed: 2026-10-02 — main-thread 리뷰: `git diff --numstat` 두 수정 파일 모두 0 insertions 순수 삭제, 테스트 수 파리티 426=426)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — gofmt 555 files 0 unformatted, ci-lint 0 issues)

## Evidence

분할 결과(물리): validate_warnings.go 1575→650, validate_warnings_test.go
1740→362, 신규 interaction_walk 63 / section_order 114 /
validate_warnings_health 103 / validate_warnings_interaction 186 /
validate_warnings_compose_split 488, 테스트 신규 section_order_test 80 /
health_test 211 / interaction_test 443 / compose_split_test 352 /
stack_order_test 272 / env_test 61.

계획 대비 판단 기록: (1) compose_split 테스트 봉합이 536 코드라인으로 한도를
넘어 내부 경계에서 2분할 — stack-order/heavy-infra 테스트 3개를
`validate_warnings_stack_order_test.go`(신규)로. (2) `sameStringSet`은 잔존 —
유일한 호출자 replaceHookIsComposeCandidate가 잔존 파일에 있다. (3) 테스트
헬퍼 `nestedInteractionConfig`는 유일 호출 테스트와 함께 interaction_test로.

공유 인프라 `eachInteractionNode`는 interaction_walk.go로 이동 — validate.go·
validate_warnings_refs.go의 기존 호출이 무변경으로 컴파일됨을 go build ./...
으로 확인.

게이트: `go build ./...` ok · `go test ./internal/config/ -count=1` ok · vet
silent · gofmt clean · `make lint` 0 issues · filesize 13파일 High 0.
