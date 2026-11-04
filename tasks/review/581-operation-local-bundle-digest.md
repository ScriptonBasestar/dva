---
id: TASK-581
title: "Isolate skill bundle digest state between operations"
type: refactor
priority: P2
exec-tier: standard
status: review
created: 2026-11-04
---

## Summary

스킬 설치·상태 조회에서 테스트가 교체하던 전역 currentBundleSHA를 호출별 비공개 의존성으로 격리한다. 기본 해시 계산과 영수증 기록은 sourceBundleSHA를 유지하며 downgrade·bundle drift 동작을 보존한다. 서로 다른 임시 home/state와 해시 전략을 barrier로 동시에 실행하는 회귀로 교차 오염을 검증한다.

## Completion Criteria

- [ ] Concurrent operations retain independent bundle identity and preserve receipt and downgrade behavior | verify: `go test -race ./internal/skillinstall -run 'Test(BundleDigestStrategyIsOperationLocal|StatusClassifiesBundleDrift|InstallRefusesDowngradeUnlessAllowed)$'`
