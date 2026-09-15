---
id: ISSUE-018
title: "Three parse-layer conditions silently switch the plan prose checks off"
type: bug
status: todo
priority: P2
severity: low
effort: S
exec-tier: standard
ownership: local
created: 2026-09-14
discovered-in: "review-381 independent review of TASK-381 (2026-09-14)"
discovered-at: 2026-09-14
source: "review-381의 conditional 판정, F6. 세 조건 모두 파싱 단계에서 조용히 검사를 끄는 같은 성질을 공유한다"
---

## Summary

`tools/planprogress/check.go`의 세 조건이, 카드가 흔히 쓸 법한 형태를 만나면
[[ISSUE-016]]/[[ISSUE-017]]이 다루는 산문 검사 자체를 **오류 없이 조용히
꺼버린다.** 검사가 실패하는 것보다 나쁘다 — 카드는 여전히 초록으로 보인다.

## Evidence

**(1) 접힌/블록 `scope: >`가 통째로 리터럴 `">"`가 된다** (anchor `check.go:186-187`
`case "scope": p.scope = strings.Trim(val, ...)`). frontmatter 한 줄 파서
(`frontmatterFieldRE`, `^([A-Za-z][A-Za-z0-9_-]*):\s*(.*)$`)는 `scope:` 뒤에
남은 한 줄만 값으로 잡는다. YAML의 `>`(folded)나 `|`(literal) 블록 스칼라처럼
다음 줄부터 들여쓰기로 이어지는 값은 지원하지 않고, `scope`에 문자 그대로
`">"` 하나만 들어간다. 오류는 나지 않는다 — scope 검사 둘 다 조용히 스킵된다.

측정:
```go
data := `---
id: PLAN-F6A
scope: >
  TASK-1, 2, 3 folded scalar body
children: [TASK-1, TASK-2, TASK-3]
total-tasks: 3
---
## Goal (2026)
body under a decorated heading
`
p, err := parsePlan(data)
// err == nil, p.scope == ">"
```

이것이 이 카드에서 가장 걸리기 쉬운 지점이다: `>`는 긴 scope 한 줄을 감쌀 때
쓰는 평범한 YAML이고, 지금 보드의 모든 살아있는 scope는 이미 그러고 싶을
만큼 길다.

**(2) 꾸며진 heading이 Goal을 통째로 비운다** (anchor `check.go:229`
`if trimmed == heading`, 호출부 `:214` `extractSection(lines[end+1:], "## Goal")`).
`extractSection`은 heading을 **문자열 완전 일치**로만 찾는다. `## Goal (2026)`처럼
같은 heading에 뭔가 덧붙이면 `"## Goal (2026)" != "## Goal"`이라 `in` 상태에
한 번도 들어가지 못하고, `p.goal`은 빈 문자열이 된다. Goal 검사가 조용히
스킵된다.

측정: 위 픽스처의 `p.goal == ""` (heading이 `## Goal (2026)`였다).

**(3) h1이 Goal 섹션을 끝내지 못한다** (anchor `check.go:226`
`if in && strings.HasPrefix(trimmed, "## ")`). 종료 조건이 리터럴 `"## "`
접두사 하나뿐이라, `"# Appendix"`(h1)처럼 **더 높은** 레벨의 heading이 와도
멈추지 않는다. `extractSection`의 자기 doc 코멘트(`check.go:218-219`)는 "ending
at the next heading of **the same or a higher level**"이라고 주장하는데,
코드는 `## ` 접두사만 검사하므로 이 주장은 거짓이다.

측정:
```go
body := []string{"## Goal", "", "first paragraph.", "", "# Appendix", "", "should this be excluded from Goal?"}
got := extractSection(body, "## Goal")
// got == "first paragraph.\n\n# Appendix\n\nshould this be excluded from Goal?"
```
h1 이후 텍스트가 Goal 본문에 그대로 섞여 들어간다.

