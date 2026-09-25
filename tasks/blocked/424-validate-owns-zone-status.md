---
id: TASK-424
title: "Make ce task validate own the zone status rule"
type: bug
priority: P2
effort: M
exec-tier: standard
status: blocked
created: 2026-09-24
---

## Summary

상류 `0f0a5f3c`는 live zone의 unknown, empty, zone-mismatch status를 거부한다.
독립 리뷰는 이 부분을 확인했지만, CE 공개 validator가 archive를 건너뛰고
`status` 누락과 비정규 alias를 허용하는 것도 확인했다([ISSUE-041](../issue/041-upstream-status-validator-contract-parity.md)).
따라서 현재 DVA status sweep을 제거하면 계약의 일부가 비게 된다.

방향: 상류 기준을 먼저 기록하고 ISSUE-041에서 나머지 계약을 닫는다. 그 뒤 격리된
CE 빌드가 동일한 live/archive fixture를 거부하는 것을 확인한 후에만 DVA의
`checkCardStatus` 복사본을 제거한다. 보드 readiness 판단은 공유 `ce task gate`가
소유하며, `make doc-check`는 DVA 저장소의 문서·소스 검사만 유지한다.

## Completion Criteria

- [x] Shared validator rejects unknown, empty, and zone-mismatched status values in live zones | verify: `ce-agent-kit` commit `0f0a5f3c`; `internal/usecase/task/validator_status_test.go` covers the three cases
- [ ] Shared validator rejects missing status, non-canonical aliases, and archive mismatches under the canonical contract | verify: human — ISSUE-041 upstream tests and paired fixtures
- [ ] Remove DVA `checkCardStatus` and its zone table only after an isolated shared validator rejects the same live and archive fixtures | verify: human — removal commit plus `make doc-check` on the repository

## Out of scope

- Shared board readiness verdicts in `make doc-check`; `ce task gate` owns those verdicts.

## Sources

- [ISSUE-007](../issue/007-ce-task-validate-does-not-constrain-the-status-field.md)
- [ISSUE-041](../issue/041-upstream-status-validator-contract-parity.md)
