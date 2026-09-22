---
id: ISSUE-021
title: "Prose pairing blames the wrong enumeration and a reversed range discards the run after it"
type: bug
status: todo
priority: P2
severity: low
ownership: local
created: 2026-09-14
discovered-in: "review-planprogress independent review of TASK-391 (2026-09-14)"
discovered-at: 2026-09-14
source: "[[ISSUE-016]] F3의 잔여분과 [[ISSUE-017]]가 남긴 두 미결 — TASK-391이 F1과 중복 제거를 닫으면서 남긴 것"
promoted-to: [TASK-409]
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

위 블록은 `ca5157c` 시점의 출력 그대로다. 결함 E를 고친 뒤 같은 입력은
`... beside an enumeration naming 2 ...`를 낸다 — **문구만 바뀌었고 판정은 그대로다.**
블록을 갱신하지 않고 이렇게 적는 이유는, A가 관측된 순간의 출력이 A의 증거이기
때문이다. 증거를 사후 문구로 덮어쓰면 언제 무엇을 쟀는지가 사라진다.

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

## 소유권 — 이 저장소다 (2026-09-14 명시)

[[ISSUE-016]]·[[ISSUE-017]]·[[ISSUE-018]]과 같은 칸이다 — 결함 A·B·C·D·E가 전부 이
저장소의 `tools/planprogress` 코드 안에 있고, 이 저장소가 고칠 수 있고 고쳐야 한다.

이 표기가 처음 이 카드에 빠져 있었다는 사실 자체를 남긴다. 세 형제 카드에는 있고
이 카드에만 없었고, 그 사이 이 보드의 이슈 칸은 상류 소유가 다수가 됐다 — 표기를
빠뜨린 새 카드는 기본값으로 그 다수에 섞여 "이 보드가 어쩔 수 없는 것"처럼 읽힌다.
자기 소유 결함이 그 자리에 섞이면 고칠 수 있는 일이 영원히 대기열에 남는다.

## Resolution Criteria

- [ ] A: 어순을 바꾼 반례가 더 이상 오탐을 내지 않거나, 왜 내는지가 규칙의
      의도로 코드에 적혀 있다 | verify: human — `prose.go`의 파일 코멘트를 읽고,
      "다섯 장 — TASK-1·2와 TASK-3·4·5"에 대한 코드의 실제 동작이 코멘트가
      서술하는 동작과 일치하는지 확인한다
- [x] B: 역방향 범위 뒤의 나열이 보존되거나, 버린다는 사실이 보고된다 —
      어느 쪽이든 `"reversed range is ignored"` 기대값이 그 결정에 맞게 갱신된다
      | verify: `/usr/bin/grep -q 'TASK-365\.\.358' tools/planprogress/prose_test.go`
- [x] C: 위키링크 나열이 인식되거나, 인식하지 못한다는 사실이 테스트로 못박혀
      있다 | verify: `/usr/bin/grep -q 'TASK-1\]\]' tools/planprogress/prose_test.go`
- [x] 패키지 테스트와 문서 게이트가 통과한다 | verify: `go test ./tools/planprogress/ && make doc-check`

## 부가 관찰 — 바인딩을 달지 않은 둘 (2026-09-14)

`review-planprogress`의 리뷰 후반부가 낸 minor 둘이다. **A·B·C와 달리 오판이
아니다** — 검사의 답은 옳고, 그 답을 설명하는 말이 실제보다 크다. 결함이 아닌 것에
기준을 다는 것은 기준을 장식으로 만드는 일이라 바인딩 없이 관찰로만 남긴다.

**D — `findCountPhrases`의 개/건 출력은 오늘 아무 데도 닿지 않는다.** 유일한 프로덕션
호출부가 곧바로 `cardCounts(...)`로 감싼다(`prose.go:102`). 그래서 파일 코멘트의
"the narrowing lives in the pairing, not in the vocabulary"는 구조적으로는 참이지만
관측 가능한 차이가 없다 — 개/건 매치는 테스트에만 보인다. 어휘가 아니라 짝짓기에서
거른다는 설계는 나중에 `건`이 카드를 세는 코퍼스가 오면 값을 하지만, 그 날이 오기
전까지 이 문장은 실제보다 무겁게 읽힌다. 직접 측정:

```
$ /usr/bin/grep -n 'findCountPhrases(' tools/planprogress/prose.go
102:		counts := cardCounts(findCountPhrases(sentence))
213:func findCountPhrases(sentence string) []countMatch {
```

**E — 중복 제거 후 "enumeration of 1"이 나올 수 있다.** `len(ids) >= 2` 판정이 중복
제거 **전**의 raw run에서 이뤄지기 때문이다. 측정:

```
findCountedEnumerations("TASK-1, 1 두 장이다")
  -> {ids: ["TASK-1"], count: 2, text: "TASK-1, 1"}
  -> counts 2 card(s) beside an enumeration naming 1
```

