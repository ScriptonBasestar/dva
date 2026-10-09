---
id: TASK-502
title: "Decide the docs/ reading index and numbering convention"
type: decision
priority: P3
effort: S
needs-human: true
execution-mode: decision
human-grade: human
status: Proposed
created: 2026-10-10
---

## Summary

`docs/` 설계 문서에는 읽는 순서를 알려 주는 색인이 없다. 번호 체계도 선언되지 않았다.
에이전트와 개발자가 핵심 문서 다음에 어떤 설계 문서를 읽을지 정해야 한다.
색인을 어느 문서가 소유할지도 함께 정한다.

## Context

2026-10-10 문서 정합성 검토에서 확인했다.

- `docs/README.md`는 없다. [AGENTS.md](../../AGENTS.md)의 Documentation Ownership 표는
  루트 문서의 읽기 순서만 정한다.
- 핵심 문서가 링크하는 설계 문서는 일부뿐이다. ARCHITECTURE는 30·31·40·43,
  PRODUCT는 30·31·40을 링크한다. 57~61 등은 핵심 문서 어디에서도 닿지 않는다.
- `53-ci-profiles.md`와 `53-command-surface-agent-execution.md`가 번호 53을 공유한다.
  기계 색인은 없어 깨지는 것은 없다. 대신 "docs/53"이라는 말이 모호하다.
- `406-`·`407-`은 카드 ID를 접두어로 쓴다. 정렬하면 40과 41 사이에 놓인다.
- 44~49와 66은 비어 있다. 번호의 의미(주제 대역 또는 작성 순서)는 어디에도 적혀 있지 않다.

## Decision

- Status: Proposed

선택지:

1. **`docs/README.md` 색인 신설**: 주제별 묶음과 읽는 순서, 역사 문서 표시를 둔다.
   AGENTS.md는 그 색인에 한 줄로 링크한다. 번호는 그대로 둔다.
2. **1 + 번호 규약**: 1에 더해 53 중복과 406/407을 새 번호로 옮긴다. 인바운드 링크
   4곳 이상을 함께 고친다. 번호 대역 규약을 색인에 적는다.
3. **현행 유지**: 핵심 문서의 개별 링크만 쓴다. 고아 문서는 카드에서만 찾는다.

## Rationale

검토자 권고는 1이다. 링크를 깨지 않으면서 읽기 순서 누락을 메운다. 번호 변경은
이득보다 링크 갱신 비용이 크다. 다만 색인은 Documentation Ownership에 새 소유 문서를
더하는 결정이라 사람이 정한다.

## Consequences

- 1·2는 색인 소유 문서를 하나 더한다. Documentation Ownership 목록에 추가해야 한다.
- 2는 파일 이동이므로 `make doc-check`로 링크를 다시 확인해야 한다. 카드의 verify
  바인딩이 docs 경로를 직접 가리키는지도 함께 확인해야 한다.
- 3은 비용이 없다. 대신 설계 문서 발견을 검색에 맡긴다.

## Resolution Criteria

- [ ] 사람이 색인 신설, 색인과 번호 규약, 현행 유지 중 하나를 고르고 색인의 소유 문서를 적는다 | verify: human — choice recorded on this card before any docs/ file is added or renamed
