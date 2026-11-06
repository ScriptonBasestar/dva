---
id: TASK-587
title: "작업 워크트리 실행 대상 계약을 문서에 고정"
type: docs
priority: P1
effort: S
exec-tier: standard
allowed-paths: [USAGE.md, ARCHITECTURE.md, docs/31-execution-plan-resolution.md]
decomposable: false
status: todo
created: 2026-11-05
---

## Summary

[TASK-538](../todo/538-worktree-development-targets.md)은 작업 워크트리와 기본 체크아웃을 혼동하지 않도록 실행 대상을 고르는 요구사항을 소유한다. 명령 형태와 기록할 파일이 정해지기 전에는 그 요구사항을 실행 단위로 나눌 수 없다. 이 카드는 2026-11-05에 채택한 계약을 세 문서에 적는다. 명령을 구현하지 않고, TASK-538을 완료하거나 대체하지 않는다.

## 현재와 목표

현재 DVA는 작업 디렉터리에서 설정을 찾고, 이름 있는 plan으로 수명주기 명령을 실행한다. plan 이름은 체크아웃이 아니다. 프로세스 pid 파일은 그 설정 디렉터리의 `.sb/dva/pids/`에 있다. 후보를 고르는 명령은 없다.

목표는 같은 세 문서가 아래 계약을 적고, 그 문장이 이미 구현된 기능처럼 읽히지 않는 것이다. `docs/31-execution-plan-resolution.md`는 500줄과 10240바이트 제한 안에 남는다.

## 채택한 계약

