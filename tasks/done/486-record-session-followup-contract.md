---
id: TASK-486
title: "Record the session follow-up operating contract"
type: docs
priority: P2
effort: M
exec-tier: standard
allowed-paths: [ce-tasks.yaml, tasks/README.md, decisions/DECISION-001-changelog-append-only-vs-size-rules.md, decisions/DECISION-003-json-schemas-ref-split-vs-rules.md, decisions/DECISION-004-whole-file-config-overages.md, decisions/README.md, tasks/issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md, tasks/issue/006-preflight-reports-needs-human-cards-as-runnable.md, tasks/issue/453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md, tasks/todo/459-implement-remote-access-tunnel.md, tasks/done/486-record-session-followup-contract.md, tasks/decision/487-changelog-half-ceiling-trigger.md, tasks/issue/488-measure-expired-tunnel-token-and-live-doctor.md, tasks/decision/README.md, tasks/done/evidence/TASK-486, tools/doccheck/cardstatus.go, tools/doccheck/cardstatus_test.go]
status: done
quality-review: pass
quality-reviewed-at: 2026-10-05
quality-review-evidence: "Independent Grok 4.7 session 01a10c09-c969-7890-a71b-7c623b0cd598 PASS for TASK-491 classification correction; tasks/done/evidence/TASK-492/scope-review.json and live-acceptance-review.json; historical TASK-489 receipts preserved; tasks/done/evidence/TASK-492/final-review.json PASS; tasks/done/evidence/TASK-492/final-default-environment-review.json supplements ordinary-shell default guards"
created: 2026-10-05
---

## Summary

세션 후속 계약을 보드에 기록한다. `tasks/README.md`의 운영 계약, 수락된
DECISION-001·003·004 본문, 사람 이슈 004·006·453·488, 미결 TASK-487,
TASK-459의 외부 분류다. 제품 루프 코드는 없다. 독립 리뷰 PASS 전에는
봉인하지 않는다.

기준 1–5는 이 카드가 생기기 전 트리에서 실패했다. 기준 6–8은 그때도
통과하는 회귀 가드다.

## Steps

1. `tasks/README.md:32` — `## 자율 실행 계약`. `:36` — `policy:model-grok-4.7`, grok(기본) 또는 grok --model grok-4.7. `:37` — grok --model grok-4.7-build-fast. `:44` — `policy:human-grade-encoding`. `:48` — 이슈 제목 needs stronger review. `:49` — `policy:commit-immediate-push`. `:54` — 실패 자체와 악화되지 않은 증거. `:55` — `policy:no-intercard-questions`. `:56` — PASS 자동 통합, 소스 push, cleanup, ce가 처리한다. `:58` — `policy:no-second-board-grader`.
2. `tasks/README.md:14` — 설치본 commit `62db34ea9291c0cab9cc03222136e8a32fcaed16`. `:15` — `1270e1dc`와 `3656558b`의 자손. `:17` — 시작 기준선(START) `23047d07`. 통합 뒤 master를 이 해시로 단정하지 않는다. `:23` — ISSUE-488. `:30` — TASK-459는 todo, TASK-486은 독립 PASS 전에 봉인하지 않는다.
3. `ce-tasks.yaml:2` — `strict-status: true` 유지. CE 내장 decision kind를 사용하며 zones와 zone-status를 선언하지 않는다.
4. `decisions/DECISION-001-changelog-append-only-vs-size-rules.md:34` — 수락된 옵션 A. `:43` — 미결은 TASK-487. 마커와 ceiling은 그대로다.
5. `decisions/DECISION-003-json-schemas-ref-split-vs-rules.md:36` — 수락된 옵션 B와 `1270e1dc47bc7f3a2421de2074b92f619e4298a7`. schema.json 1882줄, expectedExit 1, errorLimit 1000. 로더 리팩터가 아니다.
6. `decisions/DECISION-004-whole-file-config-overages.md:28` — ceiling 280과 320. 마커는 이미 적용됐다.
7. `decisions/README.md:21` — TASK-487. DECISION-005를 만들지 않는다.
8. `tasks/issue/004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md:8` — `needs-human: true`, `execution-mode: external`, `human-grade: human`. `:166` — 2026-10-05 보드 메모. W13 terminal/rollback, ISSUE-453과 같은 의존.
9. `tasks/issue/006-preflight-reports-needs-human-cards-as-runnable.md:8` — 같은 사람 플래그. `:192` — W14 잔여. `exec-tier` 없음.
10. `tasks/issue/453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md:7` — 사람 플래그. `:95` — 공개 승인 필드 일곱 개. `:108` — `mutationAuthorized` 단독 승인 금지.
11. `tasks/todo/459-implement-remote-access-tunnel.md:7` — 외부 사람 카드. `:26` — 만료 토큰 기준은 `[ ]`. `:41` — 역사적 allowed-paths. done으로 옮기지 않는다.
12. `tasks/decision/487-changelog-half-ceiling-trigger.md:10` — `status: Proposed`. `:30` — `Status: Proposed`. 선택 미적용. `tasks/issue/488-measure-expired-tunnel-token-and-live-doctor.md:25` — `## 소유권 — 이 저장소다`. `:29` — Reproduction. `:54` — cloudflared access token 서브셸. `:64` — `DVA_FILE` doctor. 외부 확인은 실행하지 않았다.
13. `tools/doccheck/cardstatus.go:37` — `tasks/decision/`를 identity zone에 넣는다. skip이 아니다. `tools/doccheck/cardstatus_test.go:41` — 그 경로를 커버한다. doc-check가 undeclared board directory를 보고한 뒤에만 고친다.
14. `tasks/decision/README.md:5` — `현재 최대: ADR-0000, 로그 487. 다음은 ADR-0001, 로그 488.` `:9` — 색인 행 status proposed. 색인은 카드가 아니다. `tasks/README.md:71` — 내장 decision kind와 Proposed. 별도 zones와 zone-status는 없다.

