---
id: ISSUE-005
title: "ce task run-* receipts never reach a terminal state"
type: bug
status: done
priority: P2
effort: M
exec-tier: standard
severity: low
discovered-in: "2026-09-13 board currentization review"
discovered-at: 2026-09-13
ownership: upstream
created: 2026-09-13
upstream-ref: "ce-agent-kit#2"
resolution: fixed
resolved-at: 2026-09-24T12:42:44Z
resolution-summary: "Resolved as fixed."
---

## Summary

`ce task run-finish` retires a worktree and reports integration success, but the
execution receipt it was retiring stays in the run registry unchanged. The
registry therefore accumulates entries that describe work already integrated,
and `ce task run-list` refuses to report anything at all because it reads that
accumulation as blocked executions. The lifecycle commands are the declared
mutation path for every CE-adopting repository, so an operator has no
first-party way to see which executions are live.

The receipt schema is the root cause, not the accumulation: an entry carries
`task`, `type`, `source`, `owner`, `branch`, `worktree`, `startedAt`, and
`updatedAt` — and no terminal-state field. There is nowhere for `run-finish` to
record that it finished. `updatedAt` cannot stand in for one; it equals
`startedAt` on every entry, so it is a creation timestamp under another name.

Owner is external. `run-*` and its registry belong to ce-agent-kit; DVA owns
only the cards it files. This line named ce-workbook/task_management alongside
ce-agent-kit until 2026-09-15; the 소유권 section below carries the correction.
This is filed
here on the ISSUE-004 precedent — a DVA-observed defect in the shared runtime
gets a DVA card so the observation is not lost, and the fix lands upstream.

## Reproduction

1. In this repository, run `ce task run-list --json`.
2. Read `executions[]`: 20 entries on 2026-09-13.
3. Test each `worktree` path for existence: 19 of the 20 no longer exist. The
   one that does is the execution making the observation.
4. Compare `updatedAt` to `startedAt` on each: equal on all 20.
5. Run `ce task run-list` without `--json`: it prints
   `BLOCKED: one or more task executions are blocked` and exits non-zero,
   printing no execution at all.

Two of the 19 are `suppression-review-followups` and `record-309-verdict`,
whose `run-finish` invocations reported integration and recovery as completed
in this same session. Their commits are on `master` (`1e15275`, `99680ac`) and
their branches and worktrees are gone. The work is done; only the receipt
disagrees.

## Expected vs Actual

- Expected: a successful `run-finish` marks its receipt terminal, and
  `run-list` distinguishes live executions from retired ones.
- Actual: no field can express the distinction. Every receipt ever created is
  still live by the registry's own reading, and `run-list` degrades to a
  single BLOCKED line that names nothing.

## Impact

Two failures, and the second is the worse one:

- The registry cannot answer "what is running now", which is the question it
  exists to answer.
- BLOCKED is indistinguishable between a stale receipt and a genuinely
  abandoned worktree that needs recovery. A real abandonment arrives as one
  more line in a backlog of 19 false ones, so the signal that should stop an
  operator is the signal they have learned to skip.

## Recommended Resolution

Add an explicit terminal state to the receipt and have `run-finish` and
`run-abort` write it. A derived liveness check — "the worktree path is gone,
therefore it finished" — is not equivalent: it cannot tell integration from a
worktree someone deleted by hand, and it reads the answer off the filesystem
the receipt exists to be independent of.

Retiring the 19 existing entries is migration, not fix; a fix that leaves them
is still a fix, and this repository will accept whichever ordering upstream
prefers.

## 현행화 (2026-09-14, ce-agent-kit 0.8.4)

**이 이슈의 운영 고통은 사라졌다. 남은 것은 스키마 한 줄이다.** 그래서 P1 → P2,
severity medium → low. P3로 내리려 했으나 이슈 카드의 priority 어휘는 P0–P2가
전부다(`canonicalEnumRule{name: "priority", allowed: []string{"P0", "P1", "P2"}}`) —
task 카드에는 P3가 있는데 issue 카드에는 없다. 그 비대칭은 [[ISSUE-019]]로 따로 떴다.
P2가 이 보드에서 표현 가능한 가장 낮은 등급이고, severity로 실제 무게를 적었다.

파일 당시의 핵심 증상은 "`ce task run-list`가 아무것도 보고하지 못한다"였다. 0.8.4에서
그 명령은 rc 0으로 살아 있는 실행과 정산된 건수를 모두 낸다(이 저장소, 이 날 실측):

