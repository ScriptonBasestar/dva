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

2026-10-02에 수락된 역사적 결정이다. **옵션 A**가 채택됐다. CHANGELOG.md
최상단 마커와 ceiling `87296 bytes, 1000 lines`가 그 승인이다. ceiling은
당시 약 67KB/710 lines에 약 30% 여유를 더한 값이다.

2026-10-05에 사용자가 [TASK-487](../tasks/decision/487-changelog-half-ceiling-trigger.md) 옵션 2를 골랐다. 아카이브 트리거는 실제 ceiling이다. 릴리스 시점에 바이트가 87296 이상이거나 물리 줄이 1000 이상이면 v0.2.x 이하를 아카이브한다. 이 비교는 문서화된 릴리스 트리거다. CE 파서가 그 비교를 기계적으로 집행한다고 말하지 않는다. 승인된 상한은 마커의 ceiling `87296 bytes, 1000 lines`다.

선택을 열 때의 측정은 67110 bytes, 744 lines였다. 절반은 43648 bytes, 500 lines이고, 그 측정은 절반을 넘고 ceiling은 넘지 않았다. 마커 문장을 실제 ceiling으로 바꾼 뒤 파일은 67170 bytes, 744 physical lines다. 두 상한 아래이므로 이번 변경은 아카이브하지 않는다. 적용 카드는 [TASK-489](../tasks/todo/489-apply-approved-changelog-ceiling.md)다.

## Rationale

A가 유일하게 "철학을 지키면서 성장을 수렴"시킨다. USAGE.md(TASK-356)가
만든 선례와 동일 구조다: 면제는 설계 의도의 기록이고 ceiling은 진짜
상한이며, 닿는 순간 분할 근거가 생긴다. B는 지금 비용을 치르고 경로를
깨는 반면, 그 필요성은 ceiling이 닿기 전까지는 입증되지 않는다.

절반 트리거는 그 30% 여유보다 앞에서 이미 참이었다. 그래서 2026-10-05 옵션 2가 트리거를 승인된 ceiling로 옮긴다. ceiling 숫자는 다시 쓰지 않는다.

## Consequences

- ceiling `87296 bytes, 1000 lines`는 그대로다. 바뀐 것은 아카이브 문장뿐이다.
- 문서화된 트리거는 릴리스에서 바이트 >= 87296 또는 물리 줄 >= 1000이다. 지금 측정은 그 아래라 아카이브하지 않는다.
- `tools/changelogcheck`는 Unreleased 섹션만 본다.
