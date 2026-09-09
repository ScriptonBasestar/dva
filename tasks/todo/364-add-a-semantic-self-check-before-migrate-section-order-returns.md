---
id: TASK-364
title: "fix: add a semantic self-check before MigrateSectionOrder returns"
type: bug
priority: P1
effort: M
exec-tier: strong
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview) SHOULD-FIX 6"
status: todo
depends-on: []
needs-human: true
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

## 결정이 필요한 지점

- **비용은 측정되지 않았다.** 리뷰어가 "성능. 파일당 파스 1회 추가의 비용을 측정하지
  않았습니다"라고 명시적으로 미측정으로 남겼다. 파일당 파스 1회는 직관적으로 싸 보이지만,
  이 카드를 닫기 전에 실제 dva.yml 크기 분포에서 재 볼 것. **추정치를 측정치인 것처럼
  카드에 적지 말 것.**
- **비교의 정밀도.** `%#v` 문자열 비교는 프로브 하네스에서는 충분했지만 프로덕션 코드로는
  거칠다. `reflect.DeepEqual`을 `map[string]any`에 쓸지, 노드 트리를 정규화해 비교할지
  결정해야 한다. 주석은 의미가 아니므로 비교 대상에서 빠져야 하는데, 주석 보존이 이
  함수의 존재 이유라는 점과 충돌하지 않게 할 것 — **의미 동등성**과 **주석 보존**은
  각각 다른 단언이다.
- **차단이 수정을 대체하지 않는다.** 이 self-check가 들어오면 TASK-358·363은 "조용한
  손실"에서 "차단됨"으로 바뀐다. 차단은 손실보다 낫지만, 사용자 입장에서는 파일이
  여전히 안 고쳐진 것이다. 두 카드를 닫을 이유가 아니라 우선순위를 낮출 이유일 뿐이다.
- **개별 가드를 제거할 것인가.** 하지 말 것. 우산 검사는 "무엇이 잘못됐는지" 말해 주지
  못한다. M1·M7·M8·M9의 가드는 각각 구체적인 이유를 남긴다.

## Completion Criteria

- [ ] 반환 직전 의미 비교가 있고, 불일치 시 `src`를 반환하며 `Blocked`에 이유를 남긴다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderSelfChecksSemanticEquivalence(' internal/config`
- [ ] 이 검사가 실제로 결함을 잡는다 — TASK-363의 `|+` 입력을 개별 수정 없이 넣었을 때 차단된다 | verify: `/usr/bin/grep -rq 'func TestMigrateSectionOrderSelfCheckCatchesKeepChomped(' internal/config`
- [ ] 정상 파일이 이 검사에 걸리지 않는다 — 기존 테스트 전부 통과 | verify: `go test ./internal/config/` (regression-guard)
- [ ] 추가 파스 1회의 비용을 실제로 측정하고 결과를 카드에 기록한다 (추정 금지) | verify: human — 벤치마크 결과 첨부
- [ ] 게이트 통과 | verify: `make doc-check`

## 참고

- `internal/config/migrate_section_order.go` — 반환 직전
- `internal/config/migrate_report.go` — `Blocked` 채널
- TASK-358 (anchor/alias), TASK-363 (`|+`) — 이 검사가 함께 막는 것들
- TASK-318 카드의 M1·M7·M8·M9 절 — 같은 결함 클래스의 네 변종
