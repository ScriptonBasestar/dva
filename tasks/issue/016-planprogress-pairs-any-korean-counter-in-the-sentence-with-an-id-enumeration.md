---
id: ISSUE-016
title: "planprogress pairs any Korean counter in the sentence with an id enumeration"
type: bug
status: todo
priority: P2
severity: medium
effort: S
exec-tier: standard
created: 2026-09-14
discovered-in: "review-381 independent review of TASK-381 (2026-09-14)"
discovered-at: 2026-09-14
source: "review-381의 conditional 판정, F1/F2/F3. 세 지적 모두 count/enumeration 짝짓기 규칙이라는 한 지점에서 나온다"
---

## Summary

`개`/`건`/`장`은 범용 한국어 수량 단위다. `tools/planprogress/prose.go`는 같은
문장 안에 id 나열과 이 단위 중 하나가 함께 있으면 무조건 "그 나열의 카드 수"로
읽는다(`countPhraseRE`, `nearestCount`). 세 가지 서로 다른 실패가 이 한 지점에서
나온다.

## Evidence

**F1 — 무관한 수량이 나열의 카드 수로 오인된다** (anchor `prose.go:56` `countPhraseRE`,
`:183` `nearestCount`, `:217-231` 짝짓기 후 비교). 측정된 오탐(현재 `HEAD`, 재현
가능):

- Goal `TASK-1·2는 check.go의 Result 구조체(필드 32개)를 고친다` →
  `## Goal counts 32 card(s) beside an enumeration of 2`
- scope `TASK-1, 2, 3 — check.go(19-51행, 필드 32개)` →
  `scope: counts 32 card(s) beside an enumeration of 3`
- scope `TASK-1, 2, 3 — 관련 문서 8건을 정리한다` →
  `scope: counts 8 card(s) beside an enumeration of 3`

세 문장 모두 이 카드와 함께 `tools/planprogress/known_issues_test.go`(빌드
태그 `knownbroken`)에 `TestIssue016FalsePositives`로 추가했다. 이 파일은
기본 빌드에서 제외되므로 `go test ./tools/planprogress/`는 여전히 초록이다 —
`go test -tags=knownbroken -run TestIssue016FalsePositives
./tools/planprogress/`를 돌리면 이 셋에서 FAIL한다:

```
--- FAIL: TestIssue016FalsePositives/a_struct_field_count_in_Goal_is_not_an_enumeration_count
    known_issues_test.go:63: checkPlanProse() = [PLAN-ISSUE-016-A (): ## Goal counts 32 card(s) beside an enumeration of 2 ("TASK-1·2")] (1 defects), want 0
--- FAIL: TestIssue016FalsePositives/a_struct_field_count_beside_a_line_range_in_scope_is_not_an_enumeration_count
    known_issues_test.go:63: checkPlanProse() = [PLAN-ISSUE-016-B (): scope: counts 32 card(s) beside an enumeration of 3 ("TASK-1, 2, 3")] (1 defects), want 0
--- FAIL: TestIssue016FalsePositives/a_document_count_in_scope_is_not_an_enumeration_count
    known_issues_test.go:63: checkPlanProse() = [PLAN-ISSUE-016-C (): scope: counts 8 card(s) beside an enumeration of 3 ("TASK-1, 2, 3")] (1 defects), want 0
```

(`prose_test.go` 자체는 손대지 않았다 — TASK-381의 리뷰 대상 파일을 리뷰 영수증과
같은 커밋에서 움직이지 않기 위해서다. 처음에는 이 픽스처를 `prose_test.go`에
직접 넣어 `go test ./tools/planprogress/`가 실패하도록 했으나, 그러면
`make test`/`make lint`가 전체 저장소 기준으로 빨간불이 되어 무관한 다른
task 브랜치의 integration까지 막는다는 문제가 있어 `knownbroken` 태그 뒤로
격리했다. `t.Skip`으로 조용히 건너뛰지 않은 이유도 같은 자리에 적는다 —
그러면 바인딩이 결함 유무와 무관하게 항상 exit 0이 되어, 애초에 피하려던
vacuous binding으로 되돌아간다. 아래 세 번째 Resolution Criterion이 이
격리가 영구화되지 않도록 한다.)

