---
id: TASK-619
title: "호스트 전역 무거운 CI 단계 직렬화 슬롯"
type: feature
priority: P1
effort: M
exec-tier: strong
status: review
created: 2026-11-17
---

## Summary

여러 에이전트 세션이 서로 다른 저장소·워크트리에서 같은 호스트의 `dva ci`를 실행할
때, 무거운 실행이 서로 겹쳐 commit 게이트를 밀어내는 문제를 단계 단위 옵트인
`host_slot` 마커로 해소한다. 표시한 native run 단계는 실행 내내 호스트 전체
advisory 슬롯(CI 상태 루트의 고정 `locks/host-slot.lock`)을 잡고, 점유 중이면 점유자
신원을 밝히며 기다린다. 표시하지 않은 단계·프로필의 동작은 바뀌지 않는다.

## Scope

- 설정 키 `steps[].host_slot`(boolean). native run 전용이며 `oci`와 공존하면
  설정 검증·내장 스키마·직렬화 세 계층에서 거절한다. manifest SchemaVersion 1.12.
- 타임아웃 회계: 슬롯 대기는 단계 `timeout` 예산에서 제외된다
  (`runConditionedStep`이 예산 산정 전 문맥을 보관하고 `runStep`이 그 문맥으로
  획득). profile `timeout`과 외부 취소만 대기를 제한하며 어떤 timeout 값도
  올리지 않는다.
- 재사용 hit과 조건 거짓은 슬롯을 잡지 않는다. 조상이 슬롯을 잡은 구성 child의
  재선언은 교착 대신 즉시 실패한다.
- 메커니즘 결정 기록은 [decisions/619](../../decisions/619-host-slot-serialization.md),
  사용법은 [dva-ci 설정 참조](../../skills/dva-ci/references/configuration.md)에 둔다.
  `docs/53-ci-profiles.md`는 크기 상한 17바이트 이내라 문단을 넣지 않았다.

## Out of Scope

- 이 저장소 자체 프로필에 `host_slot` 채택 여부 — 저장소별 제품 결정이다.
- `dva` 밖 프로세스의 통제. 협력적 직렬화다.

## Completion Criteria

- [x] 표시한 native 단계만 호스트 슬롯을 잡고, 표시 없는 단계는 점유 중에도 영향 없이 실행한다 | verify: `go test ./internal/cirun -run TestHostSlot -count=1` (observed: 2026-11-17)
- [x] 슬롯 대기가 단계 timeout 예산에서 제외된다 | verify: `go test ./internal/cirun -run TestHostSlotWaitExcludedFromStepTimeout -count=1` (observed: 2026-11-17)
- [x] 재사용 경로에서도 대기가 예산에서 제외되고 실행 창 예산은 여전히 강제한다 | verify: `go test ./internal/cirun -run TestHostSlotReuse -count=1` (observed: 2026-11-17)
- [x] host_slot은 OCI 단계 공존과 비 boolean 값을 거절한다 | verify: `go test ./internal/config -run HostSlot -count=1` (observed: 2026-11-17)

## Verification

실제 보조 프로세스가 진짜 슬롯 파일을 잡는 프로세스 수준 테스트
(internal/cirun/host_slot_test.go, 9개)가 녹색이다. 변이 확인(green-both-ways):

- `cirun_steps.go`의 획득 게이트를 `step.HostSlot && false`로 무력화하면
  `TestHostSlotStepWaitsForRealHolderProcess`가 "slot step executed while another
  process held the host slot"로 실패했다(2026-11-17).
- 예산 산정 전 문맥 대신 산정된 문맥으로 대기하게 하면(원본 origin 무시 변이)
  `TestHostSlotWaitExcludedFromStepTimeout`가 "context deadline exceeded"(1.006s)로
  실패했다(2026-11-17). 이 테스트는 조건 단계로 wrapper deadline이 대기 위에
  걸리는 형태를 쓴다.
- 독립 리뷰가 재사용 경로의 예산 제외 누락을 지목했다(a8be9ca0). 재현 테스트
  `TestHostSlotReuseWaitExcludedFromStepTimeout`를 수정 전 코드에서 먼저 실행해
  wait 1.924s·실제 점유 28ms인 정상 직렬 실행이 "timed_out"으로 오분류되는 것을
  확인했고(2026-11-17), `ci_reuse_step.go`가 대기 후 검증 창을 pre-budget 문맥에서
  재산정하는 수정 뒤 통과했다. `TestHostSlotReuseExecutionDeadlineStillBites`는
  대기 제외가 예산 강제를 비활성화하지 않음을 같은 경로에서 확인한다.

모든 변이 복원 후 전체 `TestHostSlot` 슈트가 통과했다.

## Delivery

구현·문서·이 카드를 같은 태스크 브랜치에서 커밋하고 `make test`, `make build`,
`make lint`, `make doc-check`, `ce task gate --bindings-since` 통과 뒤 push한다.
독립 리뷰는 별도 세션에서 수행한다.
