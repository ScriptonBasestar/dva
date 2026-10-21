---
id: ISSUE-515
title: "Hosted candidate test cancellation and rerun authorization remain open"
type: bug
priority: P1
effort: S
exec-tier: strong
status: done
severity: medium
ownership: local
needs-human: true
execution-mode: external
human-grade: human
discovered-in: "Hosted candidate test cancellation and rerun authorization remain open"
discovered-at: 2026-10-14
created: 2026-10-14
resolution: fixed
resolved-at: 2026-10-21T00:00:13Z
resolution-summary: "Resolved as fixed."
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
[API·요청 증거](../../../docs/hosted-candidate-followup.json)는 인증 토큰을 포함하지 않는다.

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

- 같은 지점은 후보 밖에서도 반복된다. master 실행 `38907704289`(`3070ea1c`),
  `39372724570`(`129a2627`), `39379552072`(`89caebc9`)의 test job도 cancelled이고
  Test `in_progress`·`Preserve test progress` `pending`이며, 확인한 두 실행의
  마지막 `run` 이벤트도 같은 `malformed/execution`이다. 출력 정지까지의 시간도
  Test 시작 후 약 90초로 같다.

해석(가설): 테스트 프로세스 안의 정지라면 5분 Go deadline이 스택을 남겨야 하므로,
관측은 test 바이너리 정지보다 runner와 control plane의 통신 상실에 더 부합한다.
다만 세 commit·네 실행에서 같은 하위 테스트에 수렴하므로 무작위 heartbeat 기아로는
설명이 부족하다. 이 하위 테스트(또는 같은 시점의 다른 동작)가 Linux runner에서
runner 프로세스 자체를 잃게 만드는 동작을 일으킨다는 가설을 우선 검증 대상으로 둔다.
원인은 미확정이다.

원본 로그는 primary checkout의 무시된
`tmp/hosted-followup-20261020/`(`test-job-attempt1.log`, `test-job-attempt2.log`,
`local-internal-cli-89caebc9.log`)에 보존한다.

## 2026-10-20 조사 — 정지 지점은 테스트가 아니라 로그 크기에 묶여 있다

Linux 재현(golang:1.26.5 linux/arm64 컨테이너, 비root, `GOMAXPROCS=1`,
`GOTESTFLAGS=-json`, `CI=true`, CI와 같은 `timeout 10m sh scripts/without-dva.sh make test`):

- `internal/cli` 단독(`-race -json`)은 19.2s에 통과했고 컨테이너도 정상 유지됐다.
- 전체 `make test`도 정지 없이 끝났다(약 3분 20초). 실패한 `internal/jobrun`,
  `internal/secretpush`, `tools/installcheck`는 컨테이너에 없는 외부 도구 탓으로 보이며
  hosted 정지와는 무관하다(미검증).
- 이 실행에서 `malformed/execution`의 `run` 이벤트는 출력 8,228번째 줄,
  1,839,615바이트 지점이다. hosted Test 단계에서 출력이 멈춘 위치(시작 후 8,237~8,238줄,
  약 1.84MB)와 같다.

hosted 취소 실행 7개의 job 로그(`gh run view --job <id> --log`)는 commit과 마지막 테스트가
달라도 같은 크기에서 끝난다.

| 실행 | commit | 마지막 테스트 | 줄 | 바이트 |
|---|---|---|---|---|
| 37552832725 | 38934c4b | `TestManifestLeavesNonReservedInteractionAlone` | 8,566 | 2,260,151 |
| 37554447670 | 406324b7 | `TestManifestLeavesNonReservedInteractionAlone` | 8,568 | 2,260,099 |
| 38533431297 a1 | 6c37f680 | `…/malformed/execution` | 8,564 | 2,259,970 |
| 38533431297 a2 | 6c37f680 | `…/malformed/execution` | 8,566 | 2,260,209 |
| 38907704289 | 3070ea1c | `…/malformed/execution` | 8,564 | 2,260,162 |
| 39372724570 | 129a2627 | `…/malformed/execution` | 8,565 | 2,259,992 |
| 39379552072 | 89caebc9 | `…/malformed/execution` | 8,565 | 2,260,006 |

