---
id: TASK-489
title: "Apply the approved CHANGELOG ceiling trigger"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tasks/decision/491-queue-acceptance-scope-and-historical-evidence.md, decisions/README.md, CHANGELOG.md, decisions/DECISION-001-changelog-append-only-vs-size-rules.md, tasks/decision/487-changelog-half-ceiling-trigger.md, tasks/decision/README.md, tasks/README.md, tasks/done/486-record-session-followup-contract.md, tasks/issue/488-measure-expired-tunnel-token-and-live-doctor.md, tasks/todo/489-apply-approved-changelog-ceiling.md, tasks/done/489-apply-approved-changelog-ceiling.md, tasks/issue/490-selected-taskchain-rejects-native-decision-directory.md, tasks/done/evidence/TASK-489]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-05
quality-review-evidence: "Independent grok-4.7 session eaafc426-c64b-4074-96b9-7679057fc6d1 PASS; tasks/done/evidence/TASK-489/independent-review-attempt-2.json"
created: 2026-10-05
---

## Summary

2026-10-05에 사용자가 TASK-487 옵션 2를 골랐다. ceiling은 `87296 bytes, 1000 lines` 그대로다. 절반 아카이브 문장을 실제 ceiling 트리거로 바꾼다. 릴리스에서 바이트가 87296 이상이거나 물리 줄이 1000 이상이면 v0.2.x 이하를 아카이브한다. 이 비교는 문서화된 트리거다. CE 파서가 그 비교를 기계적으로 집행한다고 말하지 않는다. 현재 측정은 67170 bytes, 744 physical lines라 아카이브하지 않는다.

작성 세션의 이력과 조율된 범위를 구분한다. 이 작성 세션은 ISSUE-488·ISSUE-490·TASK-491을 고치지 않는다. 조율된 범위에서 ISSUE-488은 코디네이터의 실제 읽기 전용 확인 기록이다. ISSUE-490은 실제 큐 조회 실패 기록이다. TASK-491은 Proposed이고 선택은 적용하지 않았다. 세 경로는 이미 `allowed-paths`에 있다. 비밀은 없다.

기준 1–6과 기준 8은 변경 전 HEAD에서 실패한다. 기준 7과 기준 9는 회귀 가드이며 지금 통과한다. 기준 9의 `make doc-check`가 지금 실패한다는 문장은 현재 Summary가 아니다.

## Steps

1. `CHANGELOG.md:2` — 첫 마커만. ceiling `87296 bytes, 1000 lines`는 유지한다. 아카이브 문장은 문서화된 트리거로, 바이트 >= 87296 또는 물리 줄 >= 1000이다. 아래 히스토리 문장은 고치지 않는다.
2. `decisions/DECISION-001-changelog-append-only-vs-size-rules.md:6` — `status: Accepted`. `:35` — 승인된 ceiling. `:38` — 2026-10-05 옵션 2와 문서화된 트리거. 파서 비집행 문장. `:40` — 67170 bytes, 744 physical lines. 아카이브하지 않는다. `:53` — ceiling 유지. `:54` — 바이트와 물리 줄 트리거.
3. `tasks/decision/487-changelog-half-ceiling-trigger.md:10` — `status: Accepted`. `:29` — `- Status: Accepted`. `:31` — 옵션 2. `:34` — 파서가 비교를 기계적으로 집행한다고 말하지 않는다. `:58` — 해결 기준은 `[x]`.
4. `tasks/decision/README.md:5` — 현재 최대는 ADR-0000, 로그 491이고 다음은 로그 492다. TASK-491이 색인에 있어서다. `:9` — TASK-487 status `accepted`. 로그 487은 TASK-491이 생기기 전 작성 스냅샷의 역사다. 그 역사 문장으로 현재 최대를 단정하지 않는다.
5. `tasks/README.md:30` — 현재 상태. TASK-487 Accepted, TASK-489 todo, 아카이브 없음. ISSUE-488은 고치지 않는다고 적는다.
6. `tasks/done/486-record-session-followup-contract.md:57` — 기준 2 바인딩은 `status: Accepted`. `:60` — 기준 5도 Accepted. `:61` — 기준 6은 실제 ceiling 마커이고 `ceiling 절반(44KB)`는 없다. 옛 exit 0은 historical이다. Steps·Attempts·리뷰 파일은 그대로다.
7. `tasks/issue/488-measure-expired-tunnel-token-and-live-doctor.md:13` — `discovered-at: 2026-10-05`. `:25` — `## 소유권 — 이 저장소다`. `:29` — `## Reproduction`. `:73` — `## 실행 기록 (2026-10-05)`. `:78` — `dva --json doctor` exit 0, 터널 인증 행은 없다. 코디네이터 갱신이다. 이 카드는 그 줄을 고치지 않는다.

