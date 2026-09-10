---
id: TASK-351
title: "Flag a narrow compose replace-hook candidate for review"
type: feature
priority: P3
effort: M
exec-tier: standard
status: done
needs-human: false
created: 2026-09-08
source: "TASK-320 item 3 분리 — docs/dogfood/scripton-nd-stack.md 개선점 2"
---

## Summary

`interaction.<builtin>.replace`는 built-in의 동작을 프로젝트 명령으로 갈아끼우는 훅이다.
문제는 **갈아끼운 것이 원래 동작과 겹쳐 보일 때**다. replace 훅은 임의의 셸을 실행하므로
validate가 두 경로의 lifecycle 의미가 같다고 증명할 수는 없다. 다만 stack의 모든 compose
파일을 그대로 나열해 `logs` 또는 `build`를 호출하는 한 단계 훅은 사람이 검토할 후보가 된다.

PLAN의 "build/logs 훅 정리" 공통 행은 이런 프로젝트를 **사람이 하나씩 읽어서** 찾고 있다.
기계적으로 찾아주는 것이 이 카드의 목적이다.

## Decision — 무엇을 "동등"으로 볼 것인가

전체 동등성 판정은 불가능하다(훅은 임의의 셸). 승인된 안전한 형태는 후보 알림만 낸다:

승인: 훅의 step이 **하나**이고, 그 `run:`이 `docker compose`(또는 `docker-compose`) 호출이며,
서브커맨드가 `logs` 또는 `build`이고, `-f`로 넘긴 compose 파일 집합이 그 프로젝트 stack의
compose 엔트리가 선언한 **전체 집합과 일치**하며, 개행·셸 구분자·추가 플래그·서비스 인수가
없을 때만 후보 warning을 낸다. 이 문구는 훅 삭제나 `dva <verb> <plan>`과의 동등을 주장하지
않고, 사람이 두 실행 경로를 검토하도록 요청한다. 그 밖의 모든 형태는 조용히 통과한다.

이 좁힘이 핵심이다. 넓히면 정당한 훅(추가 플래그, 서비스 한정, 전처리)까지 잡아 warn을
끄는 습관을 만들고, 그러면 `suggestion_ignore`가 또 한 번 110줄이 된다(TASK-309).

`clean`처럼 **이미 제거된** built-in의 훅은 이 검사 대상이 아니다 — 그건 별도 규칙이고
`clean_hook_advice`가 이미 다룬다.

## Completion Criteria

- [x] 전체 compose 파일 집합을 쓰는 단일 step `logs`/`build` 훅이 검토 후보 warning을 낸다 | verify: `/usr/bin/grep -rq 'func TestWarnsOnNarrowReplaceHookCandidate(' internal/config`
- [x] 부분집합 compose 파일은 warn되지 않는다 | verify: `/usr/bin/grep -rq 'func TestSubsetComposeFilesAreNotCandidate(' internal/config`
- [x] 플래그·서비스·전처리·인용 파일·개행 명령은 warn되지 않는다 | verify: `/usr/bin/grep -rq 'func TestNewlineReplaceHookIsNotCandidate(' internal/config`
- [x] `up`/`down`/`stop`/`restart`와 제거된 `clean` 훅은 이 검사가 건드리지 않는다 | verify: `/usr/bin/grep -rq 'func TestOnlyBuildAndLogsAreReplaceHookCandidates(' internal/config`
- [x] warning 문구가 훅 경로와 사람 검토 필요성을 함께 말하고 동등·삭제를 주장하지 않는다 | verify: `/usr/bin/grep -rq 'func TestWarnsOnNarrowReplaceHookCandidate(' internal/config`

## Notes

- 출처: `docs/dogfood/scripton-nd-stack.md` 개선점 2. 같은 문서의 `logs` 훅은 추가 `-f`
  플래그를 가지므로 이 안전한 후보 검사의 대상이 아니다. 이 검사는 사람 판단을 대체하지
  않고, 더 단순한 선언형 중복 후보만 검토 목록으로 좁힌다.
- TASK-320에서 분리됐다. 320의 나머지 세 항목은 파서·매칭 결함이었고 이것만 신규 검사다.
