---
id: TASK-513
title: "Published commit subject history handling"
type: decision
priority: P2
effort: XS
needs-human: true
execution-mode: decision
human-grade: human
status: Accepted
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

2026-10-14에 사용자가 옵션 1을 수락했다. 공개된 다음 네 immutable object에만
Git `%s`와 함께 exact-SHA 역사 처리를 적용한다. 이 수락은 baseline 이동, 72자
상한 변경, 첫 물리 줄만 검사, 일반 waiver 확대를 승인하지 않는다.

- `9c40d6ef992b3cba8507402ce00ff8680e9cf9af` — `docs(tasks): file ISSUE-506 applyenv test environment isolation - TestApplyEnv_MergesVars fails on any machine exporting DB_HOST because   MergeVars deliberately gives os.LookupEnv priority and the fixture never   isolates the process environment; file the test-fixture defect with the   baseline reproduction from the ISSUE-505 gate run.`
- `1c47851a21b53655d1ee437f63efe5f8c05cfd92` — `docs(tasks): drop the pre-archive copy of ISSUE-505 - The archive commit added tasks/_archive/issue/505-... without staging   the deletion of tasks/issue/505-..., leaving the id declared twice and   failing the ce task gate's validate step.`
- `5333e5c284eb9449b8dc13a3114aa467def84614` — `docs(tasks): archive fixed ISSUE-505 - Mark the suggestion_ignore shadowing issue resolved by c0179f06 and   move the card to tasks/_archive/issue/ as history.`
- `c0179f06b2d560dd807df71177e5b9982de43e19` — `fix(cli): report suggestion_ignore entries shadowed by interactions - ISSUE-505: an ignore pattern whose every matched name an interaction   already covers (command set, subcommand coverage, wrapped command)   suppresses nothing today, yet the stale check never fires for it because   it still matches something. - Track per-pattern matches and which of them reach the suggestion pass   while the suggestion_ignore filter runs, then emit a warning-only "redundant ignore" line naming the covered names and an ignore_redundant   row in the JSON report. - Patterns matching nothing stay with the stale check; patterns still   hiding an uncovered name stand down, so globs keep their legitimate use. - Cover the judgment rule with five tests, including the e2e printed line.`

이유는 네 커밋이 모두 제목·본문 사이 blank separator를 누락해 Git이 첫 문단 전체를
subject로 해석한 공개 object이기 때문이다. 공유 이력을 재작성하지 않으면서 미래 위반은
계속 실패시키기 위해 SHA와 실제 subject를 함께 고정한다. TASK-506의 hook 재발 방지는
이 수락과 독립적으로 유지한다.

## Rationale

옵션 1은 공유 이력을 보존하면서 알려진 불변 object와 미래 위반을 구분할 수 있다.
다만 정책 소유자가 일반 waiver growth로 판단하면 옵션 2를 유지한다. baseline 이동,
첫 줄만 검사하기, 임의 상한 증가는 옵션이 아니다.

## Consequences

선택 전에는 ISSUE-512가 human-only로 남으며 동일 full CI를 자동 재시도하지 않는다.
신규 task commit 뒤 history 검사에 같은 네 object만 남는지 확인해 회귀와 구분한다.

## Resolution Criteria

- [x] 사용자가 2026-10-14에 옵션 1을 수락했고, 네 literal SHA·실제 Git `%s`·공개 이력 보존 이유를 기록했다. | verify: human — explicit option 1 acceptance on 2026-10-14; only the four listed SHA+subject pairs are adopted

## References

- [ISSUE-512](../_archive/issue/512-published-commit-subject-length-baseline.md)
- [기존 역사 처리 정책](../_archive/2026-09/278-gate-commit-subjects-before-they-land.md)
