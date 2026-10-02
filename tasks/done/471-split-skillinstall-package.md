---
id: TASK-471
title: "Split skillinstall package over-limit files"
type: refactor
priority: P2
effort: L
exec-tier: standard
allowed-paths: [internal/skillinstall, tasks]
status: done
created: 2026-10-02
quality-review: pass
quality-reviewed-at: 2026-10-02
quality-review-evidence: "Independent main-thread review separate from the implementing agent: install.go 1921->386 + 6 files, takeover_backup.go 645->304+308, install_test.go 1779->519+4 files; 3 modified files numstat 0+ (minimal algo), test parity 44=44, 6-symbol byte checks, gates green. Integrated at 64803684."
---

## Summary

skillinstall 패키지 3개 파일이 한도를 넘는다: `install.go`(1,921 물리, 61
함수), `install_test.go`(1,779), `takeover_backup.go`(645). 함수·타입 이동만
허용한다.

install.go 봉합 계획(실행 시 실측 조정):

- `install_flow.go`(신규) — 설치 경로: Install, preflightInstall,
  installDestination, installWithReservations, updateClaimedInstall,
  ensureClaimDestinationsAbsent, restoreReceipt.
- `uninstall.go`(신규) — 제거 경로: Uninstall, preflightUninstall,
  uninstallDestination, unlinkConsumers, removeLastConsumer,
  restoreActiveTakeover, restoreBackupOnly.
- `skill_bundle.go`(신규) — 번들 클러스터: bundledBundle, bundledFiles,
  bundleFor, hasRuntime, ensureDestination, ensureNoCollision,
  hasForeignCollision, skillNames, ownedSkillNames.
- `install_paths.go`(신규) — resolve, runtimePath, resultEntry,
  setAllRuntimeStatuses, setMembershipStatuses.
- install.go 잔존 — 타입·const 선언군(Scope, Runtime, Options, Result,
  DestinationResult, RuntimeStatus, receipt, takeoverBackup, backupEntry,
  fileHash, destination, skillBundle, DefaultRuntimes)과 Status.

- `takeover_backup.go` — 19 함수의 내부 봉합선을 실행 시 선언 목록에서 판단해
  2분할(예: backup 생성/복원).
- `install_test.go` — 구현 주제(Install/Uninstall/Status/backup)를 따라
  분할. 공유 셋업 헬퍼는 잔존 또는 `install_testhelpers.go`로.

## Completion Criteria

- [x] `internal/skillinstall/install.go`가 한도 안에 있다 | verify: `ce validate filesize internal/skillinstall/install.go` (observed: 2026-10-02 — 386 물리/355 코드)
- [x] `internal/skillinstall/takeover_backup.go`가 한도 안에 있다 | verify: `ce validate filesize internal/skillinstall/takeover_backup.go` (observed: 2026-10-02 — 304 물리; 복원 클러스터는 takeover_backup_restore.go 308로 분리)
- [x] `internal/skillinstall/install_test.go`가 한도 안에 있다 | verify: `ce validate filesize internal/skillinstall/install_test.go` (observed: 2026-10-02 — 519 물리/491 코드; 나머지 테스트는 주제별 4파일로 분리)
- [x] 패키지의 어떤 파일도 kind 위반이 아니다 | verify: `! ce validate filesize --all 2>&1 | /usr/bin/grep -q '🟠 internal/skillinstall'` (observed: 2026-10-02 — 패키지 High 0, 🟡 7건은 300/400 경고선 고지)
- [x] 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/skillinstall/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 6.1s, 테스트 수 패리티 44=44)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/skillinstall/` (observed: 2026-10-02)
- [x] diff가 함수·타입 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — 수정 3파일 numstat 0+ (minimal 알고리즘), 심볼 6개 HEAD 추출 바이트 동일, 문서 주석 3개 함수 동행 확인)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — gofmt clean, ci-lint 0 issues)

## Evidence

분할 결과(물리): install.go 1921→386(타입·상수, Status, 클레임/검증) ·
install_flow 410(설치 경로) · uninstall 386(제거 경로) · install_paths 234
(경로·런타임 집합 연산) · skill_bundle 130(번들 클러스터) · receipt 258 ·
replace 233(교체·synced-fs 헬퍼) · takeover_backup 645→304(생성) +
takeover_backup_restore 308(검증·복원). 테스트: install_test 1779→519 +
takeover_backup_test 432 · install_status_test 359 · install_guard_test 340 ·
receipt_test 177.

계획 대비 판단 기록: (1) 카드의 install.go 잔존안이 실측 ~800 코드라인으로 한도
초과 — receipt.go·replace.go 2개 봉합을 추가 절단. (2) install_test.go도 1714
코드라인으로 🟠라 카드의 테스트 분할 조항대로 4분할. (3) 범용 fs 헬퍼
(syncDirectory/mkdirAllSynced)는 사용처를 따라 replace.go로.

게이트: `go build ./...` ok · `go test ./internal/skillinstall/ -count=1` ok
6.1s · vet silent · gofmt clean · `make lint` 0 issues · 패키지 High 0.
