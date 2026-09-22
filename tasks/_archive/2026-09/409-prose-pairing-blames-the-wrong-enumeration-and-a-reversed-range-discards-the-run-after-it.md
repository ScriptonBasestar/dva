---
id: TASK-409
title: "Prose pairing blames the wrong enumeration and a reversed range discards the run after it"
type: bug
priority: P2
status: done
archived-at: 2026-09-22
verified-at: 2026-09-22
verification-summary: "Re-verified 2026-09-22. C1 the prose.go file comment (b) states the nearest-neighbour rule, names the surviving false positive verbatim ('다섯 장 — TASK-1·2와 TASK-3·4·5' reporting 'counts 5 card(s) beside an enumeration naming 2'), and says why merging 와-joined enumerations would silence it at the cost of F3; TestIssue021PairingResidue pins both word orders (1 defect / 0 defects) and passes. C2 `go test ./tools/planprogress/` ok, `make doc-check` exit 0. No follow-up."
quality-review: pass
quality-reviewed-at: 2026-09-22
quality-review-evidence: "독립 리뷰 review-409(구현 세션 아님, 2026-09-22). 기준1(human): tools/planprogress/prose.go 파일 코멘트 (b)절이 서술하는 동작을 코드로 직접 재측정했다 — 패키지에 임시 프로브 테스트를 넣어 findCountedEnumerations와 checkPlanProse의 실제 반환값을 찍고 지웠다. children 1..5, goal '다섯 장 — TASK-1·2와 TASK-3·4·5' → 나열 둘이 잡히고 count 5는 가까운 TASK-1·2에만 붙는다(다른 쪽 hasCount=false), 결함 1건 '## Goal counts 5 card(s) beside an enumeration naming 2 (\"TASK-1·2\")'. 코멘트가 적은 문자열과 글자 단위로 같고, '오직 하나의 나열만 지목된다'는 대칭성 주장도 결함 1건으로 실측 확인됐다. F3 어순 'TASK-1·2와 TASK-3·4·5, 세 장이 겹친다'는 결함 0건 — 코멘트의 '와-결합 나열을 합치면 이 어순이 조용해지지만 F3가 깨진다'는 설명과 일치한다(대조 프로브 '…, 다섯 장이 겹친다'는 naming 3으로 정상 검출). 즉 오탐은 잔여가 아니라 nearest-neighbour 규칙의 명시된 의도이고, TestIssue021PairingResidue가 양 어순을 모두 고정한다. 기준2: `go test ./tools/planprogress/` ok, `make doc-check` exit 0 (doc-check: OK / cilabels OK / flowcheck OK / planprogress OK — plans_checked 4, task_cards_indexed 406 / yamlcheck 0 errors / changelogcheck OK; STALE 2건은 tasks/archive/2026-09/283 카드의 옛 경로 표기로 이 카드와 무관한 기존 경고다). blocks: 미선언 카드라 receipt는 요구되지 않는다."
effort: S
exec-tier: standard
created: 2026-09-21
---

## Summary

ISSUE-021의 남은 결함은 A뿐이다. 같은 문장에서 어순만 바꾸면 옳은 수량 주장이 오탐이 된다 (`다섯 장 — TASK-1·2와 TASK-3·4·5`, children 1..5). B(역방향 범위가 뒤를 삼킴)와 C(위키링크 침묵)는 이미 닫혔다 — 구현은 TASK-394, 위키링크 인식은 TASK-408이 별도로 다룬다.

이 카드는 짝짓기 규칙을 더 좁히거나, 오탐이 규칙의 의도임을 `tools/planprogress/prose.go` 파일 코멘트와 테스트로 못박는다. 정규식 한 줄로 "주어를 안다"고 우기지 않는다.

## Completion Criteria

- [x] 어순을 바꾼 반례가 더 이상 오탐을 내지 않거나, 왜 내는지가 규칙의 의도로 코드에 적혀 있다 | verify: human — `tools/planprogress/prose.go`의 파일 코멘트를 읽고, "다섯 장 — TASK-1·2와 TASK-3·4·5"에 대한 코드의 실제 동작이 코멘트가 서술하는 동작과 일치하는지 확인한다
- [x] 패키지 테스트와 문서 게이트가 통과한다 | verify: `go test ./tools/planprogress/ && make doc-check`

## Out of scope

- 역방향 범위 파서 (ISSUE-021 B, 이미 보존 방향으로 닫힘)
- `[[TASK-N]]` 위키링크 나열 인식 (ISSUE-021 C / TASK-408)
- `findCountPhrases`의 개/건 출력 (ISSUE-021 D, 결함 아님)

## Sources

- ISSUE-021 — tasks/issue/021-prose-pairing-blames-the-wrong-enumeration-and-a-reversed-range-discards-the-run-after-it.md
