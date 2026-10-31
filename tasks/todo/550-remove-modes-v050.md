---
id: TASK-550
title: "Remove modes and default_mode in v0.5.0"
type: refactor
priority: P2
status: todo
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

- [ ] 스키마가 두 키를 받지 않는다 | verify: `! /usr/bin/grep -q -E '"(modes|default_mode)"' internal/config/schema.json`
- [ ] 비권장 기능을 권하던 경고가 사라졌다 | verify: `! /usr/bin/grep -rq 'warnMissingDefaultMode' internal`
- [ ] CHANGELOG가 업그레이드 절차를 말한다 | verify: `/usr/bin/grep -q -E 'modes.*dva config migrate|dva config migrate.*modes' CHANGELOG.md`
- [ ] modes가 든 옛 설정을 거부하는 회귀 테스트가 통과한다 | verify: `go test -v ./internal/config/ | /usr/bin/grep -q -- '--- PASS: TestSchemaRejectsModes'`
- [ ] modes가 든 옛 설정을 migrate가 변환하는 회귀 테스트가 통과한다 | verify: `go test -v ./internal/cli/ | /usr/bin/grep -q -- '--- PASS: TestConfigMigrateConvertsModes'`
- [ ] 전체 테스트가 통과한다 | verify: `make test` (regression-guard)
