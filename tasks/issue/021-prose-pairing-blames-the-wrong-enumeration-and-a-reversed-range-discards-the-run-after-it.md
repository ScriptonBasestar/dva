---
id: ISSUE-021
title: "Prose pairing blames the wrong enumeration and a reversed range discards the run after it"
type: bug
status: todo
priority: P2
severity: low
created: 2026-09-14
discovered-in: "review-planprogress independent review of TASK-391 (2026-09-14)"
discovered-at: 2026-09-14
source: "[[ISSUE-016]] F3의 잔여분과 [[ISSUE-017]]가 남긴 두 미결 — TASK-391이 F1과 중복 제거를 닫으면서 남긴 것"
---

## Summary

[[TASK-391]]이 [[ISSUE-016]] F1(무관한 수량 오짝짓기)과 [[ISSUE-017]]의 중복 수집을
닫았다. 닫히지 않은 것 세 가지를 여기 모은다 — 셋 다 **조용한 오판**이라는 같은
모양이다. 검사가 틀린 답을 내거나 아무 답도 내지 않으면서, 두 경우 모두 초록이다.

## Evidence

전부 2026-09-14에 `ca5157c`(TASK-391 통합 커밋) 위에서 직접 측정했다.

**A — 대칭 짝짓기는 F3를 좁혔을 뿐 없애지 않았다.** `review-planprogress`가 낸
반례를 그대로 재현했다. 같은 문장의 어순만 바꾸면 오탐이 돌아온다:

```
findCountedEnumerations("다섯 장 — TASK-1·2와 TASK-3·4·5")   (children 1..5)
  -> ## Goal counts 5 card(s) beside an enumeration of 2 ("TASK-1·2")
```

다섯 장을 두 묶음으로 나눠 쓴 **옳은 문장**이 결함으로 보고된다. 대칭성은 문장의
모든 나열이 함께 붉어지지 않게 할 뿐, 지목된 하나가 그 수의 주어라는 것을 보장하지
않는다. [[ISSUE-016]]이 원래 코드에 던진 지적("지금의 초록은 규칙의 속성이 아니라
줄바꿈의 우연이다")이 어순으로 옮겨간 것이다.

**B — 역방향 범위가 뒤따르는 나열까지 통째로 버린다.** [[ISSUE-017]] 제목의 앞
절반이다. 범위 분기가 실패해도 `end`가 전진하지 않아, 뒤에 오는 정상적인 run이
파서에 닿지 못한다:

```
findEnumerations("TASK-9..1, 2")              -> []
findEnumerations("TASK-365..358, 366, 367")   -> []
```

두 번째가 아픈 쪽이다. `366, 367`은 멀쩡한 멤버십 주장인데 검사에 보이지 않는다.
현재 `TestFindEnumerations`의 `"reversed range is ignored"`는 이 동작을 **정답으로
못박고 있다** — 그 기대값은 결정과 함께 갱신돼야 한다.

**C — 위키링크 형태의 나열이 보이지 않는다.** [[ISSUE-017]]의 P5 probe를 그대로
돌렸다:

```
findEnumerations("[[TASK-1]], [[TASK-9]] 두 장이 남았다")  -> []
```

보드의 산문은 카드를 위키링크로 가리키는 것이 관례인데, 그 관례로 쓴 나열은
검사 대상이 아니다. 조용히 건너뛴다는 사실 자체가 보고되지 않는다.

## Reproduction

1. `ca5157c` 이후의 `tools/planprogress`에서 임시 테스트 파일을 만들어
   `findEnumerations`/`findCountedEnumerations`에 위 Evidence의 네 문장을 그대로
   넣고 그 임시 테스트만 골라 `-v`로 돌려 반환값을 찍는다(`-run` 패턴은 여기에
   적지 않는다 — 존재하지 않는 이름을 적으면 `go test`가 0건 매치로 exit 0을
   내고, doc-check이 그 vacuous 패턴을 막는다).
2. A는 `count: 5`가 두 개짜리 나열에 붙어 돌아오고, B와 C는 빈 슬라이스가
   돌아온다.
3. 확인 후 임시 파일을 지운다 — 이 재현은 기본 스위트에 남기지 않는다.
   남기는 자리는 위 Resolution Criteria가 정한다.

## Expected vs Actual

- Expected: 수와 나열을 짝지을 때 지목된 나열이 그 수의 주어이고, 파싱하지
  못한 입력은 조용히 사라지는 대신 보존되거나 보고된다.
- Actual: 지목은 어순에 좌우되고(A), 역방향 범위 하나가 뒤따르는 정상 나열까지
  삼키며(B), 보드의 관례인 위키링크 나열은 아예 보이지 않는다(C).

## Impact

A는 옳은 카드를 CI에서 떨어뜨린다 — 즉시 아프다. B와 C는 반대로 **틀린 카드를
통과시킨다** — 아프지 않아서 더 오래 남는다. 셋 다 검사를 믿을 근거를 깎는데,
B·C는 초록이 무엇을 뜻하는지 모르게 만든다는 점에서 A보다 나쁘다.

## Resolution Criteria

- [ ] A: 어순을 바꾼 반례가 더 이상 오탐을 내지 않거나, 왜 내는지가 규칙의
      의도로 코드에 적혀 있다 | verify: human — `prose.go`의 파일 코멘트를 읽고,
      "다섯 장 — TASK-1·2와 TASK-3·4·5"에 대한 코드의 실제 동작이 코멘트가
      서술하는 동작과 일치하는지 확인한다
- [ ] B: 역방향 범위 뒤의 나열이 보존되거나, 버린다는 사실이 보고된다 —
      어느 쪽이든 `"reversed range is ignored"` 기대값이 그 결정에 맞게 갱신된다
      | verify: `/usr/bin/grep -q 'TASK-365\.\.358' tools/planprogress/prose_test.go`
- [ ] C: 위키링크 나열이 인식되거나, 인식하지 못한다는 사실이 테스트로 못박혀
      있다 | verify: `/usr/bin/grep -q 'TASK-1\]\]' tools/planprogress/prose_test.go`
- [ ] 패키지 테스트와 문서 게이트가 통과한다 | verify: `go test ./tools/planprogress/ && make doc-check`

## Related

- [[ISSUE-016]] — F1은 닫혔고 F3의 잔여분이 여기 A로 왔다.
- [[ISSUE-017]] — 중복 제거는 닫혔고 나머지 둘이 여기 B·C로 왔다.
- [[TASK-391]] — 앞의 절반을 닫은 작업.
