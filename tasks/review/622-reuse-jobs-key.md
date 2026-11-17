---
id: TASK-622
title: "CI 결과 재사용 키에서 DVA_CI_JOBS 제외"
type: bug
priority: P2
effort: S
exec-tier: strong
status: review
created: 2026-11-18
---

## Summary

재사용 키의 환경 허용 목록(internal/cirun/ci_reuse.go)에 `DVA_CI_JOBS`가 들어
있다. 이 값이 단계 행동을 실제로 판별하는지 먼저 측정하고, 측정이 지지하면
키에서 제외한다. 측정이 행동에 영향 주는 분산을 보이면 구현 대신 보고한다.

## Measurement (2026-11-18, 이 호스트, dva ci --dry-run)

- 계산 원천: `toolWorkerBudget`(internal/cirun/cirun_workers.go:19) — profile
  `tool_jobs` pin이 없으면 `computed(cpu/max_parallel)`. 이 저장소 프로필은
  전부 pin 없음. 호스트 CPU 18개 기준: dev=4, commit=4, core=2, full=2,
  audit=2. 프로필별 3회 반복 측정에서 값은 매번 동일했다(결정론적).
- 앰비언트 차단: 셸에서 `DVA_CI_JOBS=17`을 주어도 단계 환경 값은 2로
  유지된다 — `ciEnv`(internal/cirun/cirun_steps.go:459)이 계산 값으로 무조건
  덮어쓴다. 반면 `CARGO_BUILD_JOBS=9`는 그대로 흘러들어온다(도구 변수는
  행동 노브이므로 키에 남는 것이 맞다).
- 중복: 그 예산이 반영되는 도구 환경 `CARGO_BUILD_JOBS`(기본값=예산),
  `GOMAXPROCS`, `GOFLAGS`, `RUST_TEST_THREADS`는 모두 키 목록에 별도로
  있다(pin 안 하면 `CARGO_BUILD_JOBS`가 같은 값을 이중으로 실은 셈이다).
  저장소 안의 유일한 소비자는 cimetrics의 메타데이터 기록
  (tools/internal/cimetrics/metadata.go:380)이며 관측용이다.
- 분산 원천: 호스트 CPU 수와 profile `max_parallel`·`len(steps)`. 무관한
  단계 추가가 `s`를 바꿔 건드리지 않은 단계의 키까지 churn시킬 수 있다.
  행동 차이 없는 캐시 무효화만 만든다.

측정 판정: 행동에 영향 주는 분산 없음 — 제외 지지. 단계가 이 변수를 직접
읽는 사용자는 `reuse.environment`로 단계별 판별을 복원할 수 있다(문서화됨).

## Scope

- `internal/cirun/ci_reuse.go` 키 환경 목록에서 `"DVA_CI_JOBS"` 제거.
  수신 영수증(worker_budget)의 `dva_ci_jobs` 관측은 그대로 유지한다.
- 회귀 테스트 두 개와 `TestReuseKeyMutationMatrix`의 `"workers"` 행 교체
  (느슨해진 행의 대체 계약 + 판별 대조를 짝으로 묶는다).
- CHANGELOG 항목이 측정값을 인용한다.

## Out of Scope

- 다른 키 환경 항목의 재평가.
- worker 예산 계산식 자체의 변경.

## Completion Criteria

- [x] 계산된 `DVA_CI_JOBS` 예산 차이만 있으면 재사용 키가 동일하다 | verify: `go test ./internal/cirun -run TestReuseKeyIgnoresDVAJobs -count=1` (observed: 2026-11-18)
- [x] 도구 환경(`GOFLAGS -p`) 차이는 여전히 키를 판별한다 | verify: `go test ./internal/cirun -run TestReuseKeyStillDifferentiatesToolEnvironment -count=1` (observed: 2026-11-18)
- [x] 키 목록에서 `DVA_CI_JOBS`가 제거되고 이웃 항목들은 그대로다 | verify: `/usr/bin/grep -q '"RUST_TEST_THREADS", "PYTHONPATH"' internal/cirun/ci_reuse.go` (observed: 2026-11-18)

## Verification

- 수정 전 코드에서 `TestReuseKeyIgnoresDVAJobs`가 "reuse key changed with
  the computed DVA_CI_JOBS budget"로 실패했다(2026-11-18, green-both-ways).
  제거 뒤 통과했고 `TestReuseKeyStillDifferentiatesToolEnvironment`는
  전후 모두 통과한다(대조).
- 변이 확인: 세 번째 binding의 인접쌍 grep은 수정 전 레이아웃
  (`"RUST_TEST_THREADS", "DVA_CI_JOBS", "PYTHONPATH"`)에서 실패한다.
- `TestReuseKeyMutationMatrix`는 `"workers"` 행을 빼고 녹색이며, 그 계약은
  위 두 테스트가 대체한다. 전체 `TestReuse*`/`TestStepReuse*` 슈트 통과.

## Delivery

구현·이 카드를 같은 태스크 브랜치에서 커밋하고 scoped 게이트
(`make build`·`make test`·`make lint`·`make doc-check`,
`env -u POSTGRES_USER`)와 `ce task gate`를 통과한 뒤 push한다.
독립 리뷰는 별도 세션에서 수행한다.
