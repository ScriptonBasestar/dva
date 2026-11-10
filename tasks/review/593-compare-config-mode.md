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

- [x] 같은 체크아웃의 두 비교안이 서로 다른 pid 경로와 compose 이름을 쓰고, 나누지 않은 빌드 상태 경로는 실행 전 실패가 되며, 기존 dva up의 경로는 그대로임이 테스트로 고정되고 전체 테스트가 통과한다 | verify: `go test ./internal/lifecycle/ -run 'TestPluginContextInstancePaths|TestProcessPluginInstancesDoNotShareState|TestInstanceComposeProjectName|TestInstanceProjectNameIsNormalized|TestComposePluginsRunInstanceProject|TestOrchestratorReadsInstancePidFile' -count=1 && go test ./internal/cli/ -run 'TestCompare(UpConfigVariant|DownConfigVariant|UpMixesConfig|StatusConfigVariant)|TestLifecycleRunnerRunsInstances|TestParseVariantSpecsRejectsNamesDifferingByCase|TestAnnotateProcessesReadsTheInstanceDirectory' -count=1`
- [ ] config 모드 두 비교안이 동시에 뜨고 각각 down된다 | verify: human — 예시 프로젝트에서 a=config b=config로 서로 다른 포트를 띄워 두 URL과 status를 확인하고 하나씩 down한다

## Scope Notes

- 격리의 단위는 "인스턴스"다. `PluginContext.Instance`가 비면 일반 실행이고 경로·이름은 그대로다(`TestPluginContextInstancePaths`가 고정). 값이 있으면 pid·로그가 `.sb/dva/{pids,logs}/<인스턴스>/`로, compose·podman-compose 프로젝트는 `<디렉터리 이름>-<인스턴스>`로 나뉘며 compose가 명시 이름에 요구하는 형식(소문자·숫자·`-`·`_`)으로 정규화한다. per_variant compose 항목의 `project_name:`은 기존 검증이 거부하므로 접두사 경로는 없다.
- process·serverless·tilt·sam 플러그인과 composition orchestrator의 pid/log 경로 조립을 `PluginContext.PidFile/LogFile/StateDir` 하나로 모았다. 경로 계산이 한곳에 있어 인스턴스를 놓친 곳이 없다.
- 대소문자만 다른 비교안 이름(`A`와 `a`)은 대소문자를 구분하지 않는 파일시스템에서 한 디렉터리가 되므로 거부한다. OS 환경에 `DVA_VARIANT`가 있으면 모든 비교안의 값을 덮어쓰므로 포트 var처럼 실행 전에 거부한다.
- 출처 표기 `config`는 호출한 체크아웃 자체를 가리키며 기록에 `mode: config`, `instance: <비교안>`으로 남는다. 동일성은 모드+출처+경로로 본다. 브랜치 이름이 `config`인 경우는 `refs/heads/config`로 쓴다.
- 비교안 이름이 plan var `DVA_VARIANT`로 들어간다(config 모드만). 빌드 상태 판정은 알려진 도구 목록(`next`, `nuxt`, `nuxi`, `astro`, `gatsby`)이며, 그 도구를 실행하는 `per_variant` 엔트리의 명령·dir·env가 `DVA_VARIANT`를 쓰지 않으면 어떤 것도 시작하기 전에 실패한다. 선언 필드는 만들지 않았다. 목록에 없는 도구는 판정하지 못한다(조용한 통과가 아니라 문서에 한계로 적었다).
- `down`은 config 비교안에서 프로세스만 멈추고 호출한 체크아웃을 건드리지 않는다(`OwnsCheckout()`). `status`는 인스턴스의 pid 파일을 읽고 ref drift는 계산하지 않는다.

## Observations (out of scope, 후속 후보)

- 알려진 도구 목록 밖의 명령이 체크아웃에 쓰는 상태(캐시, sqlite 파일 등)는 dva가 볼 수 없다. 선언 필드(예: per_variant 항목별 `isolated: true`)가 필요해지면 별도 카드다.
- `dva logs`·`dva status` 같은 일반 명령은 인스턴스 경로를 모른다. config 비교안의 로그는 `compare status --json`의 `log_paths`로 본다.
- `internal/cli/compare_test.go`는 이미 500줄을 넘는다(550 → 556). 새 테스트는 `compare_config_test.go`·`compare_runner_instance_test.go`로 분리했다.
- 두 번째 완료 기준(예시 프로젝트에서 a=config b=config)은 사람 확인이 필요하다. 자동 테스트는 이 기준을 대신하지 않는다.
- 1차 리뷰(FAIL, head d452226d)가 낸 F1(OS 환경 DVA_VARIANT 덮어쓰기), F2(podman-compose 미분리), F3(compose 프로젝트 이름 정규화), F4(되돌려도 통과하던 구현 줄 5곳), F5(대소문자 충돌), F7(문서 모순)은 이 수리에서 고쳤고 각 줄을 되돌려 새 테스트가 실패함을 확인했다.
- 1차 리뷰 F6(미수리, 휴리스틱 한계): `script:` 플러그인의 up/down/stop 명령과 엔트리·plan의 `vars:`는 빌드 상태 판정이 보지 않고, `sh -c "next dev"`처럼 따옴표가 앞선 도구 이름은 놓치며, dir에 `next`가 든 경로는 오탐한다. 오탐은 명령 텍스트에 `$DVA_VARIANT`를 적으면 우회된다. 선언 필드를 도입할 때 함께 다룬다.

## Review Attempts

### 1. 2026-11-10 — reviewer: claude-code-subagent (opus, 독립), executor: claude-sonnet-5-5

- 대상: head d452226d(구현 커밋). 판정: fix-needed(VERDICT: FAIL). 기준 2는 verify: human이라 NEEDS_HUMAN.
- 실행한 검증: lifecycle·compare·cli 테스트와 `go vet`, 복사본에서 돌연변이 14건과 재현.
- 발견과 처리:
  - F1 OS 환경 DVA_VARIANT가 모든 비교안 값을 덮어씀: 수정(포트 var와 같은 실행 전 거부).
  - F2 podman-compose가 인스턴스를 무시: 수정(`instanceProjectName`을 docker·podman이 공유).
  - F3 compose 프로젝트 이름 미정규화: 수정(소문자·숫자·`-`·`_`).
  - F4 되돌려도 통과하는 구현 줄 5곳: 테스트 추가 후 각 줄을 되돌려 실패 확인.
  - F5 대소문자만 다른 이름 충돌: 수정(대소문자 무시 중복 거부).
  - F6 휴리스틱 한계: 수리 없음, Observations에 기록.
  - F7 docs/75 project_name 접두사 문장 모순: 문서·카드 수정.
- 수리 방식: 같은 worktree의 후속 커밋, 새 독립 리뷰어로 재검토 예정. 다음 전이: 2차 리뷰.
