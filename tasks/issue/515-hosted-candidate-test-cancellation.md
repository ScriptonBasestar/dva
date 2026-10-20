---
id: ISSUE-515
title: "Hosted candidate test cancellation and rerun authorization remain open"
type: bug
priority: P1
effort: S
exec-tier: strong
status: todo
severity: medium
ownership: local
needs-human: true
execution-mode: external
human-grade: human
discovered-in: "Hosted candidate test cancellation and rerun authorization remain open"
discovered-at: 2026-10-14
created: 2026-10-14
---

## Summary

DVA 후보 `6c37f680`의 hosted test가 취소돼 공개 전 후보 검증이 미완료다.
현재 GitHub 인증으로 재실행도 승인되지 않는다. DVA CI 검증 후속을 추적하는
소유 카드이며 token 설정이나 CI 구조 변경을 자동 승인하지 않는다.

## 소유권 — 이 저장소다

DVA의 `.github/workflows/ci.yml`과 공개 전 후보 검증 판정을 이 저장소가 소유한다.
따라서 필요한 CI 결과와 재실행 선행 조건은 여기에서 추적한다. 인증 권한은 승인된
운영자가 준비하는 외부 실행 조건이며, 이 카드는 GitHub 권한 관리나 workstation의
인증 설정 변경을 대신 수행하지 않는다.

## Reproduction

1. 실행 `38533431297`과 `38533156427`의 test job은 cancelled다. 다른 다섯 job은
   성공했지만 전체 hosted PASS는 아니다.
2. test job은 약 35분 뒤 종료됐다. job의 선언 예산은 30분이고 내부에는 10분
   `timeout`이 있다. 마지막 출력 이후 artifact 보존 단계로 진행하지 않았다.
   deadline 취소와 일치하지만 특정 테스트·하위 프로세스의 정지는 입증되지 않았다.
3. `gh run rerun 38533431297 --job 115750477939 --repo ScriptonBasestar/dva`는
   exit 1, `job 115750477939 cannot be rerun`이다.
4. 기존 전체 workflow 재실행도 exit 1,
   `Resource not accessible by personal access token`이다. run attempt는 1로 유지됐다.

원본 job 로그·거절 결과는 소유 gzh-cli-devbox의
`tmp/dva-project-tooling-evidence-20261014/hosted-followup/`에 보존한다.

## Expected vs Actual

- Expected: 같은 후보의 hosted test가 종료 상태와 로그를 남기고, 승인된 인증으로
  필요한 검증을 재실행할 수 있다.
- Actual: test 취소 후 원인과 최종 결과가 미확정이며 현재 인증으로 재실행하지 못했다.

## 2026-10-14 재실행 후속 확인

사용자 요청으로 기존 인증에서 가장 좁은 지원 경로부터 재확인했다.
실행 `38533431297`의 API 조회로 test job database ID `115750477939`와
source `6c37f680d8dd36bbcbb38cf018d03c667054ffef`를 대조했다.

- 개별 job 재실행: exit 1, `job 115750477939 cannot be rerun`.
- `--failed` 범위 재실행: exit 1, `Resource not accessible by personal access token`.
- 요청 전후 attempt는 1, 상태는 completed/cancelled, source SHA도 동일하다.

새 실행은 시작되지 않았다. `--failed`가 취소된 job을 포함하는지는 이번 거부만으로
확인할 수 없다. 권한 거부 이후 전체 workflow 요청이나 인증 변경을
추가 시도하지 않았다. 취소 원인은 여전히 미확정이며 이 카드는 열린 상태로 유지한다.
[API·요청 증거](../../docs/hosted-candidate-followup.json)는 인증 토큰을 포함하지 않는다.

실행 인계에 필요한 미확인 입력은 승인된 Actions 재실행 인증과 그 준비 경로다.
현재 설정에서 같은 요청을 반복하거나 token 값을 수집하지 않는다. 인증이 준비되면
작업 디렉터리와 기존 `gh`의 인증 선택을 확인한 뒤 지원되는 최소 재실행 범위에서
attempt·source SHA·최종 결과를 다시 확인한다. 작업 worktree는 인계에 필요하지 않다.

## 2026-10-20 attempt 2 — 정지 재현

실행 `38533431297`의 재실행 요청이 수락돼 attempt 2가 같은 source
`6c37f680d8dd36bbcbb38cf018d03c667054ffef`에서 돌았다. 결과는 다시 cancelled이고
test 외 job은 모두 성공했다. 따라서 위 "재실행에서 정지가 재현될 때" 조건이 충족됐다.

관측 사실:

- 두 attempt 모두 Test 단계 시작 후 70~90초에 출력이 멈췄다
  (attempt 1: 08:12:36 시작 → 08:13:45 마지막, attempt 2: 11:10:54 → 11:12:24 UTC).
  마지막 JSON 이벤트는 둘 다 `internal/cli`의
  `TestRunRefusesIncompleteOwnerEnvironmentBeforePreviewExecutionAndConfirmation/malformed/execution`
  `run`이다. 다음 하위 테스트 `preview`의 `run` 이벤트는 없다.
- attempt 2는 마지막 출력 후 약 33분 뒤(11:45:08) 취소됐다. 그 사이
  `go test -timeout 5m`의 panic·goroutine dump도, `timeout 10m`의 종료 흔적도 없다.
- `if: always()`인 `Preserve test progress`는 `pending`으로 남았고, 실행의 artifact는
  0개다. 취소 뒤에도 Test 단계는 `in_progress`로 보고된다.
- 같은 테스트는 로컬(darwin, HEAD `89caebc9`, stdin `/dev/null`)에서 단독 0.01s,
  `internal/cli` 패키지 전체 25.8s로 통과한다. `6c37f680..89caebc9`에는
  `internal/cli`·`internal/config` 변경이 없다.

해석(가설): 테스트 프로세스 안의 정지라면 5분 Go deadline이 스택을 남겨야 하므로,
관측은 test 바이너리 정지보다 runner와 control plane의 통신 상실에 더 부합한다.
workflow 주석의 이전 heartbeat 기아 기록(-race, 2코어)과도 일치한다. 두 attempt가
같은 지점에서 끊긴 이유는 설명되지 않았고 원인은 여전히 미확정이다.

원본 로그는 primary checkout의 무시된
`tmp/hosted-followup-20261020/`(`test-job-attempt1.log`, `test-job-attempt2.log`,
`local-internal-cli-89caebc9.log`)에 보존한다.

## 후속 범위

Actions write 권한을 가진 승인된 인증으로 기존 지원 경로를 사용하고, 요청 뒤 실제
run attempt·source SHA·종료 결과를 확인한다. 먼저 가장 좁은 지원 범위를 확인한다.
권한 부족과 cancelled 상태의 개별 job 제약을 혼동하지 않는다. 재실행에서 정지가
재현될 때만 기존 CI의 timeout·하위 프로세스·로그 보존 경계를 조사한다.

현재 로컬 full CI 9단계와 같은 commit의 snapshot 검증은 성공했다. 이를 hosted test
성공으로 대체하지 않는다. timeout 연장, 검사 생략, 새 wrapper·validator·writer,
token 값 수집·자동 권한 확대, 게시·설치는 범위 밖이다.
