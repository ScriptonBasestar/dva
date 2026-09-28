# W07b2b — CE host lifecycle evidence

2026-09-28 실제 host 검증. DVA는 TaskChain queue/verdict를 읽고 CE가
Git/worktree와 보드 상태를 쓴다. `dva task-queue-start`는 현재 보드가
0/0 `empty`여서 실제 CE start를 호출하지 않았다. 아래 CE 수명주기
호출은 host 원어를 직접 실행한 증거이며, 브리지의 단일 후보 스텁 검증과
합쳐도 과거 외부·사람 카드의 terminal 증거가 되지는 않는다.

## 실행 환경

- CE 설치본은 처음 `dba2348`이어서 `run-discard`가 없었다. 현재 CE
  소스 `2ee38b2`의 task-runtime/CLI 관련 Go 테스트를 통과한 뒤
  지정 `make install_cli`로 설치했다. `ce --version`은
  `2ee38b2b288a48cf040adb754f3f2da77d669e04`, clean을 보고했다.
- DVA와 task-manager-devbox 모두 `ce task run-doctor --json`이 ACTIVE였다.
- TaskChain 제품 로컬 바이너리는 W07b1/W07b2a에서 사용한 `9e8fac7`
  SHA-256 `d2ca31f2880a1420efef3503dba2333f62c0ddd84648b06f4c75b5a62ad6ecd0`이다.

## 같은 owner와 discard

DVA runtime `w07b2b-host-evidence`에서 같은 owner의 `run-start` 재호출은
`ACTIVE existing execution returned`를 냈고 worktree inventory에는
해당 경로가 하나였다. 이것은 다른 owner와의 경합을 재현한 결과가 아니다.

별도 DVA runtime `w07b2b-discard-smoke`를 clean/unpushed 상태에서
`ce task run-discard --reason 'clean unpushed host rollback smoke' --json`으로
종료했다. 영수증은 `operation: discard`, `status: ABORTED`와
worktree/local/remote branch removed 모두 `true`를 보고했다.
`run-status`도 ABORTED였고 Git branch/worktree inventory에 잔여가 없었다.
`run-abort`와 `run-discard`를 같은 동작으로 취급하지 않는다.
CE 원장에는 이 한 discard 호출 안에서 cleanup 전 intent 영수증
`operation: discard`, `status: BLOCKED` 뒤 최종 `status: ABORTED`가 남는다.

## pre-integration refusal과 재시도

task-manager-devbox runtime `batch-039-host-retry`의 문서 변경이 dirty일 때
`ce task run-finish --json`은 exit 1, `reason: task worktree has uncommitted
changes`를 냈다. 이 시점 runtime과 마지막 영수증은 **ACTIVE/start**였고
worktree/branch 제거는 모두 false였다. `status: BLOCKED` 영수증으로
기록하지 않는다. 원본 응답은 task-manager-devbox의
`.omo/evidence/W07b2b/host-retry-refusal.json`에 보존했다.

변경을 `d0635bf`로 커밋하고 즉시 push한 뒤 같은 runtime의
`run-finish --json`을 재시도했다. 결과는 `operation: finish`,
`status: DONE`, `sourcePushed: true`, worktree/local/remote branch
removed 모두 `true`였다. task-manager-devbox source `master`와
`origin/master`는 `d0635bf8bf43c29edb9dc4c230cb260a3207dfaa`로
일치했고 task branch/worktree는 inventory에서 사라졌다.

## 남은 경계

현재 DVA의 TASK-451 완료 카드는 별도 done-review PASS와 CE 보드 gate
READY 후 `done`으로 이동했고, CE `run-finish`가 그 커밋 `cbcfc888`을
DVA source/origin에 통합·회수했다. 이 문서의 TASK-452도 별도 review와
gate를 거쳐 같은 전이를 검증한다.

다른 owner 경합, 실제 외부·결정 카드의 사람 증거와 terminal 전이,
별도 실패 호출이 남긴 BLOCKED 실행 상태의 후속 복구, 통합 성공 후 cleanup
실패에만 적용되는 `run-recover`는 이번 검증으로 주장하지 않는다. DVA ISSUE-004/006은
그 기준이 실제 경로에서 충족될 때까지 열린 상태다.
