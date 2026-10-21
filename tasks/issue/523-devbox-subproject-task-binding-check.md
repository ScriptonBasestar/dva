---
id: ISSUE-523
title: "No dva entry point runs a devbox's task bindings for one subproject before its deploy"
type: tech-debt
priority: P2
effort: S
exec-tier: standard
status: todo
ownership: split
upstream-ref: "ce-agent-kit ISSUE-444 (tasks/_archive/issue/444-task-gate-bindings-root-scope.md)"
severity: medium
discovered-in: "singlepageweb-devbox TASK-270 (owner of the board-wide binding run)"
discovered-at: 2026-10-20
created: 2026-10-20
---

## Summary

singlepageweb-devbox는 단일 제품이 아니다. `dva.yml` `subprojects`에 사이트마다 독립 저장소가
선언되어 있고, DevBox의 CE 보드에는 모든 사이트의 카드가 함께 있다. 2026-10-20 소유자가
다음 두 가지를 정했다.

- 보드 전체의 binding 회귀 확인은 **사이트를 배포하기 전에 1회**, **그 사이트의 카드만** 대상으로 한다.
- 실행 주체는 **dva**다.

지금 dva에는 이 실행을 맡을 진입점이 없다.

- `dva ci --project <subproject>`는 그 subproject 저장소가 선언한 CI profile을 돈다.
  DevBox 보드에서 그 사이트에 속한 카드의 binding은 돌지 않는다.
- DevBox root의 interaction으로는 범위 없는 `ce task gate` 실행만 둘 수 있다. 이렇게 하면 모든
  사이트의 binding(약 135개, 약 30분)을 돌게 되어 소유자 결정과 범위가 다르다.

## 상류 의존 — ce-agent-kit ISSUE-444

보드 판정과 카드 선택은 공유 `ce task gate`만 한다. 이 저장소의
`policy:no-second-board-grader`도 같은 규칙이다. 그래서 dva가 카드를 직접 고르면 안 된다.
ce-agent-kit ISSUE-444는 `ce task gate`가 external root 하나의 binding만 돌게 하는 옵션을
요구한다. 이 카드는 그 옵션이 생긴 뒤에 dva 쪽 진입점을 만드는 몫이다.

**2026-10-22 상류 해소.** ce-agent-kit master `e20116ba`·`07fcbde7`·`e7040d9a`에 `ce task gate
--bindings-root NAME`이 들어갔다. 진입점이 기대해도 되는 계약은 다음과 같다(상류
`docs/10-ce-cli/22-readiness-runner-adapter.md` "납품 바인딩 범위").

- NAME은 `ce-tasks.yaml` `task-bindings.external-roots`의 키다. 선언되지 않은 NAME은 rc=2
  (unavailable)이므로 dva가 이름을 미리 검증하지 않아도 실패로 끝난다.
- 카드 소속은 gate가 정한다. `allowed-paths`가 모두 선언된 루트 아래이고 하나 이상이 NAME
  아래인 카드만 돈다. DevBox 자신의 파일을 건드리는 카드는 어느 사이트에도 속하지 않아 미뤄진다.
- 다른 사이트의 체크아웃이 없어도 NAME의 검사는 막히지 않는다. `--bindings-since` 없이 속한 카드가 없는 루트도 rc=2다(`e7040d9a`).
- 통과 시 `detail`에 실행·미룬 카드 수가 적힌다. rc는 0(ready)·1(not-ready)·2(unavailable).

따라서 남은 결정은 subproject 이름과 external root 이름의 대응 규칙과 진입점 형태뿐이다.

## Reproduction

1. singlepageweb-devbox primary에서 `dva ci --help`를 실행하면 `--project`가 보인다. 하지만
   devbox 보드 binding의 범위를 정하는 옵션은 없다.
2. `dva ls`의 DevBox interaction은 `check`(`make check`)와 subproject import뿐이다.
   배포 전 사이트 범위 binding을 확인하는 항목은 없다.

## Expected vs Actual

- Expected: 다음 중 하나 또는 그에 준하는 형태로 사이트 이름을 받아, DevBox 보드에서 그 사이트에
  속한 카드의 binding만 공유 gate로 돌리는 진입점이 있다.
  - devbox 선언의 `ci.profiles`
  - subproject 이름을 받는 interaction 인자

  내부적으로는 `ce task gate --external-base <devbox primary> --bindings-root <subproject>`
  같은 공유 gate 호출로 위임한다. subproject 이름과 `ce-tasks.yaml` external root 이름의
  대응 규칙(같은 이름, 또는 선언으로 매핑)은 결정할 항목이다. 대응되지 않는 이름은 실패로 끝낸다.
- Actual: 사이트 범위 실행 진입점이 없고, 범위 없는 전체 실행만 가능하다.

## 소유권 — 갈린다 (진입점은 DVA, 판정은 상류)

- 진입점(profile·interaction·인자 전달)은 DVA가 소유한다.
- 카드 선택과 판정은 ce-agent-kit `ce task gate`가 소유한다.
- 소비 저장소인 singlepageweb-devbox는 이 진입점을 기다리며 그동안 전체 검사를 돌리지 않는다
  (소유자 결정). 배포 절차에서 이 진입점을 부르는 일은 그 저장소가 맡는다.

## Evidence and Scope

- 관측: 2026-10-20, 설치본 `dva version 0.3.0`(commit `fbb336fe`)과 ce 0.10.14(`a7c10ae5`).
- 소비자 근거: singlepageweb-devbox 문서 `docs/40-development/50-board-readiness-adoption-plan.md`
  "범위 없는 전체 binding 실행 (TASK-270)" 절, `ce-tasks.yaml` external-roots(12개 사이트).
- 범위: dva 진입점의 설계와 구현까지다. 카드 소속 규칙은 ce-agent-kit ISSUE-444의 범위다.