`gh`가 줄마다 붙이는 접두사(`test\tUNKNOWN STEP\t`, 18바이트)를 빼면 약 2.10MB로 2MiB에
가깝다. 마지막 성공 실행은 2026-09-08(`34203911056`)이며 취소는 2026-10-07 Test 단계에
`-json` 출력과 `tee`가 들어온 `38934c4b`부터 시작됐다(그 사이는 failure).

결론(관측): 정지 지점은 특정 테스트 동작이 아니라 job 로그의 누적 크기 약 2MiB에 묶여 있다.
`malformed/execution`은 현재 테스트 순서에서 그 크기에 도달하는 위치일 뿐이다.

미확정: (1) 그 크기에서 runner가 실제로 멈추는지(파이프 역압으로 test가 정지), 아니면
로그만 잘리고 job은 다른 이유로 끝나지 못하는지. (2) 후자라면 아래 고아 프로세스가 Test
단계의 stdout 파이프를 붙잡아 `tee`가 EOF를 받지 못하는 경로가 후보다. 둘을 가르려면 Test
단계의 stdout 양을 줄이는 CI 변경이 필요하며, 이는 이 카드의 범위 밖이므로 별도 승인 대상이다.

부수 발견: Linux 재현에서 `internal/cli` 테스트가 `sleep 999` 2개(각자 별도 pgid, ppid 1)와
`sleep 600` 1개(`timeout`의 pgid, ppid 1)를 남겼고, 셋 다 패키지 종료 뒤에도 살아 있었다.
별도 카드로 등록한다. 원본은 같은 디렉터리의 `linux-repro-1.log`, `linux-full.jsonl`,
`linux-full-ps.log`, `test-job-run*.log`와 재현 스크립트 `repro/`다.

## 2026-10-20 판별 실험 — Test 단계 출력 분리 (사용자 승인)

사용자가 2026-10-20 위 미확정 (1)/(2)를 가르는 CI 변경을 승인했다. 아래 "후속 범위"의
"새 wrapper" 제외는 이 변경 하나에 한해 해제된다. timeout 연장·검사 생략은 여전히 범위 밖이다.

변경: Test·Integration Test 단계가 `-json` 스트림을 `tee`로 job 로그에 쓰지 않고
`scripts/ci-go-test.sh`를 거쳐 `tmp/ci/*.jsonl` 파일(기존 artifact 업로드 대상)에만 쓴다.
job 로그에는 1분 heartbeat(누적 이벤트 수·마지막 테스트), 명령 종료 시각·exit,
실패 테스트 출력, 그리고 명령의 환경 표식(`CI_GO_TEST_MARK`)을 물려받은 채 남은 프로세스만 남는다.

판정 기준:

- 통과 + 잔존 프로세스 없음: 로그 크기 경로가 원인이었다고 본다.
- 통과 + 잔존 프로세스 있음: 출력이 파일로 가서 고아가 파이프를 붙잡는 경로도 함께
  사라졌으므로 두 가설을 가르지 못한다. ISSUE-522 수정 뒤 재판정한다.
- 여전히 정지: heartbeat의 마지막 시각·이벤트 수와 `end:` 줄의 유무로 test 자체 정지와
  종료 뒤 정지를 구분한다.

로컬 확인: macOS와 Linux(golang:1.26.5 컨테이너)에서 가짜 명령으로 heartbeat, exit 전달,
실패 출력, `setsid sleep 999` 잔존 검출을 확인했다. 원본은
`tmp/hosted-followup-20261020/cigotest/`다.

### 결과 — 로그 크기 가설 기각, 변경은 되돌림

실행 39430861306(작업 브랜치 `dev/claude/mst/chore/issue-515`, `0c818832`), test job
118655264045. test 외 job 7개는 성공했다. Test 단계는 13:30:45Z에 시작해 완료 시각이 기록되지
않았고, job은 `The job has exceeded the maximum execution time of 30m0s`로 cancelled됐다.

- job 로그가 아예 없다(`gh api …/jobs/118655264045/logs` → 404 `BlobNotFound`,
  `gh run view --log` → `log not found`). artifact도 0개다. heartbeat·`end:` 줄을 관측할 수 없다.