세 측정 모두 위 스크래치 코드 그대로 재현되며 지금 `HEAD`(TASK-381 commit
`27c4dfe`)에서 참이다.

## Reproduction

1. `scope:`를 YAML 접힌 스칼라(`>`)로 쓴 plan 카드를 `parsePlan`에 준다 —
   `p.scope == ">"`이고 `err == nil`이다.
2. `## Goal (2026)`처럼 괄호가 덧붙은 heading을 쓴 카드를 준다 — `p.goal`이
   빈 문자열이 된다.
3. `## Goal` 섹션 다음에 `# Appendix`(h1)를 두고 `extractSection`을 호출한다 —
   Appendix 본문까지 Goal에 포함된다.

## Expected vs Actual

- Expected: 이 세 형태 중 하나라도 실제로 지원하거나, 지원하지 못하면 그
  카드에 대해 크게(loud) 실패한다 — 최소한 검사가 스킵됐다는 신호가 있어야
  한다.
- Actual: 셋 다 오류 없이 검사를 끈다. 카드는 계속 초록이다.

## Impact

세 조건 각각은 해당 카드에 대해 검사 전체를 끄면서 **아무 진단도 남기지
않는다.** 조용히 측정을 멈추는 게이트는 실패하는 게이트보다 나쁘다 — 카드가
여전히 초록으로 보이기 때문이다. (1)이 가장 걸리기 쉽다: 긴 scope를 `>`로
감싸는 것은 자연스러운 다음 편집이고, 그 순간 이 카드가 검사하는 모든 규칙이
그 카드에서 조용히 사라진다.

## 소유권 — 이 저장소다 (2026-09-14 명시)

세 카드(ISSUE-016·017·018)는 `review-381`의 독립 리뷰에서 나왔고, 같은 리뷰가 낸 다른
이슈들이 대부분 `ce-agent-kit`·`ce-workbook` 소유라 서술 형식이 상류 이슈와 구분되지
않았다. **이 셋은 상류가 아니다** — 결함은 전부 이 저장소의 `tools/planprogress` 코드
안에 있고, 이 저장소가 고칠 수 있고 고쳐야 한다.

구분이 중요한 이유: 상류 소유 이슈는 이 보드가 진척을 강제할 수 없어 열린 채 남는 것이
정상이지만(PLAN-007 §External의 선례), 자기 소유 결함을 같은 칸에 넣어 두면 고칠 수 있는
일이 영원히 대기열에 남는다.

## Resolution Criteria

- [ ] `scope: >`/`scope: |` 같은 YAML 블록 스칼라를 파싱하거나, 파싱하지
      못하는 경우 그 카드에 대해 명시적으로 실패/경고한다 (조용히 `">"`로
      떨어뜨리지 않는다) | verify: human — 접힌 스칼라 scope를 가진 픽스처로
      `parsePlan`을 호출해 오류 또는 올바른 파싱 중 하나가 나오는지 확인
- [ ] `extractSection`의 heading 매칭이 접두사 또는 정규화된 비교를 쓰거나,
      정확히 일치하지 않는 heading에 대해 명시적으로 알린다 | verify: human —
      `## Goal (2026)` 같은 heading을 가진 카드에서 Goal 검사가 스킵되지
      않거나 스킵 사실이 보고되는지 확인
- [ ] `extractSection`이 실제로 "같은 레벨 이상"에서 멈추도록 고쳐지거나, doc
      코멘트가 실제 동작(`## ` 리터럴 접두사에서만 멈춘다)에 맞게 정정된다 |
      verify: human — 코멘트와 h1-이후 픽스처의 결과가 서로 정합적인지 확인

## Related

- [[TASK-381]] — 이 파서와 검사를 도입한 작업, 커밋 `27c4dfe`.
- [[ISSUE-016]] — 이 카드가 조용히 꺼버리는 scope/Goal 검사 그 자체의 결함.
- [[ISSUE-017]] — 같은 파일의 나열 파싱 결함.
