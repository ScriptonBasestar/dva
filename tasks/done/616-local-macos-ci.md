---
id: TASK-616
title: "원격 macOS 의존을 명시적 로컬 플랫폼 검증으로 전환"
type: chore
priority: P0
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-616/review.txt
created: 2026-11-17
---

## Summary

macOS runner가 없고 필요한 검사는 이 Mac에서 실행한다는 사용자 결정에 따라,
원격 Linux 두 SOPS 조합은 유지하고 macOS 준비·검사·실행 조건을 명시한다.

## Completion Criteria

- [x] 유한 직렬 macOS 프로필을 선언한다 | verify: `/usr/bin/grep -q 'macos-platform:' dva.yml` (source: absent; observed: 2026-11-17)
- [x] 원격 Linux 두 조합과 기존 이벤트·job·검사·상한을 유지한다 | verify: `go test ./tools/cilabels -run TestCIEventRoutingAndCheckoutIdentity` (observed: 2026-11-17)
- [x] 도구 준비와 검사를 분리하고 누락·불일치를 거부한다 | verify: `go test ./tools/localplatform` (observed: 2026-11-17)
- [x] 실제 Mac에서 두 SOPS 조합과 native 단위 검사를 실행한다 | verify: human — 작업 후보의 native 진단 receipt와 도구 준비 기록 대조; 최종 committed-head 검사는 별도
- [x] 실행 조건과 자동 원격 검사의 보장 차이를 문서화한다 | verify: human — 로컬 macOS 정본·원격 이벤트·예산·프로필 설명 독립 대조

## Verification

[실제 Mac 진단과 준비 기록](evidence/TASK-616/local-diagnostic.json):
macos-platform 4단계 PASS 1m8.286초. SOPS 3.13.1/3.13.3, age 1.3.1.
첫 준비의 TUF 네트워크 실패와 재시도 성공을 구분한다. 최종 검증은 별도다.

## Delivery

전역 pin·설치본과 기존 checksum/Sigstore 검증을 보존한다. 명시 override는 누락을
skip/PASS로 바꾸지 않는다. self-hosted runner·자동 hook·schedule은 추가하지 않는다.
독립 source/done 검토와 확정된 커밋의 dev/full·로컬 macOS·원격 Linux CI 뒤
CE 통합·push·설치·회수를 완료한다. 최종 실행 결과는 immutable DVA/CE receipt와
원격 run에서 확인하며 검토와 실제 실행을 구분한다.
