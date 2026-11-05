---
id: TASK-550
title: "Remove modes and default_mode in v0.5.0"
type: refactor
priority: P2
exec-tier: strong
status: done
quality-review: pass
quality-reviewed-at: 2026-11-05T03:37:39Z
quality-review-evidence: tasks/done/evidence/TASK-550/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-550/done-review-8a0250e17f08467ec8f56766604c61bc97a64ef6a6924df9775dddba0d09de78.json
created: 2026-11-01
---

## Summary

[결정 501](../../decisions/501-modes-legacy-or-removed.md)은 `modes`/`default_mode`를 v0.5.0에서 제거하기로 했다. 스키마는 두 키를 거부하고, 사용자는 `dva config migrate`로 이관한다. 이 카드는 v0.5.0 범위에서만 실행한다.

## Scope

- `internal/config/schema.json`의 최상위 `modes`, `default_mode`를 제거한다.
- `warnMissingDefaultMode`를 지우고, `warnLegacyModes`는 제거 후 동작에 맞춘다.
- `Modes`/`DefaultMode`를 읽는 `internal/config`, `internal/lifecycle`, `internal/cli` 경로와 `--mode` 플래그를 정리한다.
- `dva config migrate`는 제거 후에도 옛 파일을 읽어 변환해야 한다. 스키마 거부가 변환 입력을 막으면 안 된다.
- USAGE.md, ARCHITECTURE.md, CLAUDE.md, docs/30, docs/42의 현재형 서술과 CHANGELOG 업그레이드 주의사항을 맞춘다.

## Completion Criteria

- [x] 스키마가 두 키를 받지 않는다 | verify: `! /usr/bin/grep -q -E '"(modes|default_mode)"' internal/config/schema.json`
- [x] 비권장 기능을 권하던 경고가 사라졌다 | verify: `! /usr/bin/grep -rq 'warnMissingDefaultMode' internal`
- [x] CHANGELOG가 업그레이드 절차를 말한다 | verify: `/usr/bin/grep -q -E 'modes.*dva config migrate|dva config migrate.*modes' CHANGELOG.md`
- [x] modes가 든 옛 설정을 거부하는 회귀 테스트가 통과한다 | verify: `go test -v -run '^TestSchemaRejectsModes$' ./internal/config`
- [x] modes가 든 옛 설정을 migrate가 변환하는 회귀 테스트가 통과한다 | verify: `go test -v -run '^TestConfigMigrateConvertsModes$' ./internal/cli`
- [x] 전체 테스트가 통과한다 | verify: `make test` (regression-guard)

## Execution

The accepted TASK-501 removal requires the minor version 0.5.0. The unpublished
0.4.5 candidate remains fixed at the already integrated release-preparation baseline;
this execution neither changes its notes nor creates public tags or Releases.
Main owns cross-cutting decisions and final integration. Native workers implemented
bounded config and lifecycle/CLI portions sequentially; independent strong review
evaluates the completed source and every criterion.

The two named regression bindings run Go directly rather than masking package failures
through a grep pipeline. The full make test binding remains required. The shared CE
binding-timeout setting is 10m because this full suite exceeds its default 30s;
this deliberate setting applies board-wide. Final native full CI, hosted platform
checks and local installation are separate delivery checks after metadata freeze.

## Review Attempts

- 2026-11-05T03:37:39Z — first formal review: independent strong `/root/doctor_flake_diagnosis` in review-purpose execution `review-task-550`; executor strong `/root`, sequential standard config and lifecycle/CLI workers. All six criteria pass; guarded full race/coverage suite, named regressions, source/doc consistency and immutable logs inspected. Evidence: `tasks/done/evidence/TASK-550/independent-review.json`. No formal repair cycle. Next transition: done. Exact final-head delivery checks remain required.
