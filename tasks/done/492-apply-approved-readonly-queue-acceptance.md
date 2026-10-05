---
id: TASK-492
title: "Apply the approved read-only queue acceptance"
type: docs
priority: P2
effort: S
exec-tier: standard
allowed-paths: [tasks/doing/494-allow-local-only-issue-corpus.md, tasks/done/494-allow-local-only-issue-corpus.md, tasks/done/evidence/TASK-494, tasks/README.md, tasks/decision/491-queue-acceptance-scope-and-historical-evidence.md, tasks/decision/README.md, tasks/done/486-record-session-followup-contract.md, tasks/done/evidence/TASK-492, tasks/done/evidence/TASK-493, tasks/issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md, tasks/issue/006-preflight-reports-needs-human-cards-as-runnable.md, tasks/_archive/issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md, tasks/_archive/issue/006-preflight-reports-needs-human-cards-as-runnable.md, tasks/issue/453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md, tasks/issue/490-selected-taskchain-rejects-native-decision-directory.md, tasks/_archive/issue/490-selected-taskchain-rejects-native-decision-directory.md, tasks/todo/492-apply-approved-readonly-queue-acceptance.md, tasks/done/492-apply-approved-readonly-queue-acceptance.md, tasks/todo/493-support-native-decision-queue-kind.md, tasks/doing/493-support-native-decision-queue-kind.md, tasks/done/493-support-native-decision-queue-kind.md]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-05
quality-review-evidence: "Independent Grok 4.7 session 01a10c09-c969-7890-a71b-7c623b0cd598; tasks/done/evidence/TASK-492/scope-review.json and live-acceptance-review.json PASS; prior FAIL preserved; tasks/done/evidence/TASK-492/final-review.json PASS; tasks/done/evidence/TASK-492/final-default-environment-review.json supplements ordinary-shell default guards"
created: 2026-10-05
---

## Summary

2026-10-05에 사용자가 TASK-491 옵션 1을 수락했다. ISSUE-004·006의 현재 기준을 읽기 전용 분류, 엄격한 구현 범위, verdict의 `human_required`/`empty`로 바꾼다. 과거 기준은 Historical acceptance에 활성 verify 없이 남긴다. 공개 pin 승인은 ISSUE-453이다. 기존 Go 테스트는 회귀 가드다. 이 카드의 완료 기준은 그 테스트가 아니다.

2026-10-05 실제 `dva task-queue`와 `dva task-queue-verdict`는 exit 1이다. stderr는 `queue: unsupported task directory at board root: decision`이다. 그 실패가 현재 조회 기록이다. 생산자 수정은 이 카드 밖이며, 이후 조회 성공은 루트 갱신으로 남긴다. 두 칸은 체크하지 않는다. 작성 시점에는 독립 리뷰 전이라 todo다. 완료 기준은 그 상태나 리뷰 필드에 기대지 않는다.

## Steps

1. `tasks/decision/491-queue-acceptance-scope-and-historical-evidence.md:10` — `status: Accepted`. `:35` — `- Status: Accepted`. `:37` — 2026-10-05 옵션 1 명시 수락. pin 활성화는 아니다. `:70` — 해결 기준은 `[x]`.
2. `tasks/decision/README.md:5` — decision 로그 최대는 491, 다음은 492다. 검증기가 센 결정 로그이며 전체 task id가 아니다. `:10` — TASK-491 status `accepted`.
3. `tasks/issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md:11` — `ownership: local`. `:12` — `exec-tier: standard`. `:13` — allowed-paths는 이 카드와 `tasks/done/evidence/ISSUE-004`뿐이다. `:100` — 현재 Resolution Criteria. `:106` — Historical acceptance. `needs-human`, `execution-mode`, `human-grade`는 없다.
4. `tasks/issue/006-preflight-reports-needs-human-cards-as-runnable.md:11` — `ownership: local`. `:12` — `exec-tier: standard`. `:13` — allowed-paths는 이 카드와 `tasks/done/evidence/ISSUE-006`뿐이다. `:111` — 현재 Resolution Criteria는 `TestClassifyStates`와 `TestParseQueueRejectsInvalidInvariants`다. `:117` — Historical acceptance. 사람 플래그는 없다.
5. `tasks/README.md:21` — 004·006은 수락된 읽기 전용 범위다. 실제 조회는 ISSUE-490, pin은 ISSUE-453. `:31` — 열린 todo에 TASK-492. `:33` — TASK-491 Accepted. ISSUE-488의 대상 부재 문장은 유지한다. 만료를 단정하지 않는다.

6. `tasks/done/486-record-session-followup-contract.md:59` — TASK-491 수락으로 바뀐 004·006의 현재 분류 검증식을 정정한다. 역사적 Steps·Attempts와 이전 영수증은 보존하고 독립 재검증을 받는다.

