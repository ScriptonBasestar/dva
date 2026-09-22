---
id: ISSUE-017
title: "findEnumerations silently discards ids on a reversed range and never deduplicates"
type: bug
status: done
priority: P2
severity: low
effort: S
exec-tier: standard
ownership: local
created: 2026-09-14
discovered-in: "review-381 independent review of TASK-381 (2026-09-14)"
discovered-at: 2026-09-14
source: "review-381의 conditional 판정, F4/F5/F5b. 셋 다 tools/planprogress/prose.go의 findEnumerations(및 그 안에서 쓰는 runTailRE) 계열 안의 결함이다"
promoted-to: [TASK-408]
resolution: fixed
resolved-at: 2026-09-22T02:32:30Z
resolution-summary: "Resolved as fixed by TASK-408."
---

## Summary

`findEnumerations`(`tools/planprogress/prose.go:108-162`)에 서로 다른 세 결함이
있다: 역방향 범위는 통째로 사라지고(F4), 반복된 id는 중복 제거되지 않으며(F5),
위키링크 형태(`[[TASK-N]]`)로 쓰인 나열은 애초에 인식되지 않는다(F5b). F5는 이
도구의 핵심 주장 — "나열이 세는 카드 수를 정확히 안다" — 을 조용히 뒤집고,
F5b는 그 주장을 검사 대상에서 통째로 빼버린다: `[[TASK-NNN]]`은 이 보드 자체의
상호참조 표기법이다.

## Evidence

**F4 — 역방향/과대 범위가 뒤에 오는 나열까지 통째로 삼킨다** (anchor
`prose.go:124-135`). `rangeTailRE`가 매치됐지만 `last < first`이거나
`last-first >= maxRangeSpan`이라 범위 분기가 거부되면, `end`는 `first`를 매치한
직후(`..` 앞)에 그대로 남는다. `ids`는 `first` 하나뿐이라 `len(ids) >= 2`를
통과하지 못해 아무것도 반환되지 않고, 바깥 루프는 `pos = end`로 `..` 뒤의 나머지
텍스트를 다시 스캔하지만 그 안에 새 `TASK-` 앵커가 없으면 그대로 끝난다.

측정: `findEnumerations("TASK-9..1, 2")` → `nil`. `TASK-9`, `TASK-1`, `TASK-2`
어느 것도 나열로 잡히지 않는다.

