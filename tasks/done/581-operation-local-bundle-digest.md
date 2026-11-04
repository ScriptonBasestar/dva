---
id: TASK-581
title: "Isolate skill bundle digest state between operations"
type: refactor
priority: P2
exec-tier: standard
status: done
created: 2026-11-04
quality-review: pass
quality-reviewed-at: 2026-11-04T11:02:26Z
quality-review-evidence: tasks/done/evidence/TASK-581/independent-review.json
quality-review-receipt: tasks/done/evidence/TASK-581/done-review-76e423de10fb466b95880e1ac68c82a8dad50aa0431f74f8664c884e5b2c31f5.json
---

## Summary

스킬 설치·상태 조회에서 테스트가 교체하던 전역 currentBundleSHA를 호출별 비공개 의존성으로 격리한다. 기본 해시 계산과 영수증 기록은 sourceBundleSHA를 유지하며 downgrade·bundle drift 동작을 보존한다. 서로 다른 임시 home/state와 해시 전략을 barrier로 동시에 실행하는 회귀로 교차 오염을 검증한다.

## Completion Criteria

- [x] Concurrent operations retain independent bundle identity and preserve receipt and downgrade behavior | verify: `go test -race ./internal/skillinstall -run 'Test(BundleDigestStrategyIsOperationLocal|StatusClassifiesBundleDrift|InstallRefusesDowngradeUnlessAllowed)$'`

## Review Attempts

- 2026-11-04T11:02:26Z — first card review: reviewer `/root/doctor_flake_diagnosis` (strong), executor `/root/config_missing_and_profiles` (standard). Criterion PASS after guarded `-count=1 -v` verification; TASK-581 retained `-race`; required named tests executed. No blocking findings; source `6ef16cbc45dfd070472e0e8e7cb7e5affad6e0dd`. Evidence: `tasks/done/evidence/TASK-581/independent-review.json`; runtime `review-stop-warning-bundle` (review). Outcome: pass; next transition: done.