7. `tasks/issue/453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md:119`, ISSUE-490 현재 보정 — 확인된 생산자와 미승인 공개 절차를 분리한다. TASK-493 metadata/evidence는 제품 payload를 복제하지 않고 추적한다. 실제 읽기 전용 검증 뒤 004·006·490을 정리하고 README에 기본 PATH 설치본의 잔여를 명시한다.

## Stop conditions

- 제품 소스, pin, 자격 증명, 네트워크 작업을 하지 않는다. 생산자 저장소를 다시 찾지 않는다.
- daemon, run-all, 두 번째 보드 채점기를 만들지 않는다. terminal 전이를 검증했다고 적지 않는다.
- 기존 fixture나 과거 0/0만으로 ISSUE-004·006을 닫지 않는다. 실제 queue·verdict 칸은 exit 1인 동안 체크하지 않는다.
- 제품 payload는 TASK-493이 별도 저장소에서 처리한다. DVA 보드 코디네이터는 확인된 생산자와 공개 승인·기본 PATH 채택의 잔여 안내를 현행화할 수 있다. TASK-488의 인증 입력을 추정하지 않는다.
- 작성자는 `quality-review`를 넣지 않는다. 독립 리뷰 PASS를 주장하지 않는다.
- 이 작성 세션은 커밋과 push를 하지 않는다.

## Completion Criteria

- [x] TASK-491이 Accepted이고 2026-10-05 옵션 1 수락이 색인에 있다. 변경 전 트리는 Proposed라 이 바인딩이 exit 1이다 | verify: `/usr/bin/grep -qx 'status: Accepted' tasks/decision/491-queue-acceptance-scope-and-historical-evidence.md && /usr/bin/grep -q -F -- '- Status: Accepted' tasks/decision/491-queue-acceptance-scope-and-historical-evidence.md && /usr/bin/grep -q -F '2026-10-05에 사용자가 옵션 1을 명시적으로 수락했다' tasks/decision/491-queue-acceptance-scope-and-historical-evidence.md && /usr/bin/grep -q -F '491-queue-acceptance-scope-and-historical-evidence.md) | Queue acceptance scope and current evidence | accepted |' tasks/decision/README.md && /usr/bin/grep -q -F '현재 최대: ADR-0000, 로그 491. 다음은 ADR-0001, 로그 492.' tasks/decision/README.md` (observed: 2026-10-05 — exit 0)
- [x] ISSUE-004·006의 현재 기준이 기존 테스트 이름과 실제 조회 명령이고, 현재 소유 제목이 이 저장소다. 변경 전 기준은 human 바인딩이라 이 바인딩이 exit 1이다 | verify: `/usr/bin/grep -rq -F --include='004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md' 'TestSafeAllowedPath' tasks && /usr/bin/grep -rq -F --include='004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md' '## Historical acceptance' tasks && /usr/bin/grep -rq -F --include='004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md' '## 소유권 — 이 저장소다' tasks && /usr/bin/grep -rq -F --include='006-preflight-reports-needs-human-cards-as-runnable.md' 'TestClassifyStates' tasks && /usr/bin/grep -rq -F --include='006-preflight-reports-needs-human-cards-as-runnable.md' 'TASKCHAIN_PRODUCT_REPO' tasks && /usr/bin/grep -rq -F --include='006-preflight-reports-needs-human-cards-as-runnable.md' '## 소유권 — 이 저장소다' tasks && ! /usr/bin/grep -rq -F --include='004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md' '| verify: human' tasks && ! /usr/bin/grep -rq -F --include='006-preflight-reports-needs-human-cards-as-runnable.md' '| verify: human' tasks` (observed: 2026-10-05 — exit 0)
- [x] ISSUE-004·006 frontmatter에 사람 플래그가 없고 읽기 전용 범위가 pin 승인을 요구하지 않는다. 변경 전 frontmatter는 needs-human이라 이 바인딩이 exit 1이다 | verify: `! /usr/bin/grep -rq -E --include='004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md' '^needs-human:' tasks && ! /usr/bin/grep -rq -E --include='006-preflight-reports-needs-human-cards-as-runnable.md' '^needs-human:' tasks && ! /usr/bin/grep -rq -E --include='004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md' '^execution-mode:' tasks && ! /usr/bin/grep -rq -E --include='006-preflight-reports-needs-human-cards-as-runnable.md' '^execution-mode:' tasks && ! /usr/bin/grep -rq -E --include='004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md' '^human-grade:' tasks && ! /usr/bin/grep -rq -E --include='006-preflight-reports-needs-human-cards-as-runnable.md' '^human-grade:' tasks && /usr/bin/grep -rq -F --include='004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md' 'exec-tier: standard' tasks && /usr/bin/grep -q -F '공개 pin 승인은 ISSUE-453과 별개다' tasks/README.md` (observed: 2026-10-05 — exit 0)
- [x] 적용 기록이 보드에 있다. 카드가 어느 존에 있는지나 리뷰 필드 유무에는 기대지 않는다. 변경 전 트리에는 이 기록이 없다 | verify: `/usr/bin/grep -rq -F --include='492-apply-approved-readonly-queue-acceptance.md' 'id: TASK-492' tasks && /usr/bin/grep -rq -F --include='492-apply-approved-readonly-queue-acceptance.md' 'unsupported task directory at board root: decision' tasks && /usr/bin/grep -q -F '492-apply-approved-readonly-queue-acceptance.md' tasks/README.md && /usr/bin/grep -q -F '대상 입력이 없어 만료 토큰 명령은 실행하지 않았다' tasks/README.md` (observed: 2026-10-05 — exit 0)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-05 — exit 0)