## Stop conditions

- CHANGELOG 마커와 ceiling `87296 bytes, 1000 lines`를 바꾸지 않는다.
- TASK-459의 사람 기준을 체크하거나 done으로 옮기지 않는다. 만료 토큰 명령을 실행하지 않는다.
- TASK-487의 세 선택 중 어느 것도 적용하지 않는다. 마커를 고치지 않는다.
- 네트워크, 인증, 비밀, JWT를 다루지 않는다. ISSUE-453을 복제하지 않는다.
- 독립 리뷰 PASS 전에 `quality-review`를 넣거나 이 카드를 봉인하지 않는다.
- 제품 루프 코드를 만들지 않는다. 두 번째 보드 채점기를 만들지 않는다.
- `exec-tier: human`을 쓰지 않는다. 사람 카드에 `allowed-paths`를 넣지 않는다.
- 독립 리뷰 PASS 뒤 코디네이터가 커밋, push, 설정된 소스 통합, cleanup을 한다. ce가 처리한다. 작성 카드는 그 전에 봉인하지 않는다.

## Completion Criteria

- [x] 운영 계약의 정책 태그와 모델·통합 문장이 있다. 카드 생성 전 이 바인딩은 exit 1이었다 | verify: `python3 -c 'import pathlib; t=pathlib.Path("tasks/README.md").read_text(); tags=["policy:model-grok-4.7","policy:build-fast-cheap-standard","policy:runnable-all-default","policy:max-n","policy:dependency-order","policy:parallel-ownership-isolated","policy:one-repo-explicit-files","policy:human-excluded","policy:human-grade-encoding","policy:no-self-review","policy:review-separate-session","policy:fail-max-3","policy:blocked-push-only-continue","policy:commit-immediate-push","policy:source-branch-master","policy:run-finish-configured-branch","policy:branch-integrate-configured-source","policy:stop-own-card-conflict","policy:preexisting-nonworsening","policy:no-intercard-questions","policy:integrate-after-review-pass","policy:final-report","policy:no-second-board-grader","needs stronger review","grok --model grok-4.7-build-fast","grok --model grok-4.7","human-grade: human","실패 자체와 악화되지 않은 증거","시작 기준선(START)","23047d07"]; missing=[x for x in tags if x not in t]; assert not missing, missing; assert "현재 master는" not in t'` (observed: 2026-10-05 — exit 0)
- [x] TASK-487은 Accepted이고 ISSUE-488의 사람 플래그는 그대로다. TASK-489가 이 기준의 Proposed 바인딩을 사실 정정했다. 카드 생성 전 이 바인딩은 exit 1이었다 | verify: `test "$(/usr/bin/find tasks -name '488-measure-expired-tunnel-token-and-live-doctor.md' -type f -print | /usr/bin/grep -c .)" -eq 1 && test "$(/usr/bin/find tasks -name '487-changelog-half-ceiling-trigger.md' -type f -print | /usr/bin/grep -c .)" -eq 1 && /usr/bin/grep -rq --include='488-measure-expired-tunnel-token-and-live-doctor.md' '^id: ISSUE-488' tasks && /usr/bin/grep -rq --include='488-measure-expired-tunnel-token-and-live-doctor.md' -F '이 저장소' tasks && /usr/bin/grep -q '^status: Accepted' tasks/decision/487-changelog-half-ceiling-trigger.md && /usr/bin/grep -q '^execution-mode: decision' tasks/decision/487-changelog-half-ceiling-trigger.md && /usr/bin/grep -q -F -- '- Status: Accepted' tasks/decision/487-changelog-half-ceiling-trigger.md && ! /usr/bin/grep -q '^status: done' tasks/decision/487-changelog-half-ceiling-trigger.md` (historical: 2026-10-05 exit 0 was the Proposed binding) (observed: 2026-10-05 — corrected binding exit 0)
- [x] 수락 본문에 대기 제안 문구가 없다. 카드 생성 전 이 바인딩은 exit 1이었다 | verify: `! /usr/bin/grep -q -F '제안 — 인간 확정 대기' decisions/DECISION-001-changelog-append-only-vs-size-rules.md && ! /usr/bin/grep -q -F '제안 — 인간 확정 대기' decisions/DECISION-003-json-schemas-ref-split-vs-rules.md && ! /usr/bin/grep -q -F '제안 — 인간 확정 대기' decisions/DECISION-004-whole-file-config-overages.md && /usr/bin/grep -q '^status: Accepted' decisions/DECISION-001-changelog-append-only-vs-size-rules.md && /usr/bin/grep -q -F '**옵션 A**' decisions/DECISION-001-changelog-append-only-vs-size-rules.md && /usr/bin/grep -q -F '**옵션 B**' decisions/DECISION-003-json-schemas-ref-split-vs-rules.md && /usr/bin/grep -q -F 'ceiling은 280' decisions/DECISION-004-whole-file-config-overages.md` (observed: 2026-10-05 — exit 0)
- [x] TASK-491 수락 후 004·006은 자동 읽기 전용 검증으로 현행화했고 453·459·487·488·491의 사람 분류는 유지한다. TASK-492가 현재 바인딩을 정정한다 | verify: `python3 -c 'import pathlib; root=pathlib.Path("tasks"); hit=lambda base: (lambda hs: hs[0] if len(hs)==1 else (_ for _ in ()).throw(AssertionError(base)))(list(root.rglob(base))); human=[("453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md","external"),("459-implement-remote-access-tunnel.md","external"),("488-measure-expired-tunnel-token-and-live-doctor.md","external"),("487-changelog-half-ceiling-trigger.md","decision"),("491-queue-acceptance-scope-and-historical-evidence.md","decision")]; assert all("needs-human: true" in (fm:=hit(b).read_text().split("---",2)[1]) and "execution-mode: "+m in fm and "human-grade: human" in fm and "exec-tier:" not in fm and "allowed-paths:" not in fm for b,m in human); auto=["004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md","006-preflight-reports-needs-human-cards-as-runnable.md"]; assert all("needs-human:" not in (fm:=hit(b).read_text().split("---",2)[1]) and "exec-tier: standard" in fm and "ownership: local" in fm for b in auto); body=hit("453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md").read_text(); assert all(x in body for x in ["source commit/tree","artifact identity","build toolchain","build command","artifact SHA-256","platform/human approval","mutationAuthorized"])'` (historical: TASK-486 original binding required human flags for 004/006 before TASK-491 acceptance) (observed: 2026-10-05 — exit 0)
- [x] CE 내장 decision kind를 사용하며 487은 Accepted다. 새 카드가 없던 기준선에서 이 바인딩은 실패한다. TASK-489가 Proposed 바인딩을 사실 정정했다 | verify: `! /usr/bin/grep -q 'zones:' ce-tasks.yaml && ! /usr/bin/grep -q 'zone-status:' ce-tasks.yaml && test -f tasks/decision/487-changelog-half-ceiling-trigger.md && /usr/bin/grep -q '^status: Accepted' tasks/decision/487-changelog-half-ceiling-trigger.md && test -f tasks/decision/README.md` (historical: 2026-10-05 exit 0 was the Proposed binding) (observed: 2026-10-05 — corrected binding exit 0)
- [x] CHANGELOG 첫 마커는 실제 ceiling 트리거이고 절반 문장은 없다. TASK-489가 옛 마커 바인딩을 사실 정정했다 | verify: `/usr/bin/grep -q -F '<!-- size-limit: exempt, ceiling: 87296 bytes, 1000 lines -- append-only changelog은 분할 시 히스토리가 아니라 미래가 바뀐다 (DECISION-001). 문서화된 트리거: 릴리스에서 바이트가 87296 이상이거나 물리 줄이 1000 이상이면 v0.2.x 이하를 아카이브한다. -->' CHANGELOG.md && ! /usr/bin/grep -q -F 'ceiling 절반(44KB)' CHANGELOG.md` (historical: 2026-10-05 exit 0 was the half-ceiling marker binding, then labeled regression-guard) (observed: 2026-10-05 — corrected binding exit 0)
- [x] TASK-459의 사람 기준 두 줄이 미체크다 | verify: `/usr/bin/grep -rq -F --include='459-implement-remote-access-tunnel.md' -e '- [ ] 만료된 토큰에서' tasks && /usr/bin/grep -rq -F --include='459-implement-remote-access-tunnel.md' -e 'verify: human — run dva doctor on a tunnel config' tasks && /usr/bin/grep -rq -F --include='459-implement-remote-access-tunnel.md' -e 'verify: human — docs/68 §7' tasks` (regression-guard) (observed: 2026-10-05 — exit 0)
- [x] strict-status가 유지된다 | verify: `/usr/bin/grep -q -F 'strict-status: true' ce-tasks.yaml` (regression-guard) (observed: 2026-10-05 — exit 0)
- [x] 문서 게이트가 통과한다 | verify: `make doc-check` (regression-guard) (observed: 2026-10-05 — exit 0)

