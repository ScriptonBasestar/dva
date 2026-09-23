---
id: TASK-417
title: "Prepare and verify the next release candidate"
type: docs
priority: P2
effort: M
exec-tier: standard
status: todo
blocks: [TASK-370, TASK-418]
created: 2026-09-23
depends-on: [TASK-416, TASK-411]
---

## Summary

새 기능과 수정이 쌓인 Unreleased를 0.3.0으로 확정한다. 버전 상수·CHANGELOG·새 release notes를 맞추고 TASK-370의 두 오류 재발 방지를 독립 검토한다. 공개 태그 생성과 원격 Release 작성은 TASK-418이 소유한다.

## Completion Criteria

- [ ] 버전·CHANGELOG·노트가 같은 릴리스 범위를 설명한다 | verify: human — internal/config/version.go, CHANGELOG.md와 새 release-notes를 대조
- [ ] 생성물 및 snapshot 검증 통과 | verify: `make check-generate && make release-check` (regression-guard)
- [ ] 전체 CI가 완료되고 동일 입력 attestation을 남긴다 | verify: human — dva ci full의 run ID, succeeded, elapsed, attestation을 기록
