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
TASK-423 blocked until the product owner names the canonical repository, CLI route, and
disposition contract. Keep TASK-436 blocked until the shared lifecycle supplies a supported
cleanup route and the TASK-407 owner decides what to do with its active branch. TASK-431
remains blocked until current host measurements and durable install evidence satisfy its two
failed criteria.

## Children

| 순서 | 작업 | 의존성·병렬 경계 |
|---|---|---|
| 완료 | TASK-425 stale-base 이유/rebase 안내 | 실제 격리 fixture 로그를 `tasks/done/evidence/TASK-425/`에 기록하고 독립 리뷰 PASS |
| 완료 | TASK-426 move/resolve/archive frontmatter status | 일반 이동·no-body-status·resolve·archive 테스트와 독립 PASS 확인 |
| 완료 | TASK-435 issue-promote guard | 상류 master 통합과 독립 리뷰를 마쳤다 |
| 차단 | TASK-431 host tool pins | source worktrunk pin은 devenv master/origin 7273dd62에 통합·push됐고 readiness 비교는 make check 22→22, make lint 23→23, changed-path 진단 없음. 그러나 독립 재측정에서 golangci-lint는 2.12.2를 선택했고 wt 경로는 아직 latest; force install·11개 입력 보존·설치 source commit도 durable evidence로 검증되지 않았다. 기준 1·3을 닫을 추적 가능한 evidence가 필요 |
| 완료 | TASK-427, TASK-428, TASK-429, TASK-434 | 상류 5d70c9d8 구현을 재검증하고 카드별 독립 done-review PASS |
| 완료 | TASK-422 terminal run receipt | 상류 master에 이미 통합; 독립 리뷰 PASS |
| 완료 | TASK-430 finalize evidence path | `ed1f4574` fixes receipt validity/tracking checks, negatives, and preview/apply parity; exact implementation CI and upstream independent review PASS, integrated/pushed in `1e408857`; tracked DVA previews for TASK-391/393/394 and TASK-410 all returned `WOULD REMOVE` with unchanged worktree; fresh DVA independent done-review PASS, ISSUE-024 fixed |
| 완료 | TASK-432 digest, TASK-433 finish tool version | 독립 구현·리뷰 및 commit CI 통과. TASK-432 `80efba95`, TASK-433 `33deb918`가 master에 push됐고 두 task worktree·브랜치 회수 완료 |
| 완료 | TASK-432 → TASK-420 legacy done-review receipts | 첫 리뷰에서 validator round-trip gap을 찾아 수정. `988e7de6`가 3 legacy 입력형태와 fresh reviewer evidence를 validator까지 왕복 검증하고 CI·독립 done-review PASS 뒤 master에 통합·회수됨. TASK-430의 dry-run 계약은 별도 blocker |
| 완료 | TASK-425 + TASK-433 → TASK-438 | `71b1d169`가 두 회귀를 고쳐 exact-HEAD full CI PASS 후 master/origin에 통합·push됐고 task worktree/branch 회수, 두 fixture acceptance, 독립 done-review PASS까지 확인했다. ISSUE-042 fixed; ISSUE-043은 별도 recovery follow-up |
| 완료 | TASK-438 → TASK-439 | `7eaf596a`가 exact-tree full CI·독립 리뷰 PASS 뒤 master/origin에 통합·push되고 task worktree/branch 회수됐다. ISSUE-043 fixed/archive와 DVA done-review PASS 완료 |
| 완료 | TASK-424 validate zone/status ownership | CE `dba2348b` strict policy and producer round trips; DVA paired legacy/strict fixtures, final strict board validation 492/0, duplicate-ID regression restored, `make doc-check`, and identity-scope checks pass; CE resolved-issue terminal status difference is documented; independent done-review PASS |
| 차단 | TASK-421 controller disposition | entry_selection.py는 퇴역했다. 제품 owner가 canonical repository, CLI route, disposition contract를 정해야 재개 가능 (ISSUE-040) |
| 차단 | TASK-423 needs-human 실행 구분 | TASK-421의 제품 결정 뒤에만 진행. ce-agent-kit#1은 역사적 보고처이며 현행 구현 owner로 확인되지 않았다 |
| 차단 | TASK-436 duplicate TASK-407 worktree 회수 | source rebase가 AGENTS.md와 TASK-407 rename/add/delete history에서 충돌했다. shared lifecycle은 cleanup route를 제공하지 않으며 기존 worktree/branch의 owner disposition이 필요 |

## 계획 밖 상류 이슈 (2026-09-27)

- ISSUE-001은 TASK-420의 canonical receipt 생성 기능과 구분된다. ce-workbook은 TASK-045에서 legacy Python engine/execution을 제거했고, ce task review-receipt는 정본 digest를 계산할 뿐 독립 verdict·근거·저장 경로·카드 전이를 소유하지 않는다. 남은 TASK-312 마이그레이션은 [TASK-444](../todo/444-review-task-312-and-record-a-current-durable-receipt.md)로 등록해 DVA host에서 별도 리뷰어의 새 검토와 durable receipt로 처리하며 PLAN-011 child가 아니다.
- 완료/계획 밖 | ISSUE-030 / CE TASK-331 | 구현 `16af8443` 독립 리뷰 PASS 뒤 `f3cfa169`(done-review evidence 포함)가 CE master/origin에 통합·push되고 run-finish가 branch/worktree를 회수했다. exact final-tree `make ci` run ci-20260927-175252-77713 exit 0. source-built CE v0.8.4-372-gf3cfa169에서 validate 16/0, gate READY+revision, review-receipt canonical digest+same tool stamp를 확인했다. mismatch/unknown advisory와 digest hard error는 exact CI regression tests에서 확인했고 ISSUE-030을 fixed/archive했다.
