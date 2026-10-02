---
id: TASK-464
title: "Split migrate_section_order_test.go comment-preservation cluster"
type: refactor
priority: P3
effort: S
exec-tier: standard
allowed-paths: [internal/config, tasks]
status: todo
created: 2026-10-02
---

## Summary

`internal/config/migrate_section_order_test.go`가 test kind 한도를 넘는다
(972 물리 라인, 31 테스트). 구현 `migrate_section_order.go`는 한도 안이다.
테스트를 주제 봉합으로 나눈다. 함수 이동만 허용한다.

봉합 계획(실행 시 실측 조정):

- `migrate_section_order_preservation_test.go`(신규) — 주석·공백·chomping·
  CRLF/LF·slot separator·footer comment 보존 클러스터: KeepsSlotSeparators,
  PreservesCRLFSeparator, PreservesLFSeparator, KeepsKeepChompedTrailingBlank,
  KeepsExplicitIndentKeepChompedTrailingBlank, NestedSequenceExplicitIndent…,
  ExplicitKeyIndentKeepsTrailingBlank, KeepChompedScalarBeforeTail…,
  KeepMarkerInCommentDoesNotClaimSeparator, KeepsFooterCommentAtEOF,
  KeepsEveryTrailingCommentParagraph, DoesNotTreatQuotedScalarContentAsFooter,
  TrustsHeadCommentOverColumnZero, BoundsTheBannerWalkOnCRLFToo.
- 잔존 — 핵심 마이그레이션 의미론(PreservesComments, NonCanonicalKeyKeepsSlot,
  CommentAboveBlankStaysWithPreviousSection, AlreadyCanonical*, Idempotent,
  SelfCheck*, ClearsTheValidateWarning, BailsOn*, Blocks*Reason,
  AlreadyCanonicalWithAnchor, StopsAtDocumentBoundary,
  RefusesWhenLineBreaksDisagree).
- 두 파일이 모두 test kind 한도 안에 들면 성공. 공유 헬퍼는 잔존 파일에 두고
  이동 테스트 전용 헬퍼만 따라간다.

## Completion Criteria

- [x] `internal/config/migrate_section_order_test.go`가 test kind 한도 안에 있다 | verify: `ce validate filesize internal/config/migrate_section_order_test.go` (observed: 2026-10-02 — 558 물리/431 코드, exit 0, Medium 고지는 400 경고선)
- [x] 새 파일도 test kind 한도 안에 있다 | verify: `ce validate filesize internal/config/migrate_section_order_preservation_test.go` (observed: 2026-10-02 — 423 물리)
- [x] 패키지 전체 테스트가 통과한다 | verify: `go test ./internal/config/ 2>&1 | /usr/bin/grep -q '^ok'` (observed: 2026-10-02 — `-count=1` ok 1.2s)
- [x] vet이 깨끗하다 | verify: `go vet ./internal/config/` (observed: 2026-10-02)
- [x] diff가 테스트 이동만 포함한다 | verify: human — split commit diff review (observed: 2026-10-02 — `git diff --numstat` 0+/414- 순수 삭제, 이동 본문은 HEAD 512–925행과 바이트 동일, 테스트 수 파리티 31=17+14)
- [x] 전체 lint가 통과한다 | verify: human — `make lint` output is linked in Evidence (exceeds the 30s binding budget) (observed: 2026-10-02 — ci-lint 0 issues)

## Evidence

분할 결과(물리): migrate_section_order_test.go 972→558,
migrate_section_order_preservation_test.go 423(신규). 이동 블록은 연속 구간
(HEAD 512–925행)이라 나열 외 판단 없이 이동; 공유 헬퍼 없음(전부 테스트 지역
클로저). BenchmarkMigrateSectionOrderSemanticSelfCheck는 셀프체크 계열로 잔존.

게이트: `go build ./...` ok · `go test ./internal/config/ -count=1` ok · vet
silent · gofmt clean · `make lint` 0 issues · filesize 2파일 High 0.
