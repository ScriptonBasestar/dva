---
id: TASK-395
title: "Record where an upstream-owned issue was reported"
type: chore
priority: P2
effort: M
exec-tier: strong
status: todo
created: 2026-09-14
source: "2026-09-14 보드 현행화 — 상류 소유 이슈 다수, 보고 흔적 0장"
depends-on: []
completion-summary: "상류 소유 이슈에 `upstream-ref:` 필드를 두고 값이 비어 있으면 doccheck가 세는 자리와 계량기를 만들었다(채널은 2026-09-15 ce-agent-kit 저장소 이슈로 확정). 24장 이슈 카드 전부에 `## 소유권` 절을 명시했고, 그중 상류·갈림 11장에 빈 `upstream-ref:`를 두어 계량기가 잡는다. 측정 결과 ownership_unmarked 0 / upstream_owned 15 / upstream_unref 15 — 게이트는 초록인 채로 미보고 상류 카드가 매 실행마다 보인다"
verification-status: verified
verification-evidence: "완료 기준 7개의 verify 바인딩 전부 bare 실행 exit 0 (09f681b). 기준 1: `## 소유권` 절 보유 24장 = 이슈 카드 24장, 파일별 정확히 1절. 기준 2–4: grep·TestUpstreamRef 세 축 테스트 존재. 기준 5: go test ./tools/doccheck/ 2.489s ok. 기준 6–7: make doc-check OK, GOSUMDB=sum.golang.org make lint 0 issues. ce task gate READY"
---

## Summary

이 보드의 이슈 칸은 **한쪽으로만 자란다.** 2026-09-14 기준 19장 중 다수가 이
저장소가 고칠 수 없는 결함이다 — 고치는 코드가 `ce-agent-kit`에 있다. 그리고
**어느 카드도 상류에 보고했다는 사실을 적고 있지 않다.**

```
$ /usr/bin/grep -lE 'upstream-ref|upstream-issue|보고 완료|상류에 보고' tasks/issue/*.md | wc -l
       0
```

이게 왜 문제냐면, 보고하지 않은 것과 보고했는데 안 고쳐진 것이 **보드에서 똑같이
보이기 때문이다.** 둘 다 `status: todo`이고 둘 다 게이트에서 초록이다. 구분이
없으면 "상류 소유"라는 표기가 사실상 면죄부가 된다 — 카드를 쌓아두는 이유가 되고,
아무도 그 더미가 실제로 상류에 닿았는지 묻지 않는다.

**해법은 필드 하나다.** 상류 소유 이슈 카드에 `upstream-ref:`를 두고, 값이 비어
있으면 문서 게이트가 그 카드를 센다. 그러면 "보고 안 됨"이 초록 뒤에 숨지 못한다.

### 채널은 열린 파라미터다

`upstream-ref:`가 **무엇을 가리키는지**는 이 카드가 정하지 않는다. 사용자가 셋 중
하나를 고른다:

| 채널 | `upstream-ref:` 값의 모양 | 특징 |
|---|---|---|
| `ce-agent-kit` 저장소 이슈 | `ce-agent-kit#123` | 상류가 직접 읽는다. 에이전트가 이슈를 열 수 있는지는 별개 확인 필요 |
| 별도 보고 보드 | 그 보드의 카드 ID | 이 저장소가 통제한다. 상류가 볼 보장은 없다 |
| 사용자 직접 전달 | `relayed:2026-09-14` 같은 날짜 표기 | 채널 비용 0. 추적성은 가장 약하다 |

검사는 **값의 존재**만 본다. 형식 검증은 채널이 정해진 뒤에 좁힌다 — 지금 형식을
못박으면 아직 고르지 않은 두 채널을 미리 배제하게 된다.

### 채널 결정 — 2026-09-15

**`ce-agent-kit` 저장소 이슈**로 정했다(`upstream-ref:` 값은 `ce-agent-kit#N`
형식). 근거: 상류가 직접 읽는 경로가 유일한 것은 이 채널이고, 나머지 둘(별도 보드,
사용자 전달)은 추적성을 이 저장소 밖 어딘가에 두는 방식이라 "보고했다"의 증거가
되지 못한다. 이 결정은 2026-09-15 권장안 승인("위 권장안대로 진행")에 따른
것이고, 형식 검증은 이 카드의 원칙대로 아직 좁히지 않는다 — 값 존재만 검사한다.

### 검사를 어디에 두는가 — 새 보드 게이트가 아니다

개인 정책은 공유 `ce task gate` 옆에 **저장소 로컬 보드 검사를 새로 두는 것**을
금지한다. 그래서 이 검사는 `tools/doccheck`에 붙인다. doccheck는 이미 카드
frontmatter를 읽고 zone 규칙(`status:`가 zone과 맞는지)을 판정하고 있다 — 같은
자리에 필드 규칙 하나를 더 얹는 것은 새 판정자를 만드는 것이 아니라 있는 판정자를
넓히는 것이다. `ce task lint`가 언젠가 같은 필드를 세게 되면 그때 이쪽을 걷어낸다.

## 이 카드가 하지 않는 것

- **실제 보고를 하지 않는다.** 채널이 정해지기 전에는 보고할 곳이 없고, 채널이
  정해진 뒤에도 보고 행위 자체는 별도 작업이다. 이 카드는 **자리와 계량기**만
  만든다.
- **어느 이슈가 상류 소유인지 판정하지 않는다.** 카드마다 이미 `## 소유권` 절이
  정본이다(19장 중 4장만 적혀 있다 — 나머지를 채우는 것이 아래 기준 1이다).
- **형식을 검증하지 않는다.** 위에 적은 이유로 존재만 본다.

## Completion Criteria

- [x] 상류 소유 여부가 모든 이슈 카드에 표기된다 | verify: human — `## 소유권`
      절이 있는 카드 수와 이슈 카드 총 수를 세어 둘이 같은지 확인한다
- [x] `upstream-ref:` frontmatter 필드가 상류 소유 카드에 존재한다 | verify: `/usr/bin/grep -rlq 'upstream-ref:' tasks/issue`
- [x] doccheck가 값 없는 상류 소유 카드를 센다 | verify: `/usr/bin/grep -rq 'upstream-ref' tools/doccheck`
- [x] 그 검사에 세 축 테스트가 있다 — 값 없으면 nonzero, 값 있으면 zero | verify: `/usr/bin/grep -rq 'TestUpstreamRef' tools/doccheck`
- [x] doccheck 패키지 테스트 전부 통과 | verify: `go test ./tools/doccheck/`
- [x] 문서 게이트가 초록 | verify: `make doc-check` (regression-guard)
- [x] 저장소 게이트 통과 | verify: `make lint` (regression-guard)

## Sources

- 2026-09-14 보드 현행화에서 측정한 보고 흔적 0장
- [[ISSUE-021]]의 `## 소유권 — 이 저장소다` 절 — 표기가 빠진 카드가 기본값으로
  상류 다수에 섞이는 문제를 이 카드가 구조로 막는다

## Related

- [[ISSUE-013]] · [[ISSUE-011]] · [[ISSUE-019]] — 상류 소유 이슈의 대표 사례
- [[TASK-394]] — 같은 날 등록된 자기 소유 결함 카드. 대비가 이 카드의 논지다
