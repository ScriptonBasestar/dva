---
id: TASK-466
title: "Split cli package over-limit command files"
type: refactor
priority: P2
effort: L
exec-tier: standard
allowed-paths: [internal/cli, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: compose.go 1364->359, provision.go 668->305, validate.go 1192->261 + 12 seam/test files; numstat pure, gates green, High cleared. Integrated at 519bd704."
---

## Summary

cli 패키지의 4개 파일이 한도를 넘는다: `compose.go`(1,364 물리, 13 함수 —
함수가 크다), `validate.go`(1,192, 40), `provision.go`(668, 15),
`validate_test.go`(1,166). cobra 명령 파일을 하위 주제 봉합으로 나눈다.
함수 이동만 허용한다.

실행 시 각 파일의 선언 목록을 먼저 뽑아 봉합선을 정한다. 판단 기준:

- cobra 명령 파일(`New*Cmd`)은 명령별 파일 유지가 원칙 — 한 명령 구현이
  500 코드라인을 넘으면 그 명령의 플래그·런·헬퍼를 `compose_run.go` 같은
  보조 파일로 나눈다.
- validate.go의 40개 함수는 검증 주제별(출력/플랜/스택 등) 클러스터로 나눈다.
- validate_test.go는 구현 주제를 따른다.
- 결과적으로 패키지의 모든 .go 파일이 각 kind 한도 안이면 성공.

## Completion Criteria

- [x] `internal/cli/compose.go`가 한도 안에 있다 | verify: `ce validate filesize internal/cli/compose.go` (observed: 2026-10-02 — 359 물리, High 0)
- [x] `internal/cli/validate.go`가 한도 안에 있다 | verify: `ce validate filesize internal/cli/validate.go` (observed: 2026-10-02 — 261 물리)
- [x] `internal/cli/provision.go`가 한도 안에 있다 | verify: `ce validate filesize internal/cli/provision.go` (observed: 2026-10-02 — 305 물리)
- [x] `internal/cli/validate_test.go`가 한도 안에 있다 | verify: `ce validate filesize internal/cli/validate_test.go` (observed: 2026-10-02 — 44 물리)
- [x] cli 패키지의 다른 어떤 파일도 한도 위반이 아니다 | verify: `! ce validate filesize --all 2>&1 | /usr/bin/grep -q '🟠 internal/cli'` (observed: 2026-10-02 — High 마커 없음; 원 바인딩은 Skipped 안내 줄의 internal/cli 문자열까지 매칭하는 false positive)
- [x] 이동 후 cli 패키지 테스트가 통과한다 | verify: `go test ./internal/cli/ 2>&1 | /usr/bin/grep -q '^ok'`
- [x] vet이 깨끗하다 | verify: `go vet ./internal/cli/`
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget)

## Evidence

분할 결과(물리): compose.go 1364→359, provision.go 668→305, validate.go
1192→261, validate_test.go 1166→44. 신규 12파일: compose_lifecycle 393 /
compose_build 224 / compose_flags 329 / compose_exec 97 / provision_steps 372 /
validate_interaction 104 / validate_drift 384 / validate_sources 477 /
validate_drift_test 645 / validate_drift_print_test 39 /
validate_compose_name_test 79 / validate_sources_test 397.

계획 대비 판단 기록: (1) validate_drift_test가 폴백 후에도 607 코드라인이라
TestPrintConfigDriftWarnings를 validate_drift_print_test.go(신규)로 1개 더
이동 — 유일한 계획 외 봉합. (2) resolveProvisionProfile 문서 주석은 함수와
함께 provision.go 잔존부에 유지.

이동 검증: `git diff --numstat` 4개 원본 전부 0 insertions, 함수 단위 바이트
비교 4표본 OK(downCmd/parseDvaFlags/collectDocumentedTargetNames/
executeParallelBatch), 테스트 수 파리티 39=39.

게이트: `go build ./...` ok · `go test ./internal/cli/ -count=1` ok 22.8s ·
vet silent · gofmt clean · `make lint` 0 issues · filesize 16파일 High 0
(Medium 2건: validate_sources 348, validate_drift_test 582 코드라인 고지).
