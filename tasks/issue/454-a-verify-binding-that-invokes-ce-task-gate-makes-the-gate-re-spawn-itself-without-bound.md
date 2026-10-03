---
id: ISSUE-454
title: "A verify binding that invokes ce task gate makes the gate re-spawn itself without bound"
type: bug
priority: P1
status: todo
severity: medium
discovered-in: "2026-09-30 TASK-458 integration readiness check"
discovered-at: 2026-10-01
created: 2026-10-01
ownership: split
upstream-ref: "ce-agent-kit (report pending — next upstream wave)"
---

## Summary

`ce task gate`의 마지막 단계(`bindings`)는 체크된 `[x]` 기계 바인딩을 30초
timeout으로 재실행한다. 그 바인딩이 `ce task gate`를 담고 있으면 gate는 자기
자신을 다시 호출하고, timeout은 직계 자식만 죽이므로 손자 프로세스는 PID 1에
재부모화되어 쌓인다. 2026-09-30 DVA에서 TASK-456의 done 카드 바인딩
(`make doc-check && ce task gate`)이 약 29분에 335개 프로세스를 쌓았고
([[TASK-456]] 통합 준비 검사가 10분 timeout으로 겉보기 "느려" 보였던 것이
실제 원인이었다). done 카드의 바인딩은 게이트가 재실행하므로 이 결함은
작성 시점이 아니라 완료 시점에 터진다.

## Reproduction

1. done 카드의 체크된 바인딩에 `ce task gate`를 넣는다.
2. `ce task gate`를 실행한다.
3. `pgrep -f "ce task gate"` 개수가 실행 시간에 비례해 증가하는지 관찰한다.

## Expected vs Actual

- Expected: 게이트는 재귀 진입을 감지해 바인딩을 거부하거나(환경 표식 등),
  timeout 시 프로세스 그룹 전체를 종료한다.
- Actual: 무한 재귀. 게이트는 실패하지 않고 영원히 느려 보이며, 고아
  프로세스가 기계 자원을 소진한다.

## 소유권 — 갈린다 (2026-10-01 명시)

- **런타임 가드(재귀 진입 차단, 프로세스 그룹 종료)는 상류 ce-agent-kit
  소유다.** 게이트의 바인딩 재실행은 `ce`가 구현한다.
- **DVA는 작성 시점 검출을 소유한다.** `tools/doccheck`가 활성 존(todo, done,
  issue, plan) 카드의 verify 바인딩에서 `ce task gate` 호출을 거부한다.
  아카이브는 게이트가 재실행하지 않는 관측 사실(2026-09-30 READY 보드에
  `ce task gate` 바인딩을 가진 아카이브 카드가 다수 존재)과 역사 기록
  불변 원칙에 따라 검사에서 제외한다.
- DVA 측 구현은 [[TASK-460]]이 소유한다.

## Resolution Criteria

- [ ] The gate refuses or fails fast when a binding it executes would itself invoke the gate (recursion entry guard), and a timed-out binding leaves no orphaned grandchildren | verify: human — upstream ce-agent-kit guard observed on a reproduction case
- [x] DVA doccheck rejects a verify binding invoking `ce task gate` in active-zone cards at authoring time | verify: `/usr/bin/grep -rq 'func TestBindingGateRecursion(' tools/doccheck && go test ./tools/doccheck` (observed: 2026-10-03 — exit 0, [[TASK-460]])

## 2026-10-03 재측정

DVA 측 기준 2는 [[TASK-460]]이 `tools/doccheck`에 구현했고 바인딩이 exit 0으로
통과해 체크했다. 남은 기준 1(런타임 재귀 가드·프로세스 그룹 종료)은 상류
ce-agent-kit 소유이고 보고 대기 상태다. DVA가 할 수 있는 일은 없으므로 이 이슈는
upstream-waiting으로 열어 둔다.

## 2026-10-03 상류 재측정 — resolved-in-part

기준 1은 두 요구를 묶는다. 상류가 그중 **재귀 진입 가드만** 출하했다.

- **해결됨 — 재귀 진입 가드.** 다른 저장소(primeno1-devbox)에서 같은 결함이 상류
  ISSUE-101로 따로 보고됐고, ce-agent-kit `58e2b79d`("refuse a gate started inside a
  gate's bindings step", TASK-365)가 고쳤다. bindings 단계는 각 바인딩을
  `CE_TASK_GATE_ACTIVE=1`로 실행하고, 그 표식이 있는 상태에서 시작된 게이트는 즉시
  exit 2로 끝난다. 설치된 `ce`(`vcs.revision=f3a8ba78`, `58e2b79d`를 조상으로 포함)에서
  실측했다:

  ```
  $ CE_TASK_GATE_ACTIVE=1 ce task gate
  ce task gate: refusing to run inside another gate's bindings step (CE_TASK_GATE_ACTIVE is set); a verify binding must not invoke the gate
  exit=2   (0.008s)
  ```

- **미해결 — timeout 시 손자 프로세스 정리.** `internal/usecase/task/gate_reexec.go`의
  `runCheckedBinding`은 여전히 `exec.CommandContext(runCtx, "sh", "-c", command)`에
  `Setpgid`·프로세스 그룹 kill·`WaitDelay`가 없다. 30초 timeout은 `sh`만 죽이고,
  바인딩이 띄운 손자 프로세스는 남는다. ce-agent-kit의 열린 카드 중 이 요구를
  추적하는 것은 없다(`orphan|process group|grandchild|setpgid` 검색 0건) — 상류
  ISSUE-101은 재귀만 요구했고 fixed로 보관됐다.

재귀 폭주라는 이 이슈의 관측 피해는 이제 재현되지 않는다. 남은 것은 장시간 바인딩이
timeout 뒤 자식을 남기는 일반 결함이며, 그 보고는 아직 상류에 없다. 기준 1은 두
요구가 모두 충족될 때 체크하므로 미체크로 둔다.
