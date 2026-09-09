---
id: TASK-362
title: "Make the flow-style fixture reproduce the panic it claims"
type: test
priority: P3
effort: S
exec-tier: standard
created-at: 2026-09-08T16:40:00+09:00
source: "TASK-318 재리뷰 (t318-rereview)"
status: todo
depends-on: []
needs-human: true
---

# Task 362: flow-style fixture가 자신이 막는다고 주장하는 패닉을 실제로 재현하게 한다

## 왜

`internal/config/migrate_section_order_test.go`의
`TestMigrateSectionOrderBailsOnUnrepresentableShapes`는 함수 주석에서 이렇게 말한다:

> covers the two document shapes a line-range partition cannot express. Both must
> return the source untouched — **and, for the duplicate key, must not panic**

`flow-style root mapping` 서브테스트는 이 "패닉을 막는다"는 취지를 이어받은 자리에 있지만,
그 fixture `{stack: b, version: a}\n`는 패닉을 유발하지 않는다 — 어떤 커밋 상태에서도
report assertion(재배열 없이 원본 그대로 반환됐는가)만 실패할 뿐이다. 즉 이 서브테스트는
"패닉을 막는 회귀 테스트"로서는 **빈 테스트**다 — 코드가 패닉 가드를 잃어도 이 fixture는
그 사실을 알려주지 못한다. TASK-350이 다루는 "바인딩이 뒤집혔거나 공허한가"의 테스트
버전이고, TASK-350의 카드가 문서 바인딩에 대해 하는 지적을 이 테스트 fixture에 대해
그대로 적용하는 사례다.

패닉을 실제로 재현하려면 flow-style root 앞에 배너 주석이 붙어야 한다. 배너가 있으면
`commentExtendedStart`의 되감기 로직이 flow-style 매핑의 두 키가 같은 줄(line)에 있다는
전제를 깨고 슬라이스 경계 계산에 관여하게 되어 패닉 조건이 성립한다.

## 실측

- 현재 fixture `{stack: b, version: a}\n` — 패닉 없음, 원본 바이트 그대로 반환됨.
  (`ec81723` 기준으로도 패닉 없음 — report assertion만 실패)
- 배너를 붙인 `"# banner\n{stack: b, version: a}\n"` — `ec81723`에서
  `slice bounds out of range [1:0]`로 패닉.

나머지 새 테스트 4개는 주석의 주장과 fixture가 정확히 결합돼 있다고 리뷰어가 확인했다 —
이 서브테스트 하나만 예외다.

## 무엇을

`flow-style root mapping` 서브테스트의 `src`를 `"# banner\n{stack: b, version: a}\n"`로
교체한다. 이렇게 하면 테스트가 주장하는 "패닉을 막는다"는 취지가 fixture와 실제로
결합되고, 향후 이 가드가 약화되거나 제거될 때 이 테스트가 (report assertion이 아니라)
패닉 자체로 즉시 실패하게 된다.

`{stack: b, version: a}\n`(배너 없는 버전)이 여전히 유효한 케이스라면 별도 서브테스트로
남겨도 되지만, "패닉을 막는다"는 주석의 주장은 배너 있는 fixture 쪽에 있어야 한다.

## 바인딩 주의 (2026-09-09)

두 번째 기준은 원래 이 카드가 **수정하려는 테스트 함수의 존재**에 걸려 있어 파일
시점에 통과했다. 기준의 문장은 "교체된 fixture가 패닉 없이 통과한다"이므로 증인은
이름이 아니라 실행이다 — `go test -run`으로 옮겼다. 첫 기준(fixture에 배너 주석)이
올바르게 실패하고 있어 카드 전체가 무증인은 아니었지만, 두 기준 중 하나만 일하고
있었다.

실행으로 옮긴 뒤에도 이 기준은 통과한다 — 지금 fixture도 패닉하지 않기 때문이다.
증인은 첫 기준(fixture에 배너 주석)이고 이 기준은 그 교체가 무언가를 깨지 않았음을
지키는 쪽이므로 `(regression-guard)`를 붙였다. 두 기준은 함께 읽어야 한다.

## Completion Criteria

- [ ] flow-style root mapping 서브테스트의 fixture가 배너 주석을 포함한 값으로 바뀌었다 | verify: `/usr/bin/grep -qF "# banner\\n{stack: b, version: a}" internal/config/migrate_section_order_test.go`
- [ ] 교체된 fixture가 현재 코드(패닉 가드 포함)에서는 패닉 없이 통과한다 | verify: `go test ./internal/config/ -run TestMigrateSectionOrderBailsOnUnrepresentableShapes` (regression-guard)
- [ ] 패닉 가드(같은 줄 keyLines 체크, :116-118)를 임시로 되돌렸을 때 새 fixture가 패닉으로 실패하고 이전 fixture는 실패하지 않음을 카드에 첨부해, 새 fixture가 실제로 더 강한 회귀 보호임을 근거로 남긴다 | verify: human — overlay(가드 제거) 실행 결과를 카드에 첨부

## 참고

- TASK-318 재리뷰 (t318-rereview) Q4 — flow-style root mapping 서브케이스
- TASK-350 (inverted/vacuous verify binding 거부) — 같은 범주의 테스트 버전
- `internal/config/migrate_section_order_test.go:221-251` (`TestMigrateSectionOrderBailsOnUnrepresentableShapes`), `internal/config/migrate_section_order.go:110-119` (패닉을 막는 가드)
