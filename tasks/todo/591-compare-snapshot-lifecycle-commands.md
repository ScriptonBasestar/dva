---
id: TASK-591
title: "snapshot 모드 compare up·status·down 명령"
type: feature
priority: P2
effort: L
exec-tier: strong
status: todo
created: 2026-11-06
depends-on: [TASK-590]
allowed-paths: [internal/cli/, internal/compare/, USAGE.md, docs/75-variant-comparison-design.md]
---

## Summary

`dva compare up|status|down <name>`을 snapshot 모드로 구현한다. 기존 named lifecycle(`dva up <plan>`)을 비교안마다 각 체크아웃의 설정 디렉터리로 호출하고, `shared`는 호출한 체크아웃에서 한 번만 기동한다. 사람과 에이전트가 같은 명령을 쓰도록 사람 출력과 `--json` 출력을 함께 낸다.

## Scope

- 인자: `이름=ref`. 생략하면 `compare.<name>.variants`를 쓴다. 둘 다 없으면 실패한다.
- `up`: 출처 해석, 포트 배정과 점유 검사, 표시 후 `shared` 한 번, `per_variant` 비교안마다 시작. 점유된 포트는 옮기지 않고 실패한다. `--dry-run`은 생성·시작 없이 해석 결과만 낸다. 같은 출처·커밋으로 떠 있는 비교안은 건너뛴다.
- 일부 실패: 시작한 비교안은 그대로 두고 실패로 끝낸다. 비교안별 결과(시작됨, 실패와 원인, 시도 안 함)와 되돌릴 정확한 `dva compare down` 명령을 낸다. 자동 되돌리기는 없다.
- `status`: 읽기 전용. 비교안별 출처, 커밋, 포트, URL, readiness, pid, 로그 경로, stale·drift.
- `down [<variant>...]`: `per_variant`를 먼저, 마지막 비교안 뒤 `shared`. snapshot만 회수한다.
- 기계 출력: 종료 코드는 성공 0, 실패 1. 일부 실패와 시작 전 실패는 기존 실패 envelope의 `error.code`로 구분한다(`internal/cli/root.go`). 비교안별 결과는 기존 plan 출력(`status`, `endpoints[].url`)을 감싸고, 최상위에 다음 명령을 둔다.
- 구현과 회귀 테스트가 끝나면 USAGE.md에 사용 문장을 적고 docs/75의 상태를 갱신한다.

## Decision Boundaries

- 스스로 결정해도 된다: JSON 필드 이름(기존 출력 코드에 맞춘다), `error.code` 값, 사람 출력의 배치.
- 멈추고 보고한다: 종료 코드를 0/1 밖으로 넓혀야 할 때, 기존 `dva up`·`down`의 동작이나 출력을 바꿔야 할 때.

## Out of Scope

- worktree 모드(TASK-592), config 모드(TASK-593).

## Completion Criteria

- [ ] up의 전부 성공·일부 실패·시작 전 실패, 재호출 시 빠진 비교안만 시작, down 순서, JSON envelope이 테스트로 고정되고 전체 테스트가 통과한다 | verify: human — 구현 카드가 추가한 테스트 이름을 이 줄의 go test -run 바인딩으로 바꾸고 make test 통과를 확인한다
- [ ] 실제 웹 plan으로 a=main b=<다른 ref>를 띄워 두 URL을 열고 status와 down이 설계대로 동작한다 | verify: human — 예시 프로젝트에서 up, status --json, 한 비교안 강제 실패 후 출력된 down 명령, 전체 down을 실행해 남은 프로세스와 snapshot이 없는지 확인한다
- [ ] USAGE.md가 dva compare의 사용을 적고 docs/75 상태가 구현 범위와 맞는다 | verify: `/usr/bin/grep -F -q 'dva compare' USAGE.md`
