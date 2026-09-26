---
id: ISSUE-045
title: "done-finalize scan spawns uncached git subprocesses per card and file"
type: bug
priority: P2
status: todo
severity: medium
discovered-in: "done-finalize scan spawns uncached git subprocesses per card and file"
discovered-at: 2026-09-27
ownership: upstream
created: 2026-09-27
---

## Summary

<!-- One paragraph: what is wrong and who it affects. -->

## Reproduction

1. 

## Expected vs Actual

- Expected: 
- Actual: 

## Evidence (2026-09-27, ce-agent-kit dba2348b)

`ce task done-finalize-all --dry-run`이 dva(done 19장, 추적 파일 1315개)에서 300초
넘게 출력 없이 돌았다. 120초 실측에서 user 48s + sys 57s CPU를 썼고 I/O 대기나
프롬프트가 아니었다. 원인은 `task_storage_cleanup.go:112-121`의 (done 카드 × 저장소
파일) 중첩 루프가 매 반복마다 캐시 없이 `gitTracks`(`git ls-files` 서브프로세스,
`task_storage_mutate.go:205`)와 `matchesSource`(`git show` 2회, `task_storage_cleanup.go:291`)를
실행하기 때문이다. 단일 카드 `done-finalize`도 같은 전체 스캔(`task_cleanup.go:59`)을 탄다.
`--dry-run` 자체는 비변경이다.

## Expected

경로별 결과를 호출당 한 번만 계산(메모이즈)해 비용이 카드 수에 곱해지지 않는다.

## 적용 실측 (2026-09-27)

실제 `ce task done-finalize-all`(done 20장)은 30분 38초(user 742s, sys 847s)를 돈 뒤
`cleanup source changed; preview again`으로 아무것도 적용하지 못하고 끝났다. 실행 중
다른 태스크가 master에 통합되었기 때문이다. 스캔이 길수록 동시 통합과 경합해 헛돌 확률이
커지므로, 이 비용은 성능뿐 아니라 정리 작업의 완료 가능성 문제다.