8. `tasks/issue/490-selected-taskchain-rejects-native-decision-directory.md:18` — 실제 조회 실패, 바이너리 SHA, 생산자 경로 확인과 승인 의존을 기록한다. 제품 소스와 pin은 변경하지 않는다.

9. `decisions/README.md:19` — TASK-487의 선택을 Accepted 실제 ceiling으로 현행화한다.

10. `tasks/decision/491-queue-acceptance-scope-and-historical-evidence.md:16` — 004·006 범위 현행화 옵션을 기록한다. 선택을 적용하거나 pin을 승인하지 않는다.

## Stop conditions

- CHANGELOG.md는 첫 마커만 고친다. ceiling 숫자 `87296`와 `1000`은 유지한다. v0.2.x를 아카이브하지 않는다.
- CE 파서가 바이트·물리 줄 비교를 기계적으로 집행한다고 적지 않는다.
- ISSUE-004, ISSUE-006, ISSUE-453, 자격 증명, pin을 고치지 않는다.
- ISSUE-488 본문을 고치지 않는다. 코디네이터의 발견·doctor 기록을 덮어쓰지 않는다.
- TASK-486의 Steps, Attempts, 역사적 산문, `tasks/done/evidence/TASK-486` 리뷰 파일을 고치지 않는다. `blocks:`와 `quality-review-receipt`를 넣지 않는다.
- 작성자는 `quality-review`를 넣지 않고 독립 리뷰 PASS를 주장하지 않는다. 자기 카드를 통과시키지 않는다.
- 독립 리뷰 PASS 뒤에 코디네이터가 CI, 커밋, push, done 마무리를 할 수 있다. 작성 세션은 그 전에 하지 않는다. TASK-459의 사람 기준을 체크하지 않는다.

## Completion Criteria

