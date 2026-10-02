---
id: TASK-473
title: "Split releaseworkflow main.go preflight cluster"
type: refactor
priority: P3
effort: S
exec-tier: standard
allowed-paths: [tools/releaseworkflow, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: main.go 557->312 + preflight.go 258; 5 symbols byte-identical, numstat 0+/245-, gates green. Integrated at 60ca8b23."
archived-at: 2026-10-03
---

## Summary

`tools/releaseworkflow/main.go`(557 물리)가 config kind 한도를 살짝 넘는다.
preflight·검사 클러스터를 봉합한다. 함수·타입 이동만 허용한다.

봉합 계획(실행 시 실측 조정):

- `preflight.go`(신규) — preflight, postflight, checkDetachedClean,
  checkLocalTag, requireCredential, redactCredential와 이들의 검사 헬퍼.
- main.go 잔존 — main, common/commonFlags/stringList, validateCommon,
  run/runStatus/runGH/runGHStatus/commandStatus와 워크플로우 조립.

## Completion Criteria

- [x] `tools/releaseworkflow/main.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize tools/releaseworkflow/main.go` (observed: 2026-10-02 — 312 물리, No issues)
- [x] `tools/releaseworkflow/preflight.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize tools/releaseworkflow/preflight.go` (observed: 2026-10-02 — 258 물리, No issues)
- [x] 패키지가 빌드된다 | verify: `go build -o /dev/null ./tools/releaseworkflow/` (observed: 2026-10-02)
- [x] vet이 깨끗하다 | verify: `go vet ./tools/releaseworkflow/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — `git diff --numstat` main.go 0+/245- 순수 삭제, preflight/postflight/checkLocalTag/checkGoReleaser/remoteTagTarget 5심볼 HEAD 추출과 바이트 동일)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — gofmt 581 files 0 unformatted, ci-lint 0 issues)

## Evidence

분할 결과(물리): main.go 557→312, preflight.go 258(신규). 이동 14심볼은 연속
구간이라 나열 외 판단 없이 이동; main.go에는 main/common/run/runStatus/runGH
/release 타입/finalRelease와 verifyReleaseDownloads·verifyDownloadedChecksums·
clean·checkRepositoryRoot 등 잔존.

게이트: `go build ./...` ok · `go test ./tools/releaseworkflow/ -count=1` ok
4.3s · vet silent · gofmt clean · `make lint` 0 issues · filesize 2파일 High 0.