이것은 저자가 반대 방향으로 이미 한 번 고친 것과 같은 모양이다: 범위 뒤에 쉼표로
이어지는 run이 조용히 버려지던 결함을 고쳐 `range_continued_by_a_run` 테스트로
고정했다(`prose_test.go`의 "range continued by a run" 케이스, "the tail used to
be dropped silently" 주석). 이번엔 반대 방향 — 범위 자체가 무효일 때 — 에서 같은
증상이 남아 있다.

함정을 명시한다: `TestFindEnumerations`의 `"reversed range is ignored"`
서브테스트(`prose_test.go`, `findEnumerations("TASK-365..358") == nil`을
기대)는 **지금의 결함을 정답으로 고정하고 있다.** 이 결함을 고치는 것은 곧 그
테스트의 기대값을 바꾸는 일이므로, 지나가다 손대는 패치가 아니라 카드가 필요한
이유가 여기 있다 — 무엇을 정답으로 정할지가 아직 결정되지 않았다(범위가
과대해도 같은 증상).

**F5 — id가 중복 제거되지 않는다** (anchor `prose.go:151-158`, run 연속 루프
`:138-149`). 같은 id가 쉼표로 두 번 나오면 `ids` 슬라이스에 그대로 두 번
쌓인다.

측정: `findEnumerations("TASK-1, 1, 2")` → `ids = [TASK-1 TASK-1 TASK-2]`
(개수 3, 서로 다른 카드는 2개). `checkPlanProse`에 올려보면 더 나쁘다 —
`children: [TASK-1, TASK-2, TASK-3]`, `scope: "TASK-1, 1, 2 — 세 장이 같은
파일을 고친다"`를 주면 결함이 하나도 보고되지 않는다: `len(e.ids)`가 3이라
`e.count(3) == len(e.ids)(3)`으로 우연히 맞아떨어지기 때문이다. **틀린 셈이
옳은 셈으로 읽힌다** — 이 카드가 존재하는 이유인 핵심 주장이 스스로 깨지는
지점이다.

**F5b — 위키링크 형태가 나열 탐지를 통째로 무력화한다** (anchor
`prose.go:51` `runTailRE`, 호출부 `:139`). `runTailRE`는 앵커 직후에 `,` 또는
`·`가 **곧바로** 와야 다음 id로 이어붙인다
(`^\s*[,·]\s*(?:TASK-)?0*(\d+)`). 위키링크 `[[TASK-1]]`은 앵커 뒤에 `]]`가
먼저 오므로 이 정규식이 애초에 매치하지 않는다 — 나열 자체가 성립하지 않고,
그 문장은 검사 대상에서 통째로 빠진다.

측정 (review-381의 probe P5):
```go
findEnumerations("[[TASK-1]], [[TASK-9]] 두 장이 남았다") // == nil
```
`children: [TASK-1]`(TASK-9는 이 plan의 자식이 아님)인 plan에 같은 문장을
Goal로 주고 `checkPlanProse`를 호출하면 **결함이 0개** 보고된다 — TASK-9가
자식이라고 주장하는 나열이 조용히 통과한다. 소속 위반(membership violation)이
보고되지 않는다.

이 결함은 카드의 severity 표기(`low`)가 시사하는 것보다 크게 다뤄야 한다:
`[[TASK-NNN]]`은 이 보드 자체의 상호참조 표기법이다. `[[TASK-N]]` 형태의
실측 개수는 [[TASK-381]] 본문 7회, [[PLAN-007]] 2회, PLAN-006/008/009는
0회다 — PLAN-008/009는 같은 대괄호 표기법을 `[[ISSUE-N]]`/`[[PLAN-N]]`에는
쓰지만 `[[TASK-N]]` 실례는 없고, PLAN-006에는 대괄호 위키링크 자체가 없다.

다만 **이 검사가 실제로 읽는 두 영역 — `scope:`와 `## Goal` 절 — 에는 네 plan
카드 전부에서 오늘 위키링크가 하나도 없다**(직접 확인). 그래서 이 노출은 지금
살아있는 카드를 조용히 빠뜨리고 있는 것이 아니라 **잠재적(latent)**이다: 이
결함 때문에 검사가 꺼진 채로 통과하고 있는 카드는 오늘 하나도 없다. 그러나
TASK-381·PLAN-007처럼 본문에서 `[[TASK-N]]`을 자유롭게 쓰는 저자가 그 습관을
scope/Goal로 옮기는 순간, 검사는 진단 없이 조용히 꺼진다 — 노출은 그 날
현재화된다.

이 문장들은 이 카드와 함께 회귀 픽스처로 추가했다:

- `tools/planprogress/prose_test.go`의 `TestFindEnumerations`에
  `"duplicate id in a run is deduplicated"` (F5) — 지금 `go test
  ./tools/planprogress/`에서 FAIL한다:
  ```
  --- FAIL: TestFindEnumerations/duplicate_id_in_a_run_is_deduplicated
      prose_test.go:217: findEnumerations("TASK-1, 1, 2") ids = [TASK-1 TASK-1 TASK-2], want [TASK-1 TASK-2]
  ```
- F4는 고정된 정답이 없어 회귀 픽스처를 추가하지 않았다 — 아래 Resolution
  Criteria 참조.
- F5b도 고칠 방향(`runTailRE`를 위키링크 종료(`]]`)까지 허용하도록 넓힐지,
  앵커 인식 자체를 바꿀지, 다른 접근인지)이 아직 결정되지 않아 회귀 픽스처를
  추가하지 않았다 — 아래 Resolution Criteria 참조.

## Reproduction

1. `findEnumerations("TASK-9..1, 2")`를 호출한다 — `nil`을 반환한다.
2. `children: [TASK-1, TASK-2, TASK-3]`인 plan에 `scope: "TASK-1, 1, 2 — 세
   장이 같은 파일을 고친다"`를 주고 `checkPlanProse`를 호출한다 — 결함이
   보고되지 않는다.
3. `go test ./tools/planprogress/ -run TestFindEnumerations -v`를 실행한다 —
   `duplicate_id_in_a_run_is_deduplicated` 서브테스트가 FAIL한다.
4. `children: [TASK-1]`(TASK-9는 자식이 아님)인 plan에 `goal: "[[TASK-1]],
   [[TASK-9]] 두 장이 남았다"`를 주고 `checkPlanProse`를 호출한다 — TASK-9가
   자식이 아닌데도 결함이 0개 보고된다.

## Expected vs Actual

- Expected: 역방향/과대 범위는 최소한 눈에 보이는 방식으로 실패하거나(결함으로
  보고), 뒤에 이어지는 독립된 나열까지 삼키지는 않는다. 반복된 id는 서로 다른
  카드 수를 셀 때 한 번만 센다. 위키링크로 쓰인 나열도 다른 구분자로 쓴 나열과
  동등하게 인식되거나, 인식하지 못한다는 사실이 드러난다.
- Actual: 역방향/과대 범위는 조용히 통째로 사라지고, 반복된 id는 그대로
  누적되어 틀린 카드 수를 옳은 것처럼 보이게 하며, 위키링크로 쓰인 나열은
  진단 없이 통째로 무시된다.

## Impact

F4는 낮다 — 역방향 범위는 오타에 가깝고 오늘 보드에 실례가 없다. F5는 더
직접적이다: 이 검사의 존재 이유가 "나열이 주장하는 카드 수를 정확히 센다"인데,
반복된 id 하나가 그 셈을 조용히 틀리게 만들고도 통과로 보고한다 — 검사가 자신의
핵심 계약을 어기는 입력이 실재한다.

F5b는 이 셋 중 가장 넓게 걸린다. F4/F5는 나열이 일단 인식된 뒤의 오류지만,
F5b는 인식 자체를 막아 검사를 통째로 끈다 — [[ISSUE-018]]이 다루는 파싱
단계의 "조용한 off"와 같은 성질이다. 걸리는 입력이 오타가 아니라 보드의 정상
표기법이라는 점에서 이 카드의 severity(`low`) 표기가 시사하는 것보다 크게
다뤄야 한다.

명시적으로 적는다: 이 노출은 **잠재적(latent)이지, 지금 벌어지고 있는 일이
아니다.** 검사가 실제로 읽는 `scope:`/`## Goal` 두 영역에는 PLAN-006/007/
008/009 네 카드 전부에서 오늘 위키링크가 없으므로, 이 결함 때문에 지금
조용히 통과하고 있는 살아있는 카드는 없다. 노출은 저자가 본문에서 이미 쓰는
`[[TASK-N]]` 습관(TASK-381 7회, PLAN-007 2회)을 scope/Goal로 옮기는 순간
현재화된다 — 그리고 그 날 검사는 진단도 신호도 없이 조용히 꺼진다.

## 소유권 — 이 저장소다 (2026-09-14 명시)

세 카드(ISSUE-016·017·018)는 `review-381`의 독립 리뷰에서 나왔고, 같은 리뷰가 낸 다른
이슈들이 대부분 `ce-agent-kit`·`ce-workbook` 소유라 서술 형식이 상류 이슈와 구분되지
않았다. **이 셋은 상류가 아니다** — 결함은 전부 이 저장소의 `tools/planprogress` 코드
안에 있고, 이 저장소가 고칠 수 있고 고쳐야 한다.

구분이 중요한 이유: 상류 소유 이슈는 이 보드가 진척을 강제할 수 없어 열린 채 남는 것이
정상이지만(PLAN-007 §External의 선례), 자기 소유 결함을 같은 칸에 넣어 두면 고칠 수 있는
일이 영원히 대기열에 남는다.

## Resolution Criteria

- [x] `tools/planprogress/known_issues_test.go`의 `TestIssue017Deduplication`
      (`TASK-1, 1, 2` → 두 장)이 통과한다 — `findEnumerations`가 중복 id를
      제거한다는 뜻이다. 이 바인딩은 구성상 지금은 실패하며, 결함이 고쳐졌을
      때만 통과한다. 명령이 `-v`와 PASS 줄 grep을 거치는 이유는 `go test -run`이
      0건 매치일 때 `[no tests to run]`과 함께 exit 0을 내기 때문이다 — 재현을
      지우는 것으로는 이 기준을 만족시킬 수 없다 | verify: `sh -c 'go test
      -tags=knownbroken -run "^TestIssue017Deduplication$" -v ./tools/planprogress/ 2>&1 | /usr/bin/grep -qE
      "^--- PASS: TestIssue017Deduplication "'`
- [x] 역방향/과대 범위(`maxRangeSpan` 초과 포함)를 어떻게 다룰지 — 조용히
      건너뛰되 뒤 나열은 보존, 결함으로 보고, 또는 다른 처리 — 결정되고,
      `TestFindEnumerations`의 `"reversed range is ignored"` 기대값이 그
      결정에 맞게 갱신된다 | verify: human — 결정과 갱신된 테스트가 이 카드
      또는 후속 커밋에 링크되었는지 확인
- [ ] `[[TASK-NNN]]` 위키링크 형태로 쓰인 id 나열이 `findEnumerations`에
      인식되거나, 인식하지 못하는 경우 그 사실이 명시적으로 보고된다(조용히
      전체를 건너뛰지 않는다) — 위 probe P5 문장(`[[TASK-1]], [[TASK-9]] 두
      장이 남았다`, TASK-9는 자식이 아님)을 Goal로 주면 `checkPlanProse`가
      결함을 내거나 스킵 사실을 알리는 것으로 확인한다 | verify: human — probe
      P5 픽스처를 넣고 결과를 확인
- [x] 이 결함의 재현이 기본 테스트 스위트로 옮겨졌다 — `TestIssue017Deduplication`가
      `prose_test.go`에 있고 `known_issues_test.go`에는 남아 있지 않다(태그를
      지우는 것만으로는 만족되지 않는다) | verify: `/usr/bin/grep -q 'TestIssue017Deduplication'
      tools/planprogress/prose_test.go && ! /usr/bin/grep -q 'TestIssue017Deduplication'
      tools/planprogress/known_issues_test.go`

## 2026-09-14 처분 — 제목의 뒤 절반만 닫혔다

이 카드의 제목은 "역방향 범위에서 id를 조용히 버리고 **그리고** 중복을 제거하지
않는다"였다. [[TASK-391]]이 닫은 것은 뒤 절반뿐이다.

**중복 제거 — 고쳤다.** `findEnumerations`가 `seen` 맵으로 id를 한 번만 모은다.
재현 `TestIssue017Deduplication`이 `knownbroken` 태그를 떼고 기본 스위트로
옮겨졌다. `review-planprogress`가 확인했다 — 원래 단언은 그대로 남고 거기에
`checkPlanProse` 단언이 **더해졌다**. 완화가 아니라 강화다.

**역방향 범위 — 안 닫혔다.** 오늘 `ca5157c` 위에서 직접 측정했다:

```
findEnumerations("TASK-9..1, 2")              -> []
findEnumerations("TASK-365..358, 366, 367")   -> []
```

두 번째가 이 결함의 진짜 모양이다. `366, 367`은 멀쩡한 멤버십 주장인데 앞의 잘못
쓰인 범위 하나 때문에 통째로 사라진다. 위 기준 2가 요구하는 것은 결정 **그리고**
`"reversed range is ignored"` 기대값의 갱신인데, 후자를 하지 않았으므로 기준을
체크하지 않는다. 측정 없이 미룬 것이 아니라, 측정한 결과를 근거와 함께
[[ISSUE-021]]로 옮겼다.

**위키링크 나열 — 안 닫혔다.** P5 probe를 그대로 돌렸다:

```
findEnumerations("[[TASK-1]], [[TASK-9]] 두 장이 남았다")  -> []
```

인식하지도, 건너뛴다고 알리지도 않는다. 보드 산문이 카드를 위키링크로 가리키는
것이 관례라는 점에서 이 침묵은 좁은 예외가 아니다. 같은 이유로 [[ISSUE-021]]에
C로 실었다.

**그래서 이 카드는 열려 있다.** 기계 기준 둘은 초록이고 사람 기준 둘은 붉다.
절반이 고쳐졌다고 전체를 닫으면 남은 절반이 초록 뒤로 사라진다.

## 2026-09-14 처분 2회차 — 역방향 범위 기준이 닫혔다, 위키링크 기준은 아니다

[[TASK-394]]가 [[ISSUE-021]] B·C를 처리하면서 이 카드의 남은 두 사람 기준 중
하나를 같이 닫았다.

**기준 2(역방향/과대 범위) 닫혔다.** 이 기준이 요구한 것은 고침이 아니라
**결정과 그 결정에 맞춘 기대값**이었다. 결정은 "보존"이다 — 범위는 확장하지 않되
뒤따르는 나열은 살린다. 근거와 반려한 두 대안은 [[ISSUE-021]]의 2026-09-14 처분
절에 적혀 있다. 테스트 이름도 결정에 맞게 바뀌었다: `"reversed range is ignored"`는
동작을 잘못 서술하는 이름이 됐으므로 `"reversed range contributes only its anchor"`가
됐고, 보존을 재는 케이스가 하나 추가됐다.

**기준 3(위키링크) 닫히지 않았다.** 이 기준은 두 갈래를 허용했다 — 인식하거나,
인식 못 하면 **그 사실을 보고하거나**. [[TASK-394]]는 어느 쪽도 하지 않았다.
`findEnumerations`가 위키링크 나열에 `nil`을 돌려준다는 사실을 **테스트로**
못박았을 뿐이고, `checkPlanProse`는 여전히 조용히 지나간다. 기준이 금지한 것이
바로 그 침묵이므로 이 칸은 빨갛게 둔다.

이 구분을 적어 두는 이유: 테스트로 고정하는 것과 사용자에게 보고하는 것은 둘 다
"알려진 결함"처럼 들리지만, 전자는 **우리가** 알고 후자는 **검사를 돌린 사람이**
안다. 이 기준이 요구한 것은 후자다.

## Related

- [[TASK-381]] — `findEnumerations`를 설계·구현한 작업, 커밋 `27c4dfe`. 본문이
  `[[TASK-385]]` 형태를 실제로 쓴다(F5b의 근거).
- [[PLAN-007]] — `[[TASK-381]]`/`[[TASK-388]]`로 같은 `[[TASK-N]]` 형태를
  쓴다(F5b의 근거).
- [[ISSUE-016]] — 같은 파일(`prose.go`)의 다른 함수(count/enumeration
  짝짓기)에 대한 별도 결함.
- [[ISSUE-018]] — 파싱 단계에서 검사가 조용히 꺼지는 같은 성질의 결함(F5b와
  같은 계열).
