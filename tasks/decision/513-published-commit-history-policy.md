---
id: TASK-513
title: "Published commit subject history handling"
type: decision
priority: P2
effort: XS
needs-human: true
execution-mode: decision
human-grade: human
status: Proposed
created: 2026-10-14
---

## Summary

ISSUE-512의 공개된 4개 malformed commit subject를 처리하는 정책 선택이다.
TASK-505와 TASK-506의 자동 source 개선은 이 선택을 기다릴 필요가 없으며,
기존 실패가 악화되지 않았음을 확인하고 실제 baseline 실패를 보고한다.

## Context

tools/commitcheck는 Git `%s` 문단 subject와 72자 상한을 소유한다. 네 commit은
제목·본문 사이 빈 줄이 없어 첫 문단이 긴 subject가 됐다. working HEAD 1bd93f69
에서 이미 발생했고 전체 source 검사에는 실패가 없다. 기존 exact-SHA 역사 처리
정책은 TASK-278에 있으나 일반적인 waiver 확대나 baseline 이동은 허용하지 않는다.

기준은 DVA SOUL.md §4의 실패를 성공으로 바꾸지 않는 원칙과 설치 글로벌 SOUL의
Preserve user work and trust다. 공개된 공유 이력의 재작성·정책 예외는 사람 판단이다.

## Options

1. **권장: 공개된 4개 literal SHA에 한정한 역사 처리의 채택 여부를 결정한다.**
   기존 exact-SHA 정책의 소유자에게 네 object와 원 실패를 보존하는 기록을 승인받는다.
   새 commit-msg 문단 검사는 유지하고 미래 위반은 계속 실패한다. 적용 위치는
   tools/commitcheck의 역사 처리 목록이며 비용은 작은 policy 변경과 negative regression이다.
   공유 이력은 보존되고 정책 변경은 되돌릴 수 있다. 현재 gate를 임의로 완화하는 승인으로
   해석하지 않는다. 승인된 literal SHA와 사유·날짜를 별도 owner task에서 구현한다.
2. **현 정책을 유지한다.**
   소스는 바꾸지 않고 full CI의 기존 4건 실패를 계속 보고한다. 비용은 없고 가역적이며,
   source 개선은 nonworsening 증거로 구분하되 전체 full CI가 green이라고 주장할 수 없다.
3. **공개 공유 이력을 사람이 재작성한다.**
   이력 object와 downstream 참조가 바뀌므로 비용·협업 위험이 가장 크고 되돌리기 어렵다.
   에이전트의 허용된 task-branch force-with-lease 범위 밖이다. 자동 실행하지 않는다.

## Decision

미결정. 승인 주체·날짜·literal SHA·이유를 기록하기 전에는 이력 정책을 변경하지 않는다.
TASK-506의 hook 재발 방지와 TASK-505의 source 전달은 독립적으로 진행한다.

## Rationale

옵션 1은 공유 이력을 보존하면서 알려진 불변 object와 미래 위반을 구분할 수 있다.
다만 정책 소유자가 일반 waiver growth로 판단하면 옵션 2를 유지한다. baseline 이동,
첫 줄만 검사하기, 임의 상한 증가는 옵션이 아니다.

## Consequences

선택 전에는 ISSUE-512가 human-only로 남으며 동일 full CI를 자동 재시도하지 않는다.
신규 task commit 뒤 history 검사에 같은 네 object만 남는지 확인해 회귀와 구분한다.

## Resolution Criteria

- [ ] 승인 주체·날짜·대상 literal SHA·선택과 사유를 기록한다. | verify: human — exact historical handling or policy retention explicitly adopted

## References

- [ISSUE-512](../issue/512-published-commit-subject-length-baseline.md)
- [기존 역사 처리 정책](../_archive/2026-09/278-gate-commit-subjects-before-they-land.md)
