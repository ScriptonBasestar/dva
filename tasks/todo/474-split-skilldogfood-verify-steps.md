---
id: TASK-474
title: "Split skilldogfood main.go verify cluster"
type: refactor
priority: P3
effort: S
exec-tier: standard
allowed-paths: [tools/skilldogfood, tasks]
status: todo
created: 2026-10-02
---

## Summary

`tools/skilldogfood/main.go`(1,297 물리)가 config kind 한도를 넘는다. 검증
단계 클러스터를 봉합한다. 함수·타입 이동만 허용한다.

봉합 계획(실행 시 실측 조정):

- `verify_steps.go`(신규) — verifyFlowDryRun, verifyFixtureRoundTrip,
  verifyTakeoverLifecycle.
- `skillcopy.go`(신규, 필요 시) — immutableExecutableCopy, removeAll,
  executableFile, gitRoot, limitedBuffer(+Write/String).
- main.go 잔존 — main, run, 타입 선언군(commandResult, destinationResult,
  runtimeStatus, receiptRecord, fileHash, claimRecord, treeEntry,
  gitTreeState, invocation), validSHA256, 상수·맵.

두 봉합으로 부족하면 verify 클러스터를 단계별로 3분할한다.

## Completion Criteria

- [x] `tools/skilldogfood/main.go`가 config kind 한도 안에 있다 | verify: `ce validate filesize tools/skilldogfood/main.go` (observed: 2026-10-02 — 431 물리/394 코드, 한도 500 이내; 300 경고선 Medium 고지)
- [x] 새 파일 전부를 포함한 패키지에 kind 위반이 없다 | verify: `! ce validate filesize --all 2>&1 | /usr/bin/grep -q '🟠 tools/skilldogfood'` (observed: 2026-10-02 — 패키지 High 0; skillcopy 114/verify_steps 256/verify_contract 301/snapshot 247)
- [x] 패키지가 빌드된다 | verify: `go build ./tools/skilldogfood/` (observed: 2026-10-02 — `go build ./...` 포함)
- [x] vet이 깨끗하다 | verify: `go vet ./tools/skilldogfood/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — main.go numstat 0+/866- 순수 삭제, 심볼 6개 HEAD 추출 바이트 동일 검증, 신규 파일은 사용 필터 import만 추가)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — gofmt clean, ci-lint 0 issues)

## Evidence

분할 결과(물리): main.go 1297→431(394 코드), 신규 skillcopy 114(limitedBuffer+
copy/removeAll/executableFile/gitRoot) · verify_steps 256(flow dry-run/fixture
round-trip/takeover lifecycle) · verify_contract 301(receipt/claim/artifact
계약 11심볼) · snapshot 247(gitStatus…formatSnapshot 9심볼). 두 봉합으로는
500 코드라인 미달이 확정되어 카드가 허용한 4분할로 확장 — main.go에는 타입군,
main/run, invocation 헬퍼, require* 잔존.

게이트: `go build ./...` ok · `go test ./tools/skilldogfood/ -count=1` ok · vet
silent · gofmt clean · `make lint` 0 issues · 패키지 High 0.