명령 이름은 `dva target`이다. 대상 ID 없이 호출하면 후보만 보여주고 프로세스를 시작하거나 끝내지 않는다. 실행 형태는 `dva target <target-id> up|down|status|stop <plan>`이다. 대상 ID는 provider가 정의한다. CE provider는 작업 ID, worktrunk provider는 branch를 쓴다(2026-11-06 사용자 결정, [비교안 동시 실행 설계](../../docs/75-variant-comparison-design.md#결정-기록)). 이 호출은 선택된 체크아웃의 설정 디렉터리로 기존 수명주기를 호출한다. 대상 ID가 없거나, 알 수 없거나, 후보가 여럿인데 지정되지 않거나, 조회가 실패하면 프로세스를 시작하지 않고 실패한다. 기본 체크아웃으로 조용히 돌아가지 않는다.

플래그 없는 `dva up`, `dva down`, `dva status`, `dva stop`은 지금처럼 작업 디렉터리에서 설정을 찾는다. 모든 동사에 `--worktree`를 붙이는 안은 기각한다. 플래그가 없으면 작업 디렉터리로 조용히 떨어질 수 있다. pnpm이 디렉터리를 옮긴 뒤 `dva up`만 호출하는 안도 기각한다. 선택이 DVA 밖에 남고 프로세스 관리자가 둘로 나뉜다.

상태의 키는 호출할 때마다 받는 대상 ID다. 새 레지스트리를 만들지 않고, 마지막 선택을 저장하지 않으며, pnpm 어댑터에 선택을 저장하지 않는다. 프로세스 신원은 선택된 설정 디렉터리의 `.sb/dva/pids/`에 남긴다. status와 stop은 provider를 다시 조회한다. 회수됐거나 경로, 브랜치, 저장소 신원이 바뀐 작업은 실행 전에 실패한다.

DVA는 provider 인터페이스와 실패 계약만 소유한다. CE를 import하지 않고 경로를 추정하지 않는다. 후보 조회는 CE를 DVA의 필수 의존성으로 만들지 않는다. provider가 등록되지 않았거나 조회가 실패하면 프로세스를 시작하지 않는다. provider가 없을 때의 실패는 `dva target`에만 적용하고, 기존 `dva up`은 그대로 둔다.

JSON 필드 이름과 dry-run 플래그는 새로 정하지 않는다. 나열은 프로세스를 시작하거나 끝내지 않고, 수명주기 호출의 dry-run은 기존 `--dry-run`을 따른다.

TASK-538의 네 번째 완료 기준은 그대로 둔다. Singlepageweb 체크아웃이 생기기 전에는 DVA의 실행 단위가 아니며, 이 장비에서 실행하지 않는다. TASK-538은 그 파일럿 전에 done이 될 수 없다.

## 문서 배치

- `USAGE.md`의 Command Quick Reference에 `dva target` 행을 추가하고, Lifecycle 절에 명령 문장을 적는다. 사용 문장의 정본은 여기다.
- `ARCHITECTURE.md`의 Named lifecycle 아래에 소유 문단 하나만 추가한다. 명령 목록을 복사하지 않고 USAGE로 링크한다.
- `docs/31-execution-plan-resolution.md`의 2. 실행 대상 절 끝에 문장 하나와 USAGE 링크만 추가한다. 새 절을 만들지 않는다.

필수 문장. 주변 설명의 말투는 바꿔도 되고, 아래 네 문장은 그대로 둔다.

```text
플래그 없는 `dva up`, `dva down`, `dva status`, `dva stop`은 작업 디렉터리에서 설정을 찾는다.
후보 조회는 CE를 DVA의 필수 의존성으로 만들지 않는다.
프로세스 신원은 선택된 설정 디렉터리의 `.sb/dva/pids/`에 남긴다.
체크아웃 루트는 plan 이름과 다른 축이며, 대상 선택 계약은 [USAGE.md](../USAGE.md)가 소유한다.
```

첫 문장은 USAGE.md, 다음 두 문장은 ARCHITECTURE.md, 마지막 문장은 docs/31에 둔다. 마지막 문장의 링크는 docs/31 파일 기준이다. 세 문서는 이 명령이 이미 구현됐거나 지금 실행된다고 말하지 않는다.

## Decision Boundaries

- 스스로 결정해도 된다: 필수 문장 앞뒤의 설명 문장, 표 행의 짧은 설명.
- 바꾸면 안 된다: 명령 이름 `dva target`, 호출 형태(`<target-id>`는 provider가 정의하는 ID), 세 파일 경로, `.sb/dva/pids/`, CE를 필수 의존성으로 두지 않는다는 실패 계약, 필수 문장 네 개.

## Out of Scope

- Go 코드, schema, CLI 구현, 테스트.
- Singlepageweb 저장소와 이 장비에서의 파일럿 실행.
- TASK-538의 본문, 상태, decomposable, depends-on, blocks.
- 새 카드, 의존 간선, 마지막 선택 저장소, pnpm 어댑터 수정.

## 이후 순서

이 카드는 다음 카드를 만들지 않는다. 계약이 문서에 들어간 뒤에 따로 나눈다. 순서는 후보 나열과 실행 전 재검증, 그다음 같은 대상의 시작과 상태와 종료, 체크아웃이 생긴 뒤의 파일럿, 다른 세션의 독립 리뷰다.

## Completion Criteria

- [ ] USAGE.md가 명령 이름 dva target을 적는다 | verify: `/usr/bin/grep -F -q 'dva target' USAGE.md`
- [ ] USAGE.md가 플래그 없는 수명주기 명령의 작업 디렉터리 해석을 적는다 | verify: `/usr/bin/grep -F -q '작업 디렉터리에서 설정을 찾는다' USAGE.md`
- [ ] ARCHITECTURE.md가 후보 조회를 CE 필수 의존성으로 만들지 않는다고 적는다 | verify: `/usr/bin/grep -F -q 'CE를 DVA의 필수 의존성으로 만들지 않는다' ARCHITECTURE.md`
- [ ] ARCHITECTURE.md가 선택된 설정 디렉터리의 pid 경로를 적는다 | verify: `/usr/bin/grep -F -q '.sb/dva/pids' ARCHITECTURE.md`
- [ ] docs/31이 체크아웃 루트를 plan 이름과 다른 축으로 적는다 | verify: `/usr/bin/grep -F -q '체크아웃 루트' docs/31-execution-plan-resolution.md`
- [ ] 추가 문장이 아직 없는 명령의 계약으로 읽히고 문서 게이트를 통과한다 | verify: human — 세 문서의 추가를 읽고 dva target이 이미 구현됐거나 지금 실행된다고 말하지 않는지 확인하고, make doc-check가 통과하며 docs/31은 문장 하나와 링크 하나만 더해 500줄과 10240바이트를 넘지 않는지 확인한다

## References

- [TASK-538](../todo/538-worktree-development-targets.md)이 요구사항을 소유한다. 이 카드는 그 완료 기준을 지우거나 대신 닫지 않는다.
- 기록 위치는 [USAGE.md](../../USAGE.md), [ARCHITECTURE.md](../../ARCHITECTURE.md), [실행 계획 해석](../../docs/31-execution-plan-resolution.md)이다.
- 제품 경계는 [PRODUCT.md](../../PRODUCT.md), 채택 기준은 [SOUL.md](../../SOUL.md), 파일럿 재개 조건은 [ROADMAP.md](../../ROADMAP.md)에 있다.
