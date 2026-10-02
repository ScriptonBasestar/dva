---
id: TASK-469
title: "Split ociverify.go registry and reference clusters"
type: refactor
priority: P3
effort: S
exec-tier: standard
allowed-paths: [internal/ociverify, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: verify.go 559->284 + verify_registry/verify_reference; numstat pure, byte-parity, gates green. Integrated at 9cd7be7a."
---

## Summary

`internal/ociverify/verify.go`(559 물리)가 config kind 한도를 넘는다.
레지스트리 통신과 참조 파싱 클러스터를 봉합한다. 함수·타입 이동만 허용한다.

봉합 계획(실행 시 실측 조정):

- `verify_registry.go`(신규) — 레지스트리·네트워크 보안 클러스터: verifier
  타입, fetch, token, redirectPolicy, publicClient, publicDialContext,
  publicIP, nonPublicRanges.
- `verify_reference.go`(신규) — 참조 파싱 클러스터: reference 타입,
  ValidateReference, parseReference, repositoryComponent, imageTag 정규식,
  readBounded, digest, isSHA256.
- verify.go 잔존 — Options/Result, Verify/verify 본체, OCI 디스크립터 타입군
  (descriptor, imageIndex, imageManifest, imageConfig 등)과 조립 로직.

## Completion Criteria

- [x] `internal/ociverify/verify.go`가 한도 안에 있다 | verify: `ce validate filesize internal/ociverify/verify.go` (observed: 2026-10-02 — 284 물리, 3파일 일괄 No issues)
- [x] 새 파일 전부가 한도 안에 있다 | verify: `ce validate filesize internal/ociverify/verify_registry.go internal/ociverify/verify_reference.go` (observed: 2026-10-02 — registry 198 / reference 94 물리, No issues)
- [x] 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/ociverify/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 0.3s)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/ociverify/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — main-thread 리뷰: numstat 0 insertions, 심볼 단위 바이트 비교 4표본 OK(fetch/token/parseReference/readBounded), 삭제 비공백 라인 전수 신규 파일 존재 확인)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — run-finish 게이트 확인)

## Evidence

분할 결과(물리): verify.go 559→284, 신규 verify_registry 198 /
verify_reference 94. verify_test.go 305 미변경. import 10줄은 새 파일 import
블록으로 재배치(규칙 허용 범위), 이동 본문 16선언 전부 바이트 동일.

게이트: `go build ./...` ok · `go test ./internal/ociverify/ -count=1` ok ·
vet silent · gofmt clean · filesize 3파일 No issues (High/Medium 0).
