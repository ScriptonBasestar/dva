---
id: TASK-419
title: "Accept documented release candidates before their tag exists"
type: bug
priority: P1
effort: S
exec-tier: standard
status: done
archived-at: 2026-09-23
verified-at: 2026-09-23
verification-summary: "Re-verified 2026-09-23. TestPendingReleaseCandidate accepts only a populated dated section whose source version is newer than the reachable tag, and rejects empty, mismatched, undated, invalid-date, and same-or-older candidates. go test ./tools/changelogcheck and make doc-check passed. No follow-up."
quality-review: pass
quality-reviewed-at: 2026-09-23
quality-review-evidence: "Independent ce-judge review419: corrected leading-zero SemVer finding, rechecked source/tag/heading/date/order/content boundaries; go test -race -count=1 and go vet ./tools/changelogcheck, gofmt and diff check passed. Candidate Git fixture not independently run; real 0.3.0 candidate will exercise main path in TASK-417."
created: 2026-09-23
---

## Summary

docs/52는 tag 전에 Unreleased를 버전 섹션으로 확정하지만, changelogcheck는
이전 tag 이후 변경 수로 비어 있는 Unreleased를 거부한다. 소스 버전과 일치하고
기존 tag보다 새로운 기록된 후보만 인식하며 실제 미기록 변경 검출은 유지한다.

## Completion Criteria

- [x] 소스 버전과 일치하고 도달 가능한 tag보다 새로운 날짜가 있는 후보를 인식한다 | verify: `go test ./tools/changelogcheck`
- [x] 비어 있거나 불일치·과거 후보는 검사를 우회하지 않는다 | verify: `go test ./tools/changelogcheck`
- [x] 문서 게이트 통과 | verify: `make doc-check` (regression-guard)

## Verification

`dva ci commit` de4b306596f165d0af7f811028ac5d53 succeeded in 4m40.353677625s.
Earlier run 0abc7a33517b29df26414bcecbe2abdd failed on another session’s golangci-lint lock;
that process ended before retry. No lock was bypassed or deleted.

## Archive

- [x] 아카이브
- 후속: 없음
