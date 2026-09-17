---
id: TASK-391
title: "Stop planprogress from pairing any Korean counter with an enumeration"
type: bug
priority: P2
effort: S
exec-tier: strong
status: done
archived-at: 2026-09-17
verified-at: 2026-09-17
verification-summary: "Re-verified 2026-09-17. No knownbroken tag; ISSUE-016/017 repros live in prose_test.go; cardCounter and nearestEnum in prose.go; go test ./tools/planprogress/ ok."
quality-review: pass
quality-review-date: 2026-09-14
quality-review-session: review-planprogress (independent subagent; implementer was fix-planprogress)
quality-review-evidence: "verdict SOUND WITH CAVEATS. 7개 기준 바인딩 전부 rc0 (knownbroken 부재; ISSUE-016·017 재현이 prose_test.go에 존재; cardCounter·nearestEnum 존재; go test ./tools/planprogress/; make doc-check; make lint). 리뷰어가 재현 4건이 삭제가 아니라 이동임을 HEAD 대비 diff로 확인했고 단언이 완화 없이 추가만 됐음을 확인했다. 카브아트 둘은 이 카드의 기준 밖이며 ISSUE-021로 분리됐다 — (1) 대칭 짝짓기가 F3를 좁힐 뿐 없애지 못한다(어순 반례 측정됨), (2) 장-한정 narrowing은 코퍼스 적합이라 5개라고 쓰는 첫 저자부터 검사가 꺼진다(오늘 참 양성 손실 0건으로 측정)"
created: 2026-09-14
source: "ISSUE-016·017을 파일할 때 재현을 `//go:build knownbroken` 뒤에 빨갛게 심어 뒀다. 태그를 떼는 것이 수정의 기계적 증명이다"
depends-on: []
---

## Summary

`tools/planprogress`의 산문 검사가 **두 가지 방식으로 잘못 센다.** 둘 다 같은 파일
(`prose.go`)에 있고 둘 다 빨간 재현이 이미 `knownbroken` 태그 뒤에 심겨 있다.

**1. 문장을 공유한다는 것만으로 짝을 짓는다([[ISSUE-016]]).** `개`·`건`·`장`은 모두
한국어 수 분류사지만, 이 코퍼스에서 카드를 세는 것은 `장` 하나다. "필드 32개",
"관련 문서 8건", "23개 devbox 저장소"는 각각 필드·문서·저장소를 세고, 그런 문장이
옆에 TASK-id 나열을 데리고 있는 것은 정상이다. 그런데 검사는 같은 문장 안이기만
하면 그 수를 나열의 카드 수 주장으로 읽는다.

**2. 나열을 셀 때 중복 id를 그대로 센다([[ISSUE-017]]).** `TASK-1, 1`은 나열이지만
카드는 하나다. 중복을 세면 옆의 "두 장"이 거짓으로 충족된다.

**수정의 방향은 어휘가 아니라 짝짓기에 있다.** `findCountPhrases`는 셋을 다 계속
보고한다 — `8건`은 실재하는 수량이고, 다만 카드 수가 아닐 뿐이다. 좁히는 곳은
나열과 수를 맺는 자리다. 짝짓기는 **대칭**이어야 한다: 수는 그 나열의 최근접이고
동시에 나열도 그 수의 최근접일 때만 맺는다. 한쪽 방향만 보면 한 문장 안의 모든
나열이 같은 수를 나눠 갖는다.

## 이 카드가 하지 않는 것

검사를 끄지 않는다. 재현을 초록으로 만들려고 판정 기준을 느슨하게 바꾸는 것은
[[ISSUE-018]]이 이미 경고한 실패 형태 — 파싱 단계에서 조용히 검사가 꺼지는 것 —
과 같은 결과를 낳는다. 기존 true-positive·true-negative 픽스처가 전부 같은 뜻으로
남아 있어야 한다.

## Completion Criteria

- [x] `knownbroken` 빌드 태그가 `tools/planprogress`에서 사라진다 — 빨간 재현이 기본 테스트에 들어왔다는 뜻이다 | verify: `! /usr/bin/grep -rq 'go:build knownbroken' tools/planprogress`
- [x] 그 재현들이 삭제가 아니라 이동으로 살아 있다 | verify: `/usr/bin/grep -q 'ISSUE-016' tools/planprogress/prose_test.go && /usr/bin/grep -q 'ISSUE-017' tools/planprogress/prose_test.go`
- [x] 카드를 세는 분류사가 코드에 명시적으로 적혀 있다 — 암묵 규칙으로 남기지 않는다 | verify: `/usr/bin/grep -q 'cardCounter' tools/planprogress/prose.go`
- [x] 짝짓기가 대칭이라는 근거가 코드에 남는다 | verify: `/usr/bin/grep -q 'nearestEnum' tools/planprogress/prose.go`
- [x] 패키지 테스트 전부 통과 | verify: `go test ./tools/planprogress/`
- [x] 산문 검사가 살아 있는 plan 코퍼스에서 여전히 초록 | verify: `make doc-check` (regression-guard)
- [x] 저장소 게이트 통과 | verify: `make lint` (regression-guard)

## Sources

- [[ISSUE-016]] — 분류사 무차별 짝짓기.
- [[ISSUE-017]] — 중복 id를 세는 나열.

## Related

- [[ISSUE-018]] — 같은 파일의 세 번째 결함(파싱 조건이 검사를 조용히 끈다). 이
  카드는 건드리지 않는다 — 성격이 다르고, 같이 고치면 어느 수정이 무엇을 고쳤는지
  분간할 수 없다.
