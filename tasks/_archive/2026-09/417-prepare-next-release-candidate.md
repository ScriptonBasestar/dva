---
id: TASK-417
title: "Prepare and verify the next release candidate"
type: docs
priority: P2
effort: M
exec-tier: standard
status: done
archived-at: 2026-09-23
verified-at: 2026-09-23
verification-summary: "Re-verified 2026-09-23. internal/config.Version, CHANGELOG [0.3.0] - 2026-09-23, and release-notes/v0.3.0.md describe that release. make check-generate and make release-check exited 0. Full CI e6b7e8f8b5a327b377433845e15ffa46 was not re-run; its 2026-09-23 attestation remains the record. Notes SHA-256 unchanged. No follow-up."
quality-review: pass
quality-review-receipt: tasks/done/evidence/TASK-417/done-review-33feafc92a2a57fd6d3bf81c03aa39fb349e70a266a57a80a1390e0af58c12d0.json
quality-reviewed-at: 2026-09-23
quality-review-evidence: "Independent ce-judge review410 PASS: final product contracts, full CI attestation, snapshot/generation evidence and TASK-370 schema/frozen-note digest verified; alias wording corrected and documentation gates rerun."
blocks: [TASK-370, TASK-418]
created: 2026-09-23
depends-on: [TASK-416, TASK-411, TASK-419]
---

## Summary

새 기능과 수정이 쌓인 Unreleased를 0.3.0으로 확정한다. 버전 상수·CHANGELOG·새 release notes를 맞추고 TASK-370의 두 오류 재발 방지를 독립 검토한다. 공개 태그 생성과 원격 Release 작성은 TASK-418이 소유한다.

## Completion Criteria

- [x] 버전·CHANGELOG·노트가 같은 릴리스 범위를 설명한다 | verify: human — internal/config/version.go, CHANGELOG.md와 새 release-notes를 대조
- [x] 생성물 및 snapshot 검증 통과 | verify: `make check-generate && make release-check` (regression-guard)
- [x] 전체 CI가 완료되고 동일 입력 attestation을 남긴다 | verify: human — dva ci full의 run ID, succeeded, elapsed, attestation을 기록

## Verification evidence (2026-09-23)

- 사용자 선택: 0.3.0. Version·설치 예제·CHANGELOG·새 노트를 정합화했다.
- `dva check-generate`: exit 0, 생성물 diff 없음.
- `make release-check`: exit 0, 6개 플랫폼 archive와 checksums 및 host binary stamp 검증.
- `dva ci full`: `e6b7e8f8b5a327b377433845e15ffa46`, succeeded,
  4m15.859070917s. before/after attestation 모두
  `436dda6b202162d96733ad8f189924595bd7bbb35fa5f724b1e7074183c010b4`.
- 독립 리뷰가 찾은 alias 체인 설명을 CI 완료 후 수정했다. 구현은 바뀌지 않았으며
  수정된 문서에 `make doc-check`, `ce task gate`, `git diff --check`가 통과했다.
- 새 노트의 CI YAML을 추출한 설정에 `dva config validate`: exit 0.
- 기존 v0.2.0 노트 SHA-256은
  `7905843de27f4c3c7fcb43474165fa16bee89c0e381ff2577d4dc7e656d2a818`로 유지했다.
- 공개 tag/Release는 생성하지 않았다. 승인된 immutable identity의 공개는 TASK-418이다.

## Archive

- [x] 아카이브
- 후속: 없음
