---
id: PLAN-010
title: "개발·검증·문서 증거를 정합화하고 0.3.0을 마감한다"
type: plan
scope: "보드 게이트 복구, 완료 증거 정합화, 다음 릴리스 후보 및 공개"
progress: 62
total-tasks: 8
completed-tasks: 5
children: [TASK-415, TASK-410, TASK-411, TASK-416, TASK-419, TASK-417, TASK-370, TASK-418]
target-date: "2026-09-30"
created: 2026-09-23
---

## Goal

빈 보드가 아니라 검증된 릴리스 범위를 만든다. 현재 목록은 [보드 인덱스](../README.md),
공개 절차는 [런북](../../docs/52-manual-release-runbook.md)이 소유한다.

## Children

| 순서 | 작업 | 의존·병렬 경계 |
|---|---|---|
| 1 | TASK-415 doing/blocked zone 수정 | TASK-410 독립 리뷰 및 TASK-328 증거 대조와 병렬 |
| 1 | TASK-410 누락 receipt 복구 | 구현과 다른 리뷰어가 검증; finalize 성공은 선행 조건 아님 |
| 2 | TASK-411 fold 완료 재검증 | TASK-410 receipt 및 TASK-415 문서 게이트 복구 뒤 |
| 2 | TASK-416 보드·계획 현행화 | 선행 게이트 복구 뒤; 411 독립 리뷰와 파일 소유권 분리 |
| 2 | TASK-419 tag 전 CHANGELOG 후보 인식 | TASK-416 기록 정합화와 별도 워크트리에서 병렬 구현; 둘 다 통합 후 후보 검증 |
| 3 | TASK-417 릴리스 후보 | TASK-416·411·419 완료 뒤 버전/노트/전체 검증 |
| 3 | TASK-370 새 노트 오류 재발 확인 | TASK-417 후보 완료 뒤 독립 검토; depends-on/blocks로 연결 |
| 4 | TASK-418 공개 및 postflight | TASK-417·370 완료 및 공개 identity 승인 뒤 |

## 별도 유지

상류 17개 이슈는 DVA 개발 큐로 복제하지 않는다. ISSUE-024의 finalize 차단은
증거 경로·정당한 참조를 지워 우회하지 않는다. ISSUE-039 워크트리 폐기,
PLAN-006~009 아카이브와 `tasks/` 아래 네 `.ce` 잔재 삭제는 기존 사람 결정 대기다.
루트 `.ce/task-runtime.yaml`은 ACTIVE 런타임 선언이며 잔재가 아니다.
