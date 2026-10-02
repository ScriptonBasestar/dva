---
id: TASK-472
title: "Split config validate.go schema cluster"
type: refactor
priority: P3
effort: S
exec-tier: standard
allowed-paths: [internal/config, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: validate.go 736->525 + validate_schema.go 221; 4 symbols byte-identical vs HEAD, numstat pure, gates green. Integrated at 7c0d00d2."
---

## Summary

`internal/config/validate.go`가 config kind 한도를 15 코드라인 초과한다(515).
스키마 임베드·제거키 클러스터를 봉합하면 해소된다. 함수·타입 이동만 허용한다.

봉합 계획(실행 시 실측 조정):

- `validate_schema.go`(신규) — embeddedSchema, removedSchemaKeys, rootField,
  removedRootKeys, removedInteractionKeys, isInteractionCommandField,
  validateYAMLSchema, convertYAMLToJSON, ValidateConfigBytes.
- validate.go 잔존 — ValidationErrors, Validate 본체, validateHookPlacement,
  validateHookPlanFilters, ComposeNameWarning 클러스터(ValidateComposeProjectNames,
  FixComposeProjectName, readComposeNameKey), validatePlanAliasExtends.

## Completion Criteria

- [x] `internal/config/validate.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize internal/config/validate.go` (observed: 2026-10-02 — 525 물리/367 코드, High 없음, Medium 1건은 300 경고선 고지)
- [x] `internal/config/validate_schema.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize internal/config/validate_schema.go` (observed: 2026-10-02 — 221 물리)
- [x] 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/config/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 1.6s)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/config/` (observed: 2026-10-02)
- [x] diff가 함수·타입·var 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — main-thread 리뷰: numstat 0 insertions, 심볼 바이트 비교 4표본 OK(validateYAMLSchema/convertYAMLToJSON/ValidateConfigBytes/isInteractionCommandField), removedSchemaKeys 등 var 블록 이동 확인)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — run-finish 게이트 확인)

## Evidence

분할 결과(물리): validate.go 736→525, validate_schema.go 221(신규). 임베디드
스키마·제거 키 var 블록 + validateYAMLSchema + convertYAMLToJSON +
ValidateConfigBytes가 스키마 봉합으로 이동.

게이트: `go build ./...` ok · `go test ./internal/config/ -count=1` ok · vet
silent · gofmt clean · filesize High 0 (Medium 1건: validate 367 코드라인
고지).
