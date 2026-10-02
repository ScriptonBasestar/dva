---
id: TASK-468
title: "Split cirun.go report and step-execution clusters"
type: refactor
priority: P3
effort: S
exec-tier: standard
allowed-paths: [internal/cirun, tasks]
status: todo
created: 2026-10-02
---

## Summary

`internal/cirun/cirun.go`(848 물리)가 config kind 한도를 넘는다. 보고 타입과
단계 실행 클러스터를 봉합한다. 함수·타입 이동만 허용한다.

봉합 계획(실행 시 실측 조정):

- `cirun_report.go`(신규) — Conflict, BusyError(+Error/Unwrap), Options,
  StepReport, Attestation, Report(+MarshalJSON/UnmarshalJSON), stateDirectory.
- `cirun_steps.go`(신규) — execute, runStep, ciEnv, runID, outputOrDiscard,
  warn, lockedWriter(+Write).
- cirun.go 잔존 — Run 본체와 그 직접 헬퍼.

## Completion Criteria

- [x] `internal/cirun/cirun.go`가 한도 안에 있다 | verify: `ce validate filesize internal/cirun/cirun.go` (observed: 2026-10-02 — 478 물리/453 코드, High 없음, Medium 1건은 300 경고선 고지)
- [x] 새 파일 전부가 한도 안에 있다 | verify: `ce validate filesize internal/cirun/cirun_report.go internal/cirun/cirun_steps.go` (observed: 2026-10-02 — report 126 / steps 274 물리, High 0)
- [x] 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/cirun/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 3.3s)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/cirun/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — main-thread 리뷰: numstat 0 insertions, 심볼 단위 바이트 비교 4표본 OK(execute/runStep/stateDirectory/fingerprint), report 본문 HEAD 27–138·steps 본문 HEAD 300–553 그대로)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — run-finish 게이트 확인)

## Evidence

분할 결과(물리): cirun.go 848→478, 신규 cirun_report 126 / cirun_steps 274.
fingerprint 클러스터는 cirun.go 잔존부(453 코드라인, 한도 안)에 유지 — 4분할
불필요. 삭제 370 = 이동 367 + import 제거 3(crypto/rand, os/user, sync).

게이트: `go build ./...` ok · `go test ./internal/cirun/ -count=1` ok · vet
silent · gofmt clean · filesize 3파일 High 0 (Medium 1건: cirun 453 코드라인
고지).
