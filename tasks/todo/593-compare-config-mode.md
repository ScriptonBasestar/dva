---
id: TASK-593
title: "compare config 모드의 비교안별 격리"
type: feature
priority: P2
effort: M
exec-tier: strong
status: todo
created: 2026-11-06
depends-on: [TASK-591]
allowed-paths: [internal/compare/, internal/cli/, internal/lifecycle/, internal/runner/, USAGE.md, docs/75-variant-comparison-design.md]
---

## Summary

체크아웃 없이 호출한 체크아웃 하나에서 vars만 바꿔 N개 비교안을 띄운다. 지금 격리는 디렉터리 단위라(pid 파일 경로, compose 기본 프로젝트 이름) 같은 디렉터리에서는 비교안이 서로 덮어쓴다. DVA가 비교안 이름으로 pid 경로와 compose 프로젝트 이름을 나눈다.

## Scope

- `per_variant` 엔트리의 pid 파일을 비교안별 하위 경로로, compose 프로젝트 이름을 비교안별 이름으로 나눈다. `shared`와 기존 `dva up`의 경로·이름은 바꾸지 않는다.
- 체크아웃 안에 빌드 상태를 쓰는 도구(예: `.next`)가 그 경로를 var로 나누지 않으면 실행 전에 실패한다. 판정 방법(선언 필드 또는 알려진 도구 목록)은 구현 중 정하되 조용히 통과시키지 않는다.
- 출처 표기는 `config`로, snapshot·worktree 비교안과 섞을 수 있다.

## Decision Boundaries

- 스스로 결정해도 된다: 비교안별 경로·이름 형식.
- 멈추고 보고한다: 기존 단일 실행의 pid 경로나 compose 이름이 바뀌어야 할 때, 빌드 상태 충돌을 선언 없이 판정할 수 없을 때.

## Out of Scope

- 도구별 빌드 디렉터리 자동 재작성, snapshot·worktree 모드 변경.

## Completion Criteria

- [ ] 같은 체크아웃의 두 비교안이 서로 다른 pid 경로와 compose 이름을 쓰고, 나누지 않은 빌드 상태 경로는 실행 전 실패가 되며, 기존 dva up의 경로는 그대로임이 테스트로 고정되고 전체 테스트가 통과한다 | verify: human — 구현 카드가 추가한 테스트 이름을 이 줄의 go test -run 바인딩으로 바꾸고 make test 통과를 확인한다
- [ ] config 모드 두 비교안이 동시에 뜨고 각각 down된다 | verify: human — 예시 프로젝트에서 a=config b=config로 서로 다른 포트를 띄워 두 URL과 status를 확인하고 하나씩 down한다