- 단계의 `timeout-minutes: 25`도 안쪽 `timeout 10m`도 결과에 드러나지 않았고, 서버 쪽 job 제한(30m)만 작동했다.
- 출력을 거의 내지 않았는데도 같은 정지가 났으므로 "로그 약 2MiB에서 정지" 가설은 기각한다.
  이전 실행의 2MiB 끝점은 runner가 연락을 잃기 전까지 업로드된 양으로 본다(추정).
- 단계 timeout이 작동하지 않고 로그가 업로드되지 않는 양상은 runner 자체가 응답을 잃는 쪽
  (자원 고갈·runner 프로세스 종료)을 가리킨다. 위 GOMAXPROCS 주석의 이전 증상과 같다. 미확정이다.
- 같은 job annotation: `The ubuntu-latest label will migrate to Ubuntu 26 beginning October 19, 2026`.
  정지는 10-07부터라 이 이전만으로 설명되지 않는다.

판별에 기여하지 않았으므로 CI 변경은 되돌렸다(master에는 이 카드 기록만 남는다).
스크립트 사본은 `tmp/hosted-followup-20261020/cigotest/ci-go-test.sh`다. 다음 판별 후보
(runner 자원 관측, 패키지 분할 실행 등)는 다시 CI 변경이므로 별도 승인 대상이다.

## 2026-10-20 정지 시작 커밋 — 0111e8b8

정지는 10-07(38934c4b)이 아니라 09-08에 시작됐다. master CI 이력(`ci.yml`, 09-07..10-20)에서:

- cfe25653(2026-09-08 08:19Z, run 34203911056)까지 연속 success. Test 단계 2분 8초.
- 바로 다음 master push 0111e8b8 `feat(remote): add tracked artifact jobs and secret push`
  (run 34250221955): test job이 35분 뒤 cancelled, 기록된 step 0개. 다른 job은 끝났다
  (`config-env-platform (ubuntu-latest)` failure는 별개).
- 이후 test job은 Documentation 단계 실패(약 20초, step 기록 있음)로 일찍 끝나거나,
  Test 단계까지 가면 step이 in_progress인 채 job 제한으로 cancelled됐다. 09-08 이후
  Test 단계가 완료된 master 실행은 없다. 이력 원본은 `tmp/hosted-followup-20261020/`의
  `master-runs.txt`, `test-step-history.txt`다(앞 200건 기준).
- 0111e8b8과 cfe25653 사이 커밋은 이것 하나다. 새로 들어온 테스트:
  `internal/{jobrun,secretpush,ociverify,remotetarget}`, `internal/config/remote_test.go`,
  `internal/cli/remote_test.go`, `internal/integration/{remote_job,secret_push}_test.go`.

자원 고갈 가설은 로컬 근거가 없다: Linux 컨테이너(4 CPU, 7 GiB 제한, CI와 같은 env)에서
전체 `make test`가 244초에 끝났고 cgroup peak 1,065 MiB, 최대 pids 30이었다
(`repro2/res.log`). 컨테이너의 실패 3패키지(jobrun·secretpush·installcheck)는 위 새 테스트와
겹치지만 컨테이너 환경 차이(상태 디렉터리 안전성 검사, 도구 부재) 때문으로 보이며 정지는 아니다.
hosted runner에는 `gh`·`kubectl`·`sops` 계열 도구와 Actions 환경이 있어, 새 테스트가
로컬과 다른 경로를 탈 수 있다(미검증).

다음 판별(별도 승인 필요, 실험 브랜치 한정): `dev/**` 브랜치에서 Test 단계를 위 새 테스트
패키지 묶음과 나머지로 나눠 각각 실행해 어느 쪽이 runner를 잃는지 본다. 검사 생략이나
master 반영이 아니라 이분 탐색용 일회성 실행이며, 결과만 이 카드에 남긴다.

## 2026-10-20 hosted 이분 탐색 (사용자 승인, 실험 브랜치)

