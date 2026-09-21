---
id: TASK-409
title: "Prose pairing blames the wrong enumeration and a reversed range discards the run after it"
type: bug
priority: P2
status: todo
effort: S
exec-tier: standard
created: 2026-09-21
---

## Summary

ISSUE-021의 남은 결함은 A뿐이다. 같은 문장에서 어순만 바꾸면 옳은 수량 주장이 오탐이 된다 (`다섯 장 — TASK-1·2와 TASK-3·4·5`, children 1..5). B(역방향 범위가 뒤를 삼킴)와 C(위키링크 침묵)는 이미 닫혔다 — 구현은 TASK-394, 위키링크 인식은 TASK-408이 별도로 다룬다.

이 카드는 짝짓기 규칙을 더 좁히거나, 오탐이 규칙의 의도임을 `tools/planprogress/prose.go` 파일 코멘트와 테스트로 못박는다. 정규식 한 줄로 "주어를 안다"고 우기지 않는다.

## Completion Criteria

- [ ] 어순을 바꾼 반례가 더 이상 오탐을 내지 않거나, 왜 내는지가 규칙의 의도로 코드에 적혀 있다 | verify: human — `tools/planprogress/prose.go`의 파일 코멘트를 읽고, "다섯 장 — TASK-1·2와 TASK-3·4·5"에 대한 코드의 실제 동작이 코멘트가 서술하는 동작과 일치하는지 확인한다
- [ ] 패키지 테스트와 문서 게이트가 통과한다 | verify: `go test ./tools/planprogress/ && make doc-check`

## Out of scope

- 역방향 범위 파서 (ISSUE-021 B, 이미 보존 방향으로 닫힘)
- `[[TASK-N]]` 위키링크 나열 인식 (ISSUE-021 C / TASK-408)
- `findCountPhrases`의 개/건 출력 (ISSUE-021 D, 결함 아님)

## Sources

- ISSUE-021 — tasks/issue/021-prose-pairing-blames-the-wrong-enumeration-and-a-reversed-range-discards-the-run-after-it.md
