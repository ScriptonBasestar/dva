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

2026-09-22 현행화(1차). 405-409 파동의 구현·리뷰는 로컬 태스크 브랜치에만 있고 origin에는 없다.

2026-09-22 현행화(2차). 위 문장은 더 이상 참이 아니다 — 아래 §2026-09-22 통합 완료를 본다. 이 카드가 두 번 틀린 자리를 지우지 않고 남기는 이유는 ISSUE-010이 규정한 결함이 바로 이것이기 때문이다: 쓴 날 참이었던 문장이 통합과 함께 조용히 틀려진다.

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

## 2026-09-22 통합 완료 — 위 표 세 줄을 뒤집는다

독립 리뷰 세션이 TASK-406·409를 판정하고 통합했다. 위 §2026-09-22 파동 표의
`TASK-406`·`TASK-409`·`원격` 세 줄은 그 시점의 기록이고 지금은 아래가 정본이다.

| 항목 | 2026-09-22 현재 |
|---|---|
| TASK-406 | `tasks/done/`. `quality-review: pass` + evidence. 기준 둘 다 human, 리뷰가 직접 읽고 판정 |
| TASK-409 | `tasks/done/`. `quality-review: pass` + evidence. 코멘트 대비 실제 동작을 임시 프로브로 재측정 |
| ISSUE-010 / ISSUE-021 | `resolve fixed --by TASK-406` / `--by TASK-409` 후 `tasks/_archive/issue/` |
| 원격 | **통합 완료.** `origin/master` = `fef00791`. 아래 ISSUE-034 참조 — 교착은 이 경로에서 재현되지 않았다 |
| 워크트리·브랜치 | `ce task run-finish`가 회수. `git worktree list` 1행(프라이머리)만 남음 |

`ce task done-finalize`는 여전히 돌리지 않았다 (hang + ISSUE-024). `tasks/done/`에
405·406·407·408·409 다섯 장이 리뷰 완료 상태로 남아 있고, 이것은 미처리가 아니라
ISSUE-024가 막고 있는 상태다.

## 남은 판단거리 (2026-09-22 리뷰 세션 관측)

이 절은 처분이 아니라 **다음 실행자가 고를 것**의 목록이다.

1. **`legacy-storage-dir` 경고 — 지금 접으면 게이트가 깨진다.** `ce task gate`가
   `archive/`와 `_archive/` 두 철자를 경고한다. `ce task archive --fold --dry-run`은
   413장을 `tasks/_archive/2026-09/`로 옮기겠다고 답한다. 그런데 이 저장소의 자체
   도구가 옛 철자를 하드코딩한다 — `tools/planprogress/check.go:128`의
   `zoneFromPath`는 `archive/` 접두사만 `zoneClosed`로 보고 `_archive/`는
   `zoneOther`로 떨어뜨리며, `tools/doccheck/archive.go:52`의 `archivePrefix`와
   `cardstatus.go:30`도 같다. 접으면 PLAN-006·007·008·009의 아카이브된 자식이
   전부 미완으로 재계산되어 `planprogress`가, 따라서 `make doc-check`가 깨진다.
   **순서가 있다: 도구를 먼저 고치고, 그다음에 접는다.** 경고를 먼저 지우려는
   순서는 게이트를 깬다.
2. **`tasks/_archive/` 안이 이미 두 모양이다.** 이 세션이 만든 `_archive/issue/`는
   zone별이고, `--fold`가 만들 `_archive/2026-09/`는 월별이다. 1번을 집기 전에
   어느 쪽이 정본인지 정해야 한다.
3. **완료된 plan 넷의 거취.** PLAN-006(29/29)·007(13/13)·008(8/8)·009(7/7) 전부
   100%다. `ce task archive`는 자식이 전부 닫힌 plan을 `--force` 없이 아카이브한다.
   다만 각 plan이 스스로 "이 문서가 상태의 정본"이라 적으므로, 옮기는 것은
   정리가 아니라 정본 위치를 바꾸는 결정이다.
4. **ISSUE-033(격리 워크트리의 DUP-ID)은 여전히 미승격.** TASK-407의 가드가
   DUP-ID를 막지만 할당 자체는 그대로다.

