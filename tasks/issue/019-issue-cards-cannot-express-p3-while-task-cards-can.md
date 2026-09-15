---
id: ISSUE-019
title: "Issue cards cannot express P3 while task cards can"
type: bug
status: todo
priority: P2
severity: low
ownership: upstream
created: 2026-09-14
discovered-in: "ISSUE-005 P1 강등 (2026-09-14, ce-agent-kit 0.8.4)"
discovered-at: 2026-09-14
source: "ISSUE-005를 P1에서 내리려다 validate가 P3를 거부"
upstream-ref: "ce-agent-kit#3"
---

## Summary

`ce task validate`는 issue 카드의 `priority:`를 **P0·P1·P2**만 허용하는데, task
카드는 P3를 허용한다. 그래서 "더 이상 급하지 않지만 여전히 결함"인 이슈를 표현할
자리가 없다 — 가장 낮은 등급이 P2고, P2는 이 보드에서 "곧 처리" 구간이다.

상류 규칙(ce-agent-kit `internal/usecase/task/canonical_validator.go`,
`validateCanonicalIssue`):

```go
canonicalEnumRule{name: "priority", allowed: []string{"P0", "P1", "P2"}}
```

## Reproduction

1. 아무 issue 카드의 frontmatter를 `priority: P3`로 바꾼다.
2. `ce task validate <그 카드>` → `❌ Invalid priority: P3`, exit 1.
3. 같은 값을 task 카드에 쓰면 통과한다(이 보드의 todo 존에 P3 카드가 둘 있다).

## Expected vs Actual

- Expected: issue와 task가 같은 priority 어휘를 쓰거나, issue가 P3를 거부하는
  이유가 스키마에 명시된다.
- Actual: 같은 `priority:` 필드가 kind에 따라 다른 값 집합을 갖고, 그 차이가
  거부 메시지에만 나타난다.

## Impact

낮다. 우회는 `severity:`로 실제 무게를 적는 것이고 ISSUE-005가 그렇게 했다. 다만
우회가 자리잡으면 priority와 severity가 서로 다른 축을 재게 되어, 나중에
priority로 정렬한 큐가 실제 급함과 어긋난다. **잘못된 값이 아니라 없는 값이
문제다** — 없는 칸은 옆 칸으로 새고, 새어 들어간 칸은 원래 재던 것을 못 재게 된다.

## 소유권 — 상류다 (2026-09-15 명시)

규칙은 ce-agent-kit이 소유한다. DVA는 이 관측을 잃지 않도록 카드만 세운다
([[ISSUE-004]] 선례). 여기서 로컬 우회 검사를 만들지 않는다.

## Resolution Criteria

- [ ] 상류가 issue 카드에 P3를 허용하거나, 거부 이유를 스키마 문서에 남긴다.
      | verify: human — 상류 릴리스 노트나 `canonical_validator.go`의 해당 enum
      규칙을 읽어 확인한다. 이 저장소에서 기계로 잴 수 있는 대상이 아니다.
- [ ] 그때까지 이 보드는 severity로 무게를 적고 priority는 P0–P2에 머문다.
      | verify: `! /usr/bin/grep -lE '^priority: P3$' tasks/issue/*.md`

## Related

- [[ISSUE-005]] — 이 제약을 처음 밟은 카드. P1 → P2 + severity low로 내렸다.
- [[ISSUE-011]] — 같은 계열: 상류 검증기의 어휘·조언이 이 저장소의 다른 규칙과
  어긋나는 자리.
