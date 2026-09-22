---
id: BACKLOG-009
title: "Triage remaining open issues and stale run records after 405-408 wave"
type: idea
priority: P2
effort: M
created: 2026-09-21
---

## Description

2026-09-21 전수 분류. 열린 이슈 24장. DVA 코드로 고칠 수 없는 상류(`ce-agent-kit`) 결함은 todo로 승격하지 않았다 — 실행 불가능한 카드를 쌓는 것이 보드 오염이다.

2026-09-22 현행화. 405-409 파동의 구현·리뷰는 로컬 태스크 브랜치에만 있고 origin에는 없다.

## Expected Value

이슈 칸이 "할 일"과 "상류 관측"과 "이미 승격된 작업"으로 나뉘어, 다음 실행자가 무엇을 손댈 수 있는지 카드만 읽고 알 수 있다.

## 2026-09-21 처분

### 해결 (`ce task resolve … fixed`)

| 이슈 | 사유 |
|---|---|
| ISSUE-011 | 기준이 OR. DVA 쪽 `AGENTS.md` 우회가 이미 충족. 상류 조언은 여전히 틀리지만 이 저장소 왕복은 사라짐 |
| ISSUE-023 | 세 기준 모두 충족. 살아 있는 카드 0장이 20KiB 초과, done 다섯 장은 아카이브로 처분, 게이트 READY. oversized advisory는 더 이상 안 나옴 |

### 태스크 분리 (이미 승격, 이슈는 작업 완료까지 유지)

| 이슈 | 태스크 | 상태 |
|---|---|---|
| ISSUE-009 | TASK-405 | done (독립 리뷰 pass). ISSUE-009 archived |
| ISSUE-010 | TASK-406 | review. 구현 세션 리뷰 금지 |
| ISSUE-012 | TASK-407 | done (독립 리뷰 pass). ISSUE-012 archived. 잔여 ISSUE-033 |
| ISSUE-017 | TASK-408 | done (독립 리뷰 pass). ISSUE-017 archived |
| ISSUE-021 | TASK-409 | review. 구현 세션 리뷰 금지. 트리아지 쪽 `todo/409`는 DUP-ID라 폐기 |

### 상류 유지 (DVA에서 수정 불가, 이슈로 존치)

ISSUE-001, 004, 005, 006, 007, 008, 013, 014, 019, 020, 022, 024, 026, 028, 030, 031, 032.

P1: ISSUE-004 (컨트롤러 스코프), ISSUE-032 (`issue-promote apply`가 trunk `master`에서 trailing cleanup에 실패). 032는 태스크 워크트리에서는 apply가 통과함 (TASK-409가 그 증거).

### 하지 않은 것

- 상류 이슈를 DVA todo로 승격하지 않음
- `ce task run-list`의 낡은 실행 기록은 ISSUE-005 (터미널 상태 필드 부재). 이 저장소가 지울 수 없음
- ISSUE-023의 상류 advisory 표현 방식은 여전히 안 읽힘. 이번 해결은 로컬 기준만

## 2026-09-22 파동

| 항목 | 처분 |
|---|---|
| TASK-405·407·408 독립 리뷰 | `done/` + `quality-review: pass`. ISSUE-009·012·017 `_archive/issue/` |
| TASK-406 | 로컬 master에 합침. 카드는 `review/` |
| TASK-409 | 로컬 master에 합침. 카드는 `review/`. 트리아지 쪽 `todo/409`는 DUP-ID라 폐기 |
| TASK-370 | 다음 `release-notes/v*.md`까지 보류 |
| ISSUE-033 | 승격하지 않음. 407의 가드는 DUP-ID |
| `done-finalize` | 돌리지 않음. hang + ISSUE-024 |
| 원격 | ISSUE-034. `git push`는 integrate를 요구하고 `branch-integrate`는 upstream을 요구함 |
| ISSUE-011·023 | 이 브랜치에서 archive |
