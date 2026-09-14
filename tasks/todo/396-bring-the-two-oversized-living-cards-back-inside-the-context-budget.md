---
id: TASK-396
title: "Bring the two oversized living cards back inside the context budget"
type: chore
priority: P3
effort: M
exec-tier: standard
status: todo
created: 2026-09-15
source: "ISSUE-023 처분 기록 (2026-09-15) — 살아 있는 두 장은 매 실행마다 읽힌다"
depends-on: []
---

## Summary

활성 존에서 20000바이트 예산을 넘는 카드가 두 장이다. done 존 다섯 장과 달리
이 둘은 **매 게이트 실행마다 실제로 읽히므로** 비용이 반복해서 청구된다.

```
25821  tasks/issue/001-...md   (레거시 done 리뷰 증거 서술)
22160  tasks/plan/007-...md    (done 백로그 분류)
```

[[ISSUE-023]]의 2026-09-15 처분 기록이 done 다섯 장의 길을 정리했으니, 이 카드는
살아 있는 두 장만 담는다.

## 접근

- **ISSUE-001**: 증거 서술이 본문 대부분이다. 요약 본문 + 상세를 절 압축으로
  줄이되, 리뷰가 근거로 삼은 사실(무엇을 봤고 무엇이 잘못이었는지)은 남긴다.
  상류 소유 표기(`## 소유권`, `upstream-ref:`)는 이 작업에서 이미 채워진다.
- **PLAN-007**: 분류 표가 본문이다. 표를 done 백로그의 현재 상태로 현행화하면
  자연히 줄어든다 — 2026-09-15 기준 처리 완료·제거된 항목이 다수일 것이다.

두 장 모두 서술 대증거가 아니라 요지가 목적이므로, 압축은 증거 폐기가 아니다.
원문 전문은 Git 이력에 있다.

## Completion Criteria

- [ ] 활성 존 카드가 예산 안에 든다 | verify: `test 0 -eq $(/usr/bin/find tasks/issue tasks/plan tasks/todo -maxdepth 1 -name '*.md' -size +20000c | /usr/bin/grep -c .)`
- [ ] ISSUE-001이 소유권·보고 자리를 유지한다 | verify: `/usr/bin/grep -c '소유권\|upstream-ref' $(/usr/bin/find tasks -name '001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md')`
- [ ] 게이트 통과 | verify: `make doc-check` (regression-guard)

## Sources

- ISSUE-023 — 2026-09-15 재측정과 처분 기록
- TASK-395 — ISSUE-001에 소유권 표기와 `upstream-ref:` 자리를 만든 작업

## Related

- [[ISSUE-023]] · [[ISSUE-001]] · [[PLAN-007]]
