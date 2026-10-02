---
id: TASK-470
title: "Split skillclaim.go lock and store clusters"
type: refactor
priority: P3
effort: S
exec-tier: standard
allowed-paths: [internal/skillclaim, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: claim.go 639->429 + claim_locks.go; LockSet/Digest/LockedStore/write boundaries awk-verified then deleted, symbol byte-parity, test gates green. Integrated at 7ede2c7c."
---

## Summary

`internal/skillclaim/claim.go`(639 물리)가 config kind 한도를 넘는다. 잠금·
스토어 클러스터를 봉합한다. 함수·타입 이동만 허용한다.

봉합 계획(실행 시 실측 조정):

- `claim_locks.go`(신규) — LockSet, AcquireLocks, Release, LockedStore와 그
  메서드 전부(Begin, Close, Read, authorize, Reserve, CompareAndSwap, Remove),
  allowedTransition, sameIdentity, samePayload, sameStrings.
- claim.go 잔존 — 스키마·모델·검증·전이: Schema, FileHash, Claim, Path,
  CanonicalDestination, Read, Decode, RejectDuplicateKeys, Validate,
  token/pathRecord/digest/sortedUnique, ManifestDigest, Digest,
  writeDigestFrame, Transition, Reserve, Activate.

## Completion Criteria

- [x] `internal/skillclaim/claim.go`가 한도 안에 있다 | verify: `ce validate filesize internal/skillclaim/claim.go` (observed: 2026-10-02 — 429 물리/407 코드, High 없음, Medium 1건은 300 경고선 고지)
- [x] `internal/skillclaim/claim_locks.go`가 한도 안에 있다 | verify: `ce validate filesize internal/skillclaim/claim_locks.go` (observed: 2026-10-02 — 221 물리)
- [x] 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/skillclaim/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 0.3s)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/skillclaim/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — main-thread 리뷰: numstat 0 insertions/210-, 심볼 바이트 비교 5표본 OK(LockSet/AcquireLocks/CompareAndSwap/allowedTransition/sameFiles))
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — run-finish 게이트 확인)

## Evidence

분할 결과(물리): claim.go 639→429, claim_locks.go 221(신규). LockSet 클러스터
(HEAD 288–338)와 LockedStore 클러스터(HEAD 408–565) 이동, sameFiles는
allowedTransition의 비교 헬퍼로 함께 이동(카드 나열 외 판단 — 유일 호출자가
allowedTransition).

게이트: `go build ./...` ok · `go test ./internal/skillclaim/ -count=1` ok ·
vet silent · gofmt clean · filesize High 0 (Medium 1건: claim 407 코드라인
고지).
