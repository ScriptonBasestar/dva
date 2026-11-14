---
id: TASK-608
title: "CI 이벤트별 검사 계약과 단계적 채택 계획 문서화"
type: docs
priority: P2
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-608/independent-review.md
created: 2026-11-15
---

## Summary

사용자가 합의한 이벤트별 CI 설계를 정본 문서와 채택 계획으로 보존한다.
실행 순서·현재 지원·후속 완료 기준은 [채택 계획](../../docs/ci-lifecycle-adoption.md)이
소유한다. 기존 프로필과 게이트를 유지하고 스킬에는 조건부 정본 진입점만 연결한다.

## Steps

1. 기존 실행·재사용·hook·원격 CI 계약을 조사한다.
2. 이벤트별 검사와 현재 적용 현황, 후속 적용 인수 기준을 기록한다.
3. 문서와 스킬 진입점을 연결하고 생성물·링크를 확인한다.
4. 구현과 분리된 strong 이상 독립 리뷰 후 필수 검사를 수행한다.
5. 검증 결과를 보존하고 task branch push·source 통합·회수를 확인한다.

## Completion Criteria

- [x] 이벤트별 검사와 현재 지원의 경계를 정본에 기록한다 | verify: `/usr/bin/grep -q "## 이벤트와 검사 책임" docs/ci-lifecycle.md` (observed: 2026-11-15)
- [x] 채택 순서와 미구현 항목의 완료 기준을 보존한다 | verify: `/usr/bin/grep -q "## 실행 순서" docs/ci-lifecycle-adoption.md` (observed: 2026-11-15)
- [x] 모든 이벤트·예산·입력·프로필 의미를 합의 설계와 대조하고 기존 실행 설정 보존을 확인한다 | verify: human — 독립 리뷰·설정 diff와 문서 검사로 확인. 통합은 별도의 최종 full 게이트 통과 뒤에 한다

## Verification and continuation

- 문서 구현과 strong 독립 설계 리뷰: PASS (/root/ci_contract_review).
- make doc-check, make check-generate, make test-skill-dogfood: PASS.
- make lint: 작업 트리와 기준 master 모두 동일한 기존 두 오류로 FAIL.
  compare_test.go:745 modernize, compare_runner_instance_test.go:21 staticcheck.
- 사용자는 간단하면 수정하도록 허용했다. 두 테스트의 SplitSeq 전환과
  !windows 파일의 중복 Windows 분기·runtime import 제거를 적용했다.
- 수정 후 make lint: PASS (0 issues). DVA_BASE=origin/master dva ci dev: PASS,
  run 56664ec0f3952888b1050dfef16a34f7, 2m40.655s, 입력 변경 없음.
- 독립 리뷰 원문과 검증 범위는 위 quality-review-evidence에 보존한다.
- full·원격 CI·통합·설치는 이 카드의 구현 리뷰와 별도다. 최종 커밋의 full 성공과
  독립 통합 리뷰를 확인한 뒤 source에 통합하며 실행 receipt가 그 상태를 소유한다.