- [x] TASK-487이 Accepted이고 옵션 2와 2026-10-05 선택이 적혀 있다. 변경 전 트리는 Proposed라 이 바인딩이 exit 1이었다 | verify: `/usr/bin/grep -qx 'status: Accepted' tasks/decision/487-changelog-half-ceiling-trigger.md && /usr/bin/grep -q -F -- '- Status: Accepted' tasks/decision/487-changelog-half-ceiling-trigger.md && /usr/bin/grep -q -F '2026-10-05에 사용자가 옵션 2를 골랐다' tasks/decision/487-changelog-half-ceiling-trigger.md && /usr/bin/grep -q -F -- '- [x] 사람이 2026-10-05에 옵션 2' tasks/decision/487-changelog-half-ceiling-trigger.md && /usr/bin/grep -q -F '| accepted |' tasks/decision/README.md` (observed: 2026-10-05 — exit 0)
- [x] CHANGELOG 첫 마커가 실제 ceiling 트리거이고 절반 문장은 없다. 변경 전 마커에는 절반 문장이 있어 이 바인딩이 exit 1이었다 | verify: `/usr/bin/grep -q -F '<!-- size-limit: exempt, ceiling: 87296 bytes, 1000 lines -- append-only changelog은 분할 시 히스토리가 아니라 미래가 바뀐다 (DECISION-001). 문서화된 트리거: 릴리스에서 바이트가 87296 이상이거나 물리 줄이 1000 이상이면 v0.2.x 이하를 아카이브한다. -->' CHANGELOG.md && ! /usr/bin/grep -q -F 'ceiling 절반(44KB)' CHANGELOG.md` (observed: 2026-10-05 — exit 0)
- [x] DECISION-001이 옵션 A, 같은 ceiling, 옵션 2, 문서화된 트리거와 파서 비집행 문장을 함께 가진다. 변경 전 본문은 선택을 확정하지 않아 이 바인딩이 exit 1이었다 | verify: `/usr/bin/grep -qx 'status: Accepted' decisions/DECISION-001-changelog-append-only-vs-size-rules.md && /usr/bin/grep -q -F '**옵션 A**' decisions/DECISION-001-changelog-append-only-vs-size-rules.md && /usr/bin/grep -q -F '옵션 2' decisions/DECISION-001-changelog-append-only-vs-size-rules.md && /usr/bin/grep -q -F 'CE 파서가 그 비교를 기계적으로 집행한다고 말하지 않는다' decisions/DECISION-001-changelog-append-only-vs-size-rules.md && /usr/bin/grep -q -F '67170 bytes, 744 physical lines' decisions/DECISION-001-changelog-append-only-vs-size-rules.md && ! /usr/bin/grep -q -F '이 문서는 그 선택을 확정하지 않는다' decisions/DECISION-001-changelog-append-only-vs-size-rules.md` (observed: 2026-10-05 — exit 0)
- [x] TASK-486 기준 2·5는 Accepted 바인딩이고 기준 6은 실제 ceiling 마커다. 변경 전 바인딩은 Proposed와 절반 마커라 이 바인딩이 exit 1이었다 | verify: `/usr/bin/grep -rq -F --include='486-record-session-followup-contract.md' "^status: Accepted' tasks/decision/487-changelog-half-ceiling-trigger.md" tasks && ! /usr/bin/grep -rq -F --include='486-record-session-followup-contract.md' "^status: Proposed' tasks/decision/487-changelog-half-ceiling-trigger.md" tasks && /usr/bin/grep -rq -F --include='486-record-session-followup-contract.md' 'historical: 2026-10-05 exit 0 was the Proposed binding' tasks && /usr/bin/grep -rq -F --include='486-record-session-followup-contract.md' '문서화된 트리거: 릴리스에서 바이트가 87296 이상이거나 물리 줄이 1000 이상이면' tasks && /usr/bin/grep -rq -F --include='486-record-session-followup-contract.md' 'historical: 2026-10-05 exit 0 was the half-ceiling marker binding' tasks` (observed: 2026-10-05 — exit 0)
- [x] 보드 현재 상태가 옵션 2 수락과 아카이브 부재를 말한다. 변경 전 현재 상태는 미결 선택이라 이 바인딩이 exit 1이었다 | verify: `/usr/bin/grep -q -F '옵션 2(실제 ceiling)를 골라 Accepted' tasks/README.md && /usr/bin/grep -q -F 'v0.2.x 아카이브는 하지 않는다' tasks/README.md && /usr/bin/grep -q -F '489-apply-approved-changelog-ceiling.md' tasks/README.md` (observed: 2026-10-05 — exit 0)
- [x] 새 ISSUE-490에 현재 바이너리와 native decision 조회 실패가 기록된다. 시작 트리에는 이 이슈가 없다 | verify: `/usr/bin/grep -rq --include='490-selected-taskchain-rejects-native-decision-directory.md' -F 'unsupported task directory at board root: decision' tasks && /usr/bin/grep -rq --include='490-selected-taskchain-rejects-native-decision-directory.md' -F 'db6dd2d0d61373762d5623f45418913a1585650616b6cc811476c88e2a637a1b' tasks` (observed: 2026-10-05 — exit 0)
- [x] CHANGELOG가 두 상한 아래다 | verify: `test "$(wc -c < CHANGELOG.md)" -lt 87296 && test "$(wc -l < CHANGELOG.md)" -lt 1000` (regression-guard) (observed: 2026-10-05 — exit 0)
- [x] ISSUE-488의 소유권·재현·2026-10-05 doctor 기록이 남아 있다. 이 카드는 그 파일을 고치지 않는다 | verify: `/usr/bin/grep -rq -F --include='488-measure-expired-tunnel-token-and-live-doctor.md' '## 소유권 — 이 저장소다' tasks && /usr/bin/grep -rq -F --include='488-measure-expired-tunnel-token-and-live-doctor.md' '## Reproduction' tasks && /usr/bin/grep -rq -F --include='488-measure-expired-tunnel-token-and-live-doctor.md' '## 실행 기록 (2026-10-05)' tasks && /usr/bin/grep -rq -F --include='488-measure-expired-tunnel-token-and-live-doctor.md' 'dva --json doctor' tasks && /usr/bin/grep -rq -F --include='488-measure-expired-tunnel-token-and-live-doctor.md' '터널 인증 행은 없다' tasks` (시작 트리에는 실행 기록이 없어 exit 1) (observed: 2026-10-05 — exit 0)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-05 — exit 0)

