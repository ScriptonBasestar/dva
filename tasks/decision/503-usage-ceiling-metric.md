---
id: TASK-503
title: "Define the USAGE.md ceiling metric and act on it"
type: decision
priority: P3
effort: S
needs-human: true
execution-mode: decision
human-grade: human
status: Proposed
created: 2026-10-10
---

## Summary

USAGE.md 최상단 마커의 상한 `143360 bytes, 1600 lines`는 무엇을 셀지 정의하지 않는다.
측정 방법에 따라 이미 넘었다. 마커는 "여기에 닿으면 분할 근거가 생긴 것"이라고 적는다.
그래서 측정 기준과 대응을 사람이 정해야 한다.

## Context

2026-10-10 측정 결과다.

| 측정 | 값 | 상한 대비 |
|---|---|---|
| 바이트 (`wc -c`) | 133319 | 아래 |
| 물리 줄 (`wc -l`) | 2454 | 초과 |
| 펜스 밖 줄 (빈 줄 포함) | 1697 | 초과 |
| 펜스 밖 비지 않은 줄 | 1271 | 아래 |

- 마커는 도입 커밋 `1212fd26`(2026-09-13, TASK-356)에서 당시 값을 "122KB/1326 prose
  lines"로 적었다. 그 커밋의 파일을 같은 방법으로 재면 1586(펜스 밖)과 1181(비지 않은
  줄)이 나온다. 어느 방법도 1326을 재현하지 않는다.
- `make doc-check`는 USAGE.md를 크기 검사에서 면제한다(AGENTS.md 표). 그래서 이 상한을
  집행하는 기계 게이트가 없다.
- 같은 형식의 CHANGELOG 상한은 [TASK-487](487-changelog-half-ceiling-trigger.md)에서
  물리 줄로 정해졌다.

## Decision

- Status: Proposed

선택지:

1. **물리 줄 기준 채택 + 분할**: CHANGELOG와 같은 측정을 쓴다. 이미 초과했으므로 분할
   계획 카드를 연다. 마커의 근거(사용법 축을 늘리지 않는다)와 충돌하므로 분할 경계를
   함께 정한다.
2. **물리 줄 기준 채택 + 상한 재설정**: 현재치에 여유를 더해 상한을 다시 적는다.
   근거를 마커에 남긴다.
3. **비지 않은 펜스 밖 줄 기준 명시**: 지금은 아래다. 측정 명령을 마커에 적어 재현
   가능하게 한다.

## Rationale

검토자 권고는 2 또는 3이다. 마커가 이미 "한 파일이 의도된 설계"라고 적고 있어서
분할은 그 결정을 뒤집는다. 어느 쪽이든 측정 명령이 없으면 같은 모호함이 반복된다.
문서 구조 결정이라 사람이 정한다.

## Consequences

- 어느 선택이든 마커 문장에 측정 명령을 함께 적는다. 다음 검토자가 재현할 수 있어야 한다.
- 1은 USAGE.md를 가리키는 링크와 카드 verify 바인딩을 다시 확인하는 작업을 낳는다.

## Resolution Criteria

- [ ] 사람이 측정 기준과 대응(분할, 상한 재설정, 기준 명시)을 고르고 이 카드에 적는다 | verify: human — choice recorded on this card before the USAGE.md marker is edited
