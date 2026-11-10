---
id: TASK-593
title: "compare config 모드의 비교안별 격리"
type: feature
priority: P2
effort: M
exec-tier: strong
status: review
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

- [x] 같은 체크아웃의 두 비교안이 서로 다른 pid 경로와 compose 이름을 쓰고, 나누지 않은 빌드 상태 경로는 실행 전 실패가 되며, 기존 dva up의 경로는 그대로임이 테스트로 고정되고 전체 테스트가 통과한다 | verify: `go test ./internal/lifecycle/ -run 'TestPluginContextInstancePaths|TestProcessPluginInstancesDoNotShareState|TestInstanceComposeProjectName' -count=1 && go test ./internal/cli/ -run 'TestCompare(UpConfigVariant|DownConfigVariant|UpMixesConfig|StatusConfigVariant)|TestLifecycleRunnerRunsInstances' -count=1`
- [ ] config 모드 두 비교안이 동시에 뜨고 각각 down된다 | verify: human — 예시 프로젝트에서 a=config b=config로 서로 다른 포트를 띄워 두 URL과 status를 확인하고 하나씩 down한다

## Scope Notes

- 격리의 단위는 "인스턴스"다. `PluginContext.Instance`가 비면 일반 실행이고 경로·이름은 그대로다(`TestPluginContextInstancePaths`가 고정). 값이 있으면 pid·로그가 `.sb/dva/{pids,logs}/<인스턴스>/`로, compose 프로젝트는 `<선언된 project_name 또는 소문자 디렉터리 이름>-<인스턴스>`로 나뉜다. 선언된 `project_name:`은 충돌이 아니라 접두사로 쓴다(docs/75 격리 규칙의 snapshot용 문장과 다르다. config 모드는 한 디렉터리를 공유하므로 이 방식이 유일하게 일관된다).
- process·serverless·tilt·sam 플러그인과 composition orchestrator의 pid/log 경로 조립을 `PluginContext.PidFile/LogFile/StateDir` 하나로 모았다. 경로 계산이 한곳에 있어 인스턴스를 놓친 곳이 없다.
- 출처 표기 `config`는 호출한 체크아웃 자체를 가리키며 기록에 `mode: config`, `instance: <비교안>`으로 남는다. 동일성은 모드+출처+경로로 본다. 브랜치 이름이 `config`인 경우는 `refs/heads/config`로 쓴다.
- 비교안 이름이 plan var `DVA_VARIANT`로 들어간다(config 모드만). 빌드 상태 판정은 알려진 도구 목록(`next`, `nuxt`, `nuxi`, `astro`, `gatsby`)이며, 그 도구를 실행하는 `per_variant` 엔트리의 명령·dir·env가 `DVA_VARIANT`를 쓰지 않으면 어떤 것도 시작하기 전에 실패한다. 선언 필드는 만들지 않았다. 목록에 없는 도구는 판정하지 못한다(조용한 통과가 아니라 문서에 한계로 적었다).
- `down`은 config 비교안에서 프로세스만 멈추고 호출한 체크아웃을 건드리지 않는다(`OwnsCheckout()`). `status`는 인스턴스의 pid 파일을 읽고 ref drift는 계산하지 않는다.

## Observations (out of scope, 후속 후보)

- 알려진 도구 목록 밖의 명령이 체크아웃에 쓰는 상태(캐시, sqlite 파일 등)는 dva가 볼 수 없다. 선언 필드(예: per_variant 항목별 `isolated: true`)가 필요해지면 별도 카드다.
- `dva logs`·`dva status` 같은 일반 명령은 인스턴스 경로를 모른다. config 비교안의 로그는 `compare status --json`의 `log_paths`로 본다.
- `internal/cli/compare_test.go`는 이미 500줄을 넘는다(550 → 556). 새 테스트는 `compare_config_test.go`·`compare_runner_instance_test.go`로 분리했다.
- 두 번째 완료 기준(예시 프로젝트에서 a=config b=config)은 사람 확인이 필요하다. 자동 테스트는 이 기준을 대신하지 않는다.