## Attempts

- 2026-10-05 기준 1–4와 `make doc-check` 이전: 바인딩 c2–c4 exit 1. `make doc-check` exit 2. 원인은 ISSUE-004·006의 `ownership: local`과 `## 소유권 — 상류다` 불일치. `ce task validate --all`은 547 valid, 2 invalid. TASK-492는 존 경로를 직접 적은 verify, decision README는 로그 492/493이라 검증기 corpus 491/492와 어긋났다. `ce task gate` exit 1, `task_validate_failed`.
- 2026-10-05 수정 뒤: 바인딩 c1–c4 exit 0. `make doc-check` exit 0. `ce task validate --all`은 548 valid, 1 invalid. 남은 invalid는 `tasks/todo/493-support-native-decision-queue-kind.md`의 `Invalid type "fix" for task`다. 그 카드는 이 작업의 허용 경로가 아니라 고치지 않았다. TASK-492 자체는 valid. `ce task gate` exit 1, 같은 493 오류, `NOT READY — task_validate_failed`. 이 세션은 quality-review, 커밋, push를 하지 않았다.
- 같은 날 읽기 전용 조회: `go test` 두 명령 exit 0. `dva task-queue` exit 1, `dva task-queue-verdict` exit 1. 공통 stderr는 `queue: unsupported task directory at board root: decision`. ISSUE-004·006의 그 두 칸은 미체크다.
- Coordinator correction: TASK-486 is unsealed; amended only current criterion 4 for accepted TASK-491 classification. Historical Steps/Attempts and old review receipts preserved. Independent final review will cover this correction.

- Independent scope review attempt 1: FAIL (Grok 4.7 session 01a10c09-c969-7890-a71b-7c623b0cd598). Coordinator ISSUE-453 text wrote a parent-devbox card as a local inline tasks path, so make doc-check failed. Receipt scope-review-attempt-1.json is preserved. Retry 2 names the external owner and filename without pretending the parent card belongs to this board; updated coordinator scope paths cover the actual DVA metadata edits.

- Independent scope retry 2 PASS: all current TASK-492 bindings, make doc-check, ce task validate --all (549/0), ce task gate READY exit 0. This completes scope application, not live ISSUE-004/006 closure; that follows product TASK-493.

- Coordinator strengthens the live ISSUE-004/006 criteria with an existence-guarded explicit product checkout, JSON human/agent set and terminal verdict assertions. Scope criterion 2 now checks that explicit selector contract; historical first PATH failures remain. Final independent review will cover live evidence and any resolutions.

- Independent live acceptance PASS: actual clean committed-source reader SHA5163c87e..., existence-guarded query and verdict assertions, both scope regressions, current 492/486 bindings, doc-check, validate and gate all exit0. Reviewer explicitly permits narrow-scope resolution of ISSUE004/006 while public pin/default install and active TASK493 integration remain outside that acceptance.

- Final ordinary-shell gate initially failed because TASK-493 required an unset product variable. Mechanical correction supplies the already validated explicit user workspace as a default and keeps existence guards. Final independent review and ordinary-shell gate must pass; no hidden environment-only readiness claim.

- Final ordinary-environment supplement attempt 1 FAIL: archived ISSUE-004/006 guards had a default but PATH still used the unset override variable. Receipt final-default-environment-review-attempt-1.json preserved. Retry 2 changes approach: resolve one task-specific shell variable, then use that same value for existence, executable and PATH checks. All four archived live bindings must pass with the override explicitly unset, alongside normal ce task gate.

- Final ordinary-environment retry 2 independent PASS: archived 004/006 query/verdict bindings (4), TASK-493 guards (2), and ce task gate all exit0 with TASKCHAIN_PRODUCT_REPO unset in a non-login shell. Earlier FAIL and all previous review receipts preserved.

- Commit CI fae87b9a568781782aedacc065bef1d6 failed a live-corpus upstream issue quota after legitimate 004/006 local ownership and archival. TASK-494 handles the test-only correction; no issue is falsely kept upstream to satisfy the old assertion.
