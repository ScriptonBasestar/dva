---
id: TASK-406
title: "A receipt-sealed done card has no correction path for a factual error in its own record"
type: feature
priority: P2
status: done
archived-at: 2026-09-22
verified-at: 2026-09-22
verification-summary: "Re-verified 2026-09-22. C1 docs/406-sealed-card-correction.md §2 enumerates forgery (copying the validate error digest, backdating reviewed-at, editing a past receipt) and §3 puts the formal path in the opposite order — edit first and let the seal break, independent re-run, then a new receipt beside the old one. C2 TASK-376 evidence no longer claims every Low was fixed and names the unapplied half (`:441/:446` EXIT trap, deferred to TASK-387). Fixed on archiving: §4 said `tasks/archive/` after the TASK-411 fold; repointed to `tasks/_archive/`. No follow-up."
quality-review: pass
quality-reviewed-at: 2026-09-22
quality-review-evidence: "독립 리뷰 review-406(구현 세션 아님, 2026-09-22). 기준 둘 다 human이라 직접 읽고 확인했다. (1) docs/406-sealed-card-correction.md는 §2 'Forgery (forbidden)'에서 위조를 명시적으로 열거하고(validate 오류 문자열의 sha256 복사, reviewed-at 백데이트, 과거 receipt 본문 편집), §3에서 정식 경로를 그와 대조되는 순서로 못박는다 — 편집이 먼저 봉인을 깨게 두고(step 1 'Do not mint a digest to silence that'), 독립 리뷰어가 기준을 재실행한 뒤(step 2·3) 비로소 새 receipt를 발급하고 옛 receipt는 남긴다(step 4). 구분되는 지점은 훈계가 아니라 산출물이다: digest만 갱신한 위조는 receipt가 한 장이고, 정식 경로는 두 장 + 다른 리뷰어·날짜가 남는다. §4가 archive 경로를 따로 갈라 '편집해도 done receipt 위조가 아니다'라고 적은 것도 확인했고, 실측으로 뒷받침된다 — `ce task validate tasks/archive/2026-09/376-….md`는 'Skipped: archived documents are kept as history'로 digest 검사에 도달하지 않는다(exit 0). (2) TASK-376의 quality-review-evidence에서 '전부 수정했다'는 사라졌다. 현재 문장은 'Low 5건 중 후반부(`:394` 대조군 캐시)는 같은 브랜치에서 수정했고, 전반부(`:441/:446` 로그·미리보기 temp 파일 trap 부재)는 적용하지 않았다 — EXIT trap이 emit_report()가 인용하는 STEP_LOG_FILE을 지워 리포트를 깨뜨리기 때문 (TASK-387)'이다. 미적용 한 건이 어느 지적인지(`:441/:446`)와 왜 미적용인지가 한 문장 안에서 읽힌다. blocks: 미선언 카드라 receipt는 요구되지 않는다."
created: 2026-09-17
---

## Summary

`blocks:`를 선언한 done 카드는 `quality-review-receipt`의 sha256 digest로 봉인되어 카드 내용 수정 시 게이트가 실패한다. 사후에 발견된 단순 사실관계 오류(예: TASK-376의 리뷰 지적 반영 범위 서술)를 위조 없이 정식으로 정정할 수 있는 절차를 정립하고, TASK-376의 기록을 정정한다.

## Completion Criteria

- [x] 봉인된 done 카드의 사실관계 정정 절차가 문서로 존재한다 | verify: human — 절차 문서를 읽고, 위조(리뷰 없이 digest 갱신)와 구분되는 지점이 명시돼 있는지 확인
- [x] TASK-376의 `quality-review-evidence`가 Low 한 건의 전반부 미적용을 반영한다 | verify: human — 376의 quality-review-evidence를 읽고, '전부 수정했다'가 사라졌는지와 미적용 한 건이 어느 지적인지 문장 안에서 읽히는지 확인

## Sources

- ISSUE-010 — tasks/issue/010-a-sealed-done-card-has-no-correction-path-for-a-factual-error.md
