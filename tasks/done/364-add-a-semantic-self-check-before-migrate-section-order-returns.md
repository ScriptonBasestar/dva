---
id: TASK-364
title: "fix: add a semantic self-check before MigrateSectionOrder returns"
type: bug
priority: P1
effort: M
exec-tier: strong
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview) SHOULD-FIX 6"
status: done
depends-on: []
needs-human: false
verification-evidence: "2026-09-10: semantic guard regressions, internal/config tests, five-count benchmark, dva ci commit, doc-check, diff check, and independent review passed."
allowed-paths:
  - internal/config/migrate_section_order.go
  - internal/config/migrate_section_order_test.go
  - tasks/done/364-add-a-semantic-self-check-before-migrate-section-order-returns.md
  - tasks/plan/008-migrate-section-order-defects.md
---

# Task 364: 재정렬이 의미를 바꾸지 않았는지 반환 직전에 스스로 확인한다

## Summary

`MigrateSectionOrder`는 두 라운드의 리뷰에서 **아홉 개의 결함**을 냈다. 세어 보면 무엇을
고쳐야 하는지가 분명해진다.

| 변종 | 무엇이 배너의 시작을 결정하는가 |
|------|-------------------------------|
| M1 | 컬럼 0 `#` 규칙 — 들여쓴 `#`를 배너로 오인 |
| M7 | 컬럼 0 `#`가 인용 스칼라 안에 있을 때 오인 + walk 상한 없음 |
| M8 | CRLF에서 `HeadComment` 줄 수를 넘겨 세어 스칼라를 찢음 |
| M9 | 파서와 줄 분할기가 개행 규약에 합의하지 않음 |

넷 다 **"파서와 이 함수가 같은 것을 보고 있는가"** 라는 하나의 질문이고, 넷 다 개별
형태를 하나씩 막아서 고쳤다. 다섯 번째 형태가 나오지 않는다는 보장은 어디에도 없다.
리뷰어의 표현 그대로다.

> 이 함수는 두 라운드에 걸쳐 일곱 개의 결함을 냈고 그 중 다섯이 조용한 손실이었습니다.
> 개별 형태를 하나씩 막는 방식은 여덟 번째 형태를 막지 못합니다.

M8과 M9는 그 뒤에 추가로 나온 여덟 번째·아홉 번째다. 예측이 맞았다.

## 무엇을

`MigrateSectionOrder`가 반환하기 직전에 `src`와 `out`을 각각 `yaml.Unmarshal`한 뒤
비교한다. 다르면 `src`를 그대로 돌려주고 `report.Blocked`에 기록한다.

현재 유일한 검증은 CLI 레이어의 `VerifyMigrated`이고 그건 **"파싱되는가"만** 본다.
파싱되면서 값이 달라지는 경우(TASK-363의 `|+`)는 그대로 통과한다.

**효과는 측정된 값이다.** 리뷰어의 프로브 하네스가 정확히 이 방식
(`fmt.Sprintf("%#v", …)` 비교)으로 네 결함을 전부 잡아냈다. 추측이 아니라고 명시했다.
이후 독립적으로 발견된 M8에 대해서도 "이번 것도 자동으로 걸렸을 겁니다"라고 했다.

한 자리에서 막히는 것들: MUST-FIX 1·2, SHOULD-FIX 3(TASK-358, anchor/alias),
SHOULD-FIX 4(TASK-363, `|+`).

## 결정

- **비교는 YAML node tree를 사용한다.** `map[string]any` + `reflect.DeepEqual`은 non-string
  key와 YAML scalar tag를 Go 값으로 변환하며 정밀도를 잃는다. 대신 `Kind`·resolved
  `Tag`·`Value`·자식 관계를 비교한다. mapping entry 순서는 모든 깊이에서 무시하고 sequence
  순서는 보존한다. comment, scalar style, line/column은 presentation이므로 제외한다.
