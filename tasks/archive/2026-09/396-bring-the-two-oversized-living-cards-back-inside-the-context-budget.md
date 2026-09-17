---
id: TASK-396
title: "Bring the two oversized living cards back inside the context budget"
type: chore
priority: P3
effort: M
exec-tier: standard
status: done
archived-at: 2026-09-17
verified-at: 2026-09-17
verification-summary: "Re-verified 2026-09-17. Zero active-zone cards over 20000B (ISSUE-001 now 16085B); ownership and upstream-ref markers retained."
quality-review: pass
quality-reviewed-at: 2026-09-15T19:20:00+09:00
quality-review-receipt: tasks/done/evidence/TASK-396/done-review-050a53f748ddeba5a560ef35a6b558fda9364c01c645464006d7c93957d25f4d.json
quality-review-evidence: "(2026-09-16 리스트형→스칼라형 재형식화, 내용 불변 — ISSUE-029) review-396 (독립 리뷰어, 판정-final yes, head 5e69f15): 기준 1 바인딩 재실행 rc 0 + 독립 전수 크기 측정 — 활성 존 최대 16233B(issue/027), 20000B 초과 0장; 전·후 크기 4개(27599/22160→16065/12679)가 git show로 정확 일치; 충실도 대조 — 부모 커밋 원문과 diff해 판단-결정 사실 전부 보존 확인: Summary 3항목·validator 6메시지 census·376~379 핀 실측 표·TASK-401 폐기 블록·#2→#7 보고 정정·P0→P2 근거·위조 금지 원칙; PLAN-007은 배분 스냅샷·처분 원칙(컷오프=1c85d8d0)·48장 함정·External 잔여·children 13=13(전부 done, TASK-373 done 실재 확인); 죽은 주장 grep(도달하지/조용/fallback 등) — 모든 적중이 폐기 프레임 안에 있고 'fallback은 존재하지 않는다'는 올바른 부정임을 확인; TASK-401 done 카드가 재측정을 독립적으로 교차검증; 게이트 재실행 — validate --all 120 valid 0 invalid, gate READY, make doc-check/lint rc 0, 스코프 4파일(카드 이동 + 압축 2장) 소스 무변경"
completion-summary: "ISSUE-001 27599→16065B, PLAN-007 22160→12679B — 둘 다 20000B 예산 안. ISSUE-001은 시간순 경위를 요지 중심으로 재구성하되 validator 메시지 census·실측 표·폐기/정정 이력·소유권 귀속을 전부 보존했고, TASK-401로 폐기된 plain-핀 서술을 현행화했다. PLAN-007은 처분 원칙과 배분 스냅샷 중심으로 압축했다. 기준 3개 바인딩 전부 실행해 rc 0."
verification-status: verified
verification-evidence:
  - kind: automated
    command-or-step: "기준 1 바인딩 (활성 존 >20000B 카드 0장)"
    result: "rc 0 — tasks/issue·plan·todo에서 20000바이트 초과 카드 없음. ISSUE-001 16065B, PLAN-007 12679B"
  - kind: automated
    command-or-step: "기준 2 바인딩 (ISSUE-001 소유권·upstream-ref)"
    result: "rc 0 — frontmatter upstream-ref: ce-agent-kit#7과 §소유권 절 유지"
  - kind: automated
    command-or-step: "make doc-check"
    result: "rc 0 — 압축 후 게이트 재실행, ce task validate --all 120 valid 0 invalid + gate READY 동시 확인"
created: 2026-09-15
source: "ISSUE-023 처분 기록 (2026-09-15) — 살아 있는 두 장은 매 실행마다 읽힌다"
depends-on: []
---

## Summary

활성 존에서 20000바이트 예산을 넘는 카드가 두 장이다. done 존 다섯 장과 달리
이 둘은 **매 게이트 실행마다 실제로 읽히므로** 비용이 반복해서 청구된다.

```
27599  tasks/issue/001-...md   (레거시 done 리뷰 증거 서술 — 카드 초안 시점 25821B에서 이슈 정정 작업으로 성장한 뒤의 크기)
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

- [x] 활성 존 카드가 예산 안에 든다 | verify: `test 0 -eq $(/usr/bin/find tasks/issue tasks/plan tasks/todo -maxdepth 1 -name '*.md' -size +20000c | /usr/bin/grep -c .)` — 실행 rc 0
- [x] ISSUE-001이 소유권·보고 자리를 유지한다 | verify: `/usr/bin/grep -c '소유권\|upstream-ref' $(/usr/bin/find tasks -name '001-task-runtime-cannot-review-legacy-done-cards-without-verification-evidence.md')` — 실행 rc 0
- [x] 게이트 통과 | verify: `make doc-check` (regression-guard) — 실행 rc 0

## Sources

- ISSUE-023 — 2026-09-15 재측정과 처분 기록
- TASK-395 — ISSUE-001에 소유권 표기와 `upstream-ref:` 자리를 만든 작업

## Related

- [[ISSUE-023]] · [[ISSUE-001]] · [[PLAN-007]]

## Evidence (2026-09-15)

| 카드 | 이전 | 이후 | 절약 |
|---|---|---|---|
| tasks/issue/001-…md | 27,599B | 16,065B | −11,534B |
| tasks/plan/007-…md | 22,160B | 12,679B | −9,481B |

압축 원칙(카드 §접근 준수): 서술 대증거가 아니라 요지가 목적이므로 시간순 경위를 요지
중심으로 재구성했다. 남긴 것 — ISSUE-001: 세 Summary 항목, validator 메시지 census,
376~379 실측 표, TASK-401 폐기 현행화(plain 핀 fallback 사망), #2→#7 보고 정정, 소유권
귀속, Resolution Criteria 4개(미체크 2개는 상류 소유로 유지). PLAN-007: 배분 스냅샷
표, 처분 원칙(컷오프는 규칙 커밋 `1c85d8d0`, 소급 리뷰 금지), 48장 conditional 함정
기록, External 잔여(닫기 시점 강제). 두 카드 모두 압축 안내 각주를 달아 원문 전문이
Git 이력에 있음을 명시했다. 폐기된 서술("blocks: 없으면 receipt 검사 미도달" 계열)은
삭제가 아니라 날짜 있는 현행화로 교체해 ISSUE-001의 pin 절과 PLAN-007의 381 절이
TASK-401의 재측정과 모순되지 않게 했다.

## 리뷰 대응 (2026-09-15, review-396 — pass)

독립 리뷰어가 6개 검사를 전부 재실행해 pass(판정-final)를 냈다(head-reviewed
`5e69f15`). 충실도 대조에서 판단-결정 사실 전부 보존이, 죽은 주장 grep에서
현행화의 올바른 부정이, 게이트 4종에서 green이 확인됐다. findings 3건은 전부
비차단: F1(info) — 본 Summary의 이슈 크기 25821은 카드 초안 시점 스냅샷이므로
마감 커밋에서 27599(압축 직전 실측)로 갱신함; F2(low)·F3(info) — 압축으로
떨어진 부분(P0→P2의 하위 논증, awk 측정 명령)은 카드가 선언한 압축 원칙 안에서
수용, 전문은 Git 이력에 있다.
