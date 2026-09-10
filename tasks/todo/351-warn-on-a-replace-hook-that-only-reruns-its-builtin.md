---
id: TASK-351
title: "Warn on a replace: hook that only re-runs the built-in it replaces"
type: feature
priority: P3
effort: M
exec-tier: standard
status: todo
needs-human: true
created: 2026-09-08
source: "TASK-320 item 3 분리 — docs/dogfood/scripton-nd-stack.md 개선점 2"
---

## Summary

`interaction.<builtin>.replace`는 built-in의 동작을 프로젝트 명령으로 갈아끼우는 훅이다.
문제는 **갈아끼운 것이 원래 동작과 같을 때**다. scripton-nd-stack의 `logs` 훅은
`docker compose -f … logs -f`를 직접 부르는데, 이는 `dva logs <plan>`이 이미 하는 일이다.
훅은 아무 의미 없이 선언을 하나 늘리고, 나중에 dva가 `logs`를 개선해도 그 프로젝트만
옛 동작에 고정된다. validate는 지금 이 상태에 대해 아무 말도 하지 않는다.

PLAN의 "build/logs 훅 정리" 공통 행은 이런 프로젝트를 **사람이 하나씩 읽어서** 찾고 있다.
기계적으로 찾아주는 것이 이 카드의 목적이다.

## Decision needed — 무엇을 "동등"으로 볼 것인가

전체 동등성 판정은 불가능하다(훅은 임의의 셸). 판정 가능한 좁은 형태만 잡는다:

훅의 step이 **하나**이고, 그 `run:`이 `docker compose`(또는 `docker-compose`) 호출이며,
서브커맨드가 자신이 대체하는 built-in과 같은 이름(`logs`↔`logs`, `build`↔`build`)이고,
`-f`로 넘긴 compose 파일 집합이 그 프로젝트 stack의 compose 엔트리가 선언한 집합의
부분집합일 때 — 그때만 warn한다. 그 밖의 모든 형태는 조용히 통과한다.

이 좁힘이 핵심이다. 넓히면 정당한 훅(추가 플래그, 서비스 한정, 전처리)까지 잡아 warn을
끄는 습관을 만들고, 그러면 `suggestion_ignore`가 또 한 번 110줄이 된다(TASK-309).

`clean`처럼 **이미 제거된** built-in의 훅은 이 검사 대상이 아니다 — 그건 별도 규칙이고
`clean_hook_advice`가 이미 다룬다.

## Completion Criteria

- [ ] compose 파일 집합이 stack 선언의 부분집합인 단일 step `logs` replace 훅이 warn된다 | verify: `/usr/bin/grep -rq 'func TestWarnsOnReplaceHookEquivalentToBuiltin(' internal/config`
- [ ] 플래그·서비스·전처리가 붙어 동작이 다른 훅은 warn되지 않는다 | verify: `/usr/bin/grep -rq 'func TestReplaceHookWithExtraBehaviourIsNotWarned(' internal/config`
- [ ] 제거된 built-in(`clean`)의 훅은 이 검사가 건드리지 않는다 | verify: `/usr/bin/grep -rq 'func TestRemovedBuiltinHookIsOutsideEquivalenceCheck(' internal/config`
- [ ] warning 문구가 훅 경로와 "삭제하면 `dva <verb> <plan>`이 같은 일을 한다"를 함께 말한다 | verify: `/usr/bin/grep -rq 'func TestEquivalentReplaceHookWarningNamesTheReplacement(' internal/config`

## Notes

- 출처: `docs/dogfood/scripton-nd-stack.md` 개선점 2. 같은 문서 30행이 이 프로젝트의
  `logs` 훅을 "삭제 가능해 보이나 PLAN 대상 목록에 없어 유지"로 남겨 두었다 — 이 검사가
  있으면 그 판단이 사람 손을 떠난다.
- TASK-320에서 분리됐다. 320의 나머지 세 항목은 파서·매칭 결함이었고 이것만 신규 검사다.