실험 브랜치 `exp/claude/mst/chore/issue-515`에서 `ci.yml`만 바꿔 3회 실행했다. 브랜치는
통합하지 않고 증거로 보존했다(실행은 `ce task run-abort`로 닫음). 정지 판정은 앞 절과 같다:
step 제한·`timeout 12m`이 어느 것도 걸리지 않고, step이 in_progress인 채 job이 30분에
cancelled되며 job log가 남지 않는다(`log not found`).

| 회차 | run | 커밋 | 변형 | 결과 |
| --- | --- | --- | --- | --- |
| 1 | 39444449630 | 23da19bb | 의심 패키지별 단독 job 7개(jobrun·secretpush·ociverify·remotetarget·config·cli·나머지) | 정지 0/7. `rest`만 TestLockRejectsAnotherProcess(기존 flaky)·TestUpstreamRefsSweepsTheRealCorpus 실패 |
| 2 | 39445229921 | 164c5aa3 | 선행 단계 토글 후 `make test`: full·docs-only·test-only·build-only·build-dogfood | **build-dogfood만 정지**. test-only·build-only는 Test 약 4.5분에 일반 실패. full·docs-only는 master와 같이 Documentation에서 실패 |
| 3 | 39451524530 | 6ad5a78f | build→dogfood 뒤 전체 테스트 2회(repeat-a/b), 뒤 테스트를 cli / 새 패키지(cli 제외) / 나머지로 분할 | **repeat-a 정지, repeat-b는 job 3분 48초(15:46:37–15:50:25Z)에 일반 실패**. 분할 3개 정지 0/3 |

판단:

- 정지는 결정적이지 않고 간헐적이다. 같은 build→dogfood→전체 테스트 조합이 3회 중 2회
  정지했다. 2회차의 "dogfood가 선행 조건"이라는 읽기는 표본 1개에 기댄 것이라 확정이 아니다.
  dogfood 없는 변형은 2회 모두 정지하지 않았지만 표본이 작다.
- 패키지를 나눈 실행은 1·3회차 합계 10개 job에서 한 번도 정지하지 않았다. 정지는 한 프로세스
  트리에서 전체 패키지를 이어 실행할 때만 관측됐다.
- 정지하지 않은 전체 실행의 공통 실패는 `tools/doccheck` TestUpstreamRefsSweepsTheRealCorpus다.
  정지와는 별개로 보인다.
- 로컬 Linux 컨테이너에서 build→dogfood→`make test` 순서도 재현했지만 정지하지 않았다
  (약 5분, cgroup peak 1,069 MiB, pids 최대 30, dogfood 전후 `git status --ignored` 동일;
  `tmp/hosted-followup-20261020/repro3/`). dogfood는 fixture·실행 파일 복사본·dry-run HOME을
  모두 `os.MkdirTemp`로 만들고 `defer`로 지운다(`tools/skilldogfood/main.go`,
  `verify_steps.go`, `skillcopy.go`).
- 미검증 가설: runner 자체가 신호를 받는 경우다. `internal/lifecycle/process_group_unix.go`와
  `internal/cirun/os_unix.go`는 `syscall.Kill(-pid, ...)`로 프로세스 그룹에 신호를 보낸다.
  pidfile 값이나 재사용된 pid가 runner와 같은 사용자의 그룹을 가리키면 step timeout 없이
  runner를 잃는 현상과 맞는다. 간헐성(pid 재사용 타이밍)과도 맞지만 hosted 증거는 아직 없다.

원본 로그: `tmp/hosted-followup-20261020/bisect/`(`run-39444449630.log`, `run-39451524530.log`,
`job-118727293092.log`, `job-118727293214.log`). 정지한 job의 로그는 GitHub에도 남지 않았다.

다음 판별 후보(별도 승인 필요): 정지가 간헐적이므로 변형마다 여러 표본이 필요하다. 그룹 신호
가설은 hosted에서 `Kill(-pid)` 호출 직전 대상 pgid를 runner 그룹과 비교해 기록하는
진단 실행이나, pid ≤ 1·자기 그룹을 거부하는 코드 수정 뒤 반복 실행으로 판별할 수 있다.
후자는 원인 수정이라 이 카드의 조사 범위를 넘는다.

