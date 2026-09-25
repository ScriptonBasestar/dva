---
id: PLAN-011
title: "Close upstream task follow-ups in dependency order"
type: plan
priority: P2
effort: L
scope: "Close upstream task follow-ups in dependency order"
progress: 78
total-tasks: 19
completed-tasks: 15
children: [TASK-420, TASK-421, TASK-422, TASK-423, TASK-424, TASK-425, TASK-426, TASK-427, TASK-428, TASK-429, TASK-430, TASK-431, TASK-432, TASK-433, TASK-434, TASK-435, TASK-436, TASK-438, TASK-439]
target-date: "2026-10-09"
created: 2026-09-25
---

## Goal

Close the active upstream follow-up cards in an order that respects required code
dependencies, independent reviews, and explicit ownership blockers. Keep TASK-421 and
TASK-423 blocked until the product owner names the controller contract. Keep TASK-436
blocked until the shared lifecycle supplies a supported cleanup route, and keep the
remaining in-progress follow-ups visible with their existing guards.

## Children

| 순서 | 작업 | 의존성·병렬 경계 |
|---|---|---|
| 완료 | TASK-425 stale-base 이유/rebase 안내 | 실제 격리 fixture 로그를 `tasks/done/evidence/TASK-425/`에 기록하고 독립 리뷰 PASS |
| 완료 | TASK-426 move/resolve/archive frontmatter status | 일반 이동·no-body-status·resolve·archive 테스트와 독립 PASS 확인 |
| 완료/차단 | TASK-435 issue-promote guard, TASK-431 host tool pin | TASK-435는 상류 통합 완료; TASK-431 pin은 devenv master/origin `7273dd62`에 통합·push되고 run-finish에서 worktree/branch 회수까지 완료. 최종 readiness는 `make check` 22→22, `make lint` 23→23, changed-path 진단 없음. live 설정 drift를 보존할 installer 경로만 남아 TASK-431은 blocked |
| 완료 | TASK-427, TASK-428, TASK-429, TASK-434 | 상류 5d70c9d8 구현을 재검증하고 카드별 독립 done-review PASS |
| 완료 | TASK-422 terminal run receipt | 상류 master에 이미 통합; 독립 리뷰 PASS |
| 완료 | TASK-430 finalize evidence path | `ed1f4574` fixes receipt validity/tracking checks, negatives, and preview/apply parity; exact implementation CI and upstream independent review PASS, integrated/pushed in `1e408857`; tracked DVA previews for TASK-391/393/394 and TASK-410 all returned `WOULD REMOVE` with unchanged worktree; fresh DVA independent done-review PASS, ISSUE-024 fixed |
| 완료 | TASK-432 digest, TASK-433 finish tool version | 독립 구현·리뷰 및 commit CI 통과. TASK-432 `80efba95`, TASK-433 `33deb918`가 master에 push됐고 두 task worktree·브랜치 회수 완료 |
| 완료 | TASK-432 → TASK-420 legacy done-review receipts | 첫 리뷰에서 validator round-trip gap을 찾아 수정. `988e7de6`가 3 legacy 입력형태와 fresh reviewer evidence를 validator까지 왕복 검증하고 CI·독립 done-review PASS 뒤 master에 통합·회수됨. TASK-430의 dry-run 계약은 별도 blocker |
| 완료 | TASK-425 + TASK-433 → TASK-438 | `71b1d169`가 두 회귀를 고쳐 exact-HEAD full CI PASS 후 master/origin에 통합·push됐고 task worktree/branch 회수, 두 fixture acceptance, 독립 done-review PASS까지 확인했다. ISSUE-042 fixed; ISSUE-043은 별도 recovery follow-up |
| 완료 | TASK-438 → TASK-439 | `7eaf596a`가 exact-tree full CI·독립 리뷰 PASS 뒤 master/origin에 통합·push되고 task worktree/branch 회수됐다. ISSUE-043 fixed/archive와 DVA done-review PASS 완료 |
| 완료 | TASK-424 validate zone/status ownership | CE `dba2348b` strict policy and producer round trips; DVA paired legacy/strict fixtures, final strict board validation 492/0, duplicate-ID regression restored, `make doc-check`, and identity-scope checks pass; CE resolved-issue terminal status difference is documented; independent done-review PASS |
| 차단 | TASK-421 controller disposition | `entry_selection.py` 퇴역. 새 owner/CLI route/product contract 결정 필요 (ISSUE-040) |
| 차단 | TASK-423 needs-human 실행 구분 | TASK-421의 disposition contract 뒤 진행 |
| 차단 | TASK-436 duplicate TASK-407 worktree 회수 | source rebase가 AGENTS.md와 TASK-407 rename/add/delete history에서 충돌했다. 유용한 직접 링크는 active branch에 보존했지만 `run-abort`는 worktree/ref를 제거하지 않고 정책상 수동 삭제도 허용되지 않아 shared lifecycle recovery route가 필요 |