**판정 자체는 옳다** — 산문이 두 장이라 말하면서 한 장만 이름을 댔으니 결함이
맞다. 어색했던 것은 메시지였다: 이 파일은 enumeration을 "id 둘 이상"으로 정의해
놓고 "enumeration of 1"을 출력했다. 위 블록은 **수정 후**의 출력이다 — 이 절이
처음 쓰였을 때는 `of 1`이었고, 그 문구를 고친 것이 아래 처분의 E다.

## 2026-09-14 처분 — B·C·E는 닫혔고 A·D는 각각 다른 이유로 열려 있다

[[TASK-394]]가 처리했다. 처분을 결함별로 갈라 적는다 — 한 카드가 다섯 결함을
들고 있으면 "닫혔다"는 한 단어가 어느 것을 가리키는지 사라지기 때문이다.

**B 고쳤다 — 방향은 "보존"이다.** 세 안이 있었다. (1) 지금처럼 뒤를 전부 버린다,
(2) 잘못 쓰인 범위를 결함으로 보고한다, (3) 범위는 확장하지 않되 뒤따르는 나열은
살린다. (2)를 택하지 않은 이유는 `prose.go`의 파일 코멘트가 스스로 거부하는
역할이기 때문이다 — 이 도구는 style checker가 아니다. 오타를 지적하는 일을
맡기 시작하면 산문 검사의 경계가 무너진다. 그리고 **(1)이 셋 중 가장 나쁘다**:
초록을 내면서 아무것도 재지 않기 때문이다. 그래서 (3)이다. `end += m[1]`을 범위
수락 분기 밖으로 빼는 한 줄이고, 역방향 범위는 여전히 확장되지 않고 자기 anchor
id 하나만 남긴다.

독립 리뷰가 HEAD와 수정본을 같은 표에 통과시켜 재측정했다. 수락된 범위 케이스는
전부 바이트 단위로 동일하고, 달라진 것은 거부된 범위가 뒤따르는 run을 보존하는
경우뿐이다. `strconv.Atoi`가 overflow로 실패하는 경우에도 `end`가 전진하게 된 것은
덤이다.

**리뷰가 새로 측정한 것 하나를 그대로 남긴다.** `"다섯 장(TASK-5..3, 6, 7)…"`은
수정 전에는 조용했고 수정 후에는 `counts 5 card(s) beside an enumeration naming 3`을
낸다. 산문이 실제로 앞뒤가 안 맞으므로 **거짓 양성은 아니다.** 다만 메시지가
개수를 탓하는데 원인은 옆에 선 역방향 범위다. 보존 방향의 알려진 대가로 기록한다.

**C 닫았다 — 구현이 아니라 고정으로.** 위키링크 나열은 여전히 보이지 않는다.
`runTailRE`가 구분자가 anchor 바로 뒤에 오기를 요구하는데 `]]`가 사이에 끼기
때문이다. 보드 자신의 상호참조 표기를 파싱하는 일은 이 카드의 범위가 아니었다.
대신 그 침묵을 테스트로 못박았다 — 우연한 침묵과 의도한 침묵은 관측으로
구분되지 않으므로, 구분을 테스트가 만든다.

**E 고쳤다 — 이 절이 예고한 그대로.** 위에서 "A·B·C를 다룰 때 같은 파일을 열므로
그때 함께 보는 편이 낫다"고 적었고, 실제로 그렇게 됐다. `enumeration of N` →
`enumeration naming N`. 판정 로직은 건드리지 않았다. 저장소 전체에서 옛 문구를
쓰는 곳은 18군데인데 **코드와 `verify:` 바인딩에는 0건**이고 전부 닫힌 카드의
산문과 동결된 영수증이라, 그쪽은 역사로 두고 손대지 않았다.

**A 열려 있다.** 대칭 짝짓기는 F3을 좁혔을 뿐 없애지 못한다. 어순을 바꾸면 거짓
양성이 돌아온다는 반례가 `prose.go` 파일 코멘트에 박혀 있다. 정규식으로 결정
가능한 문제가 아니라는 판단이 서기 전까지 이 칸은 비워 둔다.

**D 열려 있다 — 그리고 이번에도 바인딩을 달지 않았다.** `findCountPhrases`의
개/건 출력은 여전히 테스트 말고는 아무 데도 닿지 않는다. 죽은 출력을 지울지
쓰일 자리를 만들지는 설계 판단이고, 이 카드가 혼자 정할 일이 아니다.

## Related

- [[ISSUE-016]] — F1은 닫혔고 F3의 잔여분이 여기 A로 왔다.
- [[ISSUE-017]] — 중복 제거는 닫혔고 나머지 둘이 여기 B·C로 왔다.
- [[TASK-391]] — 앞의 절반을 닫은 작업.
