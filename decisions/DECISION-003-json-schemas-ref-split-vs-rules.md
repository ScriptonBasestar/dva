---
id: DECISION-003
title: "JSON 스키마 4종 — $ref 분할 vs 규칙 kind vs 수용"
type: decision
priority: P3
status: Accepted
created: 2026-10-02
---

## Context

`internal/config/schema.json`(1,882L — config kind 200L의 9배),
`agent-mesh-flows/schemas/flow.schema.json`(742L), `step.schema.json`(380L),
`run.schema.json`(220L). JSON은 주석 문법이 없어 per-file exempt 마커가
불가능하다(DECISION-002와 같은 원리적 한계). schema.json은 생성기 없는
손수 관리 파일이며 `# yaml-language-server: $schema=../internal/config/schema.json`
헤더와 `tools/yamlcheck`, 여러 카드 바인딩이 이 경로를 직접 가리킨다.

선택지:

- **A. $ref 분할** — definitions를 파일群으로 나누고 상대 `$ref`로 연결.
  규칙 준수 회귀, 섹션별 소유 명확화. 그러나 Go 검증 로더가
  `gojsonschema.NewBytesLoader`(internal/config/validate.go:116)인데
  바이트 로더에는 상대 참조 resolution 기저가 없다 — `NewReferenceLoader`
  + file:// 기저 교체 공사가 전제되고, yaml-language-server와
  gojsonschema의 $ref 해석 차이 실측이 필요하다.
- **B. 규칙 레벨 — 스키마 kind 추가** — DECISION-002와 같은 차량
  (upstream file-size.yaml)에 스키마용 kind(JSON Schema 파일 500/1000
  lines)를 추가. 도구 체인 무위험. 스키마는 참조 데이터라 "쪼개서 읽는
  문서"의 상징성이 약하다.
- **C. 소음 수용** — schema.json은 dva.yml 스키마 작업마다 편집되는
  살아있는 파일이라 매번 P1 피드백은 비현실적.

## Decision

2026-10-02에 수락된 역사적 결정이다. **옵션 B**가 채택됐다. 스키마 kind를
upstream 규칙에 둔다. 로더를 `NewReferenceLoader`로 바꾸는 공사는 이 결정이 아니다.

`json_schema`는 정본 ce-agent-kit `1270e1dc47bc7f3a2421de2074b92f619e4298a7`에
구현되어 있다. 예산 정본은 [ADR-0070](https://gitlab.polypia.net/archmagece/ce-agent-kit/-/blob/1270e1dc47bc7f3a2421de2074b92f619e4298a7/decisions/adr/0070-artifact-size-kinds.md)이다.
이 세션은 그 소스 검사를 다시 실행하지 않았다. 저장된 증거는
[adoption.json](../tasks/done/evidence/TASK-484/adoption.json)과
[docs/70](../docs/70-generated-artifact-upstream-report.md)이다.

`internal/config/schema.json`은 1882줄이다. 저장된 채택 측정은 이 파일에
expectedExit 1, errorLimit 1000이다. 1000줄 오류를 넘는 상태는 알려진
기준선이다. 로더 리팩터의 이유가 아니다.

## Rationale

A의 실질 비용은 스키마 파일 분할이 아니라 로더 교체 공사다 — 바이트
로더에서 참조 로더로의 전환은 스키마 해석을 URL 세계로 이동시키고, 그
실측 없이는 회귀를 보장할 수 없다. 스키마의 분해 필요성(섹션별 소유)이
입증되는 시점에 로더 교체 TASK와 묶어 별도 진행하는 것이 순서다.

## Consequences

- kind는 정본 구현에 있다. 이 결정이 로더를 바꾸지 않는다.
- schema.json 1882줄과 error 1000은 알려진 기준선으로 남는다.
- 옵션 A는 나중 평가다. 이 문서가 그 공사를 열지 않는다.
