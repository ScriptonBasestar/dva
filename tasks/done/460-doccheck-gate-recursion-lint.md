---
id: TASK-460
title: "Reject ce task gate in doccheck verify bindings"
type: feature
priority: P1
effort: S
exec-tier: standard
execution-mode: implementation
allowed-paths: [tools/doccheck, tasks, ARCHITECTURE.md]
status: done
created: 2026-10-01
quality-review: pass
quality-reviewed-at: 2026-10-01
quality-review-evidence: "Independent reviewer agent (general-purpose opus, separate from the author): verdict PASS on d779c00b with 1 P2 + 3 P3. P2 fixed at 6bffd817 (archive guard now isArchivePath, not the tasks/_archive/ literal); criterion 3 wording fixed (report counter, not JSON); remaining P3s are recorded lint limitations. Reviewer ran go test ./tools/doccheck/ ok, gofmt clean, ce task validate --all 516 valid / 0 invalid, and demonstrated doc-check clears once the card sits in done/."
---

## Summary

`ce task gate`의 bindings 단계는 체크된 바인딩을 재실행하는데, 그 바인딩이
다시 `ce task gate`를 부르면 게이트는 자기 자신을 무한히 호출하고 30초
timeout은 직계 자식만 죽여 고아 프로세스가 쌓인다([[ISSUE-454]], 2026-09-30
TASK-456 바인딩에서 335개 관측). 런타임 가드는 상류 소유다. 이 카드는 DVA의
작성 시점 방어로 `tools/doccheck`에 검사를 하나 추가한다. 활성 존(todo, done,
issue, plan) 카드의 verify 바인딩이 `ce task gate`를 호출하면 doc-check가
실패한다. 아카이브는 게이트가 재실행하지 않는 관측 사실과 역사 기록 불변
원칙에 따라 제외한다.

## Completion Criteria

- [x] A verify binding that runs `ce task gate` (with flags, paths, or after `&&`) is rejected with a message naming the recursion, while quoted prose mentioning it is not | verify: `/usr/bin/grep -rq 'func TestBindingGateRecursion(' tools/doccheck && go test ./tools/doccheck`
- [x] Cards under `tasks/_archive/` keep passing: the check does not scan archived bindings, so the existing board stays green | verify: `make doc-check` (regression-guard)
- [x] The check is wired into the shared gate path (Result counter, hard error, report counter) with no repository-local board check beside it | verify: human — inspect tools/doccheck/check.go wiring and ISSUE-454 ownership section

## Out of Scope

- 런타임 재귀 가드와 프로세스 그룹 종료 — ce-agent-kit 상류 과제(ISSUE-454 기준).
- 아카이브 카드의 역사적 바인딩 정리.

## Origin

2026-09-30 TASK-458 통합 준비 검사가 timeout으로 겉보기 "느린" 것이 실제로는
TASK-456 done 카드 바인딩의 무한 재귀였던 사건. 같은 날 별도 세션도 같은
결함을 발견해 TASK-456 바인딩을 수정했으나, 재발 방지 장치는 없었다.
