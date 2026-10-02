---
id: DECISION-004
title: "config kind 200라인 상한 vs 온전해야 하는 파일 2건 (ci.yml, full-stack.yml)"
type: decision
priority: P3
status: Proposed
created: 2026-10-02
---

## Context

`ce validate filesize --all` High 중 2건이 "분할하면 오히려 목적을 해치는"
작은 초과 파일이다.

- **`.github/workflows/ci.yml`(216 물리, 상한 200)** — 16라인 초과. 진짜
  분할(reusable workflows로 lint/test/doc-check 잡 분리)은 16라인을 위해 CI
  동작 변경 리스크를 감수하는 공사다. exempt 마커(YAML 주석)는 사용 가능.
- **`examples/full-stack.yml`(243 물리)** — 예시 파일이다. `modules:` import로
  쪼개면 복사해 쓰는 사용자에게 비자기완결적 예시가 된다. 예시의 존재 이유는
  "온전한 참조"다.

둘 다 DECISION-001(CHANGELOG)·002(생성물)와 같은 모양: "파일을 쪼개라"는
처방이 대상의 목적과 충돌한다.

## Decision

(제안 — 인간 확정 대기) **둘 다 exempt 마커 + ceiling** —
`size-limit: exempt, ceiling: ...` 마커를 파일 상단에 두고, 사유에 위 Context를
한 줄씩 기록한다. DECISION-001이 채택하는 것과 동일 차량. ceiling은 현재치
약 30% 여유(각각 280 / 320).

대안: ci.yml만 reusable-workflow 실분할(리스크 감수 시). full-stack.yml을
modules 예시로 재작성(예시 성격 변경을 감수하는 경우).

## Rationale

16~43라인 초과를 없애는 공사(CI 재구성·예시 재설계)가 상한 위반 그 자체보다
비용이 크다. exempt+ceiling은 사유를 코드에 남기고 성장 상한을 두는 저장소의
기존 메커니즘이다(TASK-356 선례). 다만 이것은 "예시가 계속 자라도 된다"는
허가가 아니라 ceiling까지다.

## Consequences

- 확정 시: 마커 2줄 추가로 High 2건 해소. 별도 TASK 없음.
- ci.yml이 ceiling을 넘는 날: reusable-workflow 분할을 그때 평가.
- full-stack.yml이 ceiling을 넘는 날: 예시 자체를 재검토(무한 수용 아님).
