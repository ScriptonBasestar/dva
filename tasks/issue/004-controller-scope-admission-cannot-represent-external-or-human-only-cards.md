---
id: ISSUE-004
title: "Controller scope admission cannot represent external or human-only cards"
type: bug
status: todo
priority: P1
effort: S
needs-human: true
execution-mode: external
human-grade: human
severity: medium
discovered-in: "2026-09-10 direct queue-run preflight"
discovered-at: 2026-09-10
ownership: upstream
created: 2026-09-10
upstream-ref: "task-manager-devbox task/list.md W13"
---

## Summary

The direct queue preflight requires every non-decomposable `todo` and the selected P0 `issue` to
have non-empty `allowed-paths`. The controller rejects `tasks/` paths,
absolute paths, parent traversals, and glob patterns. This is correct for an
executor that modifies product files, but it cannot represent cards whose
truthful work is exclusively external, human-operated, or a controller-owned
state transition.

At discovery, DVA had each shape:

- ISSUE-001 tracks the remaining DVA-owned migration of a legacy done-review to
  durable CE-compatible evidence. TASK-420 closes the CE issuer contract; no live
  ce-workbook controller currently owns a cross-repository run for TASK-312.
- TASK-329 and TASK-348 require external project readiness or a real build.
- TASK-307, TASK-309, TASK-319, TASK-321, and TASK-351 await an explicit
  human design decision.

Adding a synthetic DVA file scope would authorize an implementation the card
does not own. Adding its task-card path is rejected because `tasks/` is
controller-owned. The queue therefore exits 65 after selecting the card but before provider/action
execution or a controller-owned route transition can occur.

## Reproduction

1. Run the bound `queue_preflight` command for this repository.
2. Observe `task-allowed-paths-missing` for ISSUE-001 and active todos.
3. Add a task-card path as a hypothetical scope and observe
   `task-allowed-path-invalid`: `tasks/` is forbidden by
   `entry_selection._allowed_paths()`.

## Expected vs Actual

- Expected: cards that have no repository implementation scope can receive a
  controller-owned external/manual disposition without granting a fake executor
  scope.
- Actual: preflight selects the P0 issue, then rejects it before provider/action
  execution and route transition.

## P1 Blocker

- `reason`: DVA cannot resume its direct lifecycle queue truthfully while the
  scope contract treats external/manual cards as executable implementation
  units.
- `owner`: ce-agent-kit owns queue preflight, selection, and allowed-path
  validation. DVA owns only its adopted card states after the shared contract
  supports them. This line read ce-workbook/task_management until 2026-09-15;
  the 소유권 section below carries the measurement that corrected it.
- `next_action`: define and test an explicit machine-readable disposition-only
  admission contract. It must name the authoritative discriminator, permitted
  controller-owned routes, and fail-closed behavior before it can classify an
  external or human-only card ahead of implementation-scope validation.
- `next_check`: bound `queue_preflight` exits 0 for DVA without permitting
  `tasks/`, absolute paths, parent traversals, or globbed executor scopes.

## 소유권 — 상류다 (2026-09-15 명시)

큐 preflight·선정·allowed-path 검증은 `ce-agent-kit` 소유다 — P1 Blocker의 `owner`
항목이 그대로 적는다. 세 기준 전부가 상류 selection 테스트와 회귀를 검증
대상으로 요구한다. 보고는 [[TASK-399]]가 `ce-agent-kit#1`로 수행했다.

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

- [ ] Preflight distinguishes executor-owned implementation cards from external and human-only disposition cards | verify: human — upstream selection tests cover P0 issue, external todo, and decision todo shapes
- [ ] A DVA queue run routes ISSUE-001 and the external/manual cards without a synthetic product-file scope | verify: human — fresh controller evidence is linked here
- [ ] Scope validation remains strict for real implementation cards | verify: human — upstream regression rejects `tasks/`, absolute, parent, and glob paths

## 후속 (2026-09-24)

2026-09-24 재측정에서도 상류 계약은 그대로다. 작업은
[TASK-421](../_archive/2026-09/421-admit-external-and-human-only-cards.md)가 소유한다.

## 소유권 재측정 (2026-09-27)

