---
id: TASK-601
title: "compare 락 순서 고정과 문구 보완"
type: feature
priority: P2
status: done
quality-review: pass
quality-reviewed-at: 2026-11-11T09:00:00Z
quality-review-evidence: tasks/done/evidence/TASK-601/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-601/done-review-fe81ab7e074861e85e08ba34ead7d9e0abd07ba7d4ca51482c522eea703a4e7e.json
created: 2026-11-11
---

## Summary

TASK-600 리뷰에서 나온 낮은 심각도 후속이다. (1) `compare down`의 락이 기록 읽기보다 먼저 잡혀야 한다는 순서를 고정하는 테스트가 없어, 락을 `ReadRecord` 뒤로 옮기는 변이가 살아남는다. (2) `compare up`의 거절 문구가 아직 "another `dva compare up`"이라 락을 쥔 쪽이 down일 때 안내가 틀린다. (3) TASK-599의 코드 주석과 카드가 EPERM 비율을 약 0.65%로 적었으나 리뷰어 실측은 부하에 따라 8~40%였다. 문구와 서술을 고치고 순서 테스트를 추가한다.

## Completion Criteria

- [x] 락을 기록 읽기보다 먼저 잡는 순서를 고정하는 테스트가 있고 compare up의 거절 문구가 down이 락을 쥔 경우에도 맞다 | verify: `go test ./internal/cli/ -run '^TestCompare(DownTakesTheLockBeforeReadingTheRecord|UpRefusesWhileAnotherUpHoldsTheLock)$' -count=1`

## Review Attempts

### 1. 2026-11-11 — reviewer: claude-code-subagent (독립, read-only), executor: claude-opus-5-5

- 대상: 11a96804. 판정: pass. 기준 1 PASS: 검증 바인딩이 통과했고, 락을 `ReadRecord` 뒤로 옮긴 변이는 새 테스트가, 거절 문구를 되돌린 변이는 up 테스트가 잡았다. Summary 항목 3(EPERM 주석 0.65%를 8~40%로 정정) PASS이며 done 카드 599는 바뀌지 않았다. go vet, `GOOS=windows go vet`, `go test` cli·compare `-count=2`가 통과했다.
- 비차단: 실제 down이 락을 쥔 상태에서 up을 막는지 보는 테스트는 없다. 주석 두 줄이 줄바꿈 폭을 약간 넘는다. 다음 전환: done/.
