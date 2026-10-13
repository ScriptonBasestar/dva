---
id: TASK-506
title: "Reject commit message bodies without a blank separator"
type: bug
priority: P2
effort: XS
exec-tier: standard
status: done
created: 2026-10-14
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-506/independent-review.json
archived-at: 2026-10-14
---

## Summary

commit-msg는 첫 물리적 줄만 검사하지만 history gate는 Git subject인 첫 문단을
검사한다. 제목·본문 사이 빈 줄이 빠진 4개 공개 메시지가 hook을 통과하고 history에서
실패했다. 기존 history 검사와 72자 상한을 유지하면서 hook에서 이 형식을 거부한다.
공개된 이력의 해결은 ISSUE-512/결정 513에 남긴다.

## Scope

- tools/commitcheck/main.go, main_test.go; 관련 CHANGELOG 및 이 카드.
- 구현 standard, 독립 리뷰 strong. history rewrite·waiver·baseline 변경은 제외한다.
- 제목만 있는 메시지, 정상 분리된 본문, 기존 주석 및 generated merge/revert 예외는 유지한다.

## Evidence

- CE 생성은 미래 테스트가 없어서 empty_test_match를 보고했다. 카드 경로/ID를 보존하고 회귀 테스트를 추가한 뒤 재검증한다. 이 생성 결과를 테스트 PASS로 쓰지 않는다.
- 최초 owner package `go test ./tools/commitcheck -count=1` PASS(0.421s).
- 독립 실제 diff 리뷰·generated 예외 보존 및 전달 결과는 완료 뒤 갱신한다.
- generated merge/revert 예외가 새 separator 검사보다 먼저 적용되도록 보완했다.
  최종 owner package 전체 테스트 PASS(0.423s), strong 독립 actual diff 리뷰 PASS다.
- history scanner·72자 ceiling·baseline·역사 예외 목록은 변경하지 않았다.
- 코드 commit 562cd8c2 push 후 history 검사에서 기존 4건만 남았다(exit 1,
  968 commits/4 violations). 새로운 source 변경이 baseline을 악화시키지 않았다.
  전체 full CI를 PASS로 표시하지 않으며 ISSUE-512/결정 513은 미해결이다.

## Completion Criteria

- [x] 커밋 hook이 이력 subject 검사와 같은 문단 경계를 지킨다. | verify: `go test ./tools/commitcheck -run TestCheckMessageFileRejectsUnseparatedBody -count=1`
