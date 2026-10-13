---
id: ISSUE-507
title: "Record baseline documentation gate failures during verification skill adoption"
type: bug
priority: P3
status: todo
severity: medium
ownership: local
discovered-in: "Record baseline documentation gate failures during verification skill adoption"
discovered-at: 2026-10-13
created: 2026-10-13
---

## Summary

검증 스킬 적용 중 `make doc-check`가 기존 문서 링크 2건과 ISSUE-506의
ownership 누락으로 실패했다. 기준 커밋 `9c40d6ef992b3cba8507402ce00ff8680e9cf9af`의
primary checkout과 이번 스킬 작업트리에서 같은 세 결함을 확인했다.
이번 변경은 해당 문서나 ISSUE-506을 수정하지 않는다.

## 소유권 — 이 저장소다

DVA 저장소가 보고서의 문서 링크, 이슈 카드 메타데이터와 문서 게이트를 소유하므로
`ownership: local`이다. 이 카드는 확인한 기존 게이트 결함의 증거를 보존하며,
등록 자체는 해당 보고서나 ISSUE-506의 수정 범위를 확대하지 않는다.

## Reproduction

저장소 루트에서 `make doc-check`를 실행한다. 두 실행 모두 종료 코드 2이며
doccheck 내부 종료 코드는 1이었다. 확인한 진단:

- `reports/mydevbox2/plan.md`와 `reports/mydevbox2/results.md`의 링크가
  `reports/mydevbox2/findings.md`에 없는 F-10 anchor를 가리킨다.
- `tasks/issue/506-applyenv-test-fails-when-the-machine-exports-db-host.md`에
  usable ownership 값이 없다. 이 카드가 다루는 환경 의존 테스트 문제는 기존
  ISSUE-506을 정본으로 유지한다. 이 기록은 추가 문서 게이트 진단만 보존한다.

관측 합계: `broken_links=2`, `ownership_unclassified=1`, `duplicate=0`.
스킬의 상대 참조에는 broken-link 진단이 없었다.

## Expected vs Actual

- Expected: 기존 보고서 링크와 이슈 소유권 메타데이터가 문서 게이트를 통과한다.
- Actual: 위 세 진단으로 전체 문서 게이트가 실패한다. 스킬 변경의 회귀 검증과
  통합은 baseline 비교를 요구하며, 이 카드 등록만으로 해결 범위를 넓히지 않는다.
