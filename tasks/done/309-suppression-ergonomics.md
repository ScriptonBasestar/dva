---
id: TASK-309
title: "suppression ergonomics: suggestion_ignore compaction and drift-warning ignore"
type: feature
priority: P3
effort: M
exec-tier: standard
created-at: 2026-09-05T09:00:00+09:00
source: "docs/dogfood/{dripter,flow-taskchain,flow-knowchain}.md"
status: done
needs-human: false
---

# Task 309: suggestion_ignore 축약 및 drift warning ignore 수단

## Summary

- suggestion_ignore가 70~120 항목으로 비대해져 Makefile 이중 관리 비용 발생
  (dripter 110+, flow-taskchain 120).
- 의도적인 test/prod override compose 파일이 상주 drift warning을 유발하는데
  이를 ignore할 수단이 없음 (flow-knowchain).

## Decision required

패턴/glob 기반 축약 vs 카테고리 단위 opt-out vs suggestion 소스 자체 개선 중 방향 결정.
ignore 수단은 침묵 남용으로 이어질 수 있어 설계 검토 필요.

## Completion Criteria

- [x] 설계 결정 기록 | verify: `test -f docs/56-suppression-ergonomics-design.md && /usr/bin/grep -q '^## 6. 결정' docs/56-suppression-ergonomics-design.md && /usr/bin/grep -q '결정 완료' docs/56-suppression-ergonomics-design.md && ! /usr/bin/grep -q '결정 대기' docs/56-suppression-ergonomics-design.md`
- [x] 구현 + 테스트 | verify: `go test ./internal/cli/ -run 'TestDriftIgnore|TestSuggestion|TestOmittedSuggestionCategory|TestAnAlreadyWrappedTarget|TestTargetFamiliesDVAReplaces|TestStaleSuggestionIgnore|TestStaleIgnoreWarnings|TestValidateSummary|TestSuggestIgnore' -count=1`
- [x] dripter/flow-taskchain의 ignore 목록이 유의미하게 축약된 예시 | verify: `/usr/bin/grep -rq '## Dogfood 축약 검증 (2026-09-13)' tasks/`

## Dogfood evidence (2026-09-05 실행)

- 의도적 미등록 overlay(test/prod/e2e 픽스처)로 상주 drift warning: flow-knowchain(3), sigdock-idp, familybook(5), db-orchestrator(2). `drift_ignore` 류 선언 필요.
- suggestion_ignore 규모: gizzahub 184, familybook 194, db-orchestrator ~85(glob), flow-agent-mesh 8패턴으로 정리 사례. 감지 결함은 TASK-316.

## Design record (2026-09-05)

- 문서: `docs/56-suppression-ergonomics-design.md`. 설계 선택지 문서 작성 완료, 결정 대기. 권고: suggestion은 소스 개선(이미 interaction이 감싸는 타겟 제외)+카테고리 opt-out, drift는 루트 자동탐지 규칙에만 적용되는 `drift_ignore` glob, 억제 건수를 validate 요약에 항상 표시.

## Decision (2026-09-12)

docs/56 §6에 5건 기록. 4건은 §3·§4의 권고 그대로, 1건(§5-1)은 권고를 확장했다 —
`dva validate --suggest-ignore`(선택지 A)를 **추가로** 채택한다. 권고가 A를 뺀 이유였던
"전부 무시를 한 번에 만드는 손잡이"는 억제 건수 요약을 끌 수 없다는 결정(§6-4)이 무력화한다.
침묵이 불가능한 상태에서 A가 없애는 것은 타겟 이름을 손으로 옮겨 적는 수고뿐이고,
그 수고는 남용을 막지 못한다 — 아래 dogfood의 100줄짜리 목록들이 A 없이 손으로 쌓였다.

## Dogfood 축약 검증 (2026-09-13)

새 바이너리로 `~/mydevbox/dripter-devbox`, `~/mydevbox/flow-taskchain-devbox`에서 실측.

| | dripter | flow-taskchain |
|---|---|---|
| 기존 `suggestion_ignore` 항목 수 | 103 | 98 |
| ignore 없이 — 구버전 suggestion 수 | 93 | 135 |
| ignore 없이 — C 규칙 적용 후 (신버전) | 74 | 107 |
| 채택 수단으로 옮긴 뒤 남는 선언 줄 수 | **3** | **3** |
| 그 상태에서 실제로 보이는 suggestion | 22 (전부 package.json) | 3 (전부 package.json) |
| 상주 drift 경고 | 2 → 0 | 1 → 0 |

옮긴 선언은 두 프로젝트 모두 동일한 형태다:

```yaml
suggestions:
  makefile: false
  package_json: true
drift_ignore:
  - "deploy/local/compose.app.yaml"     # flow-taskchain은 compose.e2e.yaml 1건
  - "deploy/local/compose.infra.yaml"
```

