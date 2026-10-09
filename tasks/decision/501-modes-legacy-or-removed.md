---
id: TASK-501
title: "Decide whether modes is legacy-supported or removed"
type: decision
priority: P2
effort: S
needs-human: true
execution-mode: decision
human-grade: human
status: Proposed
created: 2026-10-10
---

## Summary

`modes`/`default_mode`의 상태를 문서와 코드가 서로 다르게 말한다. 설계 문서는
"제거", ARCHITECTURE와 코드는 "legacy지만 허용", 일부 검증 경고는 오히려 사용을
권한다. 상태 하나를 골라야 문서와 경고 문구를 맞출 수 있다.

## Context

2026-10-10 문서 정합성 검토에서 확인했다.

- [docs/42](../../docs/42-migration-and-compatibility.md) 표는 `modes`를 "제거 후 분해",
  [docs/30 예시](../../docs/30-config-merge-examples.md)의 하위호환 메모는 "`modes` 제거"라고 쓴다.
- [ARCHITECTURE.md](../../ARCHITECTURE.md)의 도메인 경계 표는 `modes`/`default_mode`를
  "legacy — validate가 plans 이관을 권고"로 둔다.
- `internal/config/schema.json`은 최상위 `modes`와 `default_mode`를 아직 받는다.
  `applications`는 스키마가 거부한다.
- `warnLegacyModes`(`internal/config/validate_warnings.go`)는 "deprecated in favor of the
  new plans model"이라고 경고한다.
- 같은 검증의 `warnMissingDefaultMode`(`internal/config/validate_warnings_compose_split.go`)는
  `modes`가 있고 `default_mode`가 없으면 "set default_mode to a minimal infrastructure mode"라고
  권한다. USAGE.md `### default_mode`도 같은 권고를 싣는다. deprecated 기능을 더 쓰라고
  안내하는 셈이다.
- [ROADMAP.md](../../ROADMAP.md)는 "Legacy 설정 제거는 마이그레이션과 제거 버전이 결정된 뒤
  별도 변경"이라고 적는다. 제거 버전은 아직 정해지지 않았다.

이번 검토는 USAGE.md 표·플래그 행과 CLAUDE.md에 "legacy" 표기만 더했다. 경고 문구,
docs/30·42의 "제거" 서술, 제거 시점은 이 결정을 기다린다.

## Decision

- Status: Proposed

선택지:

1. **Legacy 유지**: 스키마는 계속 받는다. docs/30·42의 "제거"를 "deprecated, 아직
   허용"으로 고친다. `warnMissingDefaultMode`의 권고를 plan 이관 안내로 바꾼다.
2. **제거 버전 확정**: 예를 들어 v0.5.0에서 스키마가 거부하도록 정한다. 그때까지
   `dva config migrate` 변환 경로와 경고를 유지한다. 1의 문서 정리도 함께 한다.
3. **1급 기능으로 복귀**: deprecated 경고를 걷어내고 ARCHITECTURE의 legacy 표기를 지운다.
   SOUL의 "명시적 계획 우선"과 plans 단일 실행 모델에 다시 축을 하나 더한다.

## Rationale

검토자 권고는 2다. 스키마가 아직 받는 동안 "제거"라고 쓰면 사용자가 동작하는 설정을
버려야 하는지 판단할 수 없다. 제거 버전을 정해야 ROADMAP의 "제거 버전이 결정된 뒤"
조건이 풀린다. 3은 SOUL 1·2번 신념(선언과 계획 분리, 명시적 계획 우선)과 어긋난다.
제품 범위와 호환성 약속을 바꾸므로 사람이 정한다.

## Consequences

- 1·2는 docs/30·42, USAGE.md `default_mode` 절, `warnMissingDefaultMode` 문구,
  `skills/dva-config/SKILL.md`의 legacy 서술을 함께 고치는 실행 카드를 낳는다.
- 2는 CHANGELOG 업그레이드 주의사항과 마이그레이션 검증을 요구한다.
- 3은 plans/modes 두 실행 선택 모델을 계속 문서화해야 한다.

## Resolution Criteria

- [ ] 사람이 legacy 유지, 제거 버전 확정, 1급 복귀 중 하나를 고르고 적용 범위를 적는다 | verify: human — choice recorded on this card before any schema, warning, or doc wording change
