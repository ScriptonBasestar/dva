---
id: TASK-619
title: "CI 테스트 요약과 린트 도구별 실행 메트릭 수집"
type: feature
priority: P1
effort: M
exec-tier: strong
status: done
quality-review: pass
quality-review-evidence: tasks/done/evidence/TASK-619/review.txt
created: 2026-11-17
---

## Summary

사용자가 승인한 [수집안](../../docs/ci-metrics-proposal.md)을 DVA 자체 검사에 적용한다.
기존 hosted JSONL을 요약하고 같은 파서를 로컬 실제 테스트에 연결한다.
린트는 기존 DVA 단계 기록을 활용하며 수집 실패와 검사 실패를 구분한다.

## Completion Criteria

- [x] 완료·미완료·병렬·cache 힌트·잘린 JSON을 구분하고 원문 민감값을 저장하지 않는다 | verify: human — 파서 fixture와 저장 JSON을 독립 대조; source에 수집 패키지 없음 (observed: 2026-11-17)
- [x] 수집 실패·상한·취소에서도 quiet 노출 수준·검사 종료·자식 회수를 보존하고 제한된 실패 진단을 명시한다 | verify: human — 실제 subprocess·외부 supervisor 취소·고장 저장소·passing log 비노출 fixture 대조 (observed: 2026-11-17)
- [x] 새 기록만 private namespace에서 제한·회수하고 활성 실행과 crash 잔재를 구분한다 | verify: human — 동시 실행·lock·symlink·보존 한도 fixture 대조 (observed: 2026-11-17)
- [x] commit/dev/full/audit 및 Make·hosted에 연결하며 기존 선택·예산·검사를 유지한다 | verify: human — profile dependency·검사 인자·lint 실패·workflow 계약 fixture와 실제 수집기 진단 대조; 확정 head의 profile gate는 별도 (observed: 2026-11-17)
- [x] 실행 조건·오버헤드와 조회·보존·제한을 문서화하고 수집 자체를 새 필수 gate로 만들지 않는다 | verify: human — 실제 진단 기록과 독립 source/done 검토 (observed: 2026-11-17)

## Verification

대상 다섯 패키지의 `go test -race -count=1 -timeout 90s`와 scoped golangci·gopls,
문서 검사를 실행했다. [원문 진단](evidence/TASK-619/diagnostic.json)은 준비된
캐시의 작은 agentdeny 사례를 기존 quiet 명령과 수집 명령으로 교대 관측한다.
일반 반복의 외부 wall은 각각 0.649~0.716초, 0.806~0.918초였고 수집 네 실행 모두 complete다.
이 차이를 모든 CI의 비용이나 p95로 일반화하지 않는다. 측정 뒤에는 Windows 빌드
분석을 위한 테스트 helper 배치만 바뀌었으며 runtime 코드 지문은 같다.

첫 두 독립 source 리뷰의 FAIL과 수정 후 PASS 원문은 같은 evidence 디렉터리에
보존한다. 기존 TASK-617 timeout의 원인을 해결했다고 주장하지 않는다.
확정 head의 staged fast 실측·dev/full·Mac·hosted gate와 전달은 아래 별도 절차다.
진단 JSON의 planned는 작성 시점의 후속 검증 계획이며, 최종 실행 여부는 해당
DVA/CE receipt가 소유한다. 이 과거 계획 필드를 미완료 실행의 상태표로 사용하지 않는다.

## Delivery

독립 source/done 검토 후 확정된 커밋의 dev/full·원격 CI를 실행한다.
필요한 macOS 검사는 이 Mac에서 수행하고 원격 macOS runner를 가정하지 않는다.
최종 CE 통합·push·설치·회수 결과는 실행 receipt로 남긴다.