```
$ ce task run-list
ACTIVE: task execution states returned
ACTIVE close-354-and-currentize-board: branch=dev/claude/mbp/chore/… worktree=…
ACTIVE fix-planprogress-prose-false-positives: branch=… worktree=…
ACTIVE record-and-gate-the-release-backlog: branch=… worktree=…
settled: ABORTED=22 DONE=27
$ echo $?
0
```

`--json`에는 `executions[]`와 나란히 파생 배열 `states[]`(52건)가 실린다. 각 항목이
`DONE`("integration and recovery completed") 또는 `ABORTED`("execution … closed by CE
reconciliation: worktree/branch/origin branch were all reclaimed outside CE")라는 종단
상태와 사유를 갖는다. `--help`도 이 분리를 명시한다 — "executions is history with no
status field of its own, and states carries the derived status for those same records".

따라서 기준 2·3은 오늘 충족이다. **기준 1만 남는다**: raw `executions[]` 항목은 여전히
자체 상태 필드가 없고 52건 전부 `updatedAt == startedAt`이다. 종단 상태는 영수증 위가
아니라 파생 배열에 산다.

**그 차이가 왜 아직 결함인가.** 파생 상태는 관측 시점의 워크트리·브랜치 실재 여부로
재계산된다. 영수증 자체가 상태를 싣지 않으면 그 실재 여부가 바뀐 뒤에는 과거 실행이
무엇으로 끝났는지 되물을 방법이 없다 — 기록이 아니라 추론이기 때문이다. 다만 이것은
운영을 막지 않는 이력 충실도 문제이므로 P3가 맞다.

`ABORTED`가 정상 종결이라는 점도 여기에 적어 둔다. `run-start` 뒤 `branch-integrate`로
통합하면 런처가 워크트리·브랜치·원격 브랜치를 스스로 회수하고, CE는 그 실행을
reconciliation으로 닫으며 `next: no action required`를 붙인다. 낱말만 보고 실패로
읽어 다시 정리하려 들지 말 것.

## 소유권 — 상류다 (2026-09-15 명시)

`run-*`과 그 레지스트리는 `ce-agent-kit` 소유다 — Summary가 "Owner is external …
the fix lands upstream"으로 적는다. 남은 기준 1(영수증 스키마의 terminal-state 필드)도
상류 착지다. 보고는 [[TASK-399]]가 `ce-agent-kit#2`로 수행했다.

> **2026-09-15 정정.** 이 카드는 위 저장소를 `ce-workbook/task_management`로 적고
> 있었다. 측정한 결과 틀렸다 — `ce`는 `ce-agent-kit`에서 빌드되고(`go version -m
> $(which ce)` → `mod github.com/archmagece/ce-agent-kit`), 해당 서브커맨드는
> `cmd/ce/handlers_task.go`에 등록되고 `internal/adapter/cli/commands/`에 구현돼
> 있다. `ce-workbook`에 이름이 등장하는 것은 그 저장소가 `ce task *`를 **호출하는
> 계약 테스트**를 갖고 있기 때문이며, 소비자이지 소유자가 아니다. 오귀속은
> [[ISSUE-004]]의 `owner:` 줄에서 시작해 "ISSUE-004 선례"라는 인용을 타고 이 카드까지
> 번졌다 — 측정 없이 선례가 근거를 대신했다. 경위는 [[ISSUE-027]]이 기록한다.


## Resolution Criteria

- [ ] The run receipt schema carries an explicit terminal-state field, and `run-finish` and `run-abort` set it | verify: human — upstream schema and tests are linked here
- [x] `ce task run-list` reports live executions in this repository and exits 0 with no live execution outstanding | verify: human — fresh output from this repository is linked here
- [x] BLOCKED names the executions it is blocked on | verify: human — upstream output distinguishes a stale receipt from an abandoned worktree

## 2026-09-24 정리

기준 2·3은 그대로 충족이다. `ce task run-list`는 살아 있는 실행과 settled
건수를 내고 rc 0이다. 기준 1(영수증의 종단 상태 필드)은 운영 증상과 분리해
[TASK-422](../../todo/422-terminal-state-on-run-receipts.md)로 넘긴다.
이 카드는 그 운영 증상이 사라졌으므로 닫는다.
