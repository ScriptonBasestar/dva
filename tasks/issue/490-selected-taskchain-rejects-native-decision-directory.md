---
id: ISSUE-490
title: "Selected TaskChain queue rejects the native decision directory"
type: bug
priority: P1
effort: S
status: todo
severity: high
ownership: local
needs-human: true
execution-mode: external
human-grade: human
discovered-in: "ISSUE-453/004/006 live read-only recheck"
discovered-at: 2026-10-05
created: 2026-10-05
---

## Summary

현재 PATH TaskChain은 CE 내장 `tasks/decision/`을 거부한다. 따라서 이전
0/0 기록은 현재 조회 결과가 아니다. 디렉터리 제거나 gate 제외로 우회하지 않는다.

## 소유권 — 이 저장소다

이 카드는 DVA의 실제 조회 실패와 소비자 채택·상류 인계 준비를 추적한다.
원인인 보드 디렉터리 해석은 큐 생산자 TaskChain 소유다. DVA는 실패를 전달한다.
상류 저장소 확인 전에는 보고했다고 주장하거나 upstream-ref를 만들지 않는다.
생산자 canonical 저장소가 workbook에 없어서 상류 변경 경로 확인이 먼저다.
`taskchain` alias는 별개의 `flow-taskchain-devbox`를 가리키므로 사용하지 않는다.
정확한 경로가 확인되면 구현·회귀 검사·독립 리뷰는 에이전트가 처리할 수 있다.

## Reproduction

2026-10-05 DVA `5f71e8a`에서 `dva task-queue`와 `dva task-queue-verdict`는
둘 다 exit 1이었다. 공통 원인: `queue: unsupported task directory at board root: decision`.
선택 바이너리 `/Users/archmagece/go/bin/taskchain-task-manager`의 SHA-256은
`db6dd2d0d61373762d5623f45418913a1585650616b6cc811476c88e2a637a1b`.
Go build info는 `go1.27.1`, source `f53c793889ec9f5ca2a191e0aa0d52174cf2c959`,
`vcs.modified=false`, darwin/arm64였다. 이는 provenance 관찰이며 공개 승인 증거는 아니다.
보드·캐시·설정·pin은 이 조회로 변경하지 않았다.

## Expected vs Actual

- Expected: CE의 native decision directory를 해석하고 Proposed/Accepted를 제품
  정책에 맞게 분류한다. 사람 결정을 agentRunnable에 넣지 않는다.
- Actual: 디렉터리를 만나는 즉시 전체 queue가 실패한다.

## Steps

1. Workbook catalog에 미등록인 실제 TaskChain 생산자 저장소 경로와 owner를 확인한다.
2. 그 저장소의 queue directory inventory 및 decision 상태 분류를 수정한다.
   위치는 실제 소스를 확인한 뒤 상류 구현 카드에 명시한다. DVA에 두 번째 parser를 만들지 않는다.
3. native decision Proposed/Accepted, human-only 및 implementation 카드의 양성·음성
   회귀 검사를 실행한다. DVA 실제 보드 조회와 verdict를 다시 실행한다.
4. 공개 artifact/pin은 [ISSUE-453](453-dva-queue-consumer-lacks-pinned-product-binary-provenance.md)의
   provenance·승인을 거친다. [ISSUE-004](004-controller-scope-admission-cannot-represent-external-or-human-only-cards.md)·
   [ISSUE-006](006-preflight-reports-needs-human-cards-as-runnable.md)의 실제 증거는 별도로 남긴다.

## Stop conditions

- 생산자 경로를 catalog alias나 폴더 이름으로 추정하지 않는다.
- `tasks/decision/`을 숨기거나 CE gate를 약화시키지 않는다.
- Go module 존재나 SHA 일치만으로 pin 공개 승인을 만들지 않는다.

## Resolution Criteria

- [ ] 실제 DVA 보드 queue가 성공한다 | verify: `dva task-queue >/dev/null`
- [ ] 같은 보드 verdict가 성공한다 | verify: `dva task-queue-verdict >/dev/null`
- [ ] 생산자 독립 리뷰와 decision/human 분류 회귀 증거가 있다 | verify: human — canonical producer path and reviewed upstream evidence
