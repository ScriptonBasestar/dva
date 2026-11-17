---
id: TASK-619
title: "Host-wide heavy-step serialization slot"
type: decision
priority: P2
effort: S
execution-mode: implementation
status: Accepted
created: 2026-11-17
---

## Summary

여러 에이전트 세션이 같은 20-CPU 호스트의 서로 다른 저장소·워크트리에서 `dva ci`를
실행할 때 무거운 실행(full, 약 8-17분)이 서로 겹쳐 commit 게이트까지 밀려나는
문제를, 단계 단위 옵트인 `host_slot` 마커와 호스트 전체 advisory 잠금으로 해소하는
결정이다.

## Context

기존 CI 잠금(internal/cirun/locks.go)은 root별·선언 자원별 배타 잠금이고 충돌하면
기다리지 않고 `busy`로 즉시 실패한다. 이 규칙은 같은 root의 중복 실행을 막는 데는
맞지만, 서로 다른 root의 무거운 실행이 같은 호스트 CPU·메모리를 두고 겹치는 것은
통제하지 못한다. 실제로 무거운 실행 겹침으로 commit 600s timeout이 7회 초과했다.

대안이었던 "부하가 낮아질 때까지 기다리는 외부 wrapper"는 기각됐다. 슬롯이 dva의
first-class 설정이어야 하고, wrapper는 단계 타임아웃 회계를 조작할 수 없기 때문이다.

## Options

1. **채택: 고정 경로 호스트 슬롯 파일 + 단계 단위 옵트인 마커.** native run 단계에
   `host_slot: true`를 선언하면 실행 내내 CI 상태 루트의 단일 슬롯 파일을 잡고,
   점유 중이면 대기한다. 표시하지 않은 단계·프로필은 어떤 동작도 바뀌지 않는다.
2. **현행 유지(busy 규칙만).** 무거운 실행 겹침은 그대로 두고 사용자가 수동으로
   조율한다. 문제가 해결되지 않는다.
3. **외부 wrapper로 대기 후 실행.** first-class 요구사항 위반이고 단계 예산 회계를
   정확히 할 수 없어 기각됐다.

## Decision

2026-11-17 구현으로 확정했다.

- 설정 키는 `steps[].host_slot`(boolean, 기본 생략). native run 단계 전용이며
  `oci`와 함께 선언하면 설정 검증·스키마·직렬화 세 계층에서 거절한다. 조건 단계와는
  함께 쓸 수 있다(조건이 참일 때만 대기·획득).
- 잠금 파일은 CI 상태 루트의 `locks/host-slot.lock` 하나로 고정이다. root 해시가
  없으므로 호스트의 모든 `dva` 실행 — 저장소·워크트리·세션 무관 — 이 같은 파일로
  직렬화된다. flock은 기존 root/자원 잠금과 같은 build-tagged helper를 쓴다.
- 점유 중이면 100ms 폴로 대기하며, 점유 run ID·PID·profile·단계·root·점유 시각을
  로그로 밝히고 획득·해제 시각에 `host_slot=acquired waited=…`,
  `host_slot=released held=…`를 남긴다. 슬롯 잠금 파일에는 점유 metadata JSON을
  기록한다.
- **타임아웃 회계**: `runConditionedStep`이 단계 예산 산정 전 문맥을
  `hostSlotOriginKey`로 보관하고, `runStep`이 예산 산정(`context.WithTimeout`) 이전에
  그 문맥으로 슬롯을 획득한다. 대기는 단계 `timeout` 예산에서 제외되고, profile
  `timeout`과 외부 취소만 대기를 계속 제한한다. 어떤 timeout 값도 올리지 않는다.
  context deadline은 불변이므로 profile deadline을 재무장하는 대안은 timeout 인상과
  같아 금지된다. 예산 창 규칙(2026-11-17 독립 리뷰 반영): `timeout` 창은 (1) 대기
  전 탐사 — 조건 평가, 재사용 wrapper가 `ci_reuse_step.go`에서 산정한 첫 창 —
  (2) 대기 후 실행 — `runStep`이 pre-budget 문맥에서 다시 산정 — (3) 재사용 경로의
  대기 후 검증 — 같은 값으로 재산정 — 에 각각 적용된다. 재사용 wrapper의 첫 창이
  대기 중 만료되어도 실행 후 검사는 재산정된 창을 읽으므로 정상 직렬 실행이
  timed_out으로 오분류되지 않고, 실행 자체가 예산을 넘으면 여전히 실패한다.
- 슬롯 파일의 점유 metadata는 기록 길이가 고정이 아니므로 대기자가 파일 크기(상한
  8 KiB)만큼 읽는다. 조상 점유 검사와 대기자 신고가 긴 root 경로·단계 이름에서도
  정확하다.
- 재사용 hit과 조건 거짓은 단계를 실행하지 않으므로 슬롯을 잡지 않는다. 조상이
  슬롯을 잡은 상태에서 구성 child가 다시 `host_slot`을 선언하면 자기 조상을
  기다리는 교착 대신 즉시 실패한다. `child_ci`와 같은 단계에 선언하면 부모가 자식
  실행 내내 슬롯을 잡는데, `child_ci`의 구성 grant가 ancestry에 부모 run을 싣으므로
  같은 검사가 자식의 재선언을 즉시 실패시킨다.

## Rationale

가벼운 충돌에는 즉시 `busy`가 맞지만, 선언된 무거운 단계는 "실패 후 재시도"보다
"순서를 기다린 뒤 예산 안에서 실행"이 낫다. 단일 고정 파일은 모든 실행이 공유해야
하는 자원의 성격과 일치하고, 옵트인 마커는 표시하지 않은 기존 프로필의 관측 가능한
동작을 그대로 둔다.

## Consequences

- `host_slot` 단계의 총 벽시계 시간은 탐사 창 + 대기 + 실행·검증 창까지 각 창 예산의
  합을 넘을 수 있다(비-슬롯 단계는 단일 창). 대기를 예산에서 빼는 정확한 대가다.
- 슬롯 대기가 길어지면 profile `timeout`이 대기째로 실패시킬 수 있다. 이는 예산을
  올리지 않는 대가로 남는 정직한 상한이다.
- 협력적 제어다. `dva` 밖의 프로세스나 슬롯을 선언하지 않은 프로필은 통제하지
  못한다.
- 슬롯을 잡은 실행이 강제 종료되면 flock은 커널이 회수하지만 슬롯 파일의 metadata는
  다음 점유자가 덮어 쓴다.

## Resolution Criteria

- [x] `host_slot` 마커가 단계 예산 산정 이전의 슬롯 획득으로 연결되어 대기가 단계 `timeout`에서 제외된다. | verify: `/usr/bin/grep -q hostSlotOriginKey internal/cirun/cirun_steps.go` (observed: 2026-11-17)

## References

- [CI 프로필과 동시 실행 규칙](../docs/53-ci-profiles.md)
- [dva-ci 설정 참조](../skills/dva-ci/references/configuration.md)