- **원본 parse는 기존 것을 재사용한다.** 함수 시작에서 이미 읽은 source node와 반환 후보를
  한 번 추가 parse한 node를 비교한다. 따라서 실제 추가 parse는 파일당 한 번이다.
- **불일치와 candidate parse 실패는 error가 아니라 `Blocked`다.** 원본 bytes를 반환하고
  이 함수가 잠정 기록한 `Changes`를 지운다. 사용자는 `Left for you`에서 수동 재정렬 안내를
  받고, 파이프라인의 다른 migration 결과를 abort하지 않는다.
- **개별 가드는 유지한다.** self-check는 결함의 정확한 원인을 말할 수 없는 마지막 방어선이다.
  M1·M7·M8·M9와 anchor/alias, `|+` 가드는 더 구체적인 사유를 주거나 올바른 재정렬을
  계속 가능하게 하므로 제거하지 않았다.
- **주석 보존은 별도 계약이다.** self-check는 comment-only 차이를 의미 변경으로 보지 않는다.
  실제 byte/comment 보존은 기존 `TestCanonicalSectionOrderPreservesComments`와 다른 회귀들이
  계속 담당한다.

## Completion Criteria

- [x] 반환 직전 의미 비교가 있고, 불일치 시 `src`를 반환하며 `Blocked`에 이유를 남긴다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderSelfChecksSemanticEquivalence(' internal/config`
- [x] 이 검사가 실제로 결함을 잡는다 — TASK-363의 `|+` 입력을 개별 수정 없이 넣었을 때 차단된다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderSelfCheckCatchesKeepChomped(' internal/config`
- [x] 정상 파일이 이 검사에 걸리지 않는다 — 기존 테스트 전부 통과 | verify: `go test ./internal/config/` (regression-guard)
- [x] 추가 파스 1회의 비용을 실제로 측정하고 결과를 카드에 기록한다 (추정 금지) | verify: human — 벤치마크 결과 첨부
- [x] 게이트 통과 | verify: `make doc-check` (regression-guard)

## Benchmark (2026-09-10)

저장소의 추적된 `dva.yml` 13개를 재어 96–4,652 bytes, 중앙값 878 bytes임을 확인했다.
그중 최대인 루트 `dva.yml`(4,652 bytes)을 fixture로 다음 명령을 Apple M5 Max에서 5회
실행했다.

```bash
go test ./internal/config -run '^$' -bench '^BenchmarkMigrateSectionOrderSemanticSelfCheck$' -benchmem -count=5
```

5회 중앙값은 candidate parse만 **86,693 ns/op**, 전체 parse+compare가 **88,196 ns/op**였다.
비교 단계의 관측 증분은 **1,503 ns/op(1.73%)**이고, 두 sub-benchmark 모두
**1,363 allocs/op**였다(B/op 중앙값 83,566 대 83,567). 실제 반환 경로에 추가되는 전체
guard 비용은 이 최대 관측 파일에서 약 **88 µs/op**이었다.

## Resolution (2026-09-10)

- `finishSectionOrderMigration`이 반환 후보를 추가 parse하고 YAML node 의미를 비교한다.
- 과거 TASK-363 구현이 만든 `|+` 손실 candidate를 finishing boundary에 직접 넣어, 전용
  scalar 보존 로직과 독립적으로 원본 반환·`Blocked`·변경 보고 제거를 검증했다.
- 정상 재정렬, comment/style 차이, CRLF, document boundary를 포함한 기존 config test가
  모두 통과했다. 독립 리뷰와 `dva ci commit`도 통과했다.

## 참고

- `internal/config/migrate_section_order.go` — 반환 직전
- `internal/config/migrate_report.go` — `Blocked` 채널
- TASK-358 (anchor/alias), TASK-363 (`|+`) — 이 검사가 함께 막는 것들
- TASK-318 카드의 M1·M7·M8·M9 절 — 같은 결함 클래스의 네 변종
