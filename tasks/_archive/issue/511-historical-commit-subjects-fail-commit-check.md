---
id: ISSUE-511
title: "Historical commit subjects fail commit-check"
type: bug
priority: P1
exec-tier: standard
status: done
severity: medium
ownership: local
discovered-in: "Historical commit subjects fail commit-check"
discovered-at: 2026-10-14
created: 2026-10-14
resolution: duplicate
resolved-at: 2026-10-14T05:31:09Z
resolution-summary: "Resolved as duplicate."
---

## Summary

`make commit-check`가 기존 공개 커밋 네 개의 subject 길이 위반 때문에 실패한다.
Rust worker 작업에서 관측했으나 해당 작업의 변경과 무관한 저장소 이력 결함이다.
이력 재작성이나 예외 추가는 이번 작업 범위에 포함하지 않는다.

## 소유권 — 이 저장소다

DVA 저장소가 이 커밋 이력과 `tools/commitcheck`의 정책을 소유한다.
외부 CLI 설치 문제가 아니라 이 저장소의 공개 이력과 자체 게이트 사이의 불일치다.

## Reproduction

저장소에서 `make commit-check`를 실행한다. 작업 기준 HEAD
`1d96c4a34282629bb675d3eb67346ea95567185a`에서 exit 2이며,
commitcheck가 다음 기존 커밋을 보고한다.

| Commit | Subject length | Limit |
|---|---:|---:|
| `9c40d6ef` | 338 | 72 |
| `1c47851a` | 240 | 72 |
| `5333e5c2` | 159 | 72 |
| `c0179f06` | 773 | 72 |

## Expected vs Actual

- Expected: 기존 이력에 대한 저장소 선언 게이트가 통과한다.
- Actual: 네 개 위반을 출력하고 실패한다. 새 worker 변경을 추가하기 전 HEAD 이력만 검사한 결과다.

## Verification evidence

2026-10-14: `make recovery-check`, `make check-generate`, 통합 테스트와 built-binary
skill dogfood는 모두 exit 0. `make commit-check`는 위 네 이력 위반만 보고했다.
현재 source tip `1bd93f692f901a0abe4edd6068690dadc0e2dae0`에서도 동일한 네 위반과
exit 2를 확인했다. 새 worker 변경이 baseline을 악화시키지 않는다.

## 중복 정리

최신 source의 [ISSUE-512](512-published-commit-subject-length-baseline.md)가 같은 네
공개 object와 사람 결정 513을 추적한다. 이 카드는 중복 발견의 이력으로 보관하며,
후속 처리와 정책 결정은 ISSUE-512를 정본으로 따른다. 이 작업 자체는 waiver나
이력을 변경하지 않는다.