## Attempts

- 2026-10-05 측정: 마커 교체 전 67110 bytes, 744 lines. 교체 후 67170 bytes, 744 physical lines. 둘 다 87296과 1000 아래다. 아카이브하지 않았다.
- ISSUE-488은 코디네이터가 발견·doctor 기록을 먼저 갱신했다. 이 세션은 그 파일을 쓰지 않았다.
- quality-review는 넣지 않았다. 독립 리뷰를 PASS로 적지 않는다.
- 이전 작성 스냅샷(ISSUE-490 로컬 ownership 수정 전): 기준 1–8은 exit 0, `ce task validate --all`은 546 valid / 0 invalid, `make doc-check`와 `ce task gate`는 `upstream-ref` 부재로 exit 1이라고 적었다. 그 실패와 ‘허용 경로 밖’ 문장은 그 스냅샷의 역사다. 현재 Summary와 현재 게이트가 아니다.
- Coordinator finalization: ISSUE-490 ownership is local adoption/handoff tracking, not a fabricated upstream report. Relevant doc gate failure corrected. Criteria 6 now verifies the new issue rather than depending on this card remaining todo without review. Independent review is next.
- Independent scope review found that read-only 004/006 checks do not depend on approved mutation pin; historical-card/run-all acceptance changes are Proposed in TASK-491, not silently applied.

## Attempts FAIL1

- 이유: 당시 Summary가 기준 9를 현재 실패로 적었다. 라이브 기준 9 `make doc-check`는 exit 0이었고 체크와 observed도 exit 0이었다. `ce task gate`는 READY였다. Attempts의 게이트 실패 문장은 ISSUE-490 로컬 ownership 수정 전 스냅샷이다. 영수증: `tasks/done/evidence/TASK-489/independent-review.json`. reviewerSessionId `d77d288d-995b-4840-b270-cf9715f9e8e1`, verdict FAIL. `approvalForCoordinatorMetadataFinalization`은 false다.
- 바꾼 접근: 현재 Summary는 기준 1–6·8이 HEAD에서 실패하고 기준 7·9가 지금 통과하는 회귀 가드라고 적는다. 그 Summary는 역사적 Attempts와 분리한다. 수정 뒤 현재 게이트를 다시 실행한다.
- 현재 게이트 재실행(2026-10-05, 이 Summary 수정 뒤): 기준 1–8 exit 0. 기준 9 `make doc-check` exit 0. `ce task validate --all`은 547 valid, 0 invalid. `ce task gate` exit 0, READY — task_board_ready. 작성자는 quality-review를 넣지 않았다.
- Independent grok-4.7 attempt 2 PASS; receipt [independent-review-attempt-2.json](evidence/TASK-489/independent-review-attempt-2.json). Coordinator finalized done metadata as approved by that receipt.
