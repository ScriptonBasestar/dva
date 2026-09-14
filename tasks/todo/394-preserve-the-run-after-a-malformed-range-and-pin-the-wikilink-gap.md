---
id: TASK-394
title: "Preserve the run after a malformed range and pin the wikilink gap"
type: bug
priority: P2
effort: S
exec-tier: strong
status: todo
created: 2026-09-14
source: "[[ISSUE-021]] B·C와 [[ISSUE-017]]의 남은 사람 기준 — TASK-391이 앞 절반을 닫으면서 남긴 것"
depends-on: []
---

## Summary

[[ISSUE-021]]의 B와 C를 닫는다. 둘 다 **조용한 침묵**이라는 같은 모양이다 — 검사가
아무것도 재지 않으면서 초록을 낸다.

**B는 고친다. 방향은 "보존"으로 정했다.** 잘못 쓰인 범위 하나가 뒤따르는 정상 나열을
통째로 삼킨다:

```
findEnumerations("TASK-365..358, 366, 367")   -> []
```

`366, 367`은 멀쩡한 멤버십 주장인데 검사에 닿지 못한다. 범위 분기가 가드에 걸려
실패해도 커서가 전진하지 않는 것이 원인이다.

**왜 "보고"가 아니라 "보존"인가.** 선택지는 셋이었다 — 지금처럼 통째로 버리기,
뒤 나열을 보존하기, 역방향 범위 자체를 결함으로 보고하기. 보고하는 쪽은
planprogress를 문체 검사로 한 걸음 밀어내는데, 이 도구의 파일 코멘트가 그 방향을
명시적으로 거부한다("not a style checker"). 보존하는 쪽은 잘못 쓴 범위를 묵인하지만
**검사는 계속 작동한다.** 지금 동작이 셋 중 최악이다: 초록인데 아무것도 재지 않았고,
그 사실조차 알리지 않는다.

**C는 고치지 않는다. 침묵을 테스트로 못박는다.** 위키링크 나열이 보이지 않는다:

```
findEnumerations("[[TASK-1]], [[TASK-9]] 두 장이 남았다")  -> []
```

위키링크 파싱을 넣는 것은 이 카드의 범위가 아니다. 대신 현재 동작을 고정하는
테스트를 둬서, 이 침묵이 **우연이 아니라 기록된 결정**이 되게 한다. 고정되지 않은
침묵은 다음 사람이 "원래 되는 줄 알았다"고 읽는다.

## 이 카드가 하지 않는 것

- [[ISSUE-021]] A(어순에 좌우되는 지목)는 건드리지 않는다. 대칭 짝짓기의 한계는
  `prose.go` 파일 코멘트에 이미 측정된 반례와 함께 적혀 있고, 없애려면 문장의
  주어를 판정해야 한다 — 정규식으로 결정 가능한 성질이 아니다.
- 위키링크 나열 파싱을 구현하지 않는다(C 참조).

## Completion Criteria

- [ ] 잘못 쓰인 범위 뒤의 나열이 살아남는다 | verify: `/usr/bin/grep -q 'TASK-365\.\.358' tools/planprogress/prose_test.go`
- [ ] 위키링크 나열의 침묵이 테스트로 고정된다 | verify: `/usr/bin/grep -q 'TASK-1\]\]' tools/planprogress/prose_test.go`
- [ ] 기대값이 결정에 맞게 갱신됐다 | verify: human — `TestFindEnumerations`에서
      역방향 범위 케이스의 이름과 기대값을 읽고, "ignored"라는 이름이 더 이상
      동작을 잘못 서술하지 않는지, 코드의 실제 동작과 일치하는지 확인한다
- [ ] 패키지 테스트 전부 통과 | verify: `go test ./tools/planprogress/`
- [ ] 산문 검사가 살아 있는 plan 코퍼스에서 여전히 초록 | verify: `make doc-check` (regression-guard)
- [ ] 저장소 게이트 통과 | verify: `make lint` (regression-guard)

## Sources

- [[ISSUE-021]] — B·C가 여기서 닫힌다. A와 D·E는 남는다.
- [[ISSUE-017]] — 이 카드가 역방향 범위 기준을 닫는다.

## Related

- [[TASK-391]] — 앞 절반(F1·중복 제거)을 닫은 작업.
