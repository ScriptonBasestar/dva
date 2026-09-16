---
id: ISSUE-031
title: "List-form quality-review-evidence is misreported as missing by the review-stage validator"
type: bug
status: todo
priority: P2
severity: low
ownership: upstream
created: 2026-09-17
discovered-at: 2026-09-17
discovered-in: "ISSUE-029 수리 중 — done 스키마 검증기가 리스트형 evidence를 '필드 부재'로 오판"
upstream-ref: "ce-agent-kit#9"
---

## Summary

done 카드 검증이 `quality-review-evidence`를 읽을 때 스칼라 전용 `scalarField`를
쓰는데, 같은 코드베이스의 `fieldHasNonEmptyValue`는 시퀀스도 읽는다. 그 결과
evidence를 YAML 블록 리스트로 쓴 카드는 **내용이 분명히 있는데도** "accepted
quality-review verdict requires quality-review-evidence" 오류를 받는다 — 오류
메시지는 원인(형식 미지원)이 아니라 "필드가 없다"고 거짓 보고한다.

## Evidence

2026-09-16 실제 사건(dva 보드): master의 done 카드 124장 중 정본 다이얼렉트 73장은
한 줄 스칼라 evidence를 쓰고, 5장(348·379·396·397·401)만 같은 내용을 블록 리스트로
썼다. 이 5장이 전부 "evidence 부재"로 invalid가 됐고, 게이트가 red가 돼 모든
통합이 블록됐다.

소스 위치(ce-agent-kit):

- `internal/usecase/task/canonical_validator_fields.go:66` `scalarField` —
  `yaml.ScalarNode`만 읽고 시퀀스는 `("", true)` 반환
- `internal/usecase/task/canonical_validator_fields.go:46`
  `fieldHasNonEmptyValue` — 시퀀스를 포함해 비어있지 않은 값을 검출(리뷰
  스테이지는 이걸 안 씀)
- `internal/usecase/task/validator_review_stage.go` — evidence 요구에
  `scalarField` 사용

## Reproduction

1. done 카드에 리스트형 evidence를 둔다:

   ```yaml
   status: done
   quality-review: pass
   quality-review-evidence:
     - 2026-09-15 리뷰에서 승인
     - 근거: 문서 게이트 exit 0
   ```

2. `ce task validate`를 돌린다.
3. 실제 출력: `❌ accepted quality-review verdict requires
   quality-review-evidence` — evidence가 있는데도 "필드가 없다"고 나온다.

## Expected vs Actual

| | |
|---|---|
| 기대 | 리스트형을 만나면 형식을 정직하게 보고한다(예: "quality-review-evidence must be a single-line scalar") — 또는 리스트형을 아예 수용한다 |
| 실제 | "필드가 없다"는 거짓 보고로, 사용자를 존재하지 않는 결핍의 수리로 유도한다 |

## Impact

단독으로는 혼란이지만, 게이트 red 사건(이 저장소 [[ISSUE-029]])에서는 5장의
오류 메시지가 원인 규명을 한 단계 더 돌렸다 — "필드가 없다"는 말을 그대로 믿으면
evidence를 새로 쓰는 방향으로 헤매고, 실제 수리(스칼라 변환)에 도달하려면 소스를
읽어야 한다. 같은 실수를 할 다음 사용자를 위한 보고다.

## 소유권 — 상류다

`ce-agent-kit#9`로 보고했다(2026-09-17, 소스 위치·재현·수정 제안 2건 포함).
이 저장소는 사건 서사를 제공한다. [[ISSUE-030]](`ce-agent-kit#8`)과 같은
검증기 신뢰성 계열이지만 결함 대상이 다르다 — #8은 판정의 출처 기록, 이 카드는
판정의 내용 정확성이다. 실제 수리는 보드 쪽 5장을 스칼라로 변환해 끝냈고(내용
불변, 5f184c7), 이 카드는 같은 실수의 재발 방지를 추적한다.

## Resolution Criteria

- [ ] 리스트형 `quality-review-evidence`가 정직하게 보고되거나 수용된다 | verify:
      human — 상류 반영 후 이 저장소에서 리스트형 evidence 카드의 validate 출력이
      "필드 부재"가 아닌지 확인한다