가장 날카로운 부분: **정확히 이 모양의 산문이 이미 살아있는 보드에 있다.**
`tasks/plan/009-work-the-doccheck-defect-bundle-in-dependency-order.md:31-32`는
"필드 32개", "19-51행"을 쓰고, `tasks/plan/007-done-backlog-triage.md:5`의 scope는
"결함 3건"을 쓴다. 이 두 카드가 오늘 통과하는 이유는 오직 그 수사와 나열이 우연히
같은 문장을 공유하지 않기 때문이다(PLAN-009: "다섯 장"과 나열은 첫 문장, "필드
32개"는 다음 문장; PLAN-007: "결함 3건"이 있는 문장에는 애초에 id 나열이 없다).
**지금의 초록은 규칙의 속성이 아니라 줄바꿈의 우연이다.** `prose.go`의 파일
코멘트 자신이 "PLAN-007의 '결함 3건', PLAN-006의 '23개 devbox 저장소'는 전혀
다른 것을 센다"고 이 케이스를 이미 알고 회피를 주장하는데, 실제로는 회피하지
못한다 — 문장 경계 하나에 기대고 있을 뿐이다.

review-381의 제안(결정이 아니라 단서로만 기록): 세 단위 중 이 코퍼스에서 실제로
카드를 세는 것은 `장`뿐이므로 `개`/`건`을 빼거나, count phrase가 괄호/백틱 구간
밖에 있을 것을 요구한다.

**F2 — "그리고 나머지"형 부분 나열이 거부된다** (anchor `prose.go:231`
`compareTotal` 분기), severity medium. count phrase 없이 부분만 나열하고 "나머지는
…"으로 잇는 scope는 지금은 통과한다 — 하지만 그 이유가 우연이다.
`tasks/plan/006-devbox-dogfood-followup.md:5`의 실제 scope는
`TASK-311..323 …, plus the needs-human cards that gate the rest`이고
`total-tasks: 29`다. 13개짜리 범위 나열 + "나머지" 산문인데, count phrase가
없어서만 걸리지 않는다. 이 문장에 `13장`을 더하면 — 옳은 카드가 CI에서
떨어진다. TASK-381의 `## 범위 재검토`는 "scope는 정의상 계획 전체를 기술한다"고
전제하는데, PLAN-006이 바로 그 반례와 같은 디렉터리에 있다.

**F3 — 가장 가까운 나열에만 짝짓지 않는다** (anchor `prose.go:74-89`
`findCountedEnumerations`의 순회, `:183` `nearestCount`), severity low.
`nearestCount`는 문장 안의 모든 나열에 같은 count phrase를 준다. 측정:
Goal `TASK-1·2와 TASK-3·4·5, 세 장이 겹친다` → 먼 나열에 오탐,
`counts 3 card(s) beside an enumeration of 2`. 호출자는 이 나열이 그 count의
"가장 가까운 이웃"인지 확인하지 않는다 — 대칭 짝짓기(서로가 서로의 최근접일
때만 짝짓기)로 이 부류가 사라진다.

## Reproduction

1. 이 카드와 함께 추가된 `tools/planprogress/known_issues_test.go`(빌드 태그
   `knownbroken`)의 `TestIssue016FalsePositives`를 확인한다.
2. `go test -tags=knownbroken -run TestIssue016FalsePositives
   ./tools/planprogress/ -v`를 실행한다 — 위 세 서브테스트가 FAIL한다.
3. 태그 없이 `go test ./tools/planprogress/`를 실행한다 — 이 파일은 기본
   빌드에서 제외되므로 초록으로 통과한다(격리가 의도대로 작동함을 확인).
4. `tasks/plan/009-*.md`와 `tasks/plan/007-*.md`의 해당 줄이 오늘 통과하는 것은
   각주에 적은 문장 경계 때문임을 육안으로 확인한다.

## Expected vs Actual

- Expected: count/enumeration 짝짓기가 실제로 카드를 세는 수량만 나열과 묶고,
  구조체 필드 수·줄 범위·문서 건수 같은 무관한 수량과는 묶이지 않는다. "나머지"형
  부분 나열도 옳게 통과한다.
- Actual: 문장 공유 여부만으로 짝짓고, 그 안에서 어느 나열이 진짜 이웃인지도
  가리지 않는다.

## Impact

지금 당장 살아있는 카드를 잘못 떨어뜨리지는 않는다 — 우연히 문장이 갈라져
있어서다. 그러나 그 우연은 다음 편집 한 번으로 깨진다: PLAN-009나 PLAN-007의
산문을 다듬다가 문장을 합치거나 수량을 옮기는 순간, 옳은 카드가 이유 없이
CI에서 떨어진다. `prose.go`의 설계 의도(파일 코멘트)와 실제 동작이 이미
어긋나 있다는 것 자체가, 이 규칙을 처음 읽는 사람이 안전하다고 믿을 근거가
없다는 뜻이다.

## 소유권 — 이 저장소다 (2026-09-14 명시)

세 카드(ISSUE-016·017·018)는 `review-381`의 독립 리뷰에서 나왔고, 같은 리뷰가 낸 다른
이슈들이 대부분 `ce-agent-kit`·`ce-workbook` 소유라 서술 형식이 상류 이슈와 구분되지
않았다. **이 셋은 상류가 아니다** — 결함은 전부 이 저장소의 `tools/planprogress` 코드
안에 있고, 이 저장소가 고칠 수 있고 고쳐야 한다.

구분이 중요한 이유: 상류 소유 이슈는 이 보드가 진척을 강제할 수 없어 열린 채 남는 것이
정상이지만(PLAN-007 §External의 선례), 자기 소유 결함을 같은 칸에 넣어 두면 고칠 수 있는
일이 영원히 대기열에 남는다.

## Resolution Criteria

- [ ] `tools/planprogress/known_issues_test.go`의 `TestIssue016FalsePositives`
      (F1의 세 문장)가 통과한다 — 카운터 어휘 또는 짝짓기 규칙이 좁혀졌다는
      뜻이다. 이 바인딩은 구성상 지금은 실패하며, 결함이 고쳐졌을 때만
      통과한다. 명령이 `-v`와 PASS 줄 grep을 거치는 이유는 `go test -run`이
      0건 매치일 때 `[no tests to run]`과 함께 exit 0을 내기 때문이다 — 재현을
      지우는 것으로는 이 기준을 만족시킬 수 없다 | verify: `sh -c 'go test
      -tags=knownbroken -run "^TestIssue016FalsePositives$" -v ./tools/planprogress/ 2>&1 | /usr/bin/grep -qE
      "^--- PASS: TestIssue016FalsePositives "'`
- [ ] 좁히는 방식(카운터 어휘를 `장`으로만 제한, 괄호/백틱 구간 제외, 또는
      다른 방식)이 결정되고 `prose.go`의 파일 코멘트가 그 결정과 실제 동작을
      정확히 반영한다 | verify: human — 코멘트를 읽고 F1의 세 문장 각각에 대해
      코멘트가 서술하는 동작과 코드 동작이 일치하는지 확인
- [ ] F2(부분 나열 + "나머지")와 F3(다중 나열 중 최근접이 아닌 것과의 오짝짓기)가
      이 카드 안에서 각각 받아들여지거나 반려된 근거와 함께 처리된다 | verify:
      human — 두 지적 각각에 대해 이 카드 또는 후속 카드에 결정과 근거가
      남아있는지 확인
- [ ] 이 결함의 재현이 기본 테스트 스위트로 옮겨졌다 — `TestIssue016FalsePositives`가
      `prose_test.go`에 있고 `known_issues_test.go`에는 남아 있지 않다(태그를
      지우는 것만으로는 만족되지 않는다) | verify: `/usr/bin/grep -q 'TestIssue016FalsePositives'
      tools/planprogress/prose_test.go && ! /usr/bin/grep -q 'TestIssue016FalsePositives'
      tools/planprogress/known_issues_test.go`

## Related

- [[TASK-381]] — 이 규칙을 설계·구현한 작업, 커밋 `27c4dfe`.
- [[TASK-380]] — 애초에 이 검사가 필요해진 D-2 발견.