## 2026-10-20 그룹 신호 진단 (사용자 승인, 실험 브랜치)

실험 브랜치 `exp/claude/mst/test/issue-515`(base eaf757e4, run-abort로 보존)에 진단용
`internal/issue515guard`를 넣었다. 이 코드는 `Kill(-pgid)` 호출마다 대상 pgid, 자기 pgid,
조상 프로세스의 pgid를 `$ISSUE515_KILL_LOG`에 남긴다. pgid ≤ 1, 자기 그룹, 조상 그룹은
거부하고 stack을 기록한다. 판정은 job이 약 30분에 로그 없이 cancelled면 "정지"로 했다.

| 회차 | run | commit | 변형 | 정지 |
|---|---|---|---|---|
| 4 | 39457591910 | 93b39ebf | guard 전 지점 적용 | 0/4, 거부 0 |
| 5 | 39458655653 | 5cf87c6a | guard 전 지점 적용, 프로세스 트리 기록 | 0/8, 거부 0 |
| 6 | 39459642340 | 11e69502 | 같은 batch에서 guard 적용과 원본 대조 | guard 적용 0/6, 원본 5/6 |
| 7 | 39464255573 | 97b76aa4 | 한 지점만 guard: lifecycle-prod·cirun·doctor·lifecycle-tests | lifecycle-tests 0/3, 나머지 7/9 |
| 8 | 39468952208 | 9cf5bd97 | lifecycle-tests 분할: exec-cleanups·dryrun-cleanup·원본 | exec-cleanups 0/4, dryrun-cleanup 4/4, 원본 2/4 |

판단:

- 그룹 신호 가설은 기각한다. 4~8회차 동안 guard가 runner 그룹이나 조상 그룹을 겨냥한 호출을
  거부한 적이 한 번도 없다. 5회차에서 hosted 프로세스 트리를 확인했다. step bash, Runner.Worker,
  Runner.Listener, hosted-compute-agent가 같은 pgid에 있고, `timeout` 아래 make·go·test
  binary는 별도 pgid다.
- 정지와 함께 움직인 지점은 `internal/lifecycle/process_test.go`에 있는 테스트 cleanup 3곳이다.
  `t.Cleanup(func() { _ = exec.Command("kill", "-9", "-"+strconv.Itoa(pid)).Run() })` 형태이고
  TestProcessPlugin_DownWaitsForExitBeforeRemovingState,
  TestProcessPlugin_DownCancellationRetainsStateForLiveProcess,
  TestProcessPlugin_Restart_HungProcess에 있다. 이 3곳을 `syscall.Kill(-pid, SIGKILL)`로 바꾼 변형(6회차 전 지점, 7회차 lifecycle-tests, 8회차 exec-cleanups만)은
  합계 13/13 job이 정지하지 않았다. 원본이거나 다른 지점만 바꾼 변형은
  18/23 job이 정지했다.
- 8회차 정지하지 않은 원본 job(original-a·c)의 `make test` 직후 프로세스 목록에는
  `sh -c trap '' TERM; ... while true; do sleep 0.05; done` 루프 2개가 자기 pgid로 살아 있었다.
  exec-cleanups 변형에서는 같은 루프가 남지 않았다. kill log에는 cleanup 3건이 기록됐고 결과는
  ESRCH 1건, 성공 2건이었다. 즉 exec로 띄운 `kill -9 -<pid>`는 이 두 그룹을 죽이지 못했다.
  `exec.Command(...).Run()`의 오류를 버리므로 실패 원인은 로그에 없다.
- 미검증: exec로 띄운 kill이 실패하는 이유는 아직 모른다. hosted의 procps-ng 4.0.4
  `kill -9 -N`이 kill(-N, SIGKILL)을 호출함은 별도로 확인했다(`killcheck/`). 남은 TERM 무시
  루프가 runner를 잃게 하는 경로도 아직 모른다. 루프가 남아도 정지하지 않은 job이 있으므로
  루프가 남는 것은 정지의 필요조건 후보일 뿐 충분조건은 아니다. ISSUE-522의 고아 `sleep` 관측과
  관련이 있을 수 있다. 로컬 macOS에서 원본 테스트는 고아를 남기지 않았다.
