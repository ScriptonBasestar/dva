---
id: PLAN-011
title: "Close upstream task follow-ups in dependency order"
type: plan
priority: P2
effort: L
scope: "Close upstream task follow-ups in dependency order"
progress: 94
total-tasks: 19
completed-tasks: 18
children: [TASK-420, TASK-421, TASK-422, TASK-423, TASK-424, TASK-425, TASK-426, TASK-427, TASK-428, TASK-429, TASK-430, TASK-431, TASK-432, TASK-433, TASK-434, TASK-435, TASK-436, TASK-438, TASK-439]
target-date: "2026-10-09"
created: 2026-09-25
---

## Goal

Close the active upstream follow-up cards in an order that respects required code
dependencies, independent reviews, and explicit ownership blockers. TASK-421 and
TASK-423 were handed off on 2026-09-27 to taskchain-task-manager (task-manager-devbox
W13/W14) and archived as superseded; ISSUE-004/006 track adoption. TASK-436 has
recorded reclaim authorization and waits for a supported CE discard action. TASK-431
completed host pin installation with tracked local-input and receipt evidence.

## Children

| 순서 | 작업 | 의존성·병렬 경계 |
|---|---|---|
| 완료 | TASK-425 stale-base 이유/rebase 안내 | 실제 격리 fixture 로그를 `tasks/done/evidence/TASK-425/`에 기록하고 독립 리뷰 PASS |
| 완료 | TASK-426 move/resolve/archive frontmatter status | 일반 이동·no-body-status·resolve·archive 테스트와 독립 PASS 확인 |
| 완료 | TASK-435 issue-promote guard | 상류 master 통합과 독립 리뷰를 마쳤다 |
| 완료 | TASK-431 host tool pins | source `7273dd62`의 0.74.0 핀을 지정 설치기로 live에 반영했다. `/tmp`에서 명시적 wt 0.74.0과 Aqua lint 2.13.2를 선택·실행한다. 선언된 로컬 입력 10개와 receipt를 추적 증거로 대조했고 generated shared drift는 사용자 선택에 따라 소스 정본으로 복귀했다. 독립 done-review PASS |
| 완료 | TASK-427, TASK-428, TASK-429, TASK-434 | 상류 5d70c9d8 구현을 재검증하고 카드별 독립 done-review PASS |
| 완료 | TASK-422 terminal run receipt | 상류 master에 이미 통합; 독립 리뷰 PASS |
| 완료 | TASK-430 finalize evidence path | `ed1f4574` fixes receipt validity/tracking checks, negatives, and preview/apply parity; exact implementation CI and upstream independent review PASS, integrated/pushed in `1e408857`; tracked DVA previews for TASK-391/393/394 and TASK-410 all returned `WOULD REMOVE` with unchanged worktree; fresh DVA independent done-review PASS, ISSUE-024 fixed |
| 완료 | TASK-432 digest, TASK-433 finish tool version | 독립 구현·리뷰 및 commit CI 통과. TASK-432 `80efba95`, TASK-433 `33deb918`가 master에 push됐고 두 task worktree·브랜치 회수 완료 |
| 완료 | TASK-432 → TASK-420 legacy done-review receipts | 첫 리뷰에서 validator round-trip gap을 찾아 수정. `988e7de6`가 3 legacy 입력형태와 fresh reviewer evidence를 validator까지 왕복 검증하고 CI·독립 done-review PASS 뒤 master에 통합·회수됨. TASK-430의 dry-run 계약은 별도 blocker |
| 완료 | TASK-425 + TASK-433 → TASK-438 | `71b1d169`가 두 회귀를 고쳐 exact-HEAD full CI PASS 후 master/origin에 통합·push됐고 task worktree/branch 회수, 두 fixture acceptance, 독립 done-review PASS까지 확인했다. ISSUE-042 fixed; ISSUE-043은 별도 recovery follow-up |
| 완료 | TASK-438 → TASK-439 | `7eaf596a`가 exact-tree full CI·독립 리뷰 PASS 뒤 master/origin에 통합·push되고 task worktree/branch 회수됐다. ISSUE-043 fixed/archive와 DVA done-review PASS 완료 |
| 완료 | TASK-424 validate zone/status ownership | CE `dba2348b` strict policy and producer round trips; DVA paired legacy/strict fixtures, final strict board validation 492/0, duplicate-ID regression restored, `make doc-check`, and identity-scope checks pass; CE resolved-issue terminal status difference is documented; independent done-review PASS |
| 이관 | TASK-421 controller disposition | 9/27 제품 owner가 taskchain-task-manager로 지정. task-manager-devbox W13으로 이관하고 superseded로 보관 (ISSUE-040 해결) |
| 이관 | TASK-423 needs-human 실행 구분 | 9/27 task-manager-devbox W14로 이관하고 superseded로 보관. 채택은 ISSUE-006이 추적 |
| 차단 | TASK-436 duplicate TASK-407 worktree 회수 | 사용자 회수 승인과 branch 내용 판단은 기록됐다. source rebase 충돌 후 shared lifecycle의 검증 가능한 discard 경로가 필요하다 |

## 계획 밖 상류 이슈 (2026-09-27)

- ISSUE-001은 TASK-420의 canonical receipt 생성 기능과 구분됐다. ce-workbook은 TASK-045에서 legacy Python engine/execution을 제거했고, ce task review-receipt는 정본 digest를 계산할 뿐 독립 verdict·근거·저장 경로·카드 전이를 소유하지 않는다. 마지막 TASK-312 마이그레이션은 [TASK-444](../done/444-review-task-312-and-record-a-current-durable-receipt.md)에서 DVA host의 fresh independent PASS와 first durable receipt로 마쳤고 ISSUE-001을 fixed/archive했다. TASK-444는 PLAN-011 child가 아니다.
- 완료/계획 밖 | ISSUE-030 / CE TASK-331 | 구현 `16af8443` 독립 리뷰 PASS 뒤 `f3cfa169`(done-review evidence 포함)가 CE master/origin에 통합·push되고 run-finish가 branch/worktree를 회수했다. exact final-tree `make ci` run ci-20260927-175252-77713 exit 0. source-built CE v0.8.4-372-gf3cfa169에서 validate 16/0, gate READY+revision, review-receipt canonical digest+same tool stamp를 확인했다. mismatch/unknown advisory와 digest hard error는 exact CI regression tests에서 확인했고 ISSUE-030을 fixed/archive했다.