```
# dripter, 이행 후
✅ dva.yml is valid (52 suggestions, 2 drift files ignored by dva.yml)
# flow-taskchain, 이행 후
✅ dva.yml is valid (104 suggestions, 1 drift file ignored by dva.yml)
```

읽을 것:

- C(소스 개선)만으로는 93→74, 135→107 — 20% 남짓이다. 카드가 요구한 "유의미한 축약"을
  만든 것은 B(카테고리 opt-out)이고, C는 B를 켠 뒤 **남는 package.json 제안이 실제로
  볼 만한 것들만 남게** 만든 쪽으로 기여했다. dripter의 잔여 22건은 전부 `logs:*`,
  `test:*`처럼 감쌀 후보로 읽히는 이름이다.
- 표의 두 축은 서로 맞물린다: dripter는 74 = 억제 52 + 노출 22,
  flow-taskchain은 107 = 억제 104 + 노출 3.
- 억제 건수(52/104)는 목록을 3줄로 줄여도 사라지지 않는다. 원칙 1이 요구한 그대로,
  "무엇을 얼마나 가렸는지"는 선언을 줄인다고 같이 줄어들지 않는다.
- 기존 목록의 stale 항목이 플래그 없이 드러났다: dripter 7건(`env-edit-%` 등 pattern
  rule 잔재), flow-taskchain 3건. §6-5의 소급 적용이 dogfood에서 바로 값을 냈다.
- 실측 중 `suggestions`/`drift_ignore`를 파일 맨 앞에 넣자 section-order 경고가 정확히
  떴다 — `canonicalSectionOrder`에 두 키를 등록한 것이 동작한다는 부수 증거.

두 devbox의 `dva.yml`은 이번 실측에서 수정하지 않았다(백업·복원 후 `git status` 청결 확인).
실제 이행은 각 프로젝트 담당 카드에서 한다.

## review-309 최종 verdict (2026-09-13)

**`pass`** — 1라운드 `conditional`의 블로킹 2건과 비블로킹 4건을 `1e15275`가 닫았고,
2라운드 리뷰어가 독립 검증했다.

| 라운드 1 finding | 처리 |
|---|---|
| #1 `driftIgnoreName`이 심볼릭 링크 경로에서 basename으로 붕괴 | canonical base 재시도 + 회귀 테스트 |
| #2 dogfood 표의 dripter 노출 건수 11 | 22로 정정, 74 = 52 + 22 검산 줄 추가 |
| #3 `--suggest-ignore` 주석의 없는 구분 | `--json` 가드가 실제 이유임을 명시 |
| #4 suggestion stale이 빈 표본에서 전건 유죄 | `len(universe) > 0` 가드 + 테스트 |
| #5 문서·스키마의 "beside dva.yml" | 실제 스캔 집합으로 정정 |
| #7 죽은 `matchesSuggestionIgnore` | 삭제, 테스트는 `matchIgnorePattern` 직접 검증 |

**회귀 테스트는 반증으로 확인됐다.** 구현자와 리뷰어가 각각 옛 함수 본문을 되돌려
`TestDriftIgnoreMatchesThroughASymlinkedConfigDirectory`의 세 단언이 전부 FAIL 하는
것을 재현했다(리뷰어는 `git archive` 사본에서). `#4`의 가드도 같은 방식으로,
되돌리면 `TestStaleSuggestionIgnoreStandsDownWhenThereIsNothingToJudgeAgainst`가
두 패턴을 모두 유죄로 만들며 FAIL 한다. 통과가 아니라 실패가 테스트의 값이다.

**요약 줄의 숫자는 억제 건수다** — 리뷰어가 `summarySuffix`
(`internal/cli/validate_suppression.go`)를 읽어 확인했다. 그래서 74 = 억제 52 +
노출 22가 성립한다. 이 한 줄을 노출 건수로 읽으면 표의 검산 자체가 무너진다.

2라운드가 추가로 낸 low 2건도 이 카드와 함께 닫았다: docs/56 §1 표가 삭제된
`matchesSuggestionIgnore`를 계속 가리키고 있었고(`make doc-check`가 잡지 못하는
종류의 부패다), `#4` 가드의 주석이 조건을 실제보다 좁게 서술하고 있었다 —
`##` 문서화 타겟이 없는 Makefile도 빈 universe를 만든다.

**남는 한계(이번 변경 이전부터 있던 것, 함수 주석에 이미 명시)**: 스캔 디렉터리가
config 트리를 완전히 벗어난 심볼릭 링크로 도달되면 `driftIgnoreName`은 여전히
basename으로 후퇴하고, 그러면 basename 패턴이 다른 디렉터리의 동명 파일까지 덮을 수
있다. `/tmp → /private/tmp`처럼 config 디렉터리 자신이 링크 뒤에 있는 경우는
`1e15275`가 덮는다.