## Attempts

- Coordinator verification: validate was 544 valid/0 invalid, but gate rejected declaring decision under card-dialect.zones as a legacy parking zone. Current CE has a native decision kind (lint_decision_directory_test.go). Removed only the redundant declaration; kept strict-status, decision cards and index, and DVA identity coverage. No exclusions or weakened gates.

- 첫 작성 턴은 발견이 길어져 최대 턴에서 끊겼다. 이어서 구현만 마쳤다. CE 구현 소스는 더 읽지 않았다.
- 보이는 최대 카드 번호는 485였다. 486·487·488은 그 다음 빈 번호다. 제품 루프 코드는 만들지 않았다. 터널 명령은 실행하지 않았다. 커밋, push, 배포, CI, quality-review는 없다.
- HEAD 기준선: `policy:model-grok-4.7`는 exit 1, `tasks/decision/487-…`와 `tasks/issue/488-…`는 `git cat-file -e` exit 128, `zones: [decision]`는 exit 1, `제안 — 인간 확정 대기`는 DECISION-001에 있어 exit 0이라 부정 바인딩은 기준선에서 실패한다. `strict-status: true`, CHANGELOG 마커, TASK-459의 `- [ ] 만료된 토큰에서`은 HEAD에서 exit 0이라 회귀 가드다.
- 현재 바인딩 1–8은 파서로 실행해 exit 0이다. `make doc-check`는 그 다음이다. 로그는 `tmp/session-followup/bindings-before-doc-check.txt`.
- 그 전 `make doc-check`는 `tasks/decision/` undeclared 1과 ISSUE-488 소유권 불일치 1로 실패했다. `tools/doccheck/cardstatus.go`에 skip이 아닌 `tasks/decision/`을 넣었고, 제목을 `## 소유권 — 이 저장소다`로 고쳤다. 재실행은 exit 0, undeclared 0, ownership_mismatched 0. 로그는 `tmp/session-followup/doc-check.txt`.
- Steps의 487 앵커는 줄 9가 `human-grade`라 줄 10으로 고쳤다. 그 다음 `ce task validate --all`은 4건이 실패했다. 487 frontmatter `todo`는 decision Status로 거부됐다. `tasks/decision/README.md`가 없었다. ISSUE-488에 severity·discovered·Reproduction·Expected vs Actual가 없었다. 486 바인딩이 `tasks/issue/`와 `tasks/todo/` 경로를 그대로 적었다. `zone-status.decision: todo`를 뺐다. frontmatter는 Proposed다. 색인을 추가했다. 바인딩은 basename `find`/`grep --include`와 `rglob`이다. 다시 실행한 기준 1–8은 exit 0이다. 로그는 `tmp/session-followup/bindings-validator-fix.txt`, `tmp/session-followup/doc-check-2.txt`. 최대 턴 중단은 발견 구간의 중단이지 리뷰 FAIL이 아니다.
- 2026-10-05T04:56:50Z independent review PASS. Receipt: [independent-review.json](evidence/TASK-486/independent-review.json).
