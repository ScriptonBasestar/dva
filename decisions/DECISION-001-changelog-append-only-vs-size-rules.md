---
id: DECISION-001
title: "CHANGELOG.md — append-only 성장과 크기 규칙의 충돌 처리"
type: decision
priority: P2
status: Accepted
created: 2026-10-02
---

## Context

`ce validate filesize`가 CHANGELOG.md를 High(P1)로 보고한다: 66,863 bytes
(한도 46,080), 706 prose lines(한도 500). markdown 규칙의 `require_split`
조치를 그대로 따를 수 없는 유일한 부류다 — changelog는 append-only로
설계상 무한히 자라며, 임의 위치에서 자르면 히스토리가 아닌 미래가 바뀐다.
`tools/doccheck`의 크기 규칙은 docs/·workflows/만 커버하므로 루트
CHANGELOG에는 CE 규칙만 적용된다. 면제 메커니즘은 존재한다
(`size-limit: exempt, ceiling: ...` 마커, 사유 병기 — USAGE.md TASK-356 선례).

선택지:

- **A. exempt 마커 + ceiling + ceiling 도달 시 릴리스 아카이브** — 즉시
  소음 제거, 성장 수렴 장치 내장(무한 면제 아님), split-not-exempt 철학과
  정합. ceiling이 닿으면 그때 분할(아카이브) 근거가 생긴다.
- **B. 지금 릴리스 아카이브** — v0.2.x 이하를 `docs/changelog/` 등으로
  이동. 마커 불필요하지만 히스토리 경로가 쪼개지고(외부 링크/인용 깨짐)
  `tools/changelogcheck`의 `## [Unreleased]` 단일 파일 가정 재확인 필요.
  아카이브 파일도 500 prose lines를 다시 넘길 수 있다.
- **C. 소음 수용** — 무작업이지만 편집 시마다 P1 차단 피드백으로 게이트
  신뢰가 하락한다.

## Decision

(제안 — 인간 확정 대기) **옵션 A**: 파일 최상단에
`<!-- size-limit: exempt, ceiling: 87296 bytes, 1000 lines --...-->`
마커를 선언한다. ceiling은 현재치(67KB/710 lines)에 ~30% 여유를 더한 값.
ceiling 절반에 도달하는 릴리스에서 옵션 B(아카이브)를 실행하는 2단계.

## Rationale

A가 유일하게 "철학을 지키면서 성장을 수렴"시킨다. USAGE.md(TASK-356)가
만든 선례와 동일 구조다: 면제는 설계 의도의 기록이고 ceiling은 진짜
상한이며, 닿는 순간 분할 근거가 생긴다. B는 지금 비용을 치르고 경로를
깨는 반면, 그 필요성은 ceiling이 닿기 전까지는 입증되지 않는다.

## Consequences

- 확정 시: 마커 1줄 추가(별도 TASK 아님 — 마커+사유 자체가 기록).
- ceiling 절반 도달 시: 릴리스 아카이브 TASK 생성(v0.2.x 이하 이동).
- `tools/changelogcheck`는 Unreleased 섹션만 보므로 영향 없음(마커 추가로
  확인).
