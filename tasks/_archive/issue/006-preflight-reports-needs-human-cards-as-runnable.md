---
id: ISSUE-006
title: "Preflight reports needs-human cards as runnable"
type: bug
status: done
quality-review: pass
quality-reviewed-at: 2026-10-05
quality-review-evidence: "Independent Grok 4.7; tasks/done/evidence/TASK-492/live-acceptance-review.json; accepted TASK-491 read-only scope only"
priority: P2
effort: S
severity: medium
discovered-in: "2026-09-13 task:run-all loop termination"
discovered-at: 2026-09-13
ownership: local
exec-tier: standard
allowed-paths: [tasks/issue/006-preflight-reports-needs-human-cards-as-runnable.md, tasks/done/evidence/ISSUE-006]
created: 2026-09-13
upstream-ref: "task-manager-devbox task/list.md W14"
resolution: fixed
resolved-at: 2026-10-05T13:01:09Z
resolution-summary: "Resolved as fixed by TASK-492."
---

## Summary

`ce task preflight` decides runnability from card structure — criteria present,
`verify:` bindings present, no blocking advisories — and never from whether a
runner exists that can satisfy them. A card whose every remaining criterion is
`verify: human — …` is structurally perfect, so it is reported runnable to an
agent that cannot run it.

At discovery, DVA's queue was entirely this shape. `ce task preflight --zone
doing,todo --json` returns `verdict: READY`, `total: 4`, `runnable: 4`,
`unrunnable: 0`. All four cards carry `needs-human: true` in frontmatter, and
their remaining criteria are human-bound. Agent-runnable work is zero.

The information preflight would need is already in front of it, twice. It reads
the frontmatter, where `needs-human: true` is declared. It parses the `verify:`
bindings, since it counts them (`criteria: 3, bound: 3`) — and the `human —`
prefix is in the text being counted. Neither signal reaches the runnable
verdict, and the per-card record it emits carries no field for either.

The original report target is external (ce-agent-kit), but a current selector
owner and CLI route have not been confirmed after the historical controller was
retired. Filed here on the ISSUE-004 precedent — including that card's
misattribution, corrected on 2026-09-15. It is closely related to [[ISSUE-004]],
which is the same gap seen from the other side — that card is about admission
refusing human-only work, this one is about runnability admitting it.

## Reproduction

1. Run `ce task preflight --zone doing,todo --json` in this repository.
2. Observe `verdict: READY`, `runnable: 4`, `unrunnable: 0`.
3. `grep -l 'needs-human: true' tasks/todo/*.md` returns all four of those cards.
4. Read the emitted `cards[]` records: no field reports `needs-human` or the
   human binding, though `bound` counts the very lines that carry it.

## Expected vs Actual

- Expected: a card an agent cannot advance is not counted toward the runnable
  total, and the verdict distinguishes "nothing left" from "nothing an agent
  can do".
- Actual: READY with 4 runnable, while agent-runnable work is 0.

## Impact

An automated loop that trusts the verdict — which is the loop's contract with
preflight — cannot terminate on it. It reaches READY, finds no card it can
advance, and must either stop on a judgement preflight did not authorize or
attempt human-only work. This session terminated on its own reading of the
cards instead of on the verdict, which is exactly the coupling preflight exists
to remove.

The alternative failure is worse than a wrong count: a loop that resolves READY
by attempting the work fabricates evidence for a human-bound criterion.

## Recommended Resolution

Report the two populations separately rather than reclassifying. Human-only
cards are not unrunnable — a human runs them, and a human-facing queue should
still list them. Collapsing them into `unrunnable` would trade this defect for
its mirror image. A distinct count (`runnable` vs `agent-runnable`, or an
explicit `needs_human` on each card record) lets each consumer read what it
needs, and lets an agent loop terminate on the verdict.

`needs-human: true` in frontmatter should be the discriminator, since it is
declared rather than inferred; the `verify: human —` prefix is corroborating
evidence and a reasonable source for a warning when the two disagree.

## 소유권 — 이 저장소다

2026-10-05 TASK-491 옵션 1 이후 남은 일은 이 저장소의 읽기 전용 채택 확인과 증거다.
`allowed-paths`는 이 카드와 `tasks/done/evidence/ISSUE-006`뿐이다. 생산자 구현은 ISSUE-490이다.
`upstream-ref`는 2026-09-15 보고의 역사 포인터다. 새 상류 보고가 아니다.

## 과거 소유권 기록 (2026-09-15)

preflight는 `ce-agent-kit` 소유다 — Summary가 "Owner is external"로 적는다.
기준 전부가 상류가 needs-human 카드를 runnable과 구분해 내는 동작을 요구한다.
보고는 [[TASK-399]]가 `ce-agent-kit#1`로 수행했다.

**2026-09-15 정정 — 상류는 `ce-agent-kit`이다.** 이 절과 위 본문은 원래
`ce-workbook/task_management`를 지목했고, 그게 실측과 어긋났다([[ISSUE-027]]).
실행되는 `ce`의 빌드정보가 자기 모듈을 직접 증언하고, ce-workbook에는 추적되는
구현이 없다:

```
$ go version -m "$(command -v ce)"
	path	github.com/archmagece/ce-agent-kit/cmd/ce
	mod	github.com/archmagece/ce-agent-kit	v0.8.5-...

$ git -C ~/mywork/ce/ce-workbook ls-files | grep -i preflight | grep -v '^tasks/'
(없음)
```

등록 지점은 `cmd/ce/handlers_task.go`, 구현은
`internal/adapter/cli/commands/task_preflight.go`로 **같은 저장소 안에 함께** 있다.
ce-workbook의 `tmp/.../queue_preflight.py`는 `.gitignore`된 미추적 스크래치 파일이고,
`task_management/USAGE.md`의 "기존 Python preflight"는 그 문서가 같은 자리에서 "새
lifecycle 규칙은 ce-agent-kit에서만 작성"이라고 적는 별개 개념이다. 보고처는
`ssh://git@gitlab.polypia.net:2224/archmagece/ce-agent-kit.git`다.

## Resolution Criteria

- [x] 기존 회귀가 agent와 human 집합을 나누고 `human_required`와 `empty`를 종료 상태로 둔다. 사람 카드가 agent 집합에 들어가면 거부한다 | verify: `go test -count=1 -run '^(TestClassifyStates|TestParseQueueRejectsInvalidInvariants)$' ./internal/taskqueue/` (observed: 2026-10-05 — exit 0) (regression-guard)
- [x] 명시된 제품 checkout의 바이너리로 현재 보드를 읽어 사람 필드·집합·Accepted 제외를 확인한다. 기본 PATH 설치를 바꾸지 않는다 | verify: `taskchain_reader_repo="${TASKCHAIN_PRODUCT_REPO:-$HOME/mydevbox/task-manager-devbox/taskchain-task-manager}"; test -d "$taskchain_reader_repo" && test -x "$taskchain_reader_repo/build/taskchain-task-manager" && PATH="$taskchain_reader_repo/build:$PATH" dva task-queue | python3 -c 'import json,sys; d=json.load(sys.stdin); assert d["runnableCount"]==len(d["runnable"]) and d["agentRunnableCount"]==0 and d["agentRunnable"]==[]; assert all(x["needsHuman"] is True and x["executionMode"] in ("external","decision") and x["allowedPaths"]==[] for x in d["runnable"]); assert not {x["card"]["id"] for x in d["runnable"]}.intersection({"TASK-487","TASK-491"})'` (observed: 2026-10-05 — exit 0)
- [x] 같은 명시 바이너리의 읽기 전용 verdict가 human_required/empty이며 agent 후보가 없다. CE start·terminal 검사는 아니다 | verify: `taskchain_reader_repo="${TASKCHAIN_PRODUCT_REPO:-$HOME/mydevbox/task-manager-devbox/taskchain-task-manager}"; test -d "$taskchain_reader_repo" && test -x "$taskchain_reader_repo/build/taskchain-task-manager" && PATH="$taskchain_reader_repo/build:$PATH" dva task-queue-verdict | python3 -c 'import json,sys; d=json.load(sys.stdin); assert d["state"] in ("human_required","empty") and d["agentRunnableCount"]==0 and d["candidate"] is None'` (observed: 2026-10-05 — exit 0)



## Historical acceptance

2026-10-05 TASK-491 옵션 1이 수락됐다. 아래는 발견 당시 문장이다. 활성 확인 명령이 아니다. 당시의 네 카드와 run-all은 현재 보드 사실이 아니다. 합성 fixture와 과거 0/0 기록은 이 이슈를 닫지 않는다.

- [ ] Preflight distinguishes agent-runnable cards from human-only cards in its verdict and its per-card records
- [ ] A DVA preflight run reports agent-runnable 0 for the current four cards
- [ ] An automated loop can terminate on the verdict alone

과거 확인 문장: upstream tests cover a queue that is entirely human-only. fresh output from this repository is linked here. the run-all skill's exit condition reads the new field, not the card bodies.

## 후속 (2026-09-24)

판정은 여전히 `needs-human`을 보지 않는다. 작업은
[TASK-423](../2026-09/423-separate-needs-human-from-agent-runnable.md)가 소유한다.

## 소유권 재측정 (2026-09-27)

ce-agent-kit#1은 TASK-399의 역사적 보고처이며 현행 구현 owner로 확인되지 않았다.
TASK-423은 TASK-421/ISSUE-040의 controller owner와 disposition 계약 결정에 종속된다.
제품 결정 전에는 needs-human 기준의 구현 repository나 CLI route를 가정하지 않는다.

## Owner (2026-09-27)

The product owner assigned this to taskchain-task-manager, tracked as W14 in
`~/mydevbox/task-manager-devbox/task/list.md`. TASK-423 was archived as handed off;
this issue stays open until W14 ships and DVA adopts it.

## Dependency correction (2026-09-27)

The earlier TASK-421 dependency describes the pre-handoff state. The current
`task-manager-devbox` W14 row declares no W13 dependency; W14 can proceed independently.

## 2026-09-28 제품 구현

taskchain-task-manager `78ed463`이 읽기 전용 `queue`에 `runnable`과
`agentRunnable` 항목·수를 분리했다. `needs-human: true`와 YAML merge
입력을 검증하고 사람 전용 카드만 남았을 때 `agentRunnableCount: 0` 및
`agentRunnable: []`를 확인했다. 독립 리뷰 PASS, 제품 `make check`
(race taskstore 1019초)와 통합 사전검사를 통과했고 `master/origin`에
통합·push한 뒤 작업 브랜치·worktree를 회수했다. 이 이슈는 DVA 소비자가
새 큐를 채택해 실제 preflight 판정을 교체할 때까지 열린다.

## 2026-09-28 DVA 조회 채택

W07 부분 작업 [[TASK-449]]에서 `dva task-queue`가 제품 `queue`의
`runnable`/`agentRunnable` 두 집합을 변형 없이 노출한다. 제품
`9e8fac7`의 로컬 빌드 SHA-256은
`d2ca31f2880a1420efef3503dba2333f62c0ddd84648b06f4c75b5a62ad6ecd0`이다.
합성 fixture의 사람 전용 작업 두 개는 `runnableCount: 2`,
`agentRunnableCount: 0`, `agentRunnable: []`였고 혼합 fixture는 3/1이었다.
현재 DVA에는 이전의 네 todo 카드가 없으며 P1/P2 issue 두 개만 남아
실제 조회는 0/0이다. 이 0/0은 사람 전용 양성 사례가 아니므로 fixture와
구분해 기록한다. 직접 CLI와 DVA interaction의 출력·종료코드 일치,
보드 digest 불변 및 lock 부재도 확인했다.

위 Resolution Criteria의 자동 run-all 종료 조건은 아직 미충족이다. 이
저장소에는 새 큐를 해석하는 agent loop가 없고, 읽기 전용 interaction은
카드를 선택·claim·전이하지 않는다. W07의 실제 실행 소비자와 rollback
검증이 완료될 때까지 이 이슈는 열린 상태로 둔다.

## 2026-09-28 읽기 전용 verdict

[[TASK-450]]의 `dva task-queue-verdict`는 두 집합을 검증한 뒤
`empty`·`human_required`·`candidate`·`selection_required`를 구분한다.
사람용 `runnable`은 버리지 않으며 후보 한 건도 실행·claim·완료로 간주하지 않는다.
현재 보드의 실측은 0/0 `empty`다. 사람 전용 종료와 자동 루프의 terminal
계약은 아직 검증되지 않아 이 이슈는 열린 상태다.

## 2026-09-28 단일 후보 시작 브리지

[[TASK-451]]은 명시적인 CE branch type과 정확히 한 agent `candidate`가
있을 때만 `ce task run-start`로 연결한다. `human_required`와 다중 후보는
CE를 호출하지 않고, CE의 비성공 JSON도 복구 증거로 노출한다. 현재 실제
보드는 0/0이어서 시작하지 않는 경로만 실측했다. 자동 루프의 종료·재시도,
사람 전용 terminal은 W07b2b 실제 host 검증 전까지 열린다.

## 2026-09-28 CE host 증거의 범위

[[TASK-452]]는 CE의 실제 start/discard/refusal/retry를 확인했지만 현재
DVA 보드에는 사람 전용 runnable 카드가 없다. `human_required`에서 CE를
호출하지 않는 것은 합성 integration fixture가 확인했다. 자동 루프의
종료 판정과 사람 전용 카드 terminal은 아직 실제 host에서 확인되지 않아
이 이슈는 열린다.

## 2026-10-05 보드 메모

W14의 큐 분류, verdict, start는 이미 구현됐다. 남은 일은 사람 terminal과
rollback이다. 그 증거는
[ISSUE-453](../../issue/453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md)의
공개 산출물 승인과 같은 의존이다. 합성 fixture는 실제 사람 호스트 검증이
아니다. 위 Resolution Criteria는 체크하지 않는다. 이 절은 2026-10-05 보드
상태이며 새 호스트 테스트가 아니다.

## 읽기 전용 안내

네트워크와 인증을 실행하지 않는다. 비밀과 JWT를 출력하지 않는다. 현재 기준의
읽기 전용 확인에는 사람 허가가 필요 없다. `allowed-paths`는 이 카드와
`tasks/done/evidence/ISSUE-006`뿐이다. 생산자 구현은 ISSUE-490이다. 생산자가
준비되면 그 확인은 에이전트가 실행한다. 공개 pin 승인은 ISSUE-453이다.
`exec-tier: standard`. CE exec-tier는 cheap, standard, strong만 받는다.

## 현재 검증 경로 (TASK-491 수락 후)

TASKCHAIN_PRODUCT_REPO는 TASK-493이 통합된 제품 checkout을 명시한다. 그 저장소의
make build 출력 build/taskchain-task-manager를 서브프로세스 PATH 앞에 한 번만 둔다.
전역 설치나 pin은 바꾸지 않는다. 기본 PATH 설치본의 native decision 실패는
ISSUE-490에 남고, 공개 artifact/pin 승인은 ISSUE-453이다. 명시 읽기 전용 조회와
승인이 필요한 writer 채택을 같은 조건으로 묶지 않는다.

## 현재 해결 (2026-10-05)

독립 Grok 4.7이 clean source dd3ec0a 바이너리의 실제 DVA 읽기 전용 queue와
verdict를 재실행해 PASS했다. 사람 TASK-459 한 장, agent 후보 0, human_required,
Accepted 487·491 제외와 scope 회귀가 모두 종료 코드 0이다. 수락된 TASK-491
범위로 해결한다. source 통합 중인 TASK-493은 doing이며 완료로 취급하지 않는다.
기본 설치본 교체(ISSUE-490), 공개 pin 승인(ISSUE-453), CE start/terminal 검증은
이 해결에 포함되지 않는다. 역사적 기준·보고는 위 기록 그대로 보존한다.
