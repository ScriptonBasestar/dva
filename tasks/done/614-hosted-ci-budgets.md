---
id: TASK-614
title: "Hosted CI 전체 job의 유한 실행 예산과 runner 실패 근거 보존"
type: chore
priority: P1
effort: S
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-614/review.txt
created: 2026-11-17
---

## Summary

기존 master CI의 runner 확보 실패와 한 번의 해당 job 재시도 결과를 보존한다.
시간 상한이 빠진 hosted job 네 개에 명시적 상한을 추가하고 기존 test 상한은 유지한다.

## Scope

`.github/workflows/ci.yml`, `tools/cilabels/ci_events_test.go`, hosted 예산 문서와 관찰 증거.
원격 runner 선택·권한·보호 규칙·이벤트·검사 범위는 유지한다.
production 변경 실측과 profile 결과 재사용 채택은 후속 판단으로 남긴다.

## Completion Criteria

- [x] 모든 hosted job에 1~30분 유한 실행 상한을 요구한다 | verify: `/usr/bin/grep -q 'TimeoutMinutes' tools/cilabels/ci_events_test.go` (source: absent; observed: 2026-11-17)
- [x] 실제 YAML의 job 상한과 이벤트·입력·검사 보존을 검사한다 | verify: `go test ./tools/cilabels -run TestCIEventRoutingAndCheckoutIdentity` (observed: 2026-11-17)
- [x] 관측 시간·queue/실행 구분·원래 실패와 단일 재시도 결과를 보존한다 | verify: human — 관찰 JSON과 예산 문서 독립 대조 (observed: 2026-11-17)

## Delivery

작은 표본의 초기 중단 상한이며 p95·SLA 또는 모든 cold 실행의 시간 보장은 아니다.
독립 source/done 검토 후 최종 dev/full, CE 통합·push·설치·정리를 수행한다.
