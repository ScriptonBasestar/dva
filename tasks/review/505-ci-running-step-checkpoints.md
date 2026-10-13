---
id: TASK-505
title: "Persist running CI step checkpoints before supervisor termination"
type: bug
priority: P1
effort: S
exec-tier: standard
status: review
created: 2026-10-14
quality-review: pass
---

## Summary

SigDock의 native gate가 외부 90분 제한으로 종료된 뒤 DVA receipt는 running,
steps=null, duration=0s로 남았다. status는 살아 있는 lock이 없으면 stale로 표시한다.
이는 terminal 결과를 복원할 근거가 없는 상태다. 실행 step의 시작과 완료를 atomic
checkpoint로 기록해, supervisor가 강제 종료돼도 마지막으로 확인한 단계는 남긴다.
started_at은 scheduler의 배정 시각이며 final receipt에서도 동일하게 유지한다.
프로세스 spawn 성공 시각이나 실제 CPU 사용량은 측정하지 않는다.
판정·락·worker 기본값·검사 범위는 바꾸지 않고 종료 시각이나 PASS를 추정하지 않는다.

## Scope

- 구현: internal/cirun/cirun.go, cirun_steps.go, cirun_report.go 및 해당 focused regression.
- 문서: docs/53-ci-profiles.md, CHANGELOG.md와 이 카드.
- 구현 standard, 독립 리뷰 strong. 실제 라우팅·모델은 이 요구와 별도로 기록한다.
- 새 heartbeat 서비스·큐·프로필·전역 병렬도 변경은 포함하지 않는다.
- step의 allowlist worker 설정(DVA_CI_JOBS, CARGO_BUILD_JOBS, RUST_TEST_THREADS)을
  양의 정수일 때만 선택적 worker_budget에 기록한다. 임의 환경값은 기록하지 않는다.

## Steps

1. scheduler의 step 배정/완료 이벤트에서 기존 writeReport의 atomic 저장을 사용한다.
2. checkpoint I/O 실패를 드러내고 실행 취소 및 최종 receipt 흐름을 유지한다.
3. 결정적인 mid-run test에서 durable running step과 최종 정상 결과를 검증한다.
4. scoped 검증과 독립 실제 diff 리뷰 후 선언된 native 전달 게이트를 한 번 수행한다.

## Completion Criteria

- [x] 진행 중 step의 배정 시각·worker 설정을 durable receipt에 남기고 저장 오류 시 worker를 취소·회수하며 완료 판정은 기존 계약을 유지한다. | verify: `go test ./internal/cirun -run 'TestRunCheckpointsStartedStepBeforeCompletion|TestStepWorkerBudgetRecordsOnlyEffectiveNumericLimits|TestCheckpointFailureCancelsAndJoinsRunningSteps' -count=1`

## Evidence

- 이전 실행 ID: eecf0e0ebe1e1626227d8fc04f377ca1. supervisor 종료 후 initial running receipt와 0바이트 로그를 관찰했다.
- 0바이트 로그를 버퍼 유실로 단정하지 않는다. 기존 로그는 직접 io.MultiWriter이며 leaf가 자기 로그로 출력할 수 있다.
- 사전 card ID 검증용 make doc-check PASS. CE new-task admission은 shell syntax로 criterion을 실행하지 않았으며 이는 테스트 증거가 아니다.
- 구현·scoped 테스트·독립 리뷰·native 전달 결과는 실제 완료 뒤 갱신한다.
- 최초 회귀 검사에서 scheduler snapshot의 started_at이 0임을 발견해 step 시작 전
  timestamp를 기록하도록 수정했다. 수정 후 focused test PASS(0.394s).
- 저장 오류의 조기 반환이 다른 실행 worker를 남기지 않도록 취소 후 drain을 보완했다.
- 시간 기반 sleep 대신 READY handshake와 명시적 취소·bounded join으로 회귀를 검증한다.
- worker_budget의 기본값·상속·step override·비정상 값/임의 환경값 비유출 표본,
  checkpoint와 저장 오류 sibling 취소·join 회귀를 함께 실행해 최종 3개 PASS(0.742s)다.
  이전 부분 실행은 0.496s였다. 실제 CPU 사용량 측정으로 해석하지 않는다.
- strong 독립 실제 diff 리뷰 PASS. started_at의 running/final 동일성 및 dispatch 의미,
  atomic 저장·JSON 호환·error chain·worker 회수·임의 환경값 미유출을 검토했다.
- 기준 f83b4440에서 full CI run `337af221347454a77909a29a7dccd7e4`는
  4m5.854s 후 기존 Docker endpoint 전제 테스트로 FAIL했다. format/docs/vet/lint는 PASS,
  cirun race/coverage는 PASS(21.097s, 83.0%). build/integration/dogfood는 미실행이다.
- 실패는 `TestDockerSocketAndDaemonAgreeWhenDaemonReachable`의 환경 변경 전제이며
  기존 ISSUE-510이 기준 f83b4440에서 같은 실패를 기록했다. 현재 master 1bd93f69에는
  테스트 수정이 전달돼 있다. cirun은 기준 이후 변경이 없음을 actual diff로 확인했다.
- 최신 기준으로 task-owned delta를 옮기기 전 checkpoint를 보존한다. 이 실패를
  최신 기준의 PASS로 재사용하거나 동일 입력으로 전체 검증을 재실행하지 않는다.