ce-agent-kit#1은 TASK-399의 역사적 보고처다. 그 보고가 가리킨 Python
entry_selection controller는 퇴역했고 현재 selector successor는 카탈로그에 없다.
TASK-421은 새 제품 owner·canonical repository·CLI route·disposition contract가 ISSUE-040에서
결정될 때까지 blocked이며, 기존 issue reference만으로 ce-agent-kit 구현을 지시하지 않는다.

## Owner (2026-09-27)

The product owner assigned this to taskchain-task-manager, tracked as W13 in
`~/mydevbox/task-manager-devbox/task/list.md`. TASK-421 was archived as handed off;
this issue stays open until W13 ships and DVA adopts it.

## 2026-09-28 제품 구현

taskchain-task-manager `dc9f7b9`이 읽기 전용 `queue`에
`execution-mode: implementation|external|decision`과 `allowedPaths`를 추가했다.
구현 카드는 비어 있지 않은 정확한 저장소 상대 경로를 요구하고, 외부·결정 카드는
`needs-human: true`와 `allowed-paths` 부재를 요구한다. active P0 issue와 ready
todo를 구분해 라우팅하며, 부정확한 경로 범위와 terminal/unknown issue를 거부한다.
독립 리뷰 PASS, 정확한 커밋의 `make check`(taskstore race 693.430초)와 통합
사전검사 READY 후 제품 master/origin에 통합·push했고 작업 브랜치·worktree를
회수했다.

**남은 작업은 W07의 DVA 소비자 채택이다.** 현행 CE preflight와 DVA 직접 큐는
아직 이 출력을 사용하지 않으므로 위 Resolution Criteria를 완료로 표시하지
않는다. W07에서 실제 DVA 카드로 새 큐를 검증하고 외부·결정 카드의 종료 전이와
rollback 증거를 확정한다.

## 2026-09-28 DVA 조회 채택

W07 부분 작업 [[TASK-449]]가 `dva task-queue`를 저장소 소유의 읽기 전용
interaction으로 연결했다. 이 명령은 taskchain-task-manager의 `queue --dir tasks
--json`을 설정 루트에서 실행해 stdout/stderr/exit를 그대로 전달한다. 제품
`9e8fac7`에서 만든 로컬 바이너리 SHA-256은
`d2ca31f2880a1420efef3503dba2333f62c0ddd84648b06f4c75b5a62ad6ecd0`이다.
현재 DVA 보드에는 P1 ISSUE-004와 P2 ISSUE-006만 있어 결과는
`runnableCount: 0`, `agentRunnableCount: 0`이다. 이것은 과거 ISSUE-001과
외부·결정 카드의 실제 전이를 재현한 증거가 아니다. 합성 fixture에서 P0 외부,
결정 todo, 범위가 있는 구현 todo를 각각 분류하고 P1/P2를 제외했으며,
`tasks/`·절대·부모 탐색·glob 범위는 전체 결과 실패로 확인했다.

위 Resolution Criteria의 역사적 실제 카드·전이 기준은 여전히 미충족이다.
새 큐는 분류와 조회만 소유한다. W07의 실행 소비자·terminal/rollback 계약이
정해져 실제 경로에서 검증되기 전까지 이 이슈는 열린 상태로 둔다.

## 2026-09-28 CE host 증거의 범위

[[TASK-452]]는 CE worktree의 같은 owner 재시작, clean discard,
pre-integration refusal 뒤 재시도를 실제 host에서 확인한다. 이 증거는
옛 ISSUE-001 또는 외부·결정 카드의 실제 사람 disposition/terminal
전이가 아니므로 위 Resolution Criteria는 그대로 미충족이다. 해당 카드와
인간 증거가 있는 실제 경로가 준비될 때까지 이 이슈는 열린다.

## 2026-10-05 보드 메모

W13의 큐 분류, verdict, start는 이미 구현됐다. 남은 일은 사람 terminal과
rollback이다. 그 증거는
[ISSUE-453](453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md)의
공개 산출물 승인과 같은 의존이다. 합성 fixture와 injected-pin 테스트는 실제
사람 호스트 검증이 아니다. 위 Resolution Criteria는 체크하지 않는다. 이 절은
2026-10-05 보드 상태이며 새 호스트 테스트가 아니다.

## 읽기 전용 안내

네트워크와 인증을 실행하지 않는다. 비밀과 JWT를 출력하지 않는다. 사람 실측만
Resolution Criteria를 닫는다. `exec-tier`는 두지 않는다. CE exec-tier는
cheap, standard, strong만 받는다.