- 공통 일반 실패인 `tools/doccheck` TestUpstreamRefsSweepsTheRealCorpus는 정지와 무관하다.

원본 로그: `tmp/hosted-followup-20261020/diagnose/`. 8회차는 `r8-job-<id>.log`로 저장했다.

다음 후보(별도 승인 필요): 위 cleanup 3곳을 `syscall.Kill(-pid, syscall.SIGKILL)`로 바꾸고
오류를 버리지 않는 원인 수정을 하고, hosted에서 반복 실행해 정지가 없는지 확인한다.
이는 이 카드의 조사 범위를 넘는 코드 변경이다.

## 2026-10-21 원인 수정과 hosted 검증 (사용자 승인)

- 수정: master `647d56c6`. `internal/lifecycle/process_test.go`의 cleanup 3곳을
  `killProcessGroupInCleanup`(`internal/lifecycle/process_group_kill_unix_test.go`)로 바꿨다.
  이 helper는 `syscall.Kill(-pid, SIGKILL)`을 보내고, ESRCH가 아닌 오류는 테스트 실패로 만든다.
  pid ≤ 1과 자기 그룹은 거부한다. restart 테스트가 실패할 때 교체 프로세스도 정리한다.
  검토에서 발견한 운영 코드의 같은 경계 결함은 ISSUE-525로 분리했다.
- 반복 검증: 실험 run 39513943508(`exp/claude/mst/fix/issue-515` `5d61398f`, base `69c907fc`)에서
  build→dogfood→`make test`를 8개 job으로 돌렸다. **정지 0/8**이다(job 4~5분). 7개는 성공했다.
  g(job 118941445932)는 `internal/cirun` TestLockRejectsAnotherProcess
  (`child never held lock: <nil>`)로 실패했다. 이 실패는 1회차에도 관측된 기존 flaky로, 정지와는
  무관하다. 이 결함은 ISSUE-526으로 분리했다. 모든 job의 `make test` 직후 목록에 `trap '' TERM`
  루프가 남지 않았다. ISSUE-522의 `sleep 999` 2개와 `sleep 600` 1개는 계속 남았다.
- 실제 CI: 수정 후 master `69c907fc` run 39513864028과 task 브랜치 run 39513836067은 test를
  포함한 전 job이 성공했다. 같은 시각 수정 전 `3cf01c72`의 task 브랜치 run 39511894688은 test가
  약 35분 뒤 cancelled됐다. 이는 이전과 같은 정지다. master run 39511919376의 test cancelled는
  23:21:01, 곧 다음 master run이 시작된 시각에 끝났으므로 동시성 취소로 보고 정지 표본에서 뺐다.
- 판단: 수정 전 대조군은 23개 중 18개가 정지했다. 수정 후에는 실험 8개와 CI 2개, 모두 10개 중
  0개가 정지했다. 이 카드의 hosted test 정지는 해소된 것으로 본다. 원래 대상인 후보
  `6c37f680`의 재실행과 재실행 권한 문제는 정지 원인이 사라진 지금 새 후보의 CI로 대체할 수
  있다. 다만 이 카드는 `human-grade: human`이므로 종료 판정은 사람이 한다.
- 증거: `tmp/hosted-followup-20261020/verify/`.

## 후속 범위

Actions write 권한을 가진 승인된 인증으로 기존 지원 경로를 사용하고, 요청 뒤 실제
run attempt·source SHA·종료 결과를 확인한다. 먼저 가장 좁은 지원 범위를 확인한다.
권한 부족과 cancelled 상태의 개별 job 제약을 혼동하지 않는다. 재실행에서 정지가
재현될 때만 기존 CI의 timeout·하위 프로세스·로그 보존 경계를 조사한다.

현재 로컬 full CI 9단계와 같은 commit의 snapshot 검증은 성공했다. 이를 hosted test
성공으로 대체하지 않는다. timeout 연장, 검사 생략, 새 wrapper·validator·writer,
token 값 수집·자동 권한 확대, 게시·설치는 범위 밖이다.
