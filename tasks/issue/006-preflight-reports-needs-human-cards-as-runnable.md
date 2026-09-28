---
id: ISSUE-006
title: "Preflight reports needs-human cards as runnable"
type: bug
status: todo
priority: P2
effort: S
exec-tier: standard
severity: medium
discovered-in: "2026-09-13 task:run-all loop termination"
discovered-at: 2026-09-13
ownership: upstream
created: 2026-09-13
upstream-ref: "task-manager-devbox task/list.md W14"
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

## 소유권 — 상류다 (2026-09-15 명시)

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

- [ ] Preflight distinguishes agent-runnable cards from human-only cards in its verdict and its per-card records | verify: human — upstream tests cover a queue that is entirely human-only
- [ ] A DVA preflight run reports agent-runnable 0 for the current four cards | verify: human — fresh output from this repository is linked here
- [ ] An automated loop can terminate on the verdict alone | verify: human — the run-all skill's exit condition reads the new field, not the card bodies

## 후속 (2026-09-24)

판정은 여전히 `needs-human`을 보지 않는다. 작업은
[TASK-423](../_archive/2026-09/423-separate-needs-human-from-agent-runnable.md)가 소유한다.

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
