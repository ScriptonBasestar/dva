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
upstream-ref: "ce-agent-kit (not yet filed; evidence lives here until reported)"
created: 2026-09-27
---

## Summary

<!-- One paragraph: what is wrong and who it affects. -->

## Reproduction

1. 

## 소유권 — 상류다

결함은 ce-agent-kit의 `done-finalize` 정리 스캔 구현에 있다. dva는 보드 규모와 실측
증거만 제공하며, 우회용 로컬 정리 스크립트를 두지 않는다.

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

재시도(master 고정)는 4시간 37분(user 7103s, sys 7411s) 만에 19장을 적용했다. 적용
방식은 아카이브가 아니라 **삭제**여서 PLAN-010/011의 `children`이 존재하지 않는 카드를
가리키게 되어 `planprogress`와 `validate`가 실패했고 16개 링크가 깨졌다. 저장소 선례
(0ba36ee1)대로 삭제를 되돌리고 카드를 `tasks/_archive/2026-09/`에 `archived-at`과 함께
옮겨 해결했다. 계획 자식 카드를 삭제하는 finalize는 이 보드 규약과 맞지 않는다.
