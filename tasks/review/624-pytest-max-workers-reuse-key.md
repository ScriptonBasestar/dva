---
id: TASK-624
title: "PYTEST_MAX_WORKERS가 재사용 키에 없어 승인 cap이 키 밖 변수를 만든다"
type: bug
priority: P3
effort: S
exec-tier: strong
status: review
created: 2026-11-18
---

## Summary

TASK-622 2차 독립 리뷰가 확인한 코너다. admission이 켜지면
`ciEnv`(internal/cirun/cirun_steps.go:487)가 마지막에
`resourceadmission.CapWorkerEnvironment`(internal/resourceadmission/workers.go:10)을
적용하는데, 이 cap은 네 worker 변수(`GOMAXPROCS`·`CARGO_BUILD_JOBS`·
`RUST_TEST_THREADS`·`PYTEST_MAX_WORKERS`)를 limit으로 묶고 **없으면 limit 값으로
새로 만든다**(workers.go:27-34). 재사용 키는 명령이 받는 것과 같은 cap 후
환경에서 계산되지만 키 목록(internal/cirun/ci_reuse.go:156)에는
`PYTEST_MAX_WORKERS`가 없다.

## Reproduction (리뷰가 클론에서 관측, 2026-11-18)

- 키잉된 세 변수(`CARGO_BUILD_JOBS`·`GOMAXPROCS`·`RUST_TEST_THREADS`)를 모두
  limit 이하로 고정하면, 승인 실행과 비승인 실행이 같은 재사용 키를 공유하면서
  서로 다른 `PYTEST_MAX_WORKERS`(부재 vs limit)로 실행된다. `DVA_CI_JOBS`·
  `DVA_RESOURCE_WORKER_LIMIT`·`DVA_RESOURCE_SUPERVISED`도 키 밖에서 다르다.
- limit이 세 변수를 실제로 묶는 배치(예: 예산 4에 limit 2)에서는
  `CARGO_BUILD_JOBS` 파생·cap이 키에 도달해 키가 달라진다 — 간격은 묶지
  않는 코너에만 있다.

## Cause

cap이 키잉되지 않은 변수(`PYTEST_MAX_WORKERS`)를 새로 만들면서, 키 목록은
그 변수를 모른다. TASK-622 이전 master도 limit이 묶지 않을 때 같은 간격이
있었으므로 622가 새로 만든 것은 아니고(키잉 후보가 하나 줄어) 확대된 것이다.
worker 수 분산은 입력 키잉 재사용 전체가 세운 전제 아래 결과 중립적이라
심각도는 낮다.

## Mitigation 후보

1. `PYTEST_MAX_WORKERS`를 키 목록(ci_reuse.go:156)에 추가한다.
2. cap이 없는 변수를 limit으로 새로 만들지 않게 바꾼다(존재하는 변수만 묶는다).

단계가 이 변수를 직접 판별해야 한다면 `reuse.environment` 탈출구로
단계별 키잉을 복원할 수 있다(master 자체가 `DVA_RESOURCE_SUPERVISED`에
그 탈출구를 쓴다).

**선택(2026-11-18, 본 구현)**: 옵션 1 — 키 목록 추가. 옵션 2는 기각한다:
workers.go:27-34의 "없으면 limit으로 생성"이 pytest 협력적 worker 제한의
강제 수단이다. cap이 변수를 만들지 않게 하면 승인 단계에서 pytest가
제한 없이 돌아가는 자원 통제 약화로 이어진다. 키에 넣는 쪽이 강제를
그대로 두고 간격만 닫는다.

## Completion Criteria

- [x] 승인 cap이 만든 `PYTEST_MAX_WORKERS` 차이(부재 vs limit)가 재사용 키를 판별한다 | verify: `go test ./internal/cirun -run TestReuseKeyTracksAdmissionCreatedPytestWorkers -count=1` (observed: 2026-11-18)
- [x] `DVA_CI_JOBS` 무시 계약과 도구 환경 판별 대조는 유지된다 | verify: `go test ./internal/cirun -run 'TestReuseKeyIgnoresDVAJobs|TestReuseKeyStillDifferentiatesToolEnvironment' -count=1` (observed: 2026-11-18)
- [x] 키 목록에 `PYTEST_MAX_WORKERS`가 이웃 worker 노브와 함께 들어간다 | verify: `/usr/bin/grep -q '"RUST_TEST_THREADS", "PYTEST_MAX_WORKERS", "PYTHONPATH"' internal/cirun/ci_reuse.go` (observed: 2026-11-18)

## Verification

- 수정 전 트리에서 `TestReuseKeyTracksAdmissionCreatedPytestWorkers`가
  "reuse key ignored the admission-created PYTEST_MAX_WORKERS"로 실패했다
  (2026-11-18, green-both-ways). 키 목록에 변수를 추가한 뒤 통과. 두 사전
  guard(비승인=부재, 승인=cap 생성값 2)는 red 시점에도 통과해 실패가 정확히
  키 동일 assertion임을 보인다.
- 대조: `TestReuseKeyIgnoresDVAJobs`(DVA_CI_JOBS 제외 계약)와
  `TestReuseKeyStillDifferentiatesToolEnvironment`(GOFLAGS 판별)는 전후
  모두 통과. 인접쌍 grep binding은 수정 전 레이아웃
  (`"RUST_TEST_THREADS", "PYTHONPATH"`)에서 실패한다.
- `TestReuseKey*` 슈트 전체가 수정 후 녹색(ok, -count=1).

## Delivery

구현(키 목록 + 주석)과 회귀 테스트, 이 카드를 같은 태스크 브랜치에서
하나의 커밋으로 push한다. 게이트·독립 리뷰는 조정자 세션이 수행한다.
